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
)

// styleFor returns the style the gate enforces. A style set by a flag or a
// config file wins over the style recorded in the description's marker.
func styleFor(cwd, flagStyle string, ci bool) (string, error) {
	r, err := config.Resolve(config.Options{
		RepoPath: config.RepoPath(cwd), UserPath: config.UserPath(), FlagStyle: flagStyle, CI: ci,
	})
	if err != nil {
		return "", err
	}
	if r.StyleSource == config.SourceDefault {
		return "", nil
	}
	return r.Style, nil
}

func runGate(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("gate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	hook := fs.Bool("hook", false, "read a PreToolUse payload from stdin")
	ci := fs.Bool("ci", false, "read the PR from $GITHUB_EVENT_PATH")
	file := fs.String("file", "", "check a description file")
	useStdin := fs.Bool("stdin", false, "check a description from stdin")
	style := fs.String("style", "", "override the style")
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
		st, err := styleFor(cwd, *style, false)
		if err != nil {
			fmt.Fprintln(stderr, "pr-brief gate:", err)
			return 2
		}
		res := gate.Check(text, gate.Options{Style: st, Lint: steHard})
		printResult(res, *asJSON, stdout)
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
		return gate.Options{Style: st, Lint: steHard}
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
	ws := os.Getenv("GITHUB_WORKSPACE")
	if ws == "" {
		ws, _ = os.Getwd()
	}
	st, err := styleFor(ws, "", true)
	if err != nil {
		fmt.Fprintln(stderr, "pr-brief gate --ci:", err)
		return 2
	}
	res := gate.Check(text, gate.Options{Style: st, Lint: steHard})
	printResult(res, false, stdout)
	for _, f := range res.Findings {
		fmt.Fprintf(stdout, "::error title=pr-brief [%s]::%s\n", f.Rule, strings.ReplaceAll(f.Message, "\n", " "))
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
