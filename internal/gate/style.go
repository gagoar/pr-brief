package gate

import (
	"fmt"
	"strings"

	dg "github.com/gagoar/pr-brief/internal/diagram"
	"github.com/gagoar/pr-brief/internal/theme"
)

// styleProblems recomputes the style the theme requires for this diagram, from the
// diagram's own edges, and compares it with what the author wrote. Only text that
// pr-brief diagram produces passes.
func (d *diagram) styleProblems(th theme.Theme) []string {
	var problems []string

	kinds := make([]string, len(d.edges))
	for i, e := range d.edges {
		kinds[i] = e.kind
	}
	want := append([]string{th.InitLine()}, th.Defs(kinds)...)
	for i := range want {
		want[i] = strings.TrimSpace(want[i])
	}
	got := make([]string, len(d.style))
	for i, l := range d.style {
		got[i] = strings.TrimSpace(l)
	}
	for i := 0; i < len(want) || i < len(got); i++ {
		w, g := "", ""
		if i < len(want) {
			w = want[i]
		}
		if i < len(got) {
			g = got[i]
		}
		if w != g {
			problems = append(problems, fmt.Sprintf("the style does not match theme %s at style line %d.\n      expected: %s\n      found:    %s\n      Run `pr-brief diagram` and paste its output.", th.ID(), i+1, orNoneLine(w), orNoneLine(g)))
			break
		}
	}

	var fns, outs []string
	for _, id := range d.order {
		switch d.nodes[id].column {
		case "Functions":
			fns = append(fns, id)
		case "Output":
			outs = append(outs, id)
		}
	}
	pairs := make([][2]string, len(d.edges))
	for i, e := range d.edges {
		pairs[i] = [2]string{e.from, e.to}
	}
	var wantRank, gotRank []string
	for _, l := range dg.Ranker(fns, pairs, outs) {
		wantRank = append(wantRank, strings.TrimSpace(l))
	}
	for _, e := range d.invisible {
		gotRank = append(gotRank, fmt.Sprintf("%s ~~~ %s", e.from, e.to))
	}
	if strings.Join(wantRank, "|") != strings.Join(gotRank, "|") {
		problems = append(problems, fmt.Sprintf("the invisible layout links are wrong (GitHub drops Output below Functions without them).\n      expected: %s\n      found:    %s", orNoneLine(strings.Join(wantRank, "; ")), orNoneLine(strings.Join(gotRank, "; "))))
	}
	return problems
}

func orNoneLine(s string) string {
	if s == "" {
		return "(nothing)"
	}
	return s
}
