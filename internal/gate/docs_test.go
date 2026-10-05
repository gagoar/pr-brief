package gate

import (
	"encoding/json"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"

	dg "github.com/gagoar/pr-brief/internal/diagram"
	"github.com/gagoar/pr-brief/internal/theme"
)

func readDoc(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func firstMermaid(t *testing.T, doc, name string) string {
	t.Helper()
	m := regexp.MustCompile("(?s)```mermaid\n(.*?)\n```").FindStringSubmatch(doc)
	if m == nil {
		t.Fatalf("no mermaid example in %s", name)
	}
	return m[1]
}

func docFlow(t *testing.T) dg.Flow {
	t.Helper()
	data, err := os.ReadFile("../diagram/testdata/doc.flow.json")
	if err != nil {
		t.Fatal(err)
	}
	f, err := dg.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// The documented example must be exactly what `pr-brief diagram` prints, so the docs
// cannot drift from the code, and it must pass the gate.
func TestDocumentedExamplesAreTheToolsOutput(t *testing.T) {
	want, err := dg.Render(docFlow(t), theme.Default())
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"../../skills/pr-brief/references/diagram-convention.md", "../../README.md"} {
		got := firstMermaid(t, readDoc(t, path), path)
		if got != want {
			t.Errorf("%s: the example is not the tool's output.\n--- doc ---\n%s\n--- tool ---\n%s", path, got, want)
		}
		d := parseDiagram(got)
		if len(d.problems) > 0 {
			t.Errorf("%s: the example fails the gate: %v", path, d.problems)
		}
		if p := d.styleProblems(theme.Default()); len(p) > 0 {
			t.Errorf("%s: the example fails the style check: %v", path, p)
		}
	}
}

func TestDocumentedFlowJSONMatchesTheFixture(t *testing.T) {
	doc := readDoc(t, "../../skills/pr-brief/references/diagram-convention.md")
	m := regexp.MustCompile("(?s)```json\n(.*?)\n```").FindStringSubmatch(doc)
	if m == nil {
		t.Fatal("no flow JSON in diagram-convention.md")
	}
	var fromDoc dg.Flow
	if err := json.Unmarshal([]byte(m[1]), &fromDoc); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fromDoc, docFlow(t)) {
		t.Error("the flow JSON in diagram-convention.md differs from internal/diagram/testdata/doc.flow.json")
	}
}

func TestBodyTemplateUsesTheRealSectionNames(t *testing.T) {
	doc := readDoc(t, "../../skills/pr-brief/references/body-template.md")
	for _, want := range []string{"<!-- pr-brief:begin v1 style=ste+iceberg theme=github-dark -->", "## Brief", "## Change map", "## Review guide", "**What changed**", "**Read these first**", "**Review order**", "<!-- pr-brief:end -->"} {
		if !strings.Contains(doc, want) {
			t.Errorf("body-template.md lacks %q", want)
		}
	}
}

func TestDocsNameEveryBuiltInThemeAndTheThreeSettings(t *testing.T) {
	readme := readDoc(t, "../../README.md")
	conv := readDoc(t, "../../skills/pr-brief/references/diagram-convention.md")
	for _, n := range theme.Names() {
		if !strings.Contains(readme, "`"+n+"`") || !strings.Contains(conv, "`"+n+"`") {
			t.Errorf("README and diagram-convention.md must both name the built-in theme %s", n)
		}
	}
	for _, doc := range []string{readme, readDoc(t, "../../skills/pr-brief/SKILL.md"), readDoc(t, "../../skills/pr-brief/references/convention.md")} {
		if strings.Contains(strings.ToLower(doc), "two settings") || strings.Contains(strings.ToLower(doc), "only two") {
			t.Error("a document still says there are two settings")
		}
	}
}
