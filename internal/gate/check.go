// Package gate validates a PR description against the pr-brief convention.
package gate

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/gagoar/pr-brief/internal/body"
	"github.com/gagoar/pr-brief/internal/convention"
	dg "github.com/gagoar/pr-brief/internal/diagram"
	"github.com/gagoar/pr-brief/internal/host"
	"github.com/gagoar/pr-brief/internal/links"
	"github.com/gagoar/pr-brief/internal/theme"
	"github.com/gagoar/pr-brief/internal/tickets"
)

// Finding is one failed check.
type Finding struct {
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

// Result is the outcome of Check.
type Result struct {
	Findings   []Finding `json:"findings"`
	Warnings   []string  `json:"warnings,omitempty"`
	Skipped    bool      `json:"skipped,omitempty"`
	SkipReason string    `json:"skipReason,omitempty"`
	Style      string    `json:"style,omitempty"`
	Theme      string    `json:"theme,omitempty"`
}

// OK reports whether the body passes.
func (r Result) OK() bool { return len(r.Findings) == 0 }

func (r *Result) fail(rule, format string, a ...any) {
	r.Findings = append(r.Findings, Finding{rule, fmt.Sprintf(format, a...)})
}

// LintFunc returns the hard STE100 violations found in prose.
type LintFunc func(prose string) []string

// Options controls Check.
type Options struct {
	// Style overrides the style recorded in the begin marker. Set it when a
	// repo config exists, so an author cannot weaken the gate by editing the marker.
	Style string
	// Theme is the theme the diagrams must use. Set it when a config names one, so an
	// author cannot choose a different look by editing the begin marker. When nil, the
	// built-in theme named in the marker is used.
	Theme *theme.Theme
	Lint  LintFunc
	// Host is where the PR lives: host.GitHub or host.AzureDevOps. It sets the length
	// limit. Empty means GitHub.
	Host string
	// Branch is the PR's branch. When its name holds a Jira or Linear key that the description
	// does not mention, the gate warns. It does not fail: a branch name can fool the match.
	Branch string
}

var (
	skipRe      = regexp.MustCompile(`(?m)^> pr-brief skipped:[ \t]*(.*)$`)
	noDiagramRe = regexp.MustCompile(`<!-- pr-brief:no-diagram:\s*(.*?)\s*-->`)
	sentenceRe  = regexp.MustCompile(`[.!?]+(\s+|$)`)
	sepCellRe   = regexp.MustCompile(`^:?-{1,}:?$`)
	seeFlowRe   = regexp.MustCompile(`(?i)^see flow (\d+)$`)
	refIDRe     = regexp.MustCompile(`^(?:I|O|Input |Output )\d+$`)
	boldRe      = regexp.MustCompile(`\*\*`)
)

// Check validates a whole PR description.
func Check(text string, o Options) Result {
	var res Result
	text = strings.ReplaceAll(text, "\r\n", "\n")

	if n, limit := utf8.RuneCountInString(text), host.Limit(o.Host); n > limit {
		res.fail("length", "description is %d characters; %s rejects more than %d", n, host.Name(o.Host), limit)
	}

	for _, t := range tickets.Missing(text, tickets.FromBranch(o.Branch)) {
		res.Warnings = append(res.Warnings, fmt.Sprintf("the branch %q names %s, but the description does not. Jira and Linear link a PR by that key. Run `pr-brief tickets` to add it.", o.Branch, t.Key))
	}

	if m := skipRe.FindStringSubmatch(text); m != nil {
		reason := strings.TrimSpace(m[1])
		if reason == "" {
			res.fail("skip", "`> pr-brief skipped:` needs a reason after the colon")
			return res
		}
		res.Skipped, res.SkipReason = true, reason
		return res
	}

	m := body.FindMarkers(text)
	if !m.Found {
		if body.HasAnyMarker(text) {
			res.fail("markers", "the begin and end markers are broken; both `<!-- pr-brief:begin v1 style=... -->` and `%s` must be present, in that order", convention.End)
		} else {
			res.fail("markers", "no pr-brief description found; run /pr-brief (or add `%s <reason>` to skip this PR)", convention.SkipPrefix)
		}
		return res
	}
	if m.Version != "1" {
		res.fail("markers", "marker version v%s is not supported (expected v1)", m.Version)
	}
	if !isStyle(m.Style) {
		res.fail("markers", "style %q in the begin marker must be one of %s", m.Style, strings.Join(convention.Styles, ", "))
	}
	style := m.Style
	if o.Style != "" {
		style = o.Style
	}
	res.Style = style

	th := resolveTheme(m, o, &res)
	if th != nil {
		res.Theme = th.ID()
	}

	checkPrevious(text, m, &res)

	managed := text[m.BeginIdx:m.EndIdx]
	lines := strings.Split(managed, "\n")
	secs := splitSections(lines, &res)
	if secs == nil {
		return res
	}

	var prose []string
	checkBrief(secs[convention.Sections[0]], &res, &prose)
	checkChangeMap(secs[convention.Sections[1]], &res, &prose, th)
	checkReviewGuide(secs[convention.Sections[2]], &res, &prose)

	if style == convention.StyleSTE || style == convention.StyleSTEIceberg {
		if o.Lint == nil {
			res.Warnings = append(res.Warnings, "STE100 lint is not available; skipped the prose check")
		} else if len(prose) > 0 {
			for _, v := range o.Lint(strings.Join(prose, "\n\n")) {
				res.fail("ste", "%s", v)
			}
		}
	}
	return res
}

func isStyle(s string) bool {
	for _, v := range convention.Styles {
		if v == s {
			return true
		}
	}
	return false
}

func checkPrevious(text string, m body.Markers, res *Result) {
	if !strings.Contains(text, convention.PreviousOpen) {
		return
	}
	p, ok := body.FindPrevious(text)
	if !ok {
		res.fail("previous", "the `pr-brief:previous` block is malformed (needs the `v1 saved=...` header and the `%s` terminator)", convention.PreviousEnd)
		return
	}
	if p.Start < m.EndIdx {
		res.fail("previous", "the `pr-brief:previous` block must come after `%s`", convention.End)
	}
	if strings.Contains(p.Raw, "--") {
		res.fail("previous", "the `pr-brief:previous` block contains `--`, which would close the HTML comment early; encode it with `pr-brief body`")
	}
}

type fenceState struct {
	in   bool
	lang string
}

// next updates the fence state for a line and reports whether the line is a fence
// marker line (open or close).
func (f *fenceState) next(line string) (marker bool) {
	t := strings.TrimSpace(line)
	if !strings.HasPrefix(t, "```") {
		return false
	}
	if f.in {
		if t == "```" {
			f.in, f.lang = false, ""
			return true
		}
		return false
	}
	f.in, f.lang = true, strings.TrimSpace(strings.TrimPrefix(t, "```"))
	return true
}

func splitSections(lines []string, res *Result) map[string][]string {
	secs := map[string][]string{}
	var order []string
	cur := ""
	var fs fenceState
	for _, l := range lines {
		marker := fs.next(l)
		inside := fs.in || marker
		if !inside && strings.HasPrefix(l, "## ") {
			h := strings.TrimRight(l, " \t")
			cur = h
			order = append(order, h)
			if _, dup := secs[h]; dup {
				res.fail("sections", "section %q appears twice", h)
			}
			secs[h] = nil
			continue
		}
		if cur != "" {
			secs[cur] = append(secs[cur], l)
		}
	}
	// The end marker line ends up in the last section; drop it.
	if cur != "" {
		last := secs[cur]
		for len(last) > 0 && strings.TrimSpace(last[len(last)-1]) == convention.End {
			last = last[:len(last)-1]
		}
		secs[cur] = last
	}
	if strings.Join(order, "|") != strings.Join(convention.Sections, "|") {
		res.fail("sections", "sections must be exactly %s, in that order (found: %s)",
			strings.Join(convention.Sections, ", "), orNone(order))
		return nil
	}
	return secs
}

func checkBrief(lines []string, res *Result, prose *[]string) {
	var para []string
	blankAfter := false
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if t == "" {
			if len(para) > 0 {
				blankAfter = true
			}
			continue
		}
		if blankAfter {
			res.fail("brief", "the Brief is one paragraph; found a second one")
			break
		}
		para = append(para, t)
	}
	if len(para) == 0 {
		res.fail("brief", "the Brief is empty")
		return
	}
	text := strings.Join(para, " ")
	n := len(sentenceRe.FindAllString(text, -1))
	if n == 0 {
		n = 1
	}
	if n > convention.MaxBriefSentences {
		res.fail("brief", "the Brief has %d sentences; the limit is %d", n, convention.MaxBriefSentences)
	}
	*prose = append(*prose, text)
}

type table struct {
	header []string
	rows   [][]string
	next   int // index of the first line after the table
}

func splitRow(line string) []string {
	t := strings.TrimSpace(line)
	t = strings.TrimPrefix(t, "|")
	t = strings.TrimSuffix(t, "|")
	t = strings.ReplaceAll(t, `\|`, "\x00")
	cells := strings.Split(t, "|")
	for i, c := range cells {
		cells[i] = strings.TrimSpace(strings.ReplaceAll(c, "\x00", "|"))
	}
	return cells
}

// parseTable reads a markdown table that starts at the first non-blank line at or after i.
func parseTable(lines []string, i int) (table, bool) {
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}
	if i+1 >= len(lines) || !strings.HasPrefix(strings.TrimSpace(lines[i]), "|") {
		return table{}, false
	}
	sep := splitRow(lines[i+1])
	for _, c := range sep {
		if !sepCellRe.MatchString(c) {
			return table{}, false
		}
	}
	t := table{header: splitRow(lines[i])}
	j := i + 2
	for j < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[j]), "|") {
		t.rows = append(t.rows, splitRow(lines[j]))
		j++
	}
	t.next = j
	return t, true
}

func checkChangeMap(lines []string, res *Result, prose *[]string, th *theme.Theme) {
	joined := strings.Join(lines, "\n")
	noDiagram := false
	if m := noDiagramRe.FindStringSubmatch(joined); m != nil {
		if strings.TrimSpace(m[1]) == "" {
			res.fail("diagram", "`pr-brief:no-diagram:` needs a reason")
		} else {
			noDiagram = true
		}
	}

	var fs fenceState
	heading := false
	total, flows := 0, 0
	defined := map[string]int{} // ref id -> flow number that defined it

	for i := 0; i < len(lines); i++ {
		l := lines[i]
		wasIn := fs.in
		marker := fs.next(l)
		if !wasIn && !marker && strings.HasPrefix(l, "### ") {
			heading = true
		}
		if !(marker && !wasIn && fs.lang == "mermaid") {
			continue
		}

		// Collect the mermaid source.
		var src []string
		j := i + 1
		for ; j < len(lines); j++ {
			if strings.TrimSpace(lines[j]) == "```" {
				break
			}
			src = append(src, lines[j])
		}
		fs = fenceState{}
		total++
		label := fmt.Sprintf("diagram %d", total)
		if !heading {
			res.fail("diagram", "%s has no `### Flow N: ...` heading above it", label)
		}
		heading = false

		d := parseDiagram(strings.Join(src, "\n"))
		for _, p := range d.problems {
			res.fail("diagram", "%s: %s", label, p)
		}
		if th != nil && d.kind == kindFlow && len(d.problems) == 0 {
			for _, p := range d.styleProblems(*th) {
				res.fail("style", "%s: %s", label, p)
			}
		}
		if d.kind == kindFlow {
			flows++
			next := j + 1
			if len(d.problems) == 0 {
				var classes []string
				for _, id := range d.order {
					if n := d.nodes[id]; n.column == "Functions" {
						classes = append(classes, n.class)
					}
				}
				want := dg.LegendLine(classes)
				if next < len(lines) && strings.TrimSpace(lines[next]) == want {
					next++
				} else if want != "" {
					res.fail("diagram", "%s: the line directly under the chart must be the Legend: %s (run `pr-brief diagram`, which prints it)", label, want)
				}
			}
			t, ok := parseTable(lines, next)
			if !ok {
				res.fail("references", "%s: no References table (| Ref | What | Detail |) directly under the diagram", label)
				i = j
				continue
			}
			checkRefs(t, d, flows, label, defined, res, prose)
			i = t.next - 1
		} else {
			i = j
		}
	}

	if total > convention.MaxDiagrams {
		res.fail("diagram", "%d diagrams; the limit is %d (list the extra flows by name in the Review guide and consider splitting the PR)", total, convention.MaxDiagrams)
	}
	if total == 0 && !noDiagram {
		res.fail("diagram", "no mermaid diagram; add one per flow, or `<!-- pr-brief:no-diagram: <reason> -->` for a small or config-only PR")
	}
}

func checkRefs(t table, d *diagram, flow int, label string, defined map[string]int, res *Result, prose *[]string) {
	if len(t.header) < 3 || !strings.EqualFold(t.header[0], "ref") {
		res.fail("references", "%s: the table under the diagram must have the columns Ref | What | Detail", label)
		return
	}
	used := map[string]bool{}
	for _, id := range d.inputs {
		used[id] = true
	}
	for _, id := range d.outputs {
		used[id] = true
	}
	local := map[string]bool{}
	for _, r := range t.rows {
		if len(r) < 3 {
			res.fail("references", "%s: a References row needs three cells (Ref | What | Detail)", label)
			continue
		}
		id, what, detail := r[0], r[1], r[2]
		if !refIDRe.MatchString(id) {
			res.fail("references", "%s: %q is not a reference id (Input 1, Output 1, ...)", label, id)
			continue
		}
		id = shortRef(id)
		if !used[id] {
			res.fail("references", "%s: %s has a row but is not used in the diagram above it", label, id)
		}
		if m := seeFlowRe.FindStringSubmatch(what); m != nil {
			var n int
			fmt.Sscanf(m[1], "%d", &n)
			if defined[id] != n || n >= flow {
				res.fail("references", "%s: %s points at Flow %d, which does not define it", label, id, n)
			}
			local[id] = true
			continue
		}
		if prev, dup := defined[id]; dup {
			res.fail("references", "%s: %s is already defined in Flow %d; an id means one thing everywhere (use `see Flow %d`)", label, id, prev, prev)
			continue
		}
		if what == "" || detail == "" {
			res.fail("references", "%s: %s needs both What and Detail", label, id)
		}
		defined[id] = flow
		local[id] = true
		if detail != "" {
			*prose = append(*prose, detail)
		}
	}
	for _, id := range append(append([]string{}, d.inputs...), d.outputs...) {
		if !local[id] && defined[id] == 0 {
			res.fail("references", "%s: %s is in the diagram but has no References row", label, id)
		} else if !local[id] {
			res.fail("references", "%s: %s is in the diagram; add a row `| %s | see Flow %d | |`", label, id, id, defined[id])
		}
	}
}

func checkReviewGuide(lines []string, res *Result, prose *[]string) {
	state := ""
	sawChanged, sawRead, sawOrder := false, false, false
	inDetails := false
	var fs fenceState

	for i := 0; i < len(lines); i++ {
		l := lines[i]
		t := strings.TrimSpace(l)
		if fs.next(l) || fs.in {
			continue
		}
		if strings.Contains(t, "<details") {
			inDetails = true
		}
		if inDetails {
			if strings.Contains(t, "</details>") {
				inDetails = false
			}
			continue
		}
		switch {
		case strings.Contains(t, "**What changed**"):
			state, sawChanged = "changed", true
			if rest := afterLabel(t, "**What changed**"); rest != "" {
				*prose = append(*prose, rest)
			}
			continue
		case strings.Contains(t, "**Read these first**"):
			state, sawRead = "read", true
			tb, ok := parseTable(lines, i+1)
			if !ok {
				res.fail("review", "**Read these first** must be followed by a table (| File | Why it is delicate | What to check |)")
				continue
			}
			if len(tb.header) < 3 {
				res.fail("review", "the Read-these-first table needs three columns: File | Why it is delicate | What to check")
			}
			if n := len(tb.rows); n < convention.MinReadRows || n > convention.MaxReadRows {
				res.fail("review", "the Read-these-first table has %d rows; it needs %d to %d", n, convention.MinReadRows, convention.MaxReadRows)
			}
			for n, r := range tb.rows {
				if len(r) == 0 {
					continue
				}
				switch path, target, linked := links.Parse(r[0]); {
				case path == "":
					res.fail("review", "Read-these-first row %d: write the file as `path/to/file` in code font (then run `pr-brief links` to link it)", n+1)
				case !linked:
					res.fail("review", "Read-these-first row %d: `%s` must be a link, so a reviewer opens it from the description. Run `pr-brief links --body <file>`", n+1, path)
				case !links.Valid(path, target):
					res.fail("review", "Read-these-first row %d: the link for `%s` does not point at that file, or it is pinned to a commit, a range of commits or an older iteration, which newer commits replace. Run `pr-brief links --body <file>`", n+1, path)
				}
				for _, c := range r[min(1, len(r)):] {
					if c != "" {
						*prose = append(*prose, c)
					}
				}
			}
			i = tb.next - 1
			continue
		case strings.Contains(t, "**Review order**"):
			state, sawOrder = "order", true
			if rest := afterLabel(t, "**Review order**"); rest != "" {
				*prose = append(*prose, rest)
			}
			continue
		}
		if t == "" || strings.HasPrefix(t, "<") || strings.HasPrefix(t, "|") {
			continue
		}
		if state == "changed" || state == "order" {
			*prose = append(*prose, strings.TrimSpace(boldRe.ReplaceAllString(t, "")))
		}
	}
	if !sawChanged {
		res.fail("review", "the Review guide needs a **What changed** part")
	}
	if !sawRead {
		res.fail("review", "the Review guide needs a **Read these first** table")
	}
	if !sawOrder {
		res.fail("review", "the Review guide needs a **Review order** line")
	}
}

func afterLabel(line, label string) string {
	i := strings.Index(line, label)
	rest := strings.TrimSpace(line[i+len(label):])
	rest = strings.TrimSpace(strings.TrimLeft(rest, ":"))
	return strings.TrimSpace(boldRe.ReplaceAllString(rest, ""))
}

// resolveTheme picks the theme the diagrams are checked against. A configured theme
// wins over the marker. Without one, the marker must name a built-in theme.
func resolveTheme(m body.Markers, o Options, res *Result) *theme.Theme {
	if o.Theme != nil {
		return o.Theme
	}
	switch {
	case m.Theme == "":
		res.fail("markers", "the begin marker has no theme; write `<!-- pr-brief:begin v1 style=%s theme=%s -->`", m.Style, theme.DefaultName)
	case theme.IsBuiltin(m.Theme):
		t, _ := theme.Get(m.Theme)
		return &t
	default:
		res.fail("markers", "the description uses theme %q, which is not built in; the gate can only check it when the repo config names the theme file (diagram.theme)", m.Theme)
	}
	return nil
}

// shortRef turns the table's "Input 1" into the id "I1". The short form is accepted too.
func shortRef(id string) string {
	if n, ok := strings.CutPrefix(id, "Input "); ok {
		return "I" + n
	}
	if n, ok := strings.CutPrefix(id, "Output "); ok {
		return "O" + n
	}
	return id
}
