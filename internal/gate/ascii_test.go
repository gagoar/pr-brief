package gate

import (
	"strings"
	"testing"

	"github.com/gagoar/pr-brief/internal/convention"
	dg "github.com/gagoar/pr-brief/internal/diagram"
	"github.com/gagoar/pr-brief/internal/theme"
)

// goodASCII is the good body drawn as ASCII: the same text the tool prints, in a text fence,
// with format=ascii in the begin marker.
func goodASCII(t *testing.T) string {
	t.Helper()
	src, err := dg.RenderASCII(testFlow)
	if err != nil {
		t.Fatal(err)
	}
	s := strings.Replace(good, "```mermaid\n"+render(theme.Default())+"\n```", "```text\n"+src+"\n```", 1)
	s = strings.Replace(s, dg.FlowLegend(testFlow), dg.FlowASCIILegend(testFlow), 1)
	return strings.Replace(s, "theme=github-dark -->", "theme=github-dark format=ascii -->", 1)
}

func TestASCIIDescriptionPasses(t *testing.T) {
	body := goodASCII(t)
	if !strings.Contains(body, "```text") || !strings.Contains(body, "format=ascii") {
		t.Fatalf("the fixture was not built:\n%s", body)
	}
	if r := Check(body, Options{}); !r.OK() {
		t.Fatalf("an ASCII description from the tool failed: %+v", r.Findings)
	} else if r.Format != convention.FormatASCII {
		t.Errorf("format = %q", r.Format)
	}
}

func TestASCIIMarkerMayLeaveTheThemeOut(t *testing.T) {
	body := strings.Replace(goodASCII(t), " theme=github-dark", "", 1)
	if r := Check(body, Options{}); !r.OK() {
		t.Fatalf("an ASCII description has no colours, so it needs no theme: %+v", r.Findings)
	}
	// Mermaid still needs one.
	if r := Check(strings.Replace(good, " theme=github-dark", "", 1), Options{}); r.OK() {
		t.Error("a Mermaid description without a theme must still fail")
	}
}

// Like the Mermaid check, the gate checks the form and that the parts agree (the text is
// what the tool prints for the flow it reads back, and the legend and References match). It
// cannot know the flow is true. These are the edits that break the form or the agreement.
func TestASCIIHandEditsFail(t *testing.T) {
	body := goodASCII(t)
	for name, bad := range map[string]string{
		"a moved marker":             strings.Replace(body, "[+ InviteCommand.Handle]", "[+  InviteCommand.Handle]", 1),
		"a status the legend lacks":  strings.Replace(body, "[+ InviteCommand.Handle]", "[- InviteCommand.Handle]", 1),
		"a stray line":               strings.Replace(body, "\n```\n"+dg.FlowASCIILegend(testFlow), "\nsee the code\n```\n"+dg.FlowASCIILegend(testFlow), 1),
		"an edge to an unknown name": strings.Replace(body, "Input 1 ==> InviteCommand.Handle", "Input 1 ==> Nowhere", 1),
		"a reshaped output":          strings.Replace(body, "> Output 2 ]", ">> Output 2 ]", 1),
	} {
		if bad == body {
			t.Fatalf("%s: the edit changed nothing", name)
		}
		if r := Check(bad, Options{}); r.OK() {
			t.Errorf("%s: passed", name)
		}
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
