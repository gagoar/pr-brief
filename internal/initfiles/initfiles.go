// Package initfiles holds the files that `pr-brief init` writes into a repository.
package initfiles

import (
	"embed"
	"strings"
)

//go:embed templates/*
var fs embed.FS

func read(name string) string {
	b, err := fs.ReadFile("templates/" + name)
	if err != nil {
		panic(err) // the templates are embedded; a missing one is a build error
	}
	return string(b)
}

// PRTemplate is .github/pull_request_template.md: the description skeleton, so a
// contributor without the plugin sees what the gate expects.
func PRTemplate(style, themeID string) string {
	return strings.NewReplacer("{{STYLE}}", style, "{{THEME}}", themeID).Replace(read("pull_request_template.md"))
}

// Workflow is .github/workflows/pr-brief.yml. It checks out the base commit, so a PR
// cannot edit the rules that check it.
func Workflow() string { return read("workflow.yml") }

// WorkflowInline is the same workflow with the style and theme written in it, for a repo
// that keeps no .pr-brief.json. A .pr-brief.json on the base branch still wins.
func WorkflowInline(style, theme string) string {
	w := Workflow()
	const use = "      - uses: gagoar/pr-brief@v0\n"
	with := use +
		"        # Used only when the base branch has no .pr-brief.json. The file wins.\n" +
		"        with:\n" +
		"          style: " + style + "\n" +
		"          theme: " + theme + "\n"
	return strings.Replace(w, use, with, 1)
}

// AzurePipeline is .azuredevops/pr-brief.yml. version is the release to download; a
// development build downloads the latest release. With inline set, the style and theme
// are written in the pipeline.
func AzurePipeline(version string, inline bool, style, theme string) string {
	url := "https://github.com/gagoar/pr-brief/releases/latest/download"
	if version != "" && !strings.Contains(version, "dev") {
		url = "https://github.com/gagoar/pr-brief/releases/download/v" + strings.TrimPrefix(version, "v")
	}
	env := ""
	if inline {
		env = "      # Used only when the target branch has no .pr-brief.json. The file wins.\n" +
			"      PR_BRIEF_STYLE: " + style + "\n" +
			"      PR_BRIEF_THEME: " + theme + "\n"
	}
	out := strings.NewReplacer("{{RELEASE_URL}}", url, "{{INLINE_ENV}}\n", env).Replace(read("azure-pipelines.yml"))
	return out
}

// Auth names the way the rewrite job signs in to Claude. Each team picks one.
const (
	AuthAPIKey     = "api-key"    // an ANTHROPIC_API_KEY secret: simple, but a long-lived secret
	AuthFederation = "federation" // Anthropic workload identity federation over GitHub OIDC: no stored secret
	AuthBedrock    = "bedrock"    // Amazon Bedrock over GitHub OIDC
	AuthVertex     = "vertex"     // Google Vertex AI over GitHub OIDC
)

// AuthModes lists the values --auth accepts.
var AuthModes = []string{AuthFederation, AuthBedrock, AuthVertex, AuthAPIKey}

// WorkflowRewrite is .github/workflows/pr-brief.yml with a rewrite job in front of the gate.
// The rewrite job runs the Claude Code GitHub Action. It stays off until the credentials for
// auth exist. With inline set, the gate step carries the style and theme.
func WorkflowRewrite(auth string, inline bool, style, theme string) string {
	var (
		perms, probe, missing, login, inputs string
	)
	switch auth {
	case AuthAPIKey:
		probe = "          HAS_AUTH: ${{ secrets.ANTHROPIC_API_KEY != '' }}\n"
		missing = "the secret ANTHROPIC_API_KEY is not set"
		inputs = "          anthropic_api_key: ${{ secrets.ANTHROPIC_API_KEY }}\n"
	case AuthFederation:
		perms = "      id-token: write # the job swaps its GitHub OIDC token for a Claude token. No stored secret.\n"
		probe = "          HAS_AUTH: ${{ vars.ANTHROPIC_FEDERATION_RULE_ID != '' }}\n"
		missing = "the repository variables for workload identity federation are not set (ANTHROPIC_FEDERATION_RULE_ID and three more)"
		inputs = "          anthropic_federation_rule_id: ${{ vars.ANTHROPIC_FEDERATION_RULE_ID }}\n" +
			"          anthropic_organization_id: ${{ vars.ANTHROPIC_ORG_ID }}\n" +
			"          anthropic_service_account_id: ${{ vars.ANTHROPIC_SERVICE_ACCOUNT_ID }}\n" +
			"          anthropic_workspace_id: ${{ vars.ANTHROPIC_WORKSPACE_ID }}\n"
	case AuthBedrock:
		perms = "      id-token: write # the job assumes an AWS role with its GitHub OIDC token. No stored secret.\n"
		probe = "          HAS_AUTH: ${{ vars.AWS_ROLE_TO_ASSUME != '' }}\n"
		missing = "the repository variable AWS_ROLE_TO_ASSUME is not set"
		login = "      - uses: aws-actions/configure-aws-credentials@v4\n" +
			"        if: steps.auth.outputs.ready == 'true'\n" +
			"        with:\n" +
			"          role-to-assume: ${{ vars.AWS_ROLE_TO_ASSUME }}\n" +
			"          aws-region: ${{ vars.AWS_REGION }}\n"
		inputs = "          use_bedrock: 'true'\n"
	case AuthVertex:
		perms = "      id-token: write # the job signs in to Google Cloud with its GitHub OIDC token. No stored secret.\n"
		probe = "          HAS_AUTH: ${{ vars.GCP_WORKLOAD_IDENTITY_PROVIDER != '' }}\n"
		missing = "the repository variable GCP_WORKLOAD_IDENTITY_PROVIDER is not set"
		login = "      - uses: google-github-actions/auth@v2\n" +
			"        if: steps.auth.outputs.ready == 'true'\n" +
			"        with:\n" +
			"          workload_identity_provider: ${{ vars.GCP_WORKLOAD_IDENTITY_PROVIDER }}\n" +
			"          service_account: ${{ vars.GCP_SERVICE_ACCOUNT }}\n"
		inputs = "          use_vertex: 'true'\n"
	}
	gate, prompt := "", ""
	if inline {
		prompt = "            This repository has no .pr-brief.json. The settings for this run are style " + style + " and theme " + theme + ".\n"
		gate = "          # Used only when the base branch has no .pr-brief.json. The file wins.\n" +
			"          style: " + style + "\n" +
			"          theme: " + theme + "\n"
	}
	out := strings.NewReplacer(
		"{{REWRITE_PERMISSIONS}}", perms,
		"{{AUTH_PROBE}}", probe,
		"{{AUTH_MISSING}}", missing,
		"{{CLOUD_LOGIN}}", login,
		"{{AUTH_INPUTS}}", inputs,
		"{{GATE_INPUTS}}\n", gate,
		"{{PROMPT_SETTINGS}}", prompt,
	).Replace(read("workflow-rewrite.yml"))
	return strings.TrimRight(out, "\n") + "\n"
}

// Where the files go, relative to the repo root.
const (
	PRTemplatePath      = ".github/pull_request_template.md"
	AzurePRTemplatePath = ".azuredevops/pull_request_template.md"
	WorkflowPath        = ".github/workflows/pr-brief.yml"
	AzurePipelinePath   = ".azuredevops/pr-brief.yml"
)
