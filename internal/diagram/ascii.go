package diagram

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/gagoar/pr-brief/internal/convention"
)

// The ASCII form of a flow is for hosts that do not draw Mermaid. It is plain text in a
// code fence: three columns (Input, Functions, Output) and, under them, one line per edge.
//
//	INPUT          FUNCTIONS                 OUTPUT
//	( Input 1 )    [+ InviteCommand.Handle]  [( Output 1 )]
//	               [+ !CodeGenerator.Next]   > Output 2 ]
//	               [. UserQuery.Get]
//
//	Input 1 ==> InviteCommand.Handle
//	InviteCommand.Handle ==> !CodeGenerator.Next
//
// Colour cannot exist in plain text, so a status is a one-character marker in the box:
// + added, ~ modified, - removed, . unchanged context. A risky function keeps its `!`.
// Arrows are the Mermaid arrows: ==> new, --> existing, -.-> removed.
//
// The form is built so the gate can read it back: ParseASCII returns the flow, and the
// gate compares RenderASCII of that flow with the text, so only output of this package
// passes. It is a grid and a list, not routed lines, on purpose: a routed drawing
// cannot be parsed back exactly, and it would cost more of Azure DevOps' 4,000 characters.

const (
	asciiGap     = "  "
	headerInput  = "INPUT"
	headerFn     = "FUNCTIONS"
	headerOutput = "OUTPUT"
)

var statusMarker = map[string]string{"added": "+", "modified": "~", "removed": "-", "context": "."}

// edgeToken is the arrow of an edge status, the same arrows Mermaid uses.
func edgeToken(status string) string { return Arrow(status) }

var edgeSplitRe = regexp.MustCompile(` (==>|-\.->|-->) `)

// RenderASCII returns the ASCII diagram (no code fence, no trailing newline).
func RenderASCII(f Flow) (string, error) {
	if err := f.Validate(); err != nil {
		return "", err
	}
	// Edges name their ends by text, so two ends may not share a text.
	text := map[string]string{}
	seen := map[string]string{}
	add := func(id, t string) error {
		if prev, dup := seen[t]; dup {
			return fmt.Errorf("invalid flow: %s and %s are both drawn as %q; give one of them a different label", prev, id, t)
		}
		seen[t] = id
		text[id] = t
		return nil
	}
	var in, fn, out []string
	for _, p := range f.Inputs {
		if err := add(p.ID, PortLabel(p.ID)); err != nil {
			return "", err
		}
		in = append(in, "( "+PortLabel(p.ID)+" )")
	}
	for _, n := range f.Nodes {
		l := Label(n)
		if edgeSplitRe.MatchString(" " + l + " ") {
			return "", fmt.Errorf("invalid flow: label %q for %s contains an arrow", n.Label, n.ID)
		}
		if err := add(n.ID, l); err != nil {
			return "", err
		}
		fn = append(fn, "["+statusMarker[n.Status]+" "+l+"]")
	}
	for _, p := range f.Outputs {
		if err := add(p.ID, PortLabel(p.ID)); err != nil {
			return "", err
		}
		switch p.Kind {
		case "db":
			out = append(out, "[( "+PortLabel(p.ID)+" )]")
		case "event":
			out = append(out, "> "+PortLabel(p.ID)+" ]")
		default:
			out = append(out, "[ "+PortLabel(p.ID)+" ]")
		}
	}

	w1, w2 := width(headerInput, in), width(headerFn, fn)
	var b strings.Builder
	line := func(a, c, d string) {
		s := pad(a, w1) + asciiGap + pad(c, w2) + asciiGap + d
		b.WriteString(strings.TrimRight(s, " ") + "\n")
	}
	line(headerInput, headerFn, headerOutput)
	rows := max(len(in), len(fn), len(out))
	for i := 0; i < rows; i++ {
		line(at(in, i), at(fn, i), at(out, i))
	}
	b.WriteString("\n")
	for _, e := range f.Edges {
		fmt.Fprintf(&b, "%s %s %s\n", text[e.From], edgeToken(e.Status), text[e.To])
	}
	res := strings.TrimRight(b.String(), "\n")
	for _, l := range strings.Split(res, "\n") {
		if n := utf8.RuneCountInString(l); n > convention.MaxASCIIWidth {
			return "", fmt.Errorf("invalid flow: an ASCII line is %d characters wide; the limit is %d (shorten a label)", n, convention.MaxASCIIWidth)
		}
	}
	return res, nil
}

func width(header string, cells []string) int {
	w := utf8.RuneCountInString(header)
	for _, c := range cells {
		w = max(w, utf8.RuneCountInString(c))
	}
	return w
}

func pad(s string, w int) string {
	if n := utf8.RuneCountInString(s); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return s
}

func at(s []string, i int) string {
	if i < len(s) {
		return s[i]
	}
	return ""
}

var (
	asciiInRe   = regexp.MustCompile(`^\( Input (\d+) \)$`)
	asciiFnRe   = regexp.MustCompile(`^\[([+~\-.]) (.+)\]$`)
	asciiOutRe  = regexp.MustCompile(`^(?:\[\( Output (\d+) \)\]|> Output (\d+) \]|\[ Output (\d+) \])$`)
	markerToSta = map[string]string{"+": "added", "~": "modified", "-": "removed", ".": "context"}
)

// ParseASCII reads an ASCII diagram back into a flow. It is strict about the shape and
// lenient about nothing else: the caller compares RenderASCII of the result with the
// source, so any difference the parser ignores still fails the gate.
func ParseASCII(src string) (Flow, error) {
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	if len(lines) < 2 {
		return Flow{}, errors.New("an ASCII diagram needs a header line and a row")
	}
	head := lines[0]
	c1, c2, c3 := strings.Index(head, headerInput), strings.Index(head, headerFn), strings.Index(head, headerOutput)
	if c1 != 0 || c2 < 0 || c3 < 0 || c2 > c3 {
		return Flow{}, fmt.Errorf("the first line must be the header `%s  %s  %s`", headerInput, headerFn, headerOutput)
	}
	var f Flow
	f.Version = 1
	ids := map[string]string{} // drawn text -> id
	i := 1
	for ; i < len(lines) && strings.TrimSpace(lines[i]) != ""; i++ {
		cells := sliceCols(lines[i], c2, c3)
		if cells[0] != "" {
			m := asciiInRe.FindStringSubmatch(cells[0])
			if m == nil {
				return Flow{}, fmt.Errorf("input cell %q must look like `( Input 1 )`", cells[0])
			}
			id := "I" + m[1]
			f.Inputs = append(f.Inputs, Port{ID: id})
			ids[PortLabel(id)] = id
		}
		if cells[1] != "" {
			m := asciiFnRe.FindStringSubmatch(cells[1])
			if m == nil {
				return Flow{}, fmt.Errorf("function cell %q must look like `[+ Name]` (marker + ~ - or .)", cells[1])
			}
			id := fmt.Sprintf("F%d", len(f.Nodes)+1)
			n := Node{ID: id, Status: markerToSta[m[1]], Label: m[2]}
			if rest, ok := strings.CutPrefix(m[2], "!"); ok {
				if n.Status == "removed" {
					return Flow{}, fmt.Errorf("function cell %q: a removed function has no risk mark", cells[1])
				}
				n.Label, n.RiskScore = rest, convention.RiskClassThreshold
			}
			f.Nodes = append(f.Nodes, n)
			ids[m[2]] = id
		}
		if cells[2] != "" {
			m := asciiOutRe.FindStringSubmatch(cells[2])
			if m == nil {
				return Flow{}, fmt.Errorf("output cell %q must look like `[( Output 1 )]`, `> Output 1 ]` or `[ Output 1 ]`", cells[2])
			}
			p := Port{}
			switch {
			case m[1] != "":
				p = Port{ID: "O" + m[1], Kind: "db"}
			case m[2] != "":
				p = Port{ID: "O" + m[2], Kind: "event"}
			default:
				p = Port{ID: "O" + m[3]}
			}
			f.Outputs = append(f.Outputs, p)
			ids[PortLabel(p.ID)] = p.ID
		}
	}
	for ; i < len(lines); i++ {
		l := lines[i]
		if strings.TrimSpace(l) == "" {
			continue
		}
		loc := edgeSplitRe.FindStringSubmatchIndex(l)
		if loc == nil {
			return Flow{}, fmt.Errorf("edge line %q must look like `A ==> B`", l)
		}
		from, to, tok := l[:loc[0]], l[loc[1]:], l[loc[2]:loc[3]]
		fid, ok1 := ids[from]
		tid, ok2 := ids[to]
		if !ok1 || !ok2 {
			return Flow{}, fmt.Errorf("edge line %q names %q, which the columns above do not draw", l, map[bool]string{true: to, false: from}[ok1])
		}
		f.Edges = append(f.Edges, Edge{From: fid, To: tid, Status: statusOfToken(tok)})
	}
	if err := f.Validate(); err != nil {
		return Flow{}, err
	}
	return f, nil
}

func statusOfToken(tok string) string {
	switch tok {
	case "==>":
		return "new"
	case "-.->":
		return "removed"
	}
	return "existing"
}

// sliceCols cuts one grid row into its three cells at the header offsets.
func sliceCols(line string, c2, c3 int) [3]string {
	r := []rune(line)
	cut := func(a, b int) string {
		if a >= len(r) {
			return ""
		}
		if b > len(r) || b < 0 {
			b = len(r)
		}
		return strings.TrimSpace(string(r[a:b]))
	}
	// Offsets are in runes; the header is ASCII, so bytes and runes agree there.
	return [3]string{cut(0, c2), cut(c2, c3), cut(c3, -1)}
}

// ASCIILegendLine is the line under an ASCII chart. It names each marker the flow uses.
// classes holds Class() of each Functions node. It returns "" when no class is in use.
func ASCIILegendLine(classes []string, edgeStatuses []string) string {
	used := map[string]bool{}
	for _, c := range classes {
		switch c {
		case "riskadded":
			used["added"], used["risk"] = true, true
		case "added", "modified", "removed", "context", "risk":
			used[c] = true
		}
	}
	text := map[string]string{
		"added":    "[+] added",
		"modified": "[~] modified",
		"removed":  "[-] removed",
		"context":  "[.] unchanged context",
		"risk":     "! risk",
	}
	var parts []string
	for _, c := range legendOrder {
		if used[c] {
			parts = append(parts, text[c])
		}
	}
	seen := map[string]bool{}
	for _, s := range edgeStatuses {
		seen[s] = true
	}
	for _, s := range []string{"new", "existing", "removed"} {
		if seen[s] {
			parts = append(parts, edgeToken(s)+" "+s)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "**Legend:** " + strings.Join(parts, " · ")
}

// FlowASCIILegend is the ASCII legend line for a flow.
func FlowASCIILegend(f Flow) string {
	var classes, edges []string
	for _, n := range f.Nodes {
		classes = append(classes, n.Class())
	}
	for _, e := range f.Edges {
		edges = append(edges, e.Status)
	}
	return ASCIILegendLine(classes, edges)
}
