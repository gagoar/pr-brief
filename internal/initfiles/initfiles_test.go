package initfiles

import (
	"os"
	"regexp"
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

func TestInlineWorkflowAddsTheInputsAndNothingElse(t *testing.T) {
	got := WorkflowInline("ste", "dracula")
	if !strings.Contains(got, "with:\n          style: ste\n          theme: dracula\n") {
		t.Errorf("the inputs are missing:\n%s", got)
	}
	if strings.Contains(got, "secrets.") {
		t.Errorf("the check needs no secret:\n%s", got)
	}
	drop := withoutComments(got)
	drop = strings.Replace(drop, "\n        with:\n          style: ste\n          theme: dracula", "", 1)
	if drop != withoutComments(Workflow()) {
		t.Errorf("the inline workflow must equal the file workflow plus the inputs")
	}
}

func TestAzurePipelineNeedsNoPersonalAccessToken(t *testing.T) {
	for _, inline := range []bool{false, true} {
		got := AzurePipeline("0.3.0", inline, "iceberg", "alucard")
		if !strings.Contains(got, "$(System.AccessToken)") {
			t.Error("the pipeline must use the job's own System.AccessToken")
		}
		if strings.Contains(got, "{{") || strings.Contains(got, "secrets.") {
			t.Errorf("an unfilled placeholder or a secret in the pipeline:\n%s", got)
		}
		if !strings.Contains(got, "releases/download/v0.3.0") || !strings.Contains(got, "sha256sum -c") || !strings.Contains(got, "checkout: none") {
			t.Errorf("the binary must come from a pinned release and be checked:\n%s", got)
		}
		if has := strings.Contains(got, "PR_BRIEF_STYLE: iceberg") && strings.Contains(got, "PR_BRIEF_THEME: alucard"); has != inline {
			t.Errorf("inline=%v but the pipeline variables are present=%v", inline, has)
		}
	}
	if !strings.Contains(AzurePipeline("0.1.0-dev", false, "", ""), "releases/latest/download") {
		t.Error("a development build downloads the latest release")
	}
}

func TestRewriteWorkflowExampleIsTheFederationTemplate(t *testing.T) {
	want := withoutComments(WorkflowRewrite(AuthFederation, false, "", ""))
	if got := withoutComments(readFile(t, "../../examples/pr-brief-rewrite-workflow.yml")); got != want {
		t.Errorf("examples/pr-brief-rewrite-workflow.yml differs from `init --rewrite --auth federation`:\n--- example ---\n%s\n--- template ---\n%s", got, want)
	}
}

func TestRewriteWorkflowKeepsTheGateReadOnlyAndOffUntilConfigured(t *testing.T) {
	for _, auth := range AuthModes {
		got := WorkflowRewrite(auth, false, "", "")
		if m := regexp.MustCompile(`\{\{[A-Z_]+\}\}`).FindString(got); m != "" {
			t.Errorf("%s: an unfilled placeholder %s", auth, m)
		}
		rewrite, gate, _ := strings.Cut(got, "\n  description:")
		if strings.Contains(gate, ": write") || strings.Contains(gate, "claude-code-action") || strings.Contains(gate, "secrets.") {
			t.Errorf("%s: the gate job must be read-only, with no agent and no secret:\n%s", auth, gate)
		}
		if !strings.Contains(gate, "refresh: ${{ needs.rewrite.result == 'success' }}") || !strings.Contains(gate, "pull_request.base.sha") {
			t.Errorf("%s: the gate must read the live description, from the base commit:\n%s", auth, gate)
		}
		if !strings.Contains(rewrite, "HAS_AUTH") || strings.Count(rewrite, "steps.auth.outputs.ready == 'true'") < 3 {
			t.Errorf("%s: every rewrite step after the probe must wait for the credentials:\n%s", auth, rewrite)
		}
		if !strings.Contains(rewrite, "head.repo.full_name == github.repository") {
			t.Errorf("%s: fork PRs must not reach the agent", auth)
		}
		if oidc := strings.Contains(rewrite, "id-token: write"); oidc != (auth != AuthAPIKey) {
			t.Errorf("%s: id-token: write must be present only for OIDC sign-in (present=%v)", auth, oidc)
		}
		if secret := strings.Contains(rewrite, "secrets."); secret != (auth == AuthAPIKey) {
			t.Errorf("%s: only the api-key mode reads a secret (present=%v)", auth, secret)
		}
		if !strings.Contains(rewrite, "32511c6992ecb5f1971e46a2943f2e6adceedafe") {
			t.Errorf("%s: the asd-ste100 skill must be pinned to a commit", auth)
		}
	}
}

func TestRewriteWorkflowCarriesInlineSettings(t *testing.T) {
	got := WorkflowRewrite(AuthAPIKey, true, "iceberg", "dracula")
	for _, want := range []string{"          style: iceberg\n          theme: dracula\n", "style iceberg and theme dracula"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(WorkflowRewrite(AuthAPIKey, false, "", ""), "no .pr-brief.json") {
		t.Error("a repo with a config file needs no settings in the prompt")
	}
}
