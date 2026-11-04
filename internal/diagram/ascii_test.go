package diagram

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gagoar/pr-brief/internal/convention"
)

// The goldens are the reviewed ASCII output. Regenerate with UPDATE_GOLDEN=1 and read the diff.
func TestRenderASCIIMatchesTheGoldens(t *testing.T) {
	for _, spec := range []string{"doc", "states", "real"} {
		t.Run(spec, func(t *testing.T) {
			got, err := RenderASCII(load(t, spec+".flow.json"))
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join("testdata", "golden", "ascii-"+spec+".txt")
			if os.Getenv("UPDATE_GOLDEN") != "" {
				if err := os.WriteFile(path, []byte(got+"\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if got != strings.TrimRight(string(want), "\n") {
				t.Errorf("output differs from the golden\n--- got ---\n%s\n--- want ---\n%s", got, want)
			}
		})
	}
}

func TestASCIIStaysInsideTheWidthAndIsPlainASCII(t *testing.T) {
	for _, spec := range []string{"doc", "states", "real"} {
		got, err := RenderASCII(load(t, spec+".flow.json"))
		if err != nil {
			t.Fatal(err)
		}
		for _, l := range strings.Split(got, "\n") {
			if len([]rune(l)) > convention.MaxASCIIWidth {
				t.Errorf("%s: %q is wider than %d", spec, l, convention.MaxASCIIWidth)
			}
			if strings.TrimRight(l, " ") != l {
				t.Errorf("%s: %q has trailing spaces", spec, l)
			}
			for _, r := range l {
				if r > 126 {
					t.Errorf("%s: %q holds a non-ASCII character %q", spec, l, r)
				}
			}
		}
	}
}

// What the gate relies on: the text parses back to a flow that draws the same text.
func TestASCIIRoundTrips(t *testing.T) {
	for _, spec := range []string{"doc", "states", "real"} {
		src, err := RenderASCII(load(t, spec+".flow.json"))
		if err != nil {
			t.Fatal(err)
		}
		f, err := ParseASCII(src)
		if err != nil {
			t.Fatalf("%s: %v\n%s", spec, err, src)
		}
		again, err := RenderASCII(f)
		if err != nil {
			t.Fatal(err)
		}
		if again != src {
			t.Errorf("%s: parse then render changed the text\n--- first ---\n%s\n--- again ---\n%s", spec, src, again)
		}
		if FlowASCIILegend(f) != FlowASCIILegend(load(t, spec+".flow.json")) {
			t.Errorf("%s: the legend changed across the round trip", spec)
		}
	}
}

func TestASCIIDocFlowShape(t *testing.T) {
	got, err := RenderASCII(load(t, "doc.flow.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"( Input 1 )", "[+ InviteCommand.Handle]", "[+ !CodeGenerator.Next]", "[. UserQuery.Get]",
		"[( Output 1 )]", "> Output 2 ]",
		"Input 1 ==> InviteCommand.Handle", "InviteCommand.Handle --> UserQuery.Get",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in\n%s", want, got)
		}
	}
	if l := FlowASCIILegend(load(t, "doc.flow.json")); l != "**Legend:** [+] added · [.] unchanged context · ! risk · ==> new · --> existing" {
		t.Errorf("legend = %q", l)
	}
}

func TestASCIIRefusesWhatItCannotDrawBack(t *testing.T) {
	dup := load(t, "doc.flow.json")
	dup.Nodes[2].Label = dup.Nodes[0].Label
	if _, err := RenderASCII(dup); err == nil || !strings.Contains(err.Error(), "different label") {
		t.Errorf("two nodes with one label: err = %v", err)
	}
	arrow := load(t, "doc.flow.json")
	arrow.Nodes[0].Label = "a ==> b"
	if _, err := RenderASCII(arrow); err == nil {
		t.Error("a label with an arrow must be refused")
	}
}

func TestParseASCIIRejectsHandEditedShapes(t *testing.T) {
	src, _ := RenderASCII(load(t, "doc.flow.json"))
	cases := map[string]string{
		"bad header":    strings.Replace(src, "INPUT", "IN", 1),
		"bad function":  strings.Replace(src, "[+ InviteCommand.Handle]", "(+ InviteCommand.Handle)", 1),
		"bad output":    strings.Replace(src, "> Output 2 ]", ">> Output 2 ]", 1),
		"unknown edge":  src + "\nInput 1 ==> Nowhere",
		"no arrow":      src + "\nInput 1 to InviteCommand.Handle",
		"back to input": src + "\nInviteCommand.Handle ==> Input 1",
	}
	for name, bad := range cases {
		if _, err := ParseASCII(bad); err == nil {
			t.Errorf("%s: accepted\n%s", name, bad)
		}
	}
}

func TestASCIIMarkersFollowTheStatus(t *testing.T) {
	f := Flow{Version: 1,
		Inputs:  []Port{{ID: "I1"}},
		Nodes:   []Node{{ID: "F1", Label: "A", Status: "modified"}, {ID: "F2", Label: "B", Status: "removed"}, {ID: "F3", Label: "C", Status: "added", RiskScore: 2}},
		Outputs: []Port{{ID: "O1"}},
		Edges:   []Edge{{From: "I1", To: "F1", Status: "new"}, {From: "F1", To: "F2", Status: "removed"}, {From: "F1", To: "F3", Status: "new"}, {From: "F3", To: "O1", Status: "new"}},
	}
	got, err := RenderASCII(f)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"[~ A]", "[- B]", "[+ !C]", "[ Output 1 ]", "A -.-> B"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in\n%s", want, got)
		}
	}
}
