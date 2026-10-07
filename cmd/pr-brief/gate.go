package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gagoar/pr-brief/internal/config"
	"github.com/gagoar/pr-brief/internal/convention"
	"github.com/gagoar/pr-brief/internal/gate"
	"github.com/gagoar/pr-brief/internal/host"
	"github.com/gagoar/pr-brief/internal/theme"
)

// styleFor returns the style the gate enforces. A style set by a flag or a
// config file wins over the style recorded in the description's marker.
func styleFor(cwd, flagStyle string, ci bool) (string, error) {
	r, err := config.Resolve(config.Options{
		RepoPath: config.RepoPath(cwd), UserPath: config.UserPath(), FlagStyle: flagStyle, CI: ci, Workflow: workflowConfig(ci),
	})
	if err != nil {
		return "", err
	}
	if r.StyleSource == config.SourceDefault {
		return "", nil
	}
	return r.Style, nil
}

// workflowConfig reads the style and theme a CI workflow passes in as PR_BRIEF_STYLE and
// PR_BRIEF_THEME (the Action inputs `style` and `theme`). Outside CI it reads nothing.
func workflowConfig(ci bool) config.File {
	if !ci {
		return config.File{}
	}
	f := config.File{Style: strings.TrimSpace(os.Getenv("PR_BRIEF_STYLE"))}
	if t := strings.TrimSpace(os.Getenv("PR_BRIEF_THEME")); t != "" {
		f.Diagram = &config.Diagram{Theme: t}
	}
	return f
}

// shadowedInputs lists the workflow inputs the repo's .pr-brief.json overrides.
func shadowedInputs(cwd string, ci bool) []string {
	if !ci {
		return nil
	}
	r, err := config.Resolve(config.Options{RepoPath: config.RepoPath(cwd), CI: true, Workflow: workflowConfig(true)})
	if err != nil {
		return nil
	}
	return r.Shadowed
}

// themeFor returns the theme the gate enforces, or nil when no config names one. As
// with the style, an author can then pick any built-in theme through the begin marker,
// but a repo that names a theme gets exactly that theme.
func themeFor(cwd string, ci bool) (*theme.Theme, error) {
	r, err := config.Resolve(config.Options{RepoPath: config.RepoPath(cwd), UserPath: config.UserPath(), CI: ci, Workflow: workflowConfig(ci)})
	if err != nil {
		return nil, err
	}
	if r.ThemeSource == config.SourceDefault {
		return nil, nil
	}
	th, err := r.LoadTheme()
	if err != nil {
		return nil, err
	}
	return &th, nil
}

func runGate(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("gate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	hook := fs.Bool("hook", false, "read a PreToolUse payload from stdin")
	ci := fs.Bool("ci", false, "read the PR from $GITHUB_EVENT_PATH")
	file := fs.String("file", "", "check a description file")
	useStdin := fs.Bool("stdin", false, "check a description from stdin")
	style := fs.String("style", "", "override the style")
	hostFlag := fs.String("host", "", "where the PR lives: github or azure-devops (default: from the pipeline or the git remote)")
	asJSON := fs.Bool("json", false, "print the result as JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	switch {
	case *hook:
		return gateHook(stdin, stdout, stderr)
	case *ci:
		return gateCI(stdout, stderr)
	case *file != "" || *useStdin:
		text, err := readInput(*file, stdin)
		if err != nil {
			fmt.Fprintln(stderr, "pr-brief gate:", err)
			return 2
		}
		cwd, _ := os.Getwd()
		h, ok := host.Normalize(*hostFlag)
		if !ok {
			fmt.Fprintf(stderr, "pr-brief gate: unknown host %q; use github or azure-devops\n", *hostFlag)
			return 2
		}
		if h == "" {
			if h = host.FromEnv(os.Getenv); h == "" {
				h = host.FromGit(cwd)
			}
		}
		// A pipeline run reads the repo's config only, like the GitHub Action does.
		inPipeline := os.Getenv("TF_BUILD") != ""
		st, err := styleFor(cwd, *style, inPipeline)
		if err != nil {
			fmt.Fprintln(stderr, "pr-brief gate:", err)
			return 2
		}
		th, err := themeFor(cwd, inPipeline)
		if err != nil {
			fmt.Fprintln(stderr, "pr-brief gate:", err)
			return 2
		}
		if !*asJSON {
			for _, k := range shadowedInputs(cwd, inPipeline) {
				fmt.Fprintf(stdout, "##vso[task.logissue type=warning]pr-brief: the pipeline sets %s, but .pr-brief.json sets it too. The file wins.\n", k)
			}
		}
		res := gate.Check(text, gate.Options{Style: st, Theme: th, Lint: steHard, Host: h})
		printResult(res, *asJSON, stdout)
		if inPipeline && !*asJSON {
			for _, f := range res.Findings {
				fmt.Fprintf(stdout, "##vso[task.logissue type=error]%s\n", escapeVSO("pr-brief ["+f.Rule+"] "+f.Message))
			}
		}
		if !res.OK() {
			return 1
		}
		return 0
	}
	fmt.Fprintln(stderr, "pr-brief gate: choose one of --hook, --ci, --file, --stdin")
	return 2
}

func printResult(res gate.Result, asJSON bool, w io.Writer) {
	if asJSON {
		data, _ := json.MarshalIndent(res, "", "  ")
		fmt.Fprintln(w, string(data))
		return
	}
	switch {
	case res.Skipped:
		fmt.Fprintf(w, "skipped: %s\n", res.SkipReason)
	case res.OK():
		fmt.Fprintf(w, "ok (style %s)\n", res.Style)
	}
	for _, f := range res.Findings {
		fmt.Fprintf(w, "FAIL [%s] %s\n", f.Rule, f.Message)
	}
	for _, m := range res.Warnings {
		fmt.Fprintf(w, "warn: %s\n", m)
	}
}

// gateHook never blocks a call because of its own failure: a bad payload or a
// config error allows the call and says why on stderr.
func gateHook(stdin io.Reader, stdout, stderr io.Writer) int {
	var in gate.HookInput
	data, err := io.ReadAll(stdin)
	if err != nil || json.Unmarshal(data, &in) != nil {
		fmt.Fprintln(stderr, "pr-brief: cannot read the hook payload; allowing the call")
		return 0
	}
	reason := gate.Evaluate(in, func(cwd string) gate.Options {
		st, err := styleFor(cwd, "", false)
		if err != nil {
			fmt.Fprintln(stderr, "pr-brief: bad config:", err)
		}
		th, err := themeFor(cwd, false)
		if err != nil {
			fmt.Fprintln(stderr, "pr-brief: bad theme:", err)
		}
		return gate.Options{Style: st, Theme: th, Lint: steHard, Host: host.FromGit(cwd)}
	})
	if reason != "" {
		fmt.Fprintln(stdout, string(gate.DenyJSON(reason)))
	}
	return 0
}

type prEvent struct {
	PullRequest *struct {
		Number int     `json:"number"`
		Body   *string `json:"body"`
		User   struct {
			Type string `json:"type"`
		} `json:"user"`
	} `json:"pull_request"`
}

func gateCI(stdout, stderr io.Writer) int {
	path := os.Getenv("GITHUB_EVENT_PATH")
	if path == "" {
		fmt.Fprintln(stderr, "pr-brief gate --ci: $GITHUB_EVENT_PATH is not set")
		return 2
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(stderr, "pr-brief gate --ci:", err)
		return 2
	}
	var ev prEvent
	if err := json.Unmarshal(data, &ev); err != nil {
		fmt.Fprintln(stderr, "pr-brief gate --ci:", err)
		return 2
	}
	if ev.PullRequest == nil {
		fmt.Fprintln(stdout, "not a pull_request event; nothing to check")
		return 0
	}
	if ev.PullRequest.User.Type == "Bot" {
		fmt.Fprintln(stdout, "pull request opened by a bot; skipped")
		return 0
	}
	text := ""
	if ev.PullRequest.Body != nil {
		text = *ev.PullRequest.Body
	}
	// A job that rewrote the description reads the live text, not the event payload, which
	// still holds the old one. The Action's `refresh` input fetches it into this file.
	if p := os.Getenv("PR_BRIEF_BODY_FILE"); p != "" {
		data, err := os.ReadFile(p)
		if err != nil {
			fmt.Fprintln(stderr, "pr-brief gate --ci:", err)
			return 2
		}
		text = string(data)
	}
	ws := os.Getenv("GITHUB_WORKSPACE")
	if ws == "" {
		ws, _ = os.Getwd()
	}
	st, err := styleFor(ws, "", true)
	if err != nil {
		fmt.Fprintln(stderr, "pr-brief gate --ci:", err)
		return 2
	}
	th, err := themeFor(ws, true)
	if err != nil {
		fmt.Fprintln(stderr, "pr-brief gate --ci:", err)
		return 2
	}
	for _, k := range shadowedInputs(ws, true) {
		fmt.Fprintf(stdout, "::notice title=pr-brief::The workflow sets %s, but .pr-brief.json sets it too. The file wins.\n", k)
	}
	res := gate.Check(text, gate.Options{Style: st, Theme: th, Lint: steHard, Host: host.GitHub})
	printResult(res, false, stdout)
	for _, f := range res.Findings {
		fmt.Fprintf(stdout, "::error title=%s::%s\n", escapeProperty("pr-brief ["+f.Rule+"]"), escapeData(f.Message))
	}
	if sum := os.Getenv("GITHUB_STEP_SUMMARY"); sum != "" {
		writeSummary(filepath.Clean(sum), res, ev.PullRequest.Number)
	}
	if !res.OK() {
		return 1
	}
	return 0
}

func writeSummary(path string, res gate.Result, pr int) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	if res.OK() {
		fmt.Fprintln(f, "### pr-brief: ok")
		return
	}
	fmt.Fprintln(f, "### pr-brief: the description needs work")
	for _, x := range res.Findings {
		fmt.Fprintf(f, "- **%s**: %s\n", x.Rule, x.Message)
	}
	fmt.Fprintf(f, "\nFix it with `/pr-brief improve %d`, or add `%s <reason>` to skip this PR.\n", pr, convention.SkipPrefix)
}

// escapeData escapes the message of a workflow command the way GitHub requires, so text
// taken from a pull request description cannot end the command or start another one.
func escapeData(s string) string {
	return strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A").Replace(s)
}

// escapeProperty escapes a property value such as title=, which also ends at : and ,.
func escapeProperty(s string) string {
	return strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A", ":", "%3A", ",", "%2C").Replace(s)
}

// escapeVSO escapes a message for an Azure Pipelines logging command.
func escapeVSO(s string) string {
	return strings.NewReplacer("%", "%AZP25", "\r", "%0D", "\n", "%0A").Replace(s)
}
