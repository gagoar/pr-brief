package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHookCodexRepoScopeWritesHooksJSON(t *testing.T) {
	d := isolate(t)
	t.Setenv("HOME", filepath.Join(d, "home"))
	os.MkdirAll(filepath.Join(d, ".git"), 0o755)
	code, out := hookRun(t, "install", "--harness", "codex", "--scope", "repo", "--bin", "/x/pr-brief")
	if code != 0 {
		t.Fatalf("%d %s", code, out)
	}
	path := filepath.Join(d, ".codex", "hooks.json")
	if len(preEntries(readJSON(t, path))) != 1 {
		t.Errorf("want one entry in %s", path)
	}
	if !strings.Contains(out, "/hooks") {
		t.Errorf("a Codex install must tell the user to trust the hook in /hooks: %s", out)
	}
	if _, err := os.Stat(filepath.Join(d, ".claude")); err == nil {
		t.Error("a Codex install must not touch .claude")
	}
	if code, _ := hookRun(t, "remove", "--harness", "codex", "--scope", "repo"); code != 0 {
		t.Fatal("remove failed")
	}
	if m := readJSON(t, path); m["hooks"] != nil {
		t.Errorf("an empty hooks object must go: %v", m)
	}
}

func TestHookCodexUserScopeHonorsCodexHome(t *testing.T) {
	d := isolate(t)
	t.Setenv("HOME", filepath.Join(d, "home"))
	t.Setenv("CODEX_HOME", filepath.Join(d, "ch"))
	if code, out := hookRun(t, "install", "--harness", "codex", "--scope", "user", "--bin", "/x/pr-brief"); code != 0 {
		t.Fatalf("%d %s", code, out)
	}
	if _, err := os.Stat(filepath.Join(d, "ch", "hooks.json")); err != nil {
		t.Errorf("user scope must write $CODEX_HOME/hooks.json: %v", err)
	}
	t.Setenv("CODEX_HOME", "")
	if code, out := hookRun(t, "show", "--harness", "codex", "--bin", "/x"); code != 0 || !strings.Contains(out, filepath.Join(".codex", "hooks.json")) {
		t.Errorf("without CODEX_HOME the file is ~/.codex/hooks.json: %d %s", code, out)
	}
}

func TestHookRejectsAnUnknownHarness(t *testing.T) {
	isolate(t)
	if code, _ := hookRun(t, "show", "--harness", "vim"); code != 1 {
		t.Error("an unknown harness must fail")
	}
}

// The payload below is what Codex 0.167 sent to a PreToolUse hook for a shell command, captured from a
// real run. The gate must read it as it reads Claude Code's, so one hook entry serves both.
func TestGateReadsARealCodexPayload(t *testing.T) {
	payload, err := os.ReadFile(filepath.Join("testdata", "codex-pretooluse.json")) // before isolate changes the directory
	if err != nil {
		t.Fatal(err)
	}
	d := isolate(t)
	var out, errb bytes.Buffer
	if code := runGate([]string{"--hook"}, strings.NewReader(string(payload)), &out, &errb); code != 0 {
		t.Fatalf("a hook always exits 0, got %d: %s", code, errb.String())
	}
	if !strings.Contains(out.String(), `"permissionDecision":"deny"`) {
		t.Errorf("a PR create with no brief must be denied, got %q (dir %s)", out.String(), d)
	}
}
