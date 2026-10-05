package main

import (
	"bytes"
	"encoding/json"
	"os"
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
