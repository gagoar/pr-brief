package gate

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// The example in the reference doc must pass the real parser, so the docs cannot
// drift away from what the gate enforces.
func TestDiagramConventionExamplePassesTheGate(t *testing.T) {
	data, err := os.ReadFile("../../skills/pr-brief/references/diagram-convention.md")
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile("(?s)```mermaid\n(.*?)\n```")
	m := re.FindStringSubmatch(string(data))
	if m == nil {
		t.Fatal("no mermaid example found in diagram-convention.md")
	}
	d := parseDiagram(m[1])
	if len(d.problems) > 0 {
		t.Errorf("the documented example fails the gate: %v", d.problems)
	}
	for _, class := range []string{"added", "modified", "removed", "context", "risk", "riskadded"} {
		if !strings.Contains(m[1], "classDef "+class+" ") {
			t.Errorf("example lacks classDef %s", class)
		}
	}
}

func TestBodyTemplateUsesTheRealSectionNames(t *testing.T) {
	data, err := os.ReadFile("../../skills/pr-brief/references/body-template.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<!-- pr-brief:begin v1", "## Brief", "## Change map", "## Review guide", "**What changed**", "**Read these first**", "**Review order**", "<!-- pr-brief:end -->"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("body-template.md lacks %q", want)
		}
	}
}
