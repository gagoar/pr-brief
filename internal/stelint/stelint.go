// Package stelint is a Go port of ste-lint.py from
// github.com/danyuchn/asd-ste100-skill (MIT). It checks the structural
// ASD-STE100 rules that need no dictionary. See UPSTREAM.md for the pinned
// upstream commit and every deliberate deviation.
package stelint

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	// LevelHard is upstream's "advisory-free" level: it counts against the baseline.
	LevelHard = "advisory-free"
	// LevelAdvisory never fails a run.
	LevelAdvisory = "advisory"

	maxWords = 25
)

// Options mirrors upstream's --disable and --baseline flags.
type Options struct {
	// Disable lists rule ids to drop from the report (upstream splits the
	// --disable value on commas and does not trim).
	Disable []string
	// Baseline is the number of hard violations tolerated before a run fails.
	Baseline int
}

// Violation mirrors one entry of upstream's JSON "violations" array. Field
// order matches upstream's key order. Col is a 1-based code-point column.
type Violation struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Col     int    `json:"col"`
	Rule    string `json:"rule"`
	Level   string `json:"level"`
	Match   string `json:"match"`
	Message string `json:"message"`
}

// Hard reports whether the violation counts against the baseline.
func (v Violation) Hard() bool { return v.Level == LevelHard }

// Report mirrors upstream's --json document.
type Report struct {
	Violations  []Violation `json:"violations"`
	Count       int         `json:"count"`
	HardCount   int         `json:"hard_count"`
	Baseline    int         `json:"baseline"`
	Words       int         `json:"words"`
	Per100Words float64     `json:"per_100_words"`
}

// Failed reports whether hard violations exceed the baseline (upstream exit 1).
func (r Report) Failed() bool { return r.HardCount > r.Baseline }

const irregularParticiples = "given|taken|made|done|found|seen|known|shown|written|built|sent|set|run|read|kept|held|left|put|cut|hit|let|shut|split|spread|begun|become|come|gone|got|gotten|lost|met|paid|said|sold|told|thought|brought|bought|caught|taught|won|worn|torn|born|drawn|grown|thrown|flown|driven|risen|chosen|broken|spoken|frozen|hidden|ridden|forgotten|fallen|eaten|beaten|understood|stood|struck|stuck|swung|hung|led|fed|bled|fled|sped|bound|wound|dug|spun|slid|bit|lit|quit"

const passiveParticiples = "given|taken|made|done|found|seen|known|shown|written|built|sent|set|run|read|kept|held|left|put"

type rule struct {
	id, level string
	re        *regexp.Regexp
	msg       string
}

var (
	// All patterns run against normalize()d text, so Go's ASCII \w, \b and \s
	// behave like Python's Unicode-aware ones.

	// Upstream's six modal lookbehinds are dropped: modalPerfectPrefix below
	// already rejects every match they rejected.
	modalPerfectPrefix = regexp.MustCompile(`(?i)\b(?:may|might|could|should|would|must|can|will|shall)(?:\s+not|n['’]t)?\s+$`)

	// Upstream's negative lookahead (?!\s+(?:to|for|by)\s+\w+ing) is applied by
	// passiveMatches using passiveReject.
	passiveReject = regexp.MustCompile(`(?i)^\s+(?:to|for|by)\s+\w+ing`)

	rules = []rule{
		{"semicolon", LevelHard, regexp.MustCompile(`;`),
			"STE bans the semicolon (Rule 8.1). Split into separate sentences."},
		{"phrasal-verb", LevelHard,
			regexp.MustCompile(`(?i)\b(spin(?:ning|s)? up|spun up|reach(?:ing|es|ed)? out|div(?:e|es|ing|ed) into|dove into|kick(?:ing|s|ed)? off|circl(?:e|es|ing|ed) back|touch(?:ing|es|ed)? base)\b`),
			"Soft phrasal verb. Use the single plain verb (start, contact, read, begin)."},
		{"marketing-adjective", LevelHard,
			regexp.MustCompile(`(?i)\b(seamless(?:ly)?|robust(?:ly)?|cutting-edge|effortless(?:ly)?|blazing[- ]fast|world-class|state-of-the-art|game-chang(?:ing|er))\b`),
			"Marketing adjective. Delete, or replace with the measurement that earns the claim."},
		{"nominalization", LevelHard,
			regexp.MustCompile(`(?i)\b(perform|performs|performed|conduct|conducts|conducted|carry out|carries out|carried out)\s+(?:a|an|the)\s+\w+(?:tion|sion|ment|ance|ence|ysis)\b`),
			"Action frozen into a noun. Use the verb (analyze, not perform an analysis of)."},
		{"passive-voice", LevelAdvisory,
			regexp.MustCompile(`(?i)\b(is|are|was|were|been|being)\s+(\w+ed|` + passiveParticiples + `)\b`),
			"Possible passive voice. Name the actor and use an active verb, unless the actor is unknown or irrelevant."},
		{"present-perfect", LevelAdvisory,
			regexp.MustCompile(`(?i)\b(has|have|had)\s+(?:been\s+)?(?:\w+(?:ed|en)|` + irregularParticiples + `)\b`),
			"Compound tense. Use simple past/present unless current relevance is the point (then keep and flag)."},
	}

	synonymGroups = [][]string{
		{"check", "verify", "confirm", "validate"},
		{"delete", "remove", "erase"},
		{"start", "launch", "begin", "initiate"},
		{"stop", "halt", "terminate"},
		{"show", "display"},
		{"use", "utilize", "employ"},
		{"fix", "repair", "correct"},
		{"send", "transmit"},
		{"get", "retrieve", "fetch", "obtain"},
		{"change", "modify", "alter"},
	}
	synonymRes = func() map[string]*regexp.Regexp {
		m := map[string]*regexp.Regexp{}
		for _, g := range synonymGroups {
			for _, base := range g {
				m[base] = regexp.MustCompile(`(?i)\b` + base + `(?:s|es|ed|d|ing)?\b`)
			}
		}
		return m
	}()

	codeFence         = regexp.MustCompile("^(```|~~~)")
	inlineCode        = regexp.MustCompile("`[^`]*`")
	listItemStart     = regexp.MustCompile(`^( {0,3})([-*+]|[0-9]+[.)])( +)(.*)$`)
	conjunctionEnd    = regexp.MustCompile(`(?i)\b(?:and|or)\s*$`)
	tableSeparatorRe  = regexp.MustCompile(`^:?-{3,}:?$`)
	danglingMessage   = "List item ends with a coordinating conjunction. Complete the item or join it with the next item."
	longSentenceLevel = LevelHard
)

// Lint lints text, reporting file as "<stdin>" like upstream does for piped input.
func Lint(text string, opts Options) Report {
	return LintNamed(text, "<stdin>", opts)
}

// LintNamed lints text and labels every violation with filename.
func LintNamed(text, filename string, opts Options) Report {
	findings, words := lint(text, filename)
	return buildReport(findings, words, opts)
}

// Merge combines per-file results the way upstream's CLI does for several
// FILE arguments: findings concatenated in argument order, words summed.
func Merge(opts Options, reports ...Report) Report {
	var all []Violation
	words := 0
	for _, r := range reports {
		all = append(all, r.Violations...)
		words += r.Words
	}
	// Reports were built with their own Disable filter already applied; only
	// the summary is recomputed.
	return summarize(all, words, opts.Baseline)
}

func buildReport(findings []Violation, words int, opts Options) Report {
	if len(opts.Disable) > 0 {
		off := map[string]bool{}
		for _, d := range opts.Disable {
			off[d] = true
		}
		kept := findings[:0:0]
		for _, f := range findings {
			if !off[f.Rule] {
				kept = append(kept, f)
			}
		}
		findings = kept
	}
	return summarize(findings, words, opts.Baseline)
}

func summarize(findings []Violation, words, baseline int) Report {
	if findings == nil {
		findings = []Violation{}
	}
	hard := 0
	for _, f := range findings {
		if f.Hard() {
			hard++
		}
	}
	rate := 0.0
	if words > 0 {
		x := float64(len(findings)*100) / float64(words)
		rate, _ = strconv.ParseFloat(strconv.FormatFloat(x, 'f', 1, 64), 64)
	}
	return Report{Violations: findings, Count: len(findings), HardCount: hard,
		Baseline: baseline, Words: words, Per100Words: rate}
}

type cell struct {
	text string
	col  int
}

func leadingSpaces(line string) int {
	return len(line) - len(strings.TrimLeft(line, " "))
}

func isListContinuation(line string, contentIndent int) bool {
	if pyStrip(line) == "" {
		return true
	}
	if listItemStart.MatchString(line) {
		return false
	}
	return leadingSpaces(line) >= contentIndent
}

// splitTableRow mirrors _split_table_row. It returns nil when the line is not
// a table row. Columns are code-point offsets.
func splitTableRow(line string) []cell {
	rs := []rune(line)
	left := 0
	for left < len(rs) && pyIsSpace(rs[left]) {
		left++
	}
	right := len(rs)
	for right > 0 && pyIsSpace(rs[right-1]) {
		right--
	}
	var content []rune
	if left < right {
		content = rs[left:right]
	}
	hasPipe := false
	for _, c := range content {
		if c == '|' {
			hasPipe = true
			break
		}
	}
	if !hasPipe {
		return nil
	}
	if content[0] == '|' {
		content = content[1:]
		left++
	}
	if n := len(content); n > 0 && content[n-1] == '|' {
		content = content[:n-1]
	}
	// Manual split on a pipe not preceded by a backslash.
	var raw [][]rune
	start := 0
	for i, c := range content {
		if c == '|' && (i == 0 || content[i-1] != '\\') {
			raw = append(raw, content[start:i])
			start = i + 1
		}
	}
	raw = append(raw, content[start:])
	if len(raw) < 2 {
		return nil
	}
	cells := make([]cell, 0, len(raw))
	column := left
	for _, rc := range raw {
		s := string(rc)
		leading := len(rc) - len([]rune(pyLStrip(s)))
		cells = append(cells, cell{pyStrip(s), column + leading})
		column += len(rc) + 1
	}
	return cells
}

func markdownTableCells(lines []string) map[int][]cell {
	table := map[int][]cell{}
	index := 1
	for index < len(lines) {
		sep := splitTableRow(lines[index])
		header := splitTableRow(lines[index-1])
		ok := sep != nil && header != nil && len(sep) == len(header)
		if ok {
			for _, c := range sep {
				if !tableSeparatorRe.MatchString(c.text) {
					ok = false
					break
				}
			}
		}
		if !ok {
			index++
			continue
		}
		table[index-1] = header
		table[index] = []cell{}
		index++
		for index < len(lines) {
			row := splitTableRow(lines[index])
			if row == nil || len(row) != len(sep) {
				break
			}
			table[index] = row
			index++
		}
	}
	return table
}

func danglingConjunctionFindings(lines []string, filename string) []Violation {
	var findings []Violation
	inFence := false
	index := 0
	type itemLine struct {
		idx  int
		text string
	}
	for index < len(lines) {
		line := lines[index]
		if codeFence.MatchString(pyStrip(line)) {
			inFence = !inFence
			index++
			continue
		}
		if inFence {
			index++
			continue
		}
		start := listItemStart.FindStringSubmatch(line)
		if start == nil {
			index++
			continue
		}
		indent, marker, gap, body := start[1], start[2], start[3], start[4]
		contentIndent := len(indent) + len(marker) + len(gap)
		items := []itemLine{{index, body}}
		next := index + 1
		itemFence := false
		for next < len(lines) {
			cand := lines[next]
			if codeFence.MatchString(pyStrip(cand)) {
				itemFence = !itemFence
				next++
				continue
			}
			if itemFence {
				next++
				continue
			}
			if !isListContinuation(cand, contentIndent) {
				break
			}
			items = append(items, itemLine{next, cand})
			next++
		}

		var meaningful []itemLine
		for _, it := range items {
			cleaned := pyStrip(inlineCode.ReplaceAllString(it.text, " CODE "))
			if cleaned != "" {
				meaningful = append(meaningful, itemLine{it.idx, cleaned})
			}
		}
		if len(meaningful) > 0 {
			last := meaningful[len(meaningful)-1]
			if conjunctionEnd.MatchString(normalize([]rune(last.text))) {
				var findingLine, findingCol int
				if last.idx == index {
					findingLine = index + 1
					findingCol = len(indent) + 1
				} else {
					var raw string
					for _, it := range items {
						if it.idx == last.idx {
							raw = it.text
							break
						}
					}
					masked := inlineCode.ReplaceAllStringFunc(raw, func(s string) string {
						return strings.Repeat(" ", len([]rune(s)))
					})
					nm := normalize([]rune(masked))
					findingLine = last.idx + 1
					findingCol = 1
					if loc := conjunctionEnd.FindStringIndex(nm); loc != nil {
						findingCol = runeIndex(nm, loc[0]) + 1
					}
				}
				findings = append(findings, Violation{
					File: filename, Line: findingLine, Col: findingCol,
					Rule: "dangling-conjunction", Level: LevelHard,
					Match: last.text, Message: danglingMessage,
				})
			}
		}
		index = next
	}
	return findings
}

// passiveMatches emulates upstream's passive-voice regex including its
// negative lookahead. After a rejected match it resumes at match start+1,
// as Python's matcher does.
func passiveMatches(norm string, re *regexp.Regexp) [][2]int {
	var out [][2]int
	pos := 0
	for pos <= len(norm) {
		loc := re.FindStringIndex(norm[pos:])
		if loc == nil {
			break
		}
		s, e := pos+loc[0], pos+loc[1]
		// Slicing hides the character before pos; a match starting right at
		// the slice edge is only valid if that character is not a word char.
		if loc[0] == 0 && pos > 0 && isWordByte(norm[pos-1]) {
			pos = s + 1
			continue
		}
		if passiveReject.MatchString(norm[e:]) {
			pos = s + 1
			continue
		}
		out = append(out, [2]int{s, e})
		pos = e
	}
	return out
}

type synHit struct {
	line, col int
	match     string
}

func lint(text, filename string) ([]Violation, int) {
	var findings []Violation
	words := 0
	inFence := false
	lines := splitLines(text)
	tableCells := markdownTableCells(lines)
	type synKey struct {
		gi   int
		base string
	}
	seen := map[synKey]synHit{}

	for i, rawLine := range lines {
		lineno := i + 1
		if codeFence.MatchString(pyStrip(rawLine)) {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		segments, isTable := tableCells[i]
		if !isTable {
			segments = []cell{{rawLine, 0}}
		}
		for _, seg := range segments {
			line := inlineCode.ReplaceAllString(seg.text, "")
			words += pyWordCount(line)
			orig := []rune(line)
			norm := normalize(orig)
			for _, rl := range rules {
				var locs [][2]int
				if rl.id == "passive-voice" {
					locs = passiveMatches(norm, rl.re)
				} else {
					for _, l := range rl.re.FindAllStringIndex(norm, -1) {
						locs = append(locs, [2]int{l[0], l[1]})
					}
				}
				for _, l := range locs {
					if rl.id == "present-perfect" && modalPerfectPrefix.MatchString(norm[:l[0]]) {
						continue
					}
					rs, re := runeIndex(norm, l[0]), runeIndex(norm, l[1])
					findings = append(findings, Violation{
						File: filename, Line: lineno, Col: seg.col + rs + 1,
						Rule: rl.id, Level: rl.level,
						Match: string(orig[rs:re]), Message: rl.msg,
					})
				}
			}
			for gi, group := range synonymGroups {
				for _, base := range group {
					if _, ok := seen[synKey{gi, base}]; ok {
						continue
					}
					if l := synonymRes[base].FindStringIndex(norm); l != nil {
						rs, re := runeIndex(norm, l[0]), runeIndex(norm, l[1])
						seen[synKey{gi, base}] = synHit{lineno, seg.col + rs + 1, string(orig[rs:re])}
					}
				}
			}
			// Split at whitespace runs that follow [.!?].
			for _, sent := range splitSentences(orig) {
				n := pyWordCount(sent)
				if n > maxWords {
					findings = append(findings, Violation{
						File: filename, Line: lineno, Col: seg.col + 1,
						Rule: "long-sentence", Level: longSentenceLevel,
						Match:   strconv.Itoa(n) + " words",
						Message: "Sentence has " + strconv.Itoa(n) + " words (cap " + strconv.Itoa(maxWords) + "). Split it.",
					})
				}
			}
		}
	}

	for gi, group := range synonymGroups {
		type present struct {
			hit  synHit
			base string
		}
		var ps []present
		for _, b := range group {
			if h, ok := seen[synKey{gi, b}]; ok {
				ps = append(ps, present{h, b})
			}
		}
		if len(ps) > 1 {
			sort.Slice(ps, func(a, b int) bool {
				x, y := ps[a], ps[b]
				if x.hit.line != y.hit.line {
					return x.hit.line < y.hit.line
				}
				if x.hit.col != y.hit.col {
					return x.hit.col < y.hit.col
				}
				if x.hit.match != y.hit.match {
					return x.hit.match < y.hit.match
				}
				return x.base < y.base
			})
			first := ps[0].base
			for _, p := range ps[1:] {
				findings = append(findings, Violation{
					File: filename, Line: p.hit.line, Col: p.hit.col,
					Rule: "synonym-rotation", Level: LevelHard, Match: p.hit.match,
					Message: "'" + p.base + "' and '" + first + "' name the same action. Pick one and use it every time.",
				})
			}
		}
	}
	findings = append(findings, danglingConjunctionFindings(lines, filename)...)
	sort.SliceStable(findings, func(a, b int) bool {
		if findings[a].Line != findings[b].Line {
			return findings[a].Line < findings[b].Line
		}
		return findings[a].Col < findings[b].Col
	})
	return findings, words
}

// splitSentences reproduces re.split(r"(?<=[.!?])\s+", line): it splits at
// each maximal whitespace run that directly follows '.', '!' or '?'.
func splitSentences(rs []rune) []string {
	var out []string
	start := 0
	for i := 1; i < len(rs); {
		if pyIsSpace(rs[i]) && (rs[i-1] == '.' || rs[i-1] == '!' || rs[i-1] == '?') {
			j := i
			for j < len(rs) && pyIsSpace(rs[j]) {
				j++
			}
			out = append(out, string(rs[start:i]))
			start = j
			i = j + 1
			continue
		}
		i++
	}
	return append(out, string(rs[start:]))
}
