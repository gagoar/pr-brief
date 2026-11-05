package diagram

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/gagoar/pr-brief/internal/convention"
)

// The plain-text form of a flow is for hosts that do not draw Mermaid. The model draws it, as
// boxes joined by arrows, following the spec that Spec prints. A layout engine cannot beat a
// reader's eye on a small graph, and a drawing the model makes reads like a diagram.
//
// This package does the part a machine can do exactly:
//
//   - FlowComment writes the flow into one hidden HTML comment above the drawing, so the
//     description keeps the true nodes and calls even when the drawing is hand made.
//   - Spec lists the box text and every call for the model.
//   - Cells reads the box text back out of a drawing, and CellProblems compares it with the
//     flow. A missing, extra or misspelled box fails the gate.
//
// What the gate cannot check is the lines: an arrow that points the wrong way passes. The
// Mermaid form has the same limit (the gate checks its text against itself, not against the
// code). Say so in any doc that describes this check.

// Placeholder is the one line of the fence that `pr-brief diagram --format ascii` prints. It
// holds no box, so the gate fails until the drawing replaces it.
const Placeholder = "(replace this line with the boxes-and-arrows drawing: see the spec that pr-brief diagram prints)"

// FlowCommentPrefix opens the hidden comment that carries the flow.
const FlowCommentPrefix = "<!-- pr-brief:flow "

var statusMarker = map[string]string{"added": "+", "modified": "~", "removed": "-", "context": "."}
var markerStatus = map[string]string{"+": "added", "~": "modified", "-": "removed", ".": "context"}
var edgeMark = map[string]string{"new": "=", "existing": "-", "removed": "."}
var markEdge = map[string]string{"=": "new", "-": "existing", ".": "removed"}

// labelEscaper keeps a label from breaking the comment: it holds no `;`, `|`, `-` or `>`.
var labelEscaper = strings.NewReplacer("%", "%25", ";", "%3B", "|", "%7C", "-", "%2D", ">", "%3E", "\n", "%0A", "\r", "%0D")
var labelUnescaper = strings.NewReplacer("%3B", ";", "%7C", "|", "%2D", "-", "%3E", ">", "%0A", "\n", "%0D", "\r", "%25", "%")

// FlowComment is the hidden comment for a flow. Four sections, joined by `;`: the inputs, the
// outputs (an id, then `:db` or `:event` for a shaped one), the functions (id, status marker,
// `!` for a risky one or `_`, then the label) and the calls (from, a style mark, to).
//
//	<!-- pr-brief:flow I1;O1:db,O2;F1~_runDiagram|F2+!Check;I1-F1,F1=F2 -->
func FlowComment(f Flow) string {
	var in, out, fn, ed []string
	for _, p := range f.Inputs {
		in = append(in, p.ID)
	}
	for _, p := range f.Outputs {
		s := p.ID
		if p.Kind != "" {
			s += ":" + p.Kind
		}
		out = append(out, s)
	}
	for _, n := range f.Nodes {
		risk := "_"
		if n.Risky() {
			risk = "!"
		}
		fn = append(fn, n.ID+statusMarker[n.Status]+risk+labelEscaper.Replace(n.Label))
	}
	for _, e := range f.Edges {
		ed = append(ed, e.From+edgeMark[e.Status]+e.To)
	}
	return FlowCommentPrefix + strings.Join([]string{strings.Join(in, ","), strings.Join(out, ","), strings.Join(fn, "|"), strings.Join(ed, ",")}, ";") + " -->"
}

var (
	flowCommentRe = regexp.MustCompile(`^<!-- pr-brief:flow ([^\n]*?) -->$`)
	// fnTokenRe: an id such as F1, a status marker, `!` or `_`, then the label.
	fnTokenRe   = regexp.MustCompile(`^(F\d+)([+~\-.])([!_])(.+)$`)
	edgeTokenRe = regexp.MustCompile(`^([IFO]\d+)([=\-.])([IFO]\d+)$`)
)

// IsFlowComment reports whether a line is a flow comment.
func IsFlowComment(line string) bool { return flowCommentRe.MatchString(strings.TrimSpace(line)) }

// ParseFlowComment reads a flow back out of its comment and validates it.
func ParseFlowComment(line string) (Flow, error) {
	m := flowCommentRe.FindStringSubmatch(strings.TrimSpace(line))
	if m == nil {
		return Flow{}, errors.New("not a flow comment")
	}
	parts := strings.Split(m[1], ";")
	if len(parts) != 4 {
		return Flow{}, fmt.Errorf("a flow comment has 4 sections (inputs;outputs;functions;calls), found %d", len(parts))
	}
	f := Flow{Version: 1}
	if parts[0] != "" {
		for _, id := range strings.Split(parts[0], ",") {
			f.Inputs = append(f.Inputs, Port{ID: id})
		}
	}
	if parts[1] != "" {
		for _, tok := range strings.Split(parts[1], ",") {
			id, kind, _ := strings.Cut(tok, ":")
			f.Outputs = append(f.Outputs, Port{ID: id, Kind: kind})
		}
	}
	if parts[2] != "" {
		for _, tok := range strings.Split(parts[2], "|") {
			t := fnTokenRe.FindStringSubmatch(tok)
			if t == nil {
				return Flow{}, fmt.Errorf("function %q must look like F1~_Name", tok)
			}
			n := Node{ID: t[1], Status: markerStatus[t[2]], Label: labelUnescaper.Replace(t[4])}
			if t[3] == "!" {
				n.RiskScore = convention.RiskClassThreshold
			}
			f.Nodes = append(f.Nodes, n)
		}
	}
	if parts[3] != "" {
		for _, tok := range strings.Split(parts[3], ",") {
			t := edgeTokenRe.FindStringSubmatch(tok)
			if t == nil {
				return Flow{}, fmt.Errorf("call %q must look like F1=F2, F1-F2 or F1.F2", tok)
			}
			f.Edges = append(f.Edges, Edge{From: t[1], To: t[3], Status: markEdge[t[2]]})
		}
	}
	if err := f.Validate(); err != nil {
		return Flow{}, err
	}
	return f, nil
}

// boxText is the text inside the box of each node, in the order inputs, functions, outputs.
func boxTexts(f Flow) []string {
	var out []string
	for _, p := range f.Inputs {
		out = append(out, PortLabel(p.ID))
	}
	for _, n := range f.Nodes {
		out = append(out, statusMarker[n.Status]+" "+Label(n))
	}
	for _, p := range f.Outputs {
		out = append(out, PortLabel(p.ID))
	}
	return out
}

// nameOf is the text a call line shows for a node: the label, or `Input 1`.
func nameOf(f Flow) map[string]string {
	m := map[string]string{}
	for _, p := range f.Inputs {
		m[p.ID] = PortLabel(p.ID)
	}
	for _, n := range f.Nodes {
		m[n.ID] = Label(n)
	}
	for _, p := range f.Outputs {
		m[p.ID] = PortLabel(p.ID)
	}
	return m
}

// Spec is what the model needs to draw the flow: the text of every box and every call.
func Spec(f Flow) string {
	name := nameOf(f)
	var b strings.Builder
	fmt.Fprintf(&b, "Draw this flow as boxes joined by arrows, in a `text` fence, at most %d characters wide.\n\n", convention.MaxASCIIWidth)
	b.WriteString("Boxes. Write each one, with exactly this text inside, on its own line between two `|`:\n")
	for _, p := range f.Inputs {
		fmt.Fprintf(&b, "  %-30s input: rounded corners (. and ')\n", PortLabel(p.ID))
	}
	for _, n := range f.Nodes {
		note := "corners +"
		switch {
		case n.Risky():
			note = "risk: top and bottom border of ="
		case n.Status == "removed":
			note = "removed: dashed border (- - -)"
		}
		fmt.Fprintf(&b, "  %-30s %s\n", statusMarker[n.Status]+" "+Label(n), note)
	}
	for _, p := range f.Outputs {
		fmt.Fprintf(&b, "  %-30s output: rounded corners (. and ')\n", PortLabel(p.ID))
	}
	b.WriteString("\nCalls. Draw an arrow for each, from the first box to the second:\n")
	for _, e := range f.Edges {
		style := map[string]string{"new": "new: a double line (= and #)", "existing": "existing: a single line (- and |)", "removed": "removed: a dotted line (. and :)"}[e.Status]
		fmt.Fprintf(&b, "  %s -> %s   %s\n", name[e.From], name[e.To], style)
	}
	b.WriteString("\nLayout: inputs on top, outputs at the bottom, a call runs down the page. Arrowheads: v, or V for a new call. ")
	b.WriteString("Put nothing else in a box and no other box in the drawing. Keep the Legend line below the fence as printed.\n")
	return b.String()
}

// cellRe finds the text of a box: two `|` around text. A vertical arrow line has no text, so
// it does not match.
var cellRe = regexp.MustCompile(`\|[ ]+(\S(?:[^|]*?\S)?)[ ]+\|`)

// Cells returns the text of every box in a drawing, in reading order.
func Cells(drawing string) []string {
	var out []string
	for _, l := range strings.Split(strings.ReplaceAll(drawing, "\r\n", "\n"), "\n") {
		for _, m := range cellRe.FindAllStringSubmatch(l, -1) {
			out = append(out, m[1])
		}
	}
	return out
}

// CellProblems compares the boxes of a drawing with the boxes the flow needs. It names each box
// that is missing and each box that should not be there.
func CellProblems(f Flow, cells []string) []string {
	want := map[string]int{}
	for _, t := range boxTexts(f) {
		want[t]++
	}
	have := map[string]int{}
	for _, c := range cells {
		have[c]++
	}
	var missing, extra []string
	for t, n := range want {
		for i := have[t]; i < n; i++ {
			missing = append(missing, t)
		}
	}
	for t, n := range have {
		for i := want[t]; i < n; i++ {
			extra = append(extra, t)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	var out []string
	if len(missing) > 0 {
		out = append(out, "no box with the text "+quoteList(missing))
	}
	if len(extra) > 0 {
		out = append(out, "a box that is not in the flow: "+quoteList(extra))
	}
	return out
}

func quoteList(s []string) string {
	q := make([]string, len(s))
	for i, v := range s {
		q[i] = "`" + v + "`"
	}
	return strings.Join(q, ", ")
}

// WidthProblems names each line of a drawing that is too wide or holds a tab.
func WidthProblems(drawing string) []string {
	var out []string
	for i, l := range strings.Split(strings.ReplaceAll(drawing, "\r\n", "\n"), "\n") {
		if strings.Contains(l, "\t") {
			out = append(out, fmt.Sprintf("line %d of the drawing holds a tab; use spaces", i+1))
		}
		if n := len([]rune(l)); n > convention.MaxASCIIWidth {
			out = append(out, fmt.Sprintf("line %d of the drawing is %d characters wide; the limit is %d", i+1, n, convention.MaxASCIIWidth))
		}
	}
	return out
}

// ASCIILegendLine is the line under a plain-text drawing. It names each marker the flow uses.
// classes holds Class() of each Functions node. It returns "" when nothing is in use.
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
		"risk":     "! risk (= border)",
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
	edge := map[string]string{"new": "= # new call", "existing": "- | existing call", "removed": ". : removed call"}
	for _, s := range []string{"new", "existing", "removed"} {
		if seen[s] {
			parts = append(parts, edge[s])
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "**Legend:** " + strings.Join(parts, " · ")
}

// FlowASCIILegend is the plain-text legend line for a flow.
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
