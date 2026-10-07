package diagram

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gagoar/pr-brief/internal/theme"
)

func load(t *testing.T, name string) Flow {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	f, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// The goldens come from the generator that was checked on github.com (Mermaid 11.17.2)
// in light and dark mode. The Go renderer must reproduce them byte for byte.
func TestRenderMatchesTheGitHubValidatedGoldens(t *testing.T) {
	for _, th := range []string{"github-light", "github-dark", "dracula", "alucard"} {
		for _, spec := range []string{"states", "real"} {
			t.Run(th+"-"+spec, func(t *testing.T) {
				want, err := os.ReadFile(filepath.Join("testdata", "golden", th+"-"+spec+".mmd"))
				if err != nil {
					t.Fatal(err)
				}
				theme, _ := theme.Get(th)
				got, err := Render(load(t, spec+".flow.json"), theme)
				if err != nil {
					t.Fatal(err)
				}
				if got != strings.TrimRight(string(want), "\n") {
					t.Errorf("output differs from the golden\n--- got ---\n%s\n--- want ---\n%s", got, want)
				}
			})
		}
	}
}

func TestRankerPicksTheDeepestFunction(t *testing.T) {
	nodes := []string{"F1", "F2", "F3", "F4"}
	edges := [][2]string{{"I1", "F1"}, {"F1", "F2"}, {"F2", "F3"}, {"F1", "F4"}, {"F3", "O1"}}
	got := Ranker(nodes, edges, []string{"O1", "O2"})
	if strings.Join(got, "|") != "  F3 ~~~ O1|  F3 ~~~ O2" {
		t.Errorf("ranker = %q", got)
	}
	// A tie goes to the first in node order.
	if got := Ranker([]string{"F1", "F2"}, [][2]string{{"I1", "F1"}, {"I1", "F2"}}, []string{"O1"}); got[0] != "  F1 ~~~ O1" {
		t.Errorf("tie = %q", got)
	}
	// A cycle must not hang or run away.
	if got := Ranker([]string{"F1", "F2"}, [][2]string{{"F1", "F2"}, {"F2", "F1"}}, []string{"O1"}); len(got) != 1 {
		t.Errorf("cycle = %q", got)
	}
	if Ranker(nodes, edges, nil) != nil || Ranker(nil, edges, []string{"O1"}) != nil {
		t.Error("nothing to rank")
	}
}

func TestRiskLabelsAndClasses(t *testing.T) {
	cases := []struct {
		n     Node
		label string
		class string
	}{
		{Node{Label: "a", Status: "added", RiskScore: 2}, "!a", "riskadded"},
		{Node{Label: "a", Status: "modified", RiskScore: 5}, "!a", "risk"},
		{Node{Label: "a", Status: "context", RiskScore: 2}, "!a", "risk"},
		{Node{Label: "a", Status: "removed", RiskScore: 9}, "a", "removed"},
		{Node{Label: "a", Status: "added", RiskScore: 1}, "a", "added"},
		{Node{Label: `say "hi"()`, Status: "added"}, `say 'hi'()`, "added"},
	}
	for _, c := range cases {
		if Label(c.n) != c.label || c.n.Class() != c.class {
			t.Errorf("%+v -> %q %q, want %q %q", c.n, Label(c.n), c.n.Class(), c.label, c.class)
		}
	}
}

func TestValidateRejects(t *testing.T) {
	base := func() Flow {
		return Flow{
			Inputs:  []Port{{ID: "I1"}},
			Nodes:   []Node{{ID: "F1", Label: "a", Status: "added"}},
			Outputs: []Port{{ID: "O1"}},
			Edges:   []Edge{{From: "I1", To: "F1", Status: "new"}, {From: "F1", To: "O1", Status: "new"}},
		}
	}
	if err := base().Validate(); err != nil {
		t.Fatalf("base flow: %v", err)
	}
	mut := map[string]func(*Flow){
		"no output":       func(f *Flow) { f.Outputs = nil },
		"bad input id":    func(f *Flow) { f.Inputs[0].ID = "X1" },
		"duplicate id":    func(f *Flow) { f.Nodes = append(f.Nodes, Node{ID: "F1", Label: "b", Status: "added"}) },
		"bad status":      func(f *Flow) { f.Nodes[0].Status = "changed" },
		"bad edge status": func(f *Flow) { f.Edges[0].Status = "maybe" },
		"unknown id":      func(f *Flow) { f.Edges[0].To = "F9" },
		"output to input": func(f *Flow) { f.Edges = append(f.Edges, Edge{From: "O1", To: "F1", Status: "new"}) },
		"into an input":   func(f *Flow) { f.Edges = append(f.Edges, Edge{From: "F1", To: "I1", Status: "new"}) },
		"long label":      func(f *Flow) { f.Nodes[0].Label = strings.Repeat("x", 40) },
		"too many nodes": func(f *Flow) {
			for i := 2; i <= 9; i++ {
				f.Nodes = append(f.Nodes, Node{ID: "F" + string(rune('0'+i)), Label: "n", Status: "added"})
			}
		},
		"too many context": func(f *Flow) {
			f.Nodes = []Node{{ID: "F1", Label: "a", Status: "context"}, {ID: "F2", Label: "b", Status: "context"}, {ID: "F3", Label: "c", Status: "context"}}
		},
	}
	for name, m := range mut {
		f := base()
		m(&f)
		if err := f.Validate(); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
	if _, err := Parse([]byte(`{"inputs":[],"extra":1}`)); err == nil {
		t.Error("an unknown key must be rejected")
	}
}

func TestFromReport(t *testing.T) {
	report := `{"refs":{"I1":{"kind":"route"},"O1":{"kind":"db"}},"flows":[{"inputs":["I1"],"outputs":["O1"],
	  "functions":[{"node":"F1","label":"save","status":"added","riskScore":3}],
	  "edges":[{"from":"I1","to":"F1","status":"new"},{"from":"F1","to":"O1","status":"new"}]}]}`
	f, err := FromReport([]byte(report), 1)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Render(f, theme.Default())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`O1[("Output 1")]`, `F1["!save"]:::riskadded`, "I1 ==> F1", "F1 ~~~ O1"} {
		if !strings.Contains(got, want) {
			t.Errorf("render lacks %q:\n%s", want, got)
		}
	}
	if _, err := FromReport([]byte(report), 2); err == nil {
		t.Error("flow 2 does not exist")
	}
}
