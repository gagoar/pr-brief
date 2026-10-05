package initfiles

import (
	"os"
	"strings"
	"testing"

	"github.com/gagoar/pr-brief/internal/gate"
)

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// withoutComments drops comment and blank lines, so files can differ in their header notes.
func withoutComments(s string) string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		t := strings.TrimSpace(l)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		out = append(out, strings.TrimRight(l, " "))
	}
	return strings.Join(out, "\n")
}

func TestWorkflowTemplateMatchesTheExample(t *testing.T) {
	want := withoutComments(Workflow())
	if got := withoutComments(readFile(t, "../../examples/pr-brief-workflow.yml")); got != want {
		t.Errorf("examples/pr-brief-workflow.yml differs from the init template:\n--- example ---\n%s\n--- template ---\n%s", got, want)
	}
}

func TestRepoWorkflowIsTheTemplateWithTheLocalAction(t *testing.T) {
	want := withoutComments(strings.Replace(Workflow(), "gagoar/pr-brief@v0", "./", 1))
	if got := withoutComments(readFile(t, "../../.github/workflows/pr-brief.yml")); got != want {
		t.Errorf(".github/workflows/pr-brief.yml differs from the init template:\n--- repo ---\n%s\n--- template ---\n%s", got, want)
	}
}

func TestRepoPRTemplateIsTheInitTemplate(t *testing.T) {
	want := PRTemplate("ste+iceberg", "github-dark")
	if got := readFile(t, "../../.github/pull_request_template.md"); got != want {
		t.Errorf(".github/pull_request_template.md differs from what `pr-brief init --pr-template` writes")
	}
}

// The skeleton is a real description shape: the markers and sections parse, and the
// gate asks for the missing pieces instead of failing on the structure.
func TestPRTemplateParsesButIsNotAPassingDescription(t *testing.T) {
	res := gate.Check(PRTemplate("iceberg", "github-dark"), gate.Options{})
	if res.OK() {
		t.Fatal("an unfilled template must not pass the gate")
	}
	for _, f := range res.Findings {
		if f.Rule == "markers" || f.Rule == "sections" {
			t.Errorf("the template's structure should parse; got %+v", f)
		}
	}
}
