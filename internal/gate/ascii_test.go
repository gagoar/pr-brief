package gate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gagoar/pr-brief/internal/convention"
	dg "github.com/gagoar/pr-brief/internal/diagram"
	"github.com/gagoar/pr-brief/internal/theme"
)

// drawingFor is a hand-made drawing of testFlow, the kind the model makes from the spec.
func drawingFor(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "diagram", "testdata", "drawings", "gate.txt"))
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimRight(string(data), "\n")
}

// goodASCII is the good body drawn as plain text: the flow comment, the drawing in a text
// fence, the Legend line, and format=ascii in the begin marker.
func goodASCII(t *testing.T) string {
	t.Helper()
	block := dg.FlowComment(testFlow) + "\n```text\n" + drawingFor(t) + "\n```"
	s := strings.Replace(good, "```mermaid\n"+render(theme.Default())+"\n```", block, 1)
	s = strings.Replace(s, dg.FlowLegend(testFlow), dg.FlowASCIILegend(testFlow), 1)
	return strings.Replace(s, "theme=github-dark -->", "theme=github-dark format=ascii -->", 1)
}

func TestASCIIDescriptionPasses(t *testing.T) {
	body := goodASCII(t)
	if !strings.Contains(body, "```text") || !strings.Contains(body, "format=ascii") || !strings.Contains(body, dg.FlowCommentPrefix) {
		t.Fatalf("the fixture was not built:\n%s", body)
	}
	if r := Check(body, Options{}); !r.OK() {
		t.Fatalf("a plain-text description failed: %+v", r.Findings)
	} else if r.Format != convention.FormatASCII {
		t.Errorf("format = %q", r.Format)
	}
}

func TestASCIIMarkerMayLeaveTheThemeOut(t *testing.T) {
	body := strings.Replace(goodASCII(t), " theme=github-dark", "", 1)
	if r := Check(body, Options{}); !r.OK() {
		t.Fatalf("a plain-text description has no colours, so it needs no theme: %+v", r.Findings)
	}
	// Mermaid still needs one.
	if r := Check(strings.Replace(good, " theme=github-dark", "", 1), Options{}); r.OK() {
		t.Error("a Mermaid description without a theme must still fail")
	}
}

// The gate checks that the boxes are the flow's boxes. It cannot check the lines between them,
// and these tests say so: an arrow edit passes.
func TestASCIIBoxEditsFail(t *testing.T) {
	body := goodASCII(t)
	for name, c := range map[string]struct{ bad, want string }{
		"a wrong status":   {strings.Replace(body, "| + InviteCommand.Handle |", "| ~ InviteCommand.Handle |", 1), "no box with the text `+ InviteCommand.Handle`"},
		"a dropped risk":   {strings.Replace(body, "| ~ !CodeGenerator.Next |", "| ~ CodeGenerator.Next  |", 1), "no box with the text `~ !CodeGenerator.Next`"},
		"a missing box":    {strings.Replace(body, "| Output 2 |", "|          |", 1), "no box with the text `Output 2`"},
		"an extra box":     {strings.Replace(body, "\n```\n"+dg.FlowASCIILegend(testFlow), "\n| + Surprise |\n```\n"+dg.FlowASCIILegend(testFlow), 1), "a box that is not in the flow: `+ Surprise`"},
		"the placeholder":  {strings.Replace(body, drawingFor(t), dg.Placeholder, 1), "still the placeholder"},
		"a too wide line":  {strings.Replace(body, "| Input 1 |", "| Input 1 |"+strings.Repeat(" ", 80), 1), "characters wide"},
		"no flow comment":  {strings.Replace(body, dg.FlowComment(testFlow)+"\n", "", 1), "flow comment"},
		"a broken comment": {strings.Replace(body, "I1=F1", "I1~F1", 1), "flow comment is not valid"},
	} {
		if c.bad == body {
			t.Fatalf("%s: the edit changed nothing", name)
		}
		has(t, Check(c.bad, Options{}), "diagram", c.want)
	}
}

func TestASCIIArrowEditsPassBecauseTheGateCannotSeeThem(t *testing.T) {
	body := goodASCII(t)
	flipped := strings.Replace(body, "V            V", "^            ^", 1)
	if flipped == body {
		t.Fatal("the edit changed nothing")
	}
	if r := Check(flipped, Options{}); !r.OK() {
		t.Errorf("an arrowhead edit failed; the docs say the gate does not check lines: %+v", r.Findings)
	}
}

func TestASCIIDescriptionRefusesAMermaidFence(t *testing.T) {
	body := strings.Replace(good, "theme=github-dark -->", "theme=github-dark format=ascii -->", 1)
	has(t, Check(body, Options{}), "diagram", "format=ascii")
}

func TestASCIIKeepsTheLegendAndReferencesRules(t *testing.T) {
	body := goodASCII(t)
	has(t, Check(strings.Replace(body, dg.FlowASCIILegend(testFlow), "", 1), Options{}), "diagram", "Legend")
	has(t, Check(strings.Replace(body, "| Output 1 | `invites` table | The service adds one row. |\n", "", 1), Options{}), "references", "O1")
	has(t, Check(strings.Replace(body, "### Flow 1: Input 1 -> invite created\n", "", 1), Options{}), "diagram", "heading")
}

func TestConfigFormatIsEnforced(t *testing.T) {
	// The repo config says ascii; a Mermaid description fails, and the reverse.
	has(t, Check(good, Options{Format: convention.FormatASCII}), "markers", "diagram.format")
	has(t, Check(goodASCII(t), Options{Format: convention.FormatMermaid}), "markers", "diagram.format")
	if r := Check(goodASCII(t), Options{Format: convention.FormatASCII}); !r.OK() {
		t.Errorf("matching config failed: %+v", r.Findings)
	}
	// auto and empty accept what the marker says.
	for _, f := range []string{"", convention.FormatAuto} {
		if r := Check(goodASCII(t), Options{Format: f}); !r.OK() {
			t.Errorf("format %q failed: %+v", f, r.Findings)
		}
	}
}

func TestUnknownMarkerFormatFails(t *testing.T) {
	body := strings.Replace(good, "theme=github-dark -->", "theme=github-dark format=svg -->", 1)
	has(t, Check(body, Options{}), "markers", "format")
}
