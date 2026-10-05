package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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

const descTemplate = "<!-- pr-brief:begin v1 style=iceberg theme=%s -->\n" +
	"## Brief\nThe hook reads a tool call and decides if a PR description may pass.\n\n" +
	"## Change map\n### Flow 1: I1 -> a PR command is checked\n%s\n" +
	"| Ref | What | Detail |\n|---|---|---|\n| I1 | `gate --hook` | Claude Code sends the call as JSON. |\n| O1 | decision | The hook prints a deny decision or nothing. |\n\n" +
	"green added\n\n## Review guide\n**What changed**:\n- `gateHook()` reads the call.\n\n" +
	"**Read these first**\n| File | Why it is delicate | What to check |\n|---|---|---|\n| `hook.go` | It decides. | Check bad payloads. |\n\n" +
	"**Review order**: Start at `gateHook()`.\n<!-- pr-brief:end -->\n"

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
	  "functions":[{"node":"F1","label":"save()","status":"added","riskScore":3}],
	  "edges":[{"from":"I1","to":"F1","status":"new"},{"from":"F1","to":"O1","status":"new"}]}]}`
	p := filepath.Join(d, "report.json")
	os.WriteFile(p, []byte(report), 0o644)
	var out, errb bytes.Buffer
	if code := runDiagram([]string{"--report", p, "--flow-number", "1", "--raw"}, nil, &out, &errb); code != 0 {
		t.Fatalf("diagram --report: %s", errb.String())
	}
	if !strings.Contains(out.String(), `O1[("O1")]`) || strings.Contains(out.String(), "```") {
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
		for _, want := range []string{"graph LR", `I1(["I1"])`, "Api.Save()", "~~~", "classDef zone"} {
			if !strings.Contains(out.String(), want) {
				t.Errorf("%s: output lacks %q:\n%s", th, want, out.String())
			}
		}
	}
}
