package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gagoar/pr-brief/internal/host"
	"github.com/gagoar/pr-brief/internal/links"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// isolate points every config and state path at a temp dir.
func isolate(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(d, "state"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(d, "cfg"))
	t.Chdir(d)
	return d
}

func hookPayload(t *testing.T, cwd, command string) *strings.Reader {
	t.Helper()
	b, _ := json.Marshal(map[string]any{"tool_name": "Bash", "cwd": cwd, "tool_input": map[string]any{"command": command}})
	return strings.NewReader(string(b))
}

func TestHookDeniesAndAllows(t *testing.T) {
	d := isolate(t)
	var out, errb bytes.Buffer
	if code := runGate([]string{"--hook"}, hookPayload(t, d, `gh pr create --body "plain"`), &out, &errb); code != 0 {
		t.Fatalf("hook exit = %d", code)
	}
	if !strings.Contains(out.String(), `"permissionDecision":"deny"`) || !strings.Contains(out.String(), "> pr-brief skipped:") {
		t.Errorf("expected a readable denial, got %s", out.String())
	}
	out.Reset()
	runGate([]string{"--hook"}, hookPayload(t, d, "ls"), &out, &errb)
	if out.Len() != 0 {
		t.Errorf("unrelated command must print nothing, got %s", out.String())
	}
	out.Reset()
	errb.Reset()
	if code := runGate([]string{"--hook"}, strings.NewReader("not json"), &out, &errb); code != 0 || out.Len() != 0 {
		t.Errorf("a bad payload must fail open: code=%d out=%q", code, out.String())
	}
}

func TestGateFileAndSkip(t *testing.T) {
	d := isolate(t)
	p := filepath.Join(d, "b.md")
	os.WriteFile(p, []byte("> pr-brief skipped: release\n"), 0o644)
	var out, errb bytes.Buffer
	if code := runGate([]string{"--file", p}, nil, &out, &errb); code != 0 || !strings.Contains(out.String(), "skipped: release") {
		t.Errorf("skip: code=%d out=%s", code, out.String())
	}
	os.WriteFile(p, []byte("plain"), 0o644)
	out.Reset()
	if code := runGate([]string{"--file", p}, nil, &out, &errb); code != 1 || !strings.Contains(out.String(), "FAIL [markers]") {
		t.Errorf("plain body: code=%d out=%s", code, out.String())
	}
}

func TestGateCI(t *testing.T) {
	d := isolate(t)
	ev := filepath.Join(d, "event.json")
	t.Setenv("GITHUB_EVENT_PATH", ev)
	t.Setenv("GITHUB_WORKSPACE", d)
	summary := filepath.Join(d, "summary.md")
	t.Setenv("GITHUB_STEP_SUMMARY", summary)

	write := func(v any) {
		b, _ := json.Marshal(v)
		os.WriteFile(ev, b, 0o644)
	}
	var out, errb bytes.Buffer

	write(map[string]any{"pull_request": map[string]any{"number": 7, "body": nil, "user": map[string]any{"type": "User"}}})
	if code := runGate([]string{"--ci"}, nil, &out, &errb); code != 1 {
		t.Errorf("an empty body must fail in CI, got %d", code)
	}
	if !strings.Contains(out.String(), "::error title=pr-brief [markers]::") {
		t.Errorf("missing annotation: %s", out.String())
	}
	if s, _ := os.ReadFile(summary); !strings.Contains(string(s), "/pr-brief improve 7") {
		t.Errorf("summary should point at improve: %s", s)
	}

	out.Reset()
	write(map[string]any{"pull_request": map[string]any{"number": 8, "body": "x", "user": map[string]any{"type": "Bot"}}})
	if code := runGate([]string{"--ci"}, nil, &out, &errb); code != 0 {
		t.Errorf("bot PRs are skipped, got %d", code)
	}
	out.Reset()
	write(map[string]any{"push": map[string]any{}})
	if code := runGate([]string{"--ci"}, nil, &out, &errb); code != 0 {
		t.Errorf("non-PR events pass, got %d", code)
	}
}

func TestCIIgnoresUserConfig(t *testing.T) {
	d := isolate(t)
	var out, errb bytes.Buffer
	if code := runConfig([]string{"set", "style", "iceberg", "--scope", "user"}, &out, &errb); code != 0 {
		t.Fatalf("set: %s", errb.String())
	}
	if st, _ := styleFor(d, "", false); st != "iceberg" {
		t.Errorf("local gate should use the user style, got %q", st)
	}
	if st, _ := styleFor(d, "", true); st != "" {
		t.Errorf("CI must ignore the user style, got %q", st)
	}
}

func TestConfigRejectsUnknownKeys(t *testing.T) {
	isolate(t)
	var out, errb bytes.Buffer
	if code := runConfig([]string{"set", "diagram.maxNodes", "3"}, &out, &errb); code == 0 {
		t.Error("an unknown key must be rejected")
	}
	if code := runConfig([]string{"set", "style", "hemingway"}, &out, &errb); code == 0 {
		t.Error("a bad value must be rejected")
	}
}

func TestBodyImproveRoundTrip(t *testing.T) {
	d := isolate(t)
	cur := filepath.Join(d, "cur.md")
	man := filepath.Join(d, "managed.md")
	os.WriteFile(cur, []byte("Hand written -- notes\n"), 0o644)
	os.WriteFile(man, []byte("<!-- pr-brief:begin v1 style=ste -->\n## Brief\nx.\n<!-- pr-brief:end -->\n"), 0o644)

	var out, errb bytes.Buffer
	args := []string{"improve", "--managed", man, "--current", cur, "--owner", "o", "--repo", "r", "--pr", "5", "--mode", "comment"}
	if code := runBody(args, nil, &out, &errb); code != 0 {
		t.Fatalf("improve: %s", errb.String())
	}
	if !strings.Contains(out.String(), "pr-brief:previous v1") || strings.Contains(out.String(), "notes\n--") {
		t.Errorf("unexpected output: %s", out.String())
	}

	// Feed the result back in: the original text must survive a second run.
	os.WriteFile(cur, out.Bytes(), 0o644)
	out.Reset()
	if code := runBody(args, nil, &out, &errb); code != 0 {
		t.Fatalf("second improve: %s", errb.String())
	}
	if strings.Count(out.String(), "pr-brief:previous v1") != 1 {
		t.Errorf("blocks nested: %s", out.String())
	}
	out.Reset()
	runBody([]string{"restore", "--owner", "o", "--repo", "r", "--pr", "5"}, nil, &out, &errb)
	if out.String() != "Hand written -- notes\n" {
		t.Errorf("restore should return the original, got %q", out.String())
	}

	// drop mode removes the past writing from the description.
	out.Reset()
	dropArgs := append(append([]string{}, args[:len(args)-1]...), "drop")
	runBody(dropArgs, nil, &out, &errb)
	if strings.Contains(out.String(), "previous") {
		t.Errorf("drop must not keep the past: %s", out.String())
	}
}

func TestLintCommand(t *testing.T) {
	var out, errb bytes.Buffer
	if code := runLint([]string{"--file", "-"}, strings.NewReader("The service is seamless; it works.\n"), &out, &errb); code != 1 {
		t.Errorf("a semicolon and a marketing word must fail, got %d: %s", code, out.String())
	}
	out.Reset()
	if code := runLint([]string{"--file", "-"}, strings.NewReader("The service sends the event.\n"), &out, &errb); code != 0 {
		t.Errorf("clean prose must pass, got %d: %s", code, out.String())
	}
}

func fixtureFlow(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join("..", "..", "internal", "diagram", "testdata", "real.flow.json"))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

var descTemplate = "<!-- pr-brief:begin v1 style=iceberg theme=%s -->\n" +
	"## Brief\nThe hook reads a tool call and decides if a PR description may pass.\n\n" +
	"## Change map\n### Flow 1: I1 -> a PR command is checked\n%s\n" +
	"| Ref | What | Detail |\n|---|---|---|\n| I1 | `gate --hook` | Claude Code sends the call as JSON. |\n| O1 | decision | The hook prints a deny decision or nothing. |\n\n" +
	"green added\n\n## Review guide\n**What changed**:\n- `gateHook` reads the call.\n\n" +
	"**Read these first**\n| File | Why it is delicate | What to check |\n|---|---|---|\n| " + links.Cell("hook.go", links.Repo{Host: host.GitHub, Owner: "o", Name: "r"}.DiffURL(4, "hook.go")) + " | It decides. | Check bad payloads. |\n\n" +
	"**Review order**: Start at `gateHook`.\n<!-- pr-brief:end -->\n"

// draw runs `pr-brief diagram` and returns the fenced diagram.
func draw(t *testing.T, flow string, args ...string) string {
	t.Helper()
	var out, errb bytes.Buffer
	if code := runDiagram(append([]string{"--flow", flow}, args...), nil, &out, &errb); code != 0 {
		t.Fatalf("diagram: %s", errb.String())
	}
	return out.String()
}

func gateText(t *testing.T, dir, body string) (int, string) {
	t.Helper()
	p := filepath.Join(dir, "body.md")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	code := runGate([]string{"--file", p}, nil, &out, &errb)
	return code, out.String() + errb.String()
}

func TestDiagramThenGateRoundTrip(t *testing.T) {
	flow := fixtureFlow(t)
	d := isolate(t)
	fence := draw(t, flow, "--theme", "dracula")
	if code, out := gateText(t, d, fmt.Sprintf(descTemplate, "dracula", fence)); code != 0 {
		t.Fatalf("a diagram from the tool must pass: %s", out)
	}
	// A hand edit of one colour must fail, and the message must say how to fix it.
	edited := strings.Replace(fence, "#50fa7b", "#50fa7c", 1)
	code, out := gateText(t, d, fmt.Sprintf(descTemplate, "dracula", edited))
	if code != 1 || !strings.Contains(out, "does not match theme dracula") || !strings.Contains(out, "pr-brief diagram") {
		t.Errorf("a hand edit must fail with a fix: code=%d %s", code, out)
	}
	// A diagram from the default theme under a dracula marker fails too.
	if code, _ := gateText(t, d, fmt.Sprintf(descTemplate, "dracula", draw(t, flow))); code != 1 {
		t.Error("the wrong theme must fail")
	}
}

func TestConfiguredThemeIsEnforced(t *testing.T) {
	flow := fixtureFlow(t)
	d := isolate(t)
	var out, errb bytes.Buffer
	if code := runConfig([]string{"set", "diagram.theme", "alucard", "--scope", "repo"}, &out, &errb); code != 0 {
		t.Fatalf("set: %s", errb.String())
	}
	// Drawn in github-dark, marker says github-dark: the repo wants alucard, so it fails.
	if code, o := gateText(t, d, fmt.Sprintf(descTemplate, "github-dark", draw(t, flow, "--theme", "github-dark"))); code != 1 || !strings.Contains(o, "theme alucard") {
		t.Errorf("the repo's theme must win over the marker: code=%d %s", code, o)
	}
	// Drawn in alucard (from the config, no flag): passes whatever the marker says.
	if code, o := gateText(t, d, fmt.Sprintf(descTemplate, "github-dark", draw(t, flow))); code != 0 {
		t.Errorf("the configured theme should pass: %s", o)
	}
}

func TestCustomThemeFileEndToEnd(t *testing.T) {
	flow := fixtureFlow(t)
	d := isolate(t)
	if err := os.WriteFile(filepath.Join(d, "brand.json"), []byte(`{"bg":"#10141c","fg":"#e4ecf7","line":"#4a5a78","status":{"added":"#2ecc71","modified":"#f1c40f","removed":"#e74c3c"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	if code := runConfig([]string{"set", "diagram.theme", "./brand.json", "--scope", "repo"}, &out, &errb); code != 0 {
		t.Fatalf("set: %s", errb.String())
	}
	fence := draw(t, flow)
	if !strings.Contains(fence, "#2ecc71") {
		t.Error("the status override must reach the diagram")
	}
	th, _ := themeFor(d, false)
	if th == nil {
		t.Fatal("themeFor returned nil for a configured theme")
	}
	if code, o := gateText(t, d, fmt.Sprintf(descTemplate, th.ID(), fence)); code != 0 {
		t.Errorf("a custom theme from the repo config must pass: %s", o)
	}
	// Change the theme file: the same diagram no longer matches.
	if err := os.WriteFile(filepath.Join(d, "brand.json"), []byte(`{"bg":"#10141c","fg":"#e4ecf7","line":"#aa0000"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _ := gateText(t, d, fmt.Sprintf(descTemplate, th.ID(), fence)); code != 1 {
		t.Error("an old diagram must fail after the theme file changed")
	}
	// A broken theme file is an error, not a silent fallback.
	if err := os.WriteFile(filepath.Join(d, "brand.json"), []byte(`{"bg":"#888888","fg":"#777777"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, o := gateText(t, d, fmt.Sprintf(descTemplate, th.ID(), fence)); code != 2 || !strings.Contains(o, "contrast") {
		t.Errorf("a broken theme file must stop the gate with a reason: code=%d %s", code, o)
	}
}

func TestThemeCommands(t *testing.T) {
	d := isolate(t)
	var out, errb bytes.Buffer
	if code := runTheme([]string{"list"}, &out, &errb); code != 0 || !strings.Contains(out.String(), "github-dark  (default)") {
		t.Errorf("list: %s", out.String())
	}
	good := filepath.Join(d, "good.json")
	os.WriteFile(good, []byte(`{"bg":"#ffffff","fg":"#111111"}`), 0o644)
	out.Reset()
	if code := runTheme([]string{"validate", good}, &out, &errb); code != 0 || !strings.Contains(out.String(), "custom:") {
		t.Errorf("validate good: %s", out.String())
	}
	bad := filepath.Join(d, "bad.json")
	os.WriteFile(bad, []byte(`{"bg":"#888888","fg":"#777777"}`), 0o644)
	out.Reset()
	if code := runTheme([]string{"validate", bad}, &out, &errb); code != 1 || !strings.Contains(out.String(), "contrast") {
		t.Errorf("validate bad: %s", out.String())
	}
}

func TestDiagramFromShapeReport(t *testing.T) {
	d := isolate(t)
	report := `{"refs":{"I1":{"kind":"route"},"O1":{"kind":"db"}},"flows":[{"inputs":["I1"],"outputs":["O1"],
	  "functions":[{"node":"F1","label":"save","status":"added","riskScore":3}],
	  "edges":[{"from":"I1","to":"F1","status":"new"},{"from":"F1","to":"O1","status":"new"}]}]}`
	p := filepath.Join(d, "report.json")
	os.WriteFile(p, []byte(report), 0o644)
	var out, errb bytes.Buffer
	if code := runDiagram([]string{"--report", p, "--flow-number", "1", "--raw"}, nil, &out, &errb); code != 0 {
		t.Fatalf("diagram --report: %s", errb.String())
	}
	if !strings.Contains(out.String(), `O1[("Output 1")]`) || strings.Contains(out.String(), "```") {
		t.Errorf("unexpected output: %s", out.String())
	}
	if code := runDiagram([]string{"--report", p, "--flow-number", "3"}, nil, &out, &errb); code == 0 {
		t.Error("flow 3 does not exist")
	}
}

func TestShapeFeedsDiagram(t *testing.T) {
	d := isolate(t)
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = d
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(d, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git("init", "-q", "-b", "main")
	git("config", "user.email", "t@example.com")
	git("config", "user.name", "T")
	git("config", "commit.gpgsign", "false")
	ctrl := "public class Api\n{\n    [HttpPost(\"run\")]\n    public string Run()\n    {\n        return Save();\n    }\n\n    public string Save()\n    {\n        return \"a\";\n    }\n}\n"
	write("Api.cs", ctrl)
	git("add", "-A")
	git("commit", "-q", "-m", "base")
	git("checkout", "-q", "-b", "feature")
	write("Api.cs", strings.Replace(ctrl, `return "a";`, "repository.Insert(item);\n        return \"b\";", 1))
	git("add", "-A")
	git("commit", "-q", "-m", "change")

	var shape, errb bytes.Buffer
	if code := runShape([]string{"--dir", d, "--base", "main"}, &shape, &errb); code != 0 {
		t.Fatalf("shape: %s", errb.String())
	}
	report := filepath.Join(d, "report.json")
	os.WriteFile(report, shape.Bytes(), 0o644)

	for _, th := range []string{"github-dark", "alucard"} {
		var out, e2 bytes.Buffer
		if code := runDiagram([]string{"--report", report, "--theme", th, "--raw"}, nil, &out, &e2); code != 0 {
			t.Fatalf("%s: diagram --report: %s", th, e2.String())
		}
		for _, want := range []string{"graph LR", `I1(["Input 1"])`, "Api.Save", "~~~", "classDef zone"} {
			if !strings.Contains(out.String(), want) {
				t.Errorf("%s: output lacks %q:\n%s", th, want, out.String())
			}
		}
	}
}

func TestWorkflowCommandEscaping(t *testing.T) {
	if got := escapeData("100% sure\r\nnext"); got != "100%25 sure%0D%0Anext" {
		t.Errorf("escapeData = %q", got)
	}
	if got := escapeProperty("a: b, c%\n"); got != "a%3A b%2C c%25%0A" {
		t.Errorf("escapeProperty = %q", got)
	}
}

// Text from a pull request description must never end an annotation or start another command.
func TestCIAnnotationsCannotBeInjected(t *testing.T) {
	flow := fixtureFlow(t)
	d := isolate(t)
	ev := filepath.Join(d, "event.json")
	t.Setenv("GITHUB_EVENT_PATH", ev)
	t.Setenv("GITHUB_WORKSPACE", d)

	run := func(body string) string {
		t.Helper()
		b, _ := json.Marshal(map[string]any{"pull_request": map[string]any{"number": 1, "body": body, "user": map[string]any{"type": "User"}}})
		os.WriteFile(ev, b, 0o644)
		var out, errb bytes.Buffer
		if code := runGate([]string{"--ci"}, nil, &out, &errb); code != 1 {
			t.Fatalf("expected a failing gate, got %d: %s", code, out.String())
		}
		return out.String()
	}
	commands := func(out string) (n int) {
		for _, l := range strings.Split(out, "\n") {
			if strings.HasPrefix(l, "::") {
				n++
			}
		}
		return
	}

	// A theme name with a percent sign in the begin marker lands in a finding message.
	out := run("<!-- pr-brief:begin v1 style=iceberg theme=50%25x -->\n<!-- pr-brief:end -->\n")
	if !strings.Contains(out, "50%2525x") {
		t.Errorf("a percent sign must be escaped:\n%s", out)
	}

	// A style finding has a multi-line message: it must stay inside one command.
	fence := draw(t, flow, "--theme", "github-dark")
	edited := strings.Replace(fence, "#3fb950", "#3fb951", 1)
	out = run(fmt.Sprintf(descTemplate, "github-dark", edited))
	if !strings.Contains(out, "%0A") {
		t.Errorf("newlines in a message must be written as %%0A:\n%s", out)
	}
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, "::") && !strings.HasPrefix(l, "::error title=") {
			t.Errorf("an unexpected workflow command: %q", l)
		}
	}
	if n := commands(out); n < 1 || n > 3 {
		t.Errorf("one annotation per finding expected, got %d:\n%s", n, out)
	}
}

func TestInitWritesFilesAndNeverOverwrites(t *testing.T) {
	d := isolate(t)
	exec.Command("git", "init", "-q", d).Run()
	var out, errb bytes.Buffer
	if code := runInit([]string{"--dir", d, "--pr-template", "--workflow", "--style", "ste", "--theme", "dracula"}, &out, &errb); code != 0 {
		t.Fatalf("init: %s %s", out.String(), errb.String())
	}
	for _, f := range []string{".pr-brief.json", ".github/pull_request_template.md", ".github/workflows/pr-brief.yml"} {
		if _, err := os.Stat(filepath.Join(d, f)); err != nil {
			t.Errorf("%s was not written", f)
		}
	}
	cfg, _ := os.ReadFile(filepath.Join(d, ".pr-brief.json"))
	if !strings.Contains(string(cfg), `"style": "ste"`) || !strings.Contains(string(cfg), `"theme": "dracula"`) {
		t.Errorf("flags must reach a new config: %s", cfg)
	}
	tpl, _ := os.ReadFile(filepath.Join(d, ".github", "pull_request_template.md"))
	if !strings.Contains(string(tpl), "style=ste theme=dracula") {
		t.Errorf("the template must carry the repo's style and theme: %s", tpl)
	}
	wf, _ := os.ReadFile(filepath.Join(d, ".github", "workflows", "pr-brief.yml"))
	if !strings.Contains(string(wf), "pull_request.base.sha") {
		t.Error("the workflow must check out the base commit")
	}

	// A second run changes nothing, even when the user edited the files.
	os.WriteFile(filepath.Join(d, ".github", "pull_request_template.md"), []byte("my own template"), 0o644)
	os.WriteFile(filepath.Join(d, ".pr-brief.json"), []byte(`{"version":1,"style":"iceberg"}`), 0o644)
	out.Reset()
	if code := runInit([]string{"--dir", d, "--pr-template", "--workflow", "--style", "ste"}, &out, &errb); code != 0 {
		t.Fatalf("second init: %s", errb.String())
	}
	if got, _ := os.ReadFile(filepath.Join(d, ".github", "pull_request_template.md")); string(got) != "my own template" {
		t.Error("init overwrote an existing file")
	}
	if got, _ := os.ReadFile(filepath.Join(d, ".pr-brief.json")); !strings.Contains(string(got), `"iceberg"`) {
		t.Error("init overwrote an existing config")
	}
	if strings.Count(out.String(), "kept") != 3 {
		t.Errorf("every existing file should be reported as kept:\n%s", out.String())
	}
}

func TestInitRefusesABadTheme(t *testing.T) {
	d := isolate(t)
	exec.Command("git", "init", "-q", d).Run()
	var out, errb bytes.Buffer
	if code := runInit([]string{"--dir", d, "--theme", "solarized"}, &out, &errb); code == 0 {
		t.Error("an unknown theme must be refused")
	}
}

// initIn runs `pr-brief init` in a fresh git repo whose origin is remote.
func initIn(t *testing.T, remote string, args ...string) (string, string, int) {
	t.Helper()
	d := t.TempDir()
	exec.Command("git", "init", "-q", d).Run()
	if remote != "" {
		exec.Command("git", "-C", d, "remote", "add", "origin", remote).Run()
	}
	var out, errb bytes.Buffer
	code := runInit(append([]string{"--dir", d}, args...), &out, &errb)
	return d, out.String() + errb.String(), code
}

func TestInitWritesAnAzurePipelineForAnAzureRemote(t *testing.T) {
	d, out, code := initIn(t, "https://dev.azure.com/org/proj/_git/repo", "--workflow", "--pr-template")
	if code != 0 {
		t.Fatalf("code %d: %s", code, out)
	}
	if _, err := os.Stat(filepath.Join(d, ".azuredevops", "pr-brief.yml")); err != nil {
		t.Errorf("the pipeline file is missing: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(d, ".github")); err == nil {
		t.Error("an Azure DevOps repo must not get a .github directory")
	}
	if _, err := os.Stat(filepath.Join(d, ".azuredevops", "pull_request_template.md")); err != nil {
		t.Errorf("the PR template belongs in .azuredevops: %v", err)
	}
	if !strings.Contains(out, "Build Validation") || !strings.Contains(out, "4000") {
		t.Errorf("the next steps must name the policy and the limit:\n%s", out)
	}
}

func TestInitInlineWritesTheSettingsInTheWorkflowAndNoConfigFile(t *testing.T) {
	d, out, code := initIn(t, "https://github.com/a/b.git", "--workflow", "--config", "inline", "--style", "iceberg", "--theme", "dracula")
	if code != 0 {
		t.Fatalf("code %d: %s", code, out)
	}
	if _, err := os.Stat(filepath.Join(d, ".pr-brief.json")); err == nil {
		t.Error("inline mode must not write .pr-brief.json")
	}
	wf, _ := os.ReadFile(filepath.Join(d, ".github", "workflows", "pr-brief.yml"))
	if !strings.Contains(string(wf), "style: iceberg") || !strings.Contains(string(wf), "theme: dracula") {
		t.Errorf("the workflow lacks the settings:\n%s", wf)
	}
	if !strings.Contains(out, "no secret") {
		t.Errorf("the output must say no secret is needed:\n%s", out)
	}
}

func TestInitRejectsBadCIChoices(t *testing.T) {
	for _, args := range [][]string{
		{"--config", "inline"},                                    // inline needs --workflow
		{"--workflow", "--config", "toml"},                        // unknown mode
		{"--workflow", "--host", "gitlab"},                        // unknown host
		{"--workflow", "--config", "inline", "--theme", "x.json"}, // a path is not allowed inline
	} {
		if _, out, code := initIn(t, "", args...); code == 0 {
			t.Errorf("%v must fail:\n%s", args, out)
		}
	}
}

func TestGateStdinUsesTheAzureLimitInAPipeline(t *testing.T) {
	isolate(t)
	t.Setenv("TF_BUILD", "True")
	file := filepath.Join(t.TempDir(), "d.md")
	os.WriteFile(file, []byte(strings.Repeat("x", 4500)), 0o644)
	var out, errb bytes.Buffer
	code := runGate([]string{"--file", file}, nil, &out, &errb)
	if code != 1 || !strings.Contains(out.String(), "Azure DevOps rejects more than 4000") || !strings.Contains(out.String(), "##vso[task.logissue type=error]") {
		t.Errorf("code %d:\n%s%s", code, out.String(), errb.String())
	}
}

func TestGateCIAppliesWorkflowInputsOnlyWithoutAConfigFile(t *testing.T) {
	isolate(t)
	ws := t.TempDir()
	event := filepath.Join(t.TempDir(), "event.json")
	os.WriteFile(event, []byte(`{"pull_request":{"number":7,"body":"hello","user":{"type":"User"}}}`), 0o644)
	t.Setenv("GITHUB_EVENT_PATH", event)
	t.Setenv("GITHUB_WORKSPACE", ws)
	t.Setenv("PR_BRIEF_STYLE", "iceberg")

	var out, errb bytes.Buffer
	runGate([]string{"--ci"}, nil, &out, &errb)
	if strings.Contains(out.String(), "::notice") {
		t.Errorf("no file, so nothing is overridden:\n%s", out.String())
	}

	os.WriteFile(filepath.Join(ws, ".pr-brief.json"), []byte(`{"version":1,"style":"ste"}`), 0o644)
	out.Reset()
	runGate([]string{"--ci"}, nil, &out, &errb)
	if !strings.Contains(out.String(), "::notice title=pr-brief::The workflow sets style, but .pr-brief.json sets it too. The file wins.") {
		t.Errorf("the override must be reported:\n%s", out.String())
	}

	t.Setenv("PR_BRIEF_STYLE", "loud")
	if code := runGate([]string{"--ci"}, nil, &out, &errb); code != 2 {
		t.Errorf("a bad workflow input must be a usage error, got %d", code)
	}
}

func TestInitRewriteNeedsAnExplicitAuthAndGitHub(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"--rewrite"}, "needs --auth"},
		{[]string{"--rewrite", "--auth", "token"}, "needs --auth"},
		{[]string{"--auth", "api-key"}, "belongs to --rewrite"},
		{[]string{"--rewrite", "--auth", "api-key", "--host", "azure-devops"}, "GitHub only"},
	} {
		_, out, code := initIn(t, "", tc.args...)
		if code == 0 || !strings.Contains(out, tc.want) {
			t.Errorf("%v: code %d, want an error with %q:\n%s", tc.args, code, tc.want, out)
		}
	}
}

func TestInitRewriteWritesTheCombinedWorkflow(t *testing.T) {
	d, out, code := initIn(t, "https://github.com/a/b.git", "--rewrite", "--auth", "federation")
	if code != 0 {
		t.Fatalf("code %d: %s", code, out)
	}
	wf, _ := os.ReadFile(filepath.Join(d, ".github", "workflows", "pr-brief.yml"))
	if !strings.Contains(string(wf), "anthropic_federation_rule_id") || !strings.Contains(string(wf), "  description:") {
		t.Errorf("the workflow lacks the rewrite or gate job:\n%s", wf)
	}
	for _, want := range []string{"ANTHROPIC_FEDERATION_RULE_ID", "required check"} {
		if !strings.Contains(out, want) {
			t.Errorf("the next steps must mention %q:\n%s", want, out)
		}
	}
}

func TestGateCIReadsTheRefreshedDescription(t *testing.T) {
	isolate(t)
	ws := t.TempDir()
	event := filepath.Join(t.TempDir(), "event.json")
	os.WriteFile(event, []byte(`{"pull_request":{"number":7,"body":"the old text","user":{"type":"User"}}}`), 0o644)
	fresh := filepath.Join(t.TempDir(), "body.md")
	os.WriteFile(fresh, []byte(strings.Repeat("x", 70000)), 0o644)
	t.Setenv("GITHUB_EVENT_PATH", event)
	t.Setenv("GITHUB_WORKSPACE", ws)

	var out, errb bytes.Buffer
	runGate([]string{"--ci"}, nil, &out, &errb)
	if strings.Contains(out.String(), "70000") {
		t.Fatalf("without the file, the payload text is checked:\n%s", out.String())
	}
	t.Setenv("PR_BRIEF_BODY_FILE", fresh)
	out.Reset()
	if code := runGate([]string{"--ci"}, nil, &out, &errb); code != 1 || !strings.Contains(out.String(), "description is 70000 characters") {
		t.Errorf("the refreshed text must be the one checked, code %d:\n%s", code, out.String())
	}
	t.Setenv("PR_BRIEF_BODY_FILE", filepath.Join(ws, "missing.md"))
	if code := runGate([]string{"--ci"}, nil, &out, &errb); code != 2 {
		t.Errorf("an unreadable refresh file is a usage error, got %d", code)
	}
}

const unlinkedBody = "## Review guide\n**Read these first**\n| File | Why | Check |\n|---|---|---|\n| `cmd/a.go` | x | y |\n| `docs/b c.md` | x | y |\n\n**Review order**: a\n"

func runLinksOn(t *testing.T, body string, args ...string) (string, string, int) {
	t.Helper()
	var out, errb bytes.Buffer
	code := runLinks(args, strings.NewReader(body), &out, &errb)
	return out.String(), errb.String(), code
}

func TestLinksFollowsThePRNotACommit(t *testing.T) {
	const remote = "git@github.com:gagoar/pr-brief.git"
	// Before the PR exists: the file on the branch.
	out, msg, code := runLinksOn(t, unlinkedBody, "--remote", remote, "--branch", "feature/x")
	if code != 0 || !strings.Contains(out, "https://github.com/gagoar/pr-brief/blob/feature/x/cmd/a.go") || !strings.Contains(out, "docs/b%20c.md") {
		t.Fatalf("code %d, %s\n%s", code, msg, out)
	}
	// Once the PR exists: the diff in the PR, which shows the latest commit.
	final, _, code := runLinksOn(t, out, "--remote", remote, "--pr", "13")
	if code != 0 || !strings.Contains(final, "https://github.com/gagoar/pr-brief/pull/13/files#diff-") || strings.Contains(final, "/blob/") {
		t.Fatalf("code %d\n%s", code, final)
	}
}

// A description passes the gate at both stages, before and after the PR exists.
func TestLinkedDescriptionPassesTheGateAtBothStages(t *testing.T) {
	flow := fixtureFlow(t) // before isolate: it reads a file by a relative path
	d := isolate(t)
	fence := draw(t, flow, "--theme", "dracula")
	plain := strings.Replace(fmt.Sprintf(descTemplate, "dracula", fence), links.Cell("hook.go", links.Repo{Host: host.GitHub, Owner: "o", Name: "r"}.DiffURL(4, "hook.go")), "`hook.go`", 1)
	if code, out := gateText(t, d, plain); code != 1 || !strings.Contains(out, "must be a link") {
		t.Fatalf("a plain path must fail with the fix: code=%d %s", code, out)
	}
	const remote = "git@github.com:gagoar/pr-brief.git"
	onBranch, _, _ := runLinksOn(t, plain, "--remote", remote, "--branch", "feature/x")
	inPR, _, _ := runLinksOn(t, onBranch, "--remote", remote, "--pr", "13")
	for name, body := range map[string]string{"before the PR": onBranch, "after the PR": inPR} {
		if code, out := gateText(t, d, body); code != 0 {
			t.Errorf("%s: %s", name, out)
		}
	}
}

func TestLinksFailsClearly(t *testing.T) {
	for name, tc := range map[string]struct {
		body string
		args []string
		want string
	}{
		"another host":    {unlinkedBody, []string{"--remote", "https://gitlab.com/a/b.git", "--pr", "1"}, "cannot tell the host"},
		"no table":        {"just text", []string{"--remote", "git@github.com:a/b.git", "--pr", "1"}, "no Read-these-first table"},
		"unreadable cell": {strings.Replace(unlinkedBody, "`cmd/a.go`", "cmd/a.go", 1), []string{"--remote", "git@github.com:a/b.git", "--pr", "1"}, "cannot read these cells"},
	} {
		out, msg, code := runLinksOn(t, tc.body, tc.args...)
		if code != 1 || out != "" || !strings.Contains(msg, tc.want) {
			t.Errorf("%s: code %d, out %q, msg %q", name, code, out, msg)
		}
	}
}

const originalWithTickets = `Fixes the invite flow.

Closes ENG-42. Also tracked in [ABC-101](https://acme.atlassian.net/browse/ABC-101) and
https://linear.app/acme/issue/eng-7/rate-limit, plus PAY-9 for the mailer.
`

const newBlock = "<!-- pr-brief:begin v1 style=ste -->\n## Brief\nThe invite flow now sends a code.\n<!-- pr-brief:end -->\n"

func improveWith(t *testing.T, mode, current string, extra ...string) (string, string, int) {
	t.Helper()
	d := isolate(t)
	cur := filepath.Join(d, "cur.md")
	man := filepath.Join(d, "managed.md")
	os.WriteFile(cur, []byte(current), 0o644)
	os.WriteFile(man, []byte(newBlock), 0o644)
	args := append([]string{"improve", "--managed", man, "--current", cur, "--owner", "o", "--repo", "r", "--pr", "5", "--mode", mode}, extra...)
	var out, errb bytes.Buffer
	code := runBody(args, nil, &out, &errb)
	return out.String(), errb.String(), code
}

// A rewrite must never lose a ticket: Jira and Linear link a PR by the key in its description.
func TestImproveKeepsEveryTicketOfTheOldDescriptionAndTheBranch(t *testing.T) {
	for _, mode := range []string{"drop", "comment"} {
		out, msg, code := improveWith(t, mode, originalWithTickets, "--branch", "gago/eng-99-invites")
		if code != 0 {
			t.Fatalf("%s: %s", mode, msg)
		}
		for _, want := range []string{"Closes ENG-42", "[ABC-101](https://acme.atlassian.net/browse/ABC-101)", "https://linear.app/acme/issue/eng-7/rate-limit", "PAY-9", "ENG-99"} {
			if !strings.Contains(out, want) {
				t.Errorf("%s: the new description lost %q:\n%s", mode, want, out)
			}
		}
		// The line is visible and sits above the markers, so a rewrite of the block cannot take it.
		line := strings.Index(out, "**Tickets:**")
		begin := strings.Index(out, "<!-- pr-brief:begin")
		if line < 0 || begin < 0 || line > begin {
			t.Errorf("%s: the Tickets line must come before the begin marker:\n%s", mode, out)
		}
		if !strings.Contains(msg, "kept 5 tickets") {
			t.Errorf("%s: say what was kept: %q", mode, msg)
		}
	}
}

func TestImproveIsIdempotentForTickets(t *testing.T) {
	first, _, _ := improveWith(t, "drop", originalWithTickets, "--branch", "main")
	second, _, code := improveWith(t, "drop", first, "--branch", "main")
	if code != 0 || strings.Count(second, "**Tickets:**") != 1 || second != first {
		t.Errorf("a second run must not repeat or lose tickets (code %d):\n%s", code, second)
	}
}

func TestImproveWithNoTicketsAddsNoLine(t *testing.T) {
	out, _, code := improveWith(t, "drop", "Hand written notes\n", "--branch", "feature/no-ticket")
	if code != 0 || strings.Contains(out, "**Tickets:**") {
		t.Errorf("no tickets, no line (code %d):\n%s", code, out)
	}
}

func TestImproveFitsTheTicketsLineInsideTheHostLimit(t *testing.T) {
	long := originalWithTickets + strings.Repeat("An earlier paragraph that is quite long. ", 200)
	out, msg, code := improveWith(t, "comment", long, "--branch", "main", "--host", "dev.azure.com")
	if code != 0 {
		t.Fatalf("%s", msg)
	}
	if n := utf8.RuneCountInString(out); n > 4000 {
		t.Errorf("the description is %d characters; Azure DevOps takes 4000", n)
	}
	if !strings.Contains(out, "ENG-42") || !strings.Contains(out, "PAY-9") {
		t.Errorf("a long earlier description must not push a ticket out:\n%s", out)
	}
}

func TestTicketsCommandForANewPR(t *testing.T) {
	isolate(t)
	var out, errb bytes.Buffer
	if code := runTickets([]string{"--branch", "feature/ABC-12-login"}, strings.NewReader(newBlock), &out, &errb); code != 0 {
		t.Fatalf("%s", errb.String())
	}
	if !strings.HasPrefix(out.String(), "**Tickets:** ABC-12\n\n<!-- pr-brief:begin") {
		t.Errorf("the branch ticket leads the description:\n%s", out.String())
	}
	out.Reset()
	if code := runTickets([]string{"--list", "--branch", "main"}, strings.NewReader(originalWithTickets), &out, &errb); code != 0 || strings.Count(out.String(), "\n") != 4 {
		t.Errorf("--list prints one ticket per line, got %d:\n%s", code, out.String())
	}
}

func TestGateWarnsWhenTheBranchTicketIsMissingButDoesNotFail(t *testing.T) {
	isolate(t)
	file := filepath.Join(t.TempDir(), "d.md")
	os.WriteFile(file, []byte("> pr-brief skipped: release notes elsewhere\n"), 0o644)
	var out, errb bytes.Buffer
	if code := runGate([]string{"--file", file, "--branch", "feature/ENG-5-x"}, nil, &out, &errb); code != 0 || !strings.Contains(out.String(), "warn:") || !strings.Contains(out.String(), "ENG-5") {
		t.Errorf("warn, not fail: code %d\n%s", code, out.String())
	}
	out.Reset()
	os.WriteFile(file, []byte("> pr-brief skipped: see ENG-5\n"), 0o644)
	if code := runGate([]string{"--file", file, "--branch", "feature/ENG-5-x"}, nil, &out, &errb); code != 0 || strings.Contains(out.String(), "warn:") {
		t.Errorf("a described ticket gives no warning: code %d\n%s", code, out.String())
	}
}

func TestGateCIAnnotatesAMissingBranchTicket(t *testing.T) {
	isolate(t)
	event := filepath.Join(t.TempDir(), "event.json")
	os.WriteFile(event, []byte(`{"pull_request":{"number":7,"body":"> pr-brief skipped: notes elsewhere","user":{"type":"User"},"head":{"ref":"gago/eng-45-fix"}}}`), 0o644)
	t.Setenv("GITHUB_EVENT_PATH", event)
	t.Setenv("GITHUB_WORKSPACE", t.TempDir())
	var out, errb bytes.Buffer
	if code := runGate([]string{"--ci"}, nil, &out, &errb); code != 0 || !strings.Contains(out.String(), "::warning title=pr-brief::") || !strings.Contains(out.String(), "ENG-45") {
		t.Errorf("code %d\n%s", code, out.String())
	}
}
