package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gagoar/pr-brief/internal/config"
	"github.com/gagoar/pr-brief/internal/convention"
	"github.com/gagoar/pr-brief/internal/host"
	"github.com/gagoar/pr-brief/internal/initfiles"
	"github.com/gagoar/pr-brief/internal/theme"
)

// writeNew writes a file only if it does not exist. It reports what it did.
func writeNew(root, rel, content string, stdout io.Writer) error {
	p := filepath.Join(root, rel)
	if _, err := os.Stat(p); err == nil {
		fmt.Fprintf(stdout, "kept   %s (already exists, not changed)\n", rel)
		return nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "wrote  %s\n", rel)
	return nil
}

func runInit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "a directory inside the repository")
	prTemplate := fs.Bool("pr-template", false, "also write .github/pull_request_template.md")
	workflow := fs.Bool("workflow", false, "also write the CI check: .github/workflows/pr-brief.yml, or .azuredevops/pr-brief.yml for Azure DevOps")
	hostFlag := fs.String("host", "", "where CI runs: github or azure-devops (default: from the git remote)")
	rewrite := fs.Bool("rewrite", false, "write a workflow whose first job rewrites the description with the Claude Code GitHub Action (GitHub only; needs --auth)")
	auth := fs.String("auth", "", "how the rewrite job signs in to Claude: "+strings.Join(initfiles.AuthModes, ", "))
	cfgMode := fs.String("config", "file", "where CI reads its settings: file (.pr-brief.json, the default) or inline (written in the workflow)")
	style := fs.String("style", "", "prose style for the repo: ste+iceberg, ste or iceberg")
	themeFlag := fs.String("theme", "", "diagram theme for the repo: a built-in name or a path inside the repo")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfgPath := config.RepoPath(*dir)
	root := filepath.Dir(cfgPath)

	fail := func(err error) int {
		fmt.Fprintln(stderr, "pr-brief init:", err)
		return 1
	}

	if *rewrite && !*workflow {
		*workflow = true
	}
	if *cfgMode != "file" && *cfgMode != "inline" {
		fmt.Fprintf(stderr, "pr-brief init: --config %q must be file or inline\n", *cfgMode)
		return 2
	}
	inline := *cfgMode == "inline"
	if *rewrite && !slices.Contains(initfiles.AuthModes, *auth) {
		fmt.Fprintf(stderr, "pr-brief init: --rewrite needs --auth, one of %s. Each team chooses how its job signs in to Claude.\n", strings.Join(initfiles.AuthModes, ", "))
		return 2
	}
	if *auth != "" && !*rewrite {
		fmt.Fprintln(stderr, "pr-brief init: --auth belongs to --rewrite")
		return 2
	}
	if inline && !*workflow {
		fmt.Fprintln(stderr, "pr-brief init: --config inline writes the settings into the CI file; add --workflow")
		return 2
	}
	ciHost, ok := host.Normalize(*hostFlag)
	if !ok {
		fmt.Fprintf(stderr, "pr-brief init: unknown host %q; use github or azure-devops\n", *hostFlag)
		return 2
	}
	if ciHost == "" {
		if ciHost = host.FromGit(root); ciHost == "" {
			ciHost = host.GitHub
		}
	}

	created := false
	_, statErr := os.Stat(cfgPath)
	switch {
	case inline:
		fmt.Fprintln(stdout, "skip   .pr-brief.json (the settings go in the CI file)")
		if statErr == nil {
			fmt.Fprintln(stdout, "note   .pr-brief.json exists and wins over the CI file")
		}
	case errors.Is(statErr, os.ErrNotExist):
		if err := config.Init(cfgPath); err != nil {
			return fail(err)
		}
		created = true
		fmt.Fprintf(stdout, "wrote  %s\n", filepath.Base(cfgPath))
		if *style != "" {
			if err := config.Set(cfgPath, "style", *style); err != nil {
				return fail(err)
			}
		}
		if *themeFlag != "" {
			if err := config.Set(cfgPath, "diagram.theme", *themeFlag); err != nil {
				return fail(err)
			}
		}
	default:
		fmt.Fprintf(stdout, "kept   %s (already exists, not changed)\n", filepath.Base(cfgPath))
		if *style != "" || *themeFlag != "" {
			fmt.Fprintln(stderr, "pr-brief init: --style and --theme apply only to a new file; use `pr-brief config set` to change an existing one")
		}
	}

	// The files that follow use the repo's resolved style and theme.
	r, err := config.Resolve(config.Options{RepoPath: cfgPath})
	if err != nil {
		return fail(err)
	}
	if inline {
		// No file to read: the flags are the settings, checked the way the workflow input is.
		if *style != "" {
			r.Style = *style
		}
		if *themeFlag != "" {
			r.Theme = *themeFlag
		}
		if !slices.Contains(convention.Styles, r.Style) {
			return fail(fmt.Errorf("--style %q must be one of %s", r.Style, strings.Join(convention.Styles, ", ")))
		}
		if !theme.IsBuiltin(r.Theme) {
			return fail(fmt.Errorf("--theme %q must be a built-in theme (%s) with --config inline; put a custom theme in .pr-brief.json", r.Theme, strings.Join(theme.Names(), ", ")))
		}
	}
	th, err := r.LoadTheme()
	if err != nil {
		return fail(err)
	}
	if *prTemplate {
		tplPath := initfiles.PRTemplatePath
		if ciHost == host.AzureDevOps {
			tplPath = initfiles.AzurePRTemplatePath
		}
		if err := writeNew(root, tplPath, initfiles.PRTemplate(r.Style, th.ID()), stdout); err != nil {
			return fail(err)
		}
	}
	if *workflow {
		path, content := initfiles.WorkflowPath, initfiles.Workflow()
		switch {
		case *rewrite && ciHost == host.AzureDevOps:
			return fail(errors.New("--rewrite needs the Claude Code GitHub Action, which runs on GitHub only; use the gate pipeline on Azure DevOps"))
		case *rewrite:
			content = initfiles.WorkflowRewrite(*auth, inline, r.Style, r.Theme)
		case ciHost == host.AzureDevOps:
			path, content = initfiles.AzurePipelinePath, initfiles.AzurePipeline(version, inline, r.Style, r.Theme)
		case inline:
			content = initfiles.WorkflowInline(r.Style, r.Theme)
		}
		if err := writeNew(root, path, content, stdout); err != nil {
			return fail(err)
		}
	}

	fmt.Fprintf(stdout, "\nstyle %s · theme %s · earlier description %s\n", r.Style, th.ID(), r.Previous)
	fmt.Fprintln(stdout, "\nNext:")
	switch {
	case inline:
		fmt.Fprintln(stdout, "  1. The CI file holds the style and theme. A .pr-brief.json on the base branch would win over them.")
		fmt.Fprintln(stdout, "     Anyone who can edit the CI file in a PR can change these values; use .pr-brief.json to prevent that.")
	case created:
		fmt.Fprintln(stdout, "  1. Review .pr-brief.json and commit it. It applies to everyone who works in this repo,")
		fmt.Fprintln(stdout, "     and it wins over personal settings. CI reads it from the base branch.")
	default:
		fmt.Fprintln(stdout, "  1. .pr-brief.json already applies to everyone who works in this repo.")
	}
	switch {
	case !*workflow:
		fmt.Fprintln(stdout, "  2. Add the CI check: pr-brief init --workflow   (add --host azure-devops for Azure Pipelines)")
	case *rewrite:
		fmt.Fprintf(stdout, "  2. Commit %s. The gate job needs no secret. The rewrite job stays off until you add the credentials for --auth %s:\n", initfiles.WorkflowPath, *auth)
		for _, l := range rewriteSetup(*auth) {
			fmt.Fprintln(stdout, "     "+l)
		}
		fmt.Fprintln(stdout, "     Make the `description` job the required check. Add the label `pr-brief` to a PR to rewrite it again.")
	case ciHost == host.AzureDevOps:
		fmt.Fprintf(stdout, "  2. Commit %s, create a pipeline from it, and add it as a Build Validation policy on the\n", initfiles.AzurePipelinePath)
		fmt.Fprintln(stdout, "     target branch. It needs no secret: the job reads the PR with its own System.AccessToken.")
		fmt.Fprintf(stdout, "     Azure DevOps accepts %d characters in a description; the check enforces that limit.\n", convention.MaxBodyCharsAzureDevOps)
	default:
		fmt.Fprintf(stdout, "  2. Commit %s. It needs no secret and no personal access token: contents: read is enough.\n", initfiles.WorkflowPath)
	}
	if !*prTemplate {
		fmt.Fprintln(stdout, "  3. Show contributors the structure: pr-brief init --pr-template")
	}
	if need := styleNeeds(r.Style); need != "" {
		fmt.Fprintf(stdout, "  Style %s needs %s on the machine that writes the description (the CI check needs neither).\n", r.Style, need)
	}
	fmt.Fprintf(stdout, "  Limits and rules are fixed on purpose: %d diagrams per PR, %d nodes each. See the convention page.\n", convention.MaxDiagrams, convention.MaxNodes)
	return 0
}

// styleNeeds names what the skill must have installed to write in a style.
func styleNeeds(style string) string {
	switch style {
	case "ste+iceberg":
		return "the asd-ste100 skill and the iceberg plugin"
	case "ste":
		return "the asd-ste100 skill"
	case "iceberg":
		return "the iceberg plugin"
	}
	return ""
}

// rewriteSetup says what the rewrite job needs before it turns on.
func rewriteSetup(auth string) []string {
	switch auth {
	case initfiles.AuthAPIKey:
		return []string{"- The repository secret ANTHROPIC_API_KEY. It is a long-lived secret; prefer federation if your org allows it."}
	case initfiles.AuthFederation:
		return []string{"- Create a federation rule and service account in your Anthropic organisation, then set the repository variables",
			"  ANTHROPIC_FEDERATION_RULE_ID, ANTHROPIC_ORG_ID, ANTHROPIC_SERVICE_ACCOUNT_ID and ANTHROPIC_WORKSPACE_ID."}
	case initfiles.AuthBedrock:
		return []string{"- An AWS role your repo can assume over OIDC, with Bedrock access. Set the repository variables AWS_ROLE_TO_ASSUME and AWS_REGION."}
	case initfiles.AuthVertex:
		return []string{"- A Google Cloud workload identity provider and service account with Vertex AI access.",
			"  Set the repository variables GCP_WORKLOAD_IDENTITY_PROVIDER and GCP_SERVICE_ACCOUNT."}
	}
	return nil
}
