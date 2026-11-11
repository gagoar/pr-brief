package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func hookRun(t *testing.T, args ...string) (int, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := runHook(args, &out, &errb)
	return code, out.String() + errb.String()
}

func readJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("%s: %v\n%s", path, err, data)
	}
	return m
}

func preEntries(m map[string]any) []any {
	h, _ := m["hooks"].(map[string]any)
	l, _ := h["PreToolUse"].([]any)
	return l
}

func TestHookInstallMergesAndKeepsEverythingElse(t *testing.T) {
	d := isolate(t)
	t.Setenv("HOME", d)
	path := filepath.Join(d, ".claude", "settings.json")
	other := `{"model":"opus","hooks":{"PreToolUse":[{"matcher":"Write","hooks":[{"type":"command","command":"echo hi"}]}],"Stop":[{"hooks":[{"type":"command","command":"echo stop"}]}]}}`
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(path, []byte(other), 0o644)

	if code, out := hookRun(t, "install", "--scope", "user", "--bin", "/opt/pr-brief"); code != 0 || !strings.Contains(out, "added") {
		t.Fatalf("install: %d %s", code, out)
	}
	m := readJSON(t, path)
	if m["model"] != "opus" || len(preEntries(m)) != 2 || m["hooks"].(map[string]any)["Stop"] == nil {
		t.Errorf("install must keep every other setting and hook: %v", m)
	}
	if bak, err := os.ReadFile(path + ".pr-brief.bak"); err != nil || string(bak) != other {
		t.Errorf("the file as it was must be saved once: %v %q", err, bak)
	}
	last := preEntries(m)[1].(map[string]any)["hooks"].([]any)[0].(map[string]any)
	if last["type"] != "command" || last["command"] != `"/opt/pr-brief" gate --hook` {
		t.Errorf("unexpected command: %v", last)
	}

	// Again with the same binary: nothing changes. With another binary: the path is updated, not duplicated.
	if code, out := hookRun(t, "install", "--scope", "user", "--bin", "/opt/pr-brief"); code != 0 || !strings.Contains(out, "unchanged") {
		t.Errorf("a second install must change nothing: %d %s", code, out)
	}
	if code, out := hookRun(t, "install", "--scope", "user", "--bin", "/new/pr-brief"); code != 0 || !strings.Contains(out, "updated") {
		t.Errorf("a new path must update the entry: %d %s", code, out)
	}
	if n := len(preEntries(readJSON(t, path))); n != 2 {
		t.Errorf("want 2 PreToolUse entries after an update, got %d", n)
	}

	if code, out := hookRun(t, "remove", "--scope", "user"); code != 0 || !strings.Contains(out, "removed 1") {
		t.Fatalf("remove: %d %s", code, out)
	}
	m = readJSON(t, path)
	if e := preEntries(m); len(e) != 1 || m["model"] != "opus" {
		t.Errorf("remove must take out only our entry: %v", m)
	}
}

func TestHookInstallIntoNoFileThenRemoveLeavesNoHooks(t *testing.T) {
	d := isolate(t)
	t.Setenv("HOME", d)
	if code, out := hookRun(t, "install", "--scope", "user", "--bin", "/x/pr-brief"); code != 0 {
		t.Fatalf("%d %s", code, out)
	}
	path := filepath.Join(d, ".claude", "settings.json")
	if len(preEntries(readJSON(t, path))) != 1 {
		t.Error("want one entry")
	}
	if code, _ := hookRun(t, "remove", "--scope", "user"); code != 0 {
		t.Fatal("remove failed")
	}
	if m := readJSON(t, path); m["hooks"] != nil {
		t.Errorf("an empty hooks object must go: %v", m)
	}
}

func TestHookRepoScopeWritesInTheRepo(t *testing.T) {
	d := isolate(t)
	t.Setenv("HOME", filepath.Join(d, "home"))
	os.MkdirAll(filepath.Join(d, ".git"), 0o755)
	os.MkdirAll(filepath.Join(d, "sub"), 0o755)
	if code, out := hookRun(t, "install", "--scope", "repo", "--dir", filepath.Join(d, "sub"), "--bin", "/x/pr-brief"); code != 0 {
		t.Fatalf("%d %s", code, out)
	}
	if _, err := os.Stat(filepath.Join(d, ".claude", "settings.json")); err != nil {
		t.Errorf("the repo settings must be at the repository root: %v", err)
	}
}

func TestHookRefusesWhatItCannotReadAndNeedsAScope(t *testing.T) {
	d := isolate(t)
	t.Setenv("HOME", d)
	path := filepath.Join(d, ".claude", "settings.json")
	os.MkdirAll(filepath.Dir(path), 0o755)
	for name, content := range map[string]string{"not json": "{oops", "hooks not an object": `{"hooks":[]}`, "PreToolUse not a list": `{"hooks":{"PreToolUse":{}}}`} {
		os.WriteFile(path, []byte(content), 0o644)
		if code, _ := hookRun(t, "install", "--scope", "user", "--bin", "/x"); code != 1 {
			t.Errorf("%s: want exit 1, got %d", name, code)
		}
		if got, _ := os.ReadFile(path); string(got) != content {
			t.Errorf("%s: the file must not change, got %q", name, got)
		}
	}
	if code, _ := hookRun(t, "install"); code != 2 {
		t.Error("install without --scope must fail")
	}
	if code, _ := hookRun(t, "install", "--scope", "team"); code != 1 {
		t.Error("an unknown scope must fail")
	}
	if code, out := hookRun(t, "show", "--bin", "/x/pr-brief"); code != 0 || !strings.Contains(out, `"matcher"`) || !strings.Contains(out, "settings.json") {
		t.Errorf("show: %d %s", code, out)
	}
}

func TestThePluginDeclaresNoHook(t *testing.T) {
	// The plugin directory refuses a plugin that declares a command hook. A user opts in with `pr-brief hook install`.
	if _, err := os.Stat(filepath.Join("..", "..", "plugin", ".claude-plugin", "hooks.json")); err == nil {
		t.Error("plugin/.claude-plugin/hooks.json must not exist")
	}
	data, err := os.ReadFile(filepath.Join("..", "..", "plugin", ".claude-plugin", "plugin.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	if _, has := m["hooks"]; has {
		t.Error(`plugin.json must not have a "hooks" field`)
	}
}

func TestHookCommandWindowsPath(t *testing.T) {
	got := hookCommand(`C:\Users\a b\pr-brief.exe`, "windows")
	if want := `"C:/Users/a b/pr-brief.exe" gate --hook`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
	if got := hookCommand("/opt/pr-brief", "linux"); got != `"/opt/pr-brief" gate --hook` {
		t.Errorf("unix path changed: %s", got)
	}
}

func TestLaunchersHaveLFEndings(t *testing.T) {
	for _, p := range []string{"../../plugin/bin/pr-brief", "../../plugin/skills/pr-brief/scripts/pr-brief"} {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(data, []byte("\r")) {
			t.Errorf("%s has CR bytes: the #!/bin/sh line breaks on Windows checkouts", p)
		}
	}
}

func TestHookStatusAndAsked(t *testing.T) {
	d := t.TempDir()
	t.Setenv("HOME", d)
	t.Setenv("PR_BRIEF_CACHE", filepath.Join(d, "cache"))
	repo := filepath.Join(d, "repo")
	os.MkdirAll(filepath.Join(repo, ".git"), 0o755)

	if code, out := hookRun(t, "status", "--dir", repo); code != 0 || out != "user=false\nrepo=false\nasked=false\n" {
		t.Fatalf("fresh status: %d %q", code, out)
	}
	if code, out := hookRun(t, "install", "--scope", "user", "--bin", "/opt/pr-brief"); code != 0 {
		t.Fatalf("install: %d %s", code, out)
	}
	if code, out := hookRun(t, "asked"); code != 0 || out != "" {
		t.Fatalf("asked: %d %q", code, out)
	}
	if _, out := hookRun(t, "status", "--dir", repo); out != "user=true\nrepo=false\nasked=true\n" {
		t.Errorf("after install and asked: %q", out)
	}
	if code, out := hookRun(t, "status", "--harness", "nope"); code != 0 || !strings.Contains(out, "user=false") {
		t.Errorf("an unknown harness must not crash status: %d %q", code, out)
	}
}
