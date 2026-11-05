package diagram

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gagoar/pr-brief/internal/convention"
)

func drawing(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "drawings", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestFlowCommentRoundTrips(t *testing.T) {
	for _, spec := range []string{"doc", "states", "real"} {
		f := load(t, spec+".flow.json")
		c := FlowComment(f)
		if strings.Contains(c, "\n") || !strings.HasPrefix(c, FlowCommentPrefix) || !strings.HasSuffix(c, " -->") {
			t.Fatalf("%s: not a one-line comment: %q", spec, c)
		}
		got, err := ParseFlowComment(c)
		if err != nil {
			t.Fatalf("%s: %v\n%s", spec, err, c)
		}
		if FlowComment(got) != c {
			t.Errorf("%s: comment, parse, comment changed the text", spec)
		}
		if strings.Join(boxTexts(got), "|") != strings.Join(boxTexts(f), "|") {
			t.Errorf("%s: the boxes changed across the round trip", spec)
		}
		if FlowASCIILegend(got) != FlowASCIILegend(f) {
			t.Errorf("%s: the legend changed across the round trip", spec)
		}
	}
}

func TestFlowCommentEscapesLabelsThatWouldBreakIt(t *testing.T) {
	f := load(t, "doc.flow.json")
	f.Nodes[0].Label = "a;b|c-d>e %f"
	c := FlowComment(f)
	if strings.Count(c, ";") != 3 || strings.Contains(strings.TrimSuffix(c, " -->"), "-->") {
		t.Fatalf("label broke the comment: %q", c)
	}
	got, err := ParseFlowComment(c)
	if err != nil || got.Nodes[0].Label != "a;b|c-d>e %f" {
		t.Errorf("label did not survive: %v %q", err, got.Nodes[0].Label)
	}
}

func TestParseFlowCommentRejectsBadInput(t *testing.T) {
	good := FlowComment(load(t, "doc.flow.json"))
	for name, bad := range map[string]string{
		"not a comment":    "hello",
		"three sections":   strings.Replace(good, ";", ",", 1),
		"bad function":     strings.Replace(good, "F1+_", "F1?_", 1),
		"bad call":         strings.Replace(good, "I1=F1", "I1~F1", 1),
		"call to nowhere":  strings.Replace(good, "I1=F1", "I1=F9", 1),
		"call into input":  strings.Replace(good, "I1=F1", "F1=I1", 1),
		"empty everything": FlowCommentPrefix + ";;; -->",
	} {
		if bad == good {
			t.Fatalf("%s: the edit changed nothing", name)
		}
		if _, err := ParseFlowComment(bad); err == nil {
			t.Errorf("%s: accepted: %s", name, bad)
		}
	}
}

func TestCellsReadTheBoxesAndSkipTheArrows(t *testing.T) {
	cells := Cells(drawing(t, "doc.txt"))
	want := []string{"Input 1", "+ InviteCommand.Handle", "+ !CodeGenerator.Next", ". UserQuery.Get", "Output 1", "Output 2"}
	if strings.Join(cells, "|") != strings.Join(want, "|") {
		t.Errorf("cells = %q\nwant  %q", cells, want)
	}
	f := load(t, "doc.flow.json")
	if p := CellProblems(f, cells); len(p) != 0 {
		t.Errorf("the drawing for doc.flow.json has problems: %v", p)
	}
}

func TestCellProblemsNameWhatIsWrong(t *testing.T) {
	f := load(t, "doc.flow.json")
	d := drawing(t, "doc.txt")
	cases := map[string]struct{ edit, want string }{
		"a missing box":   {strings.Replace(d, "| Output 2 |", "|          |", 1), "no box with the text `Output 2`"},
		"a wrong status":  {strings.Replace(d, "| + InviteCommand.Handle |", "| ~ InviteCommand.Handle |", 1), "no box with the text `+ InviteCommand.Handle`"},
		"a misspelling":   {strings.Replace(d, "UserQuery.Get", "UserQuery.Gett", 1), "a box that is not in the flow: `. UserQuery.Gett`"},
		"a missing risk":  {strings.Replace(d, "!CodeGenerator", "CodeGenerator", 1), "no box with the text `+ !CodeGenerator.Next`"},
		"an extra box":    {d + "| + Surprise |\n", "a box that is not in the flow: `+ Surprise`"},
		"a repeated box":  {d + "| Output 1 |\n", "a box that is not in the flow: `Output 1`"},
		"the placeholder": {Placeholder, "no box with the text"},
	}
	for name, c := range cases {
		if c.edit == d {
			t.Fatalf("%s: the edit changed nothing", name)
		}
		p := strings.Join(CellProblems(f, Cells(c.edit)), " / ")
		if !strings.Contains(p, c.want) {
			t.Errorf("%s: problems = %q, want %q", name, p, c.want)
		}
	}
}

func TestWidthProblems(t *testing.T) {
	if p := WidthProblems(drawing(t, "doc.txt")); len(p) != 0 {
		t.Errorf("doc drawing: %v", p)
	}
	wide := strings.Repeat("x", convention.MaxASCIIWidth+1)
	if p := WidthProblems("ok\n" + wide); len(p) != 1 || !strings.Contains(p[0], "line 2") {
		t.Errorf("wide line: %v", p)
	}
	if p := WidthProblems("a\tb"); len(p) != 1 || !strings.Contains(p[0], "tab") {
		t.Errorf("tab: %v", p)
	}
}

func TestSpecListsEveryBoxAndCall(t *testing.T) {
	f := load(t, "doc.flow.json")
	s := Spec(f)
	for _, want := range []string{
		"Input 1", "+ InviteCommand.Handle", "+ !CodeGenerator.Next", ". UserQuery.Get", "Output 1", "Output 2",
		"InviteCommand.Handle -> !CodeGenerator.Next   new", "InviteCommand.Handle -> UserQuery.Get   existing",
		"risk: top and bottom border of =", "78 characters wide",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("the spec lacks %q:\n%s", want, s)
		}
	}
}

func TestLegendNamesOnlyWhatTheFlowUses(t *testing.T) {
	got := FlowASCIILegend(load(t, "doc.flow.json"))
	want := "**Legend:** [+] added · [.] unchanged context · ! risk (= border) · = # new call · - | existing call"
	if got != want {
		t.Errorf("legend = %q\nwant     %q", got, want)
	}
	if l := FlowASCIILegend(Flow{}); l != "" {
		t.Errorf("an empty flow has a legend: %q", l)
	}
}
