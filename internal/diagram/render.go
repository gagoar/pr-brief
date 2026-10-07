package diagram

import (
	"fmt"
	"strings"

	"github.com/gagoar/pr-brief/internal/theme"
)

// Arrow returns the Mermaid arrow of an edge status.
func Arrow(status string) string {
	switch status {
	case "new":
		return "==>"
	case "removed":
		return "-.->"
	}
	return "-->"
}

// Ranker returns the invisible layout links. On GitHub, Output drops below the
// Functions box unless the deepest function links to every output: Mermaid then ranks
// each output after the whole Functions box. The deepest function is the first, in
// node order, with the longest path of function-to-function edges.
func Ranker(nodes []string, edges [][2]string, outputs []string) []string {
	if len(outputs) == 0 || len(nodes) == 0 {
		return nil
	}
	isNode := map[string]bool{}
	depth := map[string]int{}
	for _, n := range nodes {
		isNode[n] = true
		depth[n] = 1
	}
	for i := 0; i < len(nodes); i++ { // longest path; the cap stops cycles
		changed := false
		for _, e := range edges {
			if isNode[e[0]] && isNode[e[1]] && depth[e[1]] < depth[e[0]]+1 && depth[e[0]]+1 <= len(nodes) {
				depth[e[1]] = depth[e[0]] + 1
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	deepest := nodes[0]
	for _, n := range nodes {
		if depth[n] > depth[deepest] {
			deepest = n
		}
	}
	var out []string
	for _, o := range outputs {
		out = append(out, fmt.Sprintf("  %s ~~~ %s", deepest, o))
	}
	return out
}

// Render returns the complete diagram source (no code fence, no trailing newline).
func Render(f Flow, th theme.Theme) (string, error) {
	if err := f.Validate(); err != nil {
		return "", err
	}
	var b strings.Builder
	w := func(format string, a ...any) { fmt.Fprintf(&b, format+"\n", a...) }
	w("%s", th.InitLine())
	w("graph LR")
	w(`  subgraph IN["Input"]`)
	for _, p := range f.Inputs {
		w(`    %s(["%s"])`, p.ID, PortLabel(p.ID))
	}
	w("  end")
	w(`  subgraph FN["Functions"]`)
	for _, n := range f.Nodes {
		w(`    %s["%s"]:::%s`, n.ID, Label(n), n.Class())
	}
	w("  end")
	w(`  subgraph OUT["Output"]`)
	for _, p := range f.Outputs {
		switch p.Kind {
		case "db":
			w(`    %s[("%s")]`, p.ID, PortLabel(p.ID))
		case "event":
			w(`    %s>"%s"]`, p.ID, PortLabel(p.ID))
		default:
			w(`    %s["%s"]`, p.ID, PortLabel(p.ID))
		}
	}
	w("  end")
	var kinds []string
	var pairs [][2]string
	for _, e := range f.Edges {
		w("  %s %s %s", e.From, Arrow(e.Status), e.To)
		kinds = append(kinds, kindOf(e.Status))
		pairs = append(pairs, [2]string{e.From, e.To})
	}
	var ids, outs []string
	for _, n := range f.Nodes {
		ids = append(ids, n.ID)
	}
	for _, p := range f.Outputs {
		outs = append(outs, p.ID)
	}
	for _, l := range Ranker(ids, pairs, outs) {
		w("%s", l)
	}
	for _, l := range th.Defs(kinds) {
		w("%s", l)
	}
	return strings.TrimRight(b.String(), "\n"), nil
}

func kindOf(status string) string {
	switch status {
	case "new":
		return theme.EdgeNew
	case "removed":
		return theme.EdgeRemoved
	}
	return theme.EdgeExisting
}
