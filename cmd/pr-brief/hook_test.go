package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	got := plainCommand(`C:\Users\a b\pr-brief.exe`, "windows")
	if want := `"C:/Users/a b/pr-brief.exe" gate --hook`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
	if got := plainCommand("/opt/pr-brief", "linux"); got != `"/opt/pr-brief" gate --hook` {
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

const (
	oldLauncherBin = "/home/u/.cache/pr-brief/v0.7.0/pr-brief-darwin-arm64"
	newLauncherBin = "/home/u/.cache/pr-brief/v0.8.0/pr-brief-darwin-arm64"
)

func TestHookCommandFollowsThePluginOnlyForLauncherBinaries(t *testing.T) {
	c, stable := hookCommand(newLauncherBin, false, "darwin")
	if !stable || !strings.Contains(c, stableTag) || !strings.Contains(c, hookMarker) || strings.Contains(c, "v0.8.0") {
		t.Errorf("a launcher binary gets a command with no version in it: %v %s", stable, c)
	}
	if !isOurs(hookEntry(c)) {
		t.Error("remove and a second install must still find the entry")
	}
	for name, tc := range map[string]struct {
		bin      string
		explicit bool
		goos     string
	}{
		"explicit --bin":  {newLauncherBin, true, "darwin"},
		"windows":         {newLauncherBin, false, "windows"},
		"a local build":   {"/repo/plugin/bin/pr-brief-darwin-arm64", false, "darwin"},
		"some other path": {"/opt/pr-brief", false, "linux"},
	} {
		got, stable := hookCommand(tc.bin, tc.explicit, tc.goos)
		if stable || got != `"`+tc.bin+`" `+hookMarker {
			t.Errorf("%s keeps the plain path: %v %s", name, stable, got)
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

func TestStableCommandRunsTheNewestInstalledVersion(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the command is a POSIX shell command")
	}
	if out, err := exec.Command("sh", "-c", "printf 'v0.10.0\\nv0.9.0\\n' | sort -V | tail -1").Output(); err != nil || strings.TrimSpace(string(out)) != "v0.10.0" {
		t.Skip("this sort has no -V")
	}
	home := t.TempDir()
	for _, v := range []string{"0.9.0", "0.10.0", "0.8.0"} {
		dir := filepath.Join(home, ".claude", "plugins", "cache", "gago-plugins", "pr-brief", v, "bin")
		os.MkdirAll(dir, 0o755)
		os.WriteFile(filepath.Join(dir, "pr-brief"), []byte("#!/bin/sh\necho \"plugin "+v+" $*\"\n"), 0o755)
	}
	// A plugin of the same name from another marketplace, with a higher version, must never run.
	evil := filepath.Join(home, ".claude", "plugins", "cache", "other-market", "pr-brief", "9.9.9", "bin")
	os.MkdirAll(evil, 0o755)
	os.WriteFile(filepath.Join(evil, "pr-brief"), []byte("#!/bin/sh\necho evil\n"), 0o755)
	cmd, _ := hookCommand(newLauncherBin, false, "linux")
	run := func(home string) (string, error) {
		c := exec.Command("sh", "-c", cmd)
		c.Env = []string{"HOME=" + home, "PATH=" + os.Getenv("PATH")}
		out, err := c.CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}
	if out, err := run(home); err != nil || out != "plugin 0.10.0 gate --hook" {
		t.Errorf("version order, not text order, and only the two named marketplaces: %v %q", err, out)
	}

	// No plugin: fall back to the newest binary in the launcher cache.
	home2 := t.TempDir()
	for _, v := range []string{"0.9.0", "0.10.0"} {
		dir := filepath.Join(home2, ".cache", "pr-brief", "v"+v)
		os.MkdirAll(dir, 0o755)
		os.WriteFile(filepath.Join(dir, "pr-brief-darwin-arm64"), []byte("#!/bin/sh\necho \"cache "+v+" $*\"\n"), 0o755)
	}
	if out, err := run(home2); err != nil || out != "cache 0.10.0 gate --hook" {
		t.Errorf("launcher cache fallback: %v %q", err, out)
	}

	// Nothing installed: say so, and fail.
	if out, err := run(t.TempDir()); err == nil || !strings.Contains(out, "no installed binary found") {
		t.Errorf("an empty machine must say what to do: %v %q", err, out)
	}
}

func TestHookCheckStates(t *testing.T) {
	d := isolate(t)
	t.Setenv("HOME", d)
	path := filepath.Join(d, ".claude", "settings.json")

	// No settings file at all: nothing to report, exit 0.
	if code, out := hookRun(t, "check", "--scope", "user", "--bin", newLauncherBin); code != 0 || !strings.Contains(out, "no pr-brief hook") {
		t.Errorf("no file: %d %q", code, out)
	}
	if code, out := hookRun(t, "check", "--bin", newLauncherBin); code != 0 || out != "" {
		t.Errorf("no scope and no hook is silent: %d %q", code, out)
	}

	write := func(command string) {
		m := map[string]any{}
		installHook(m, command)
		os.MkdirAll(filepath.Dir(path), 0o755)
		data, _ := json.Marshal(m)
		os.WriteFile(path, data, 0o644)
	}

	write(`"` + oldLauncherBin + `" ` + hookMarker)
	if code, out := hookRun(t, "check", "--scope", "user", "--bin", newLauncherBin); code != 1 || !strings.Contains(out, "out of date") || !strings.Contains(out, "pr-brief hook install --scope user") {
		t.Errorf("an older downloaded version is out of date, and the output gives the fix: %d %q", code, out)
	}
	if code, out := hookRun(t, "check", "--bin", newLauncherBin); code != 1 || !strings.Contains(out, "out of date") {
		t.Errorf("without --scope it looks in the user file: %d %q", code, out)
	}

	write(`"` + newLauncherBin + `" ` + hookMarker)
	if code, out := hookRun(t, "check", "--scope", "user", "--bin", newLauncherBin); code != 0 || !strings.Contains(out, "ok") {
		t.Errorf("the hook names the binary that runs: %d %q", code, out)
	}

	stableCmd, _ := hookCommand(newLauncherBin, false, "linux")
	write(stableCmd)
	if code, out := hookRun(t, "check", "--scope", "user", "--bin", newLauncherBin); code != 0 || !strings.Contains(out, "follows the newest") {
		t.Errorf("a stable hook never goes stale: %d %q", code, out)
	}

	// A path the user chose that exists is left alone. A launcher path that is gone is stale.
	custom := filepath.Join(d, "my-pr-brief")
	os.WriteFile(custom, []byte("#!/bin/sh\n"), 0o755)
	write(`"` + custom + `" ` + hookMarker)
	if code, out := hookRun(t, "check", "--scope", "user", "--bin", newLauncherBin); code != 0 || !strings.Contains(out, "you chose") {
		t.Errorf("a custom path is not judged: %d %q", code, out)
	}
	write(`"` + filepath.Join(d, "gone", "pr-brief") + `" ` + hookMarker)
	if code, out := hookRun(t, "check", "--scope", "user", "--bin", newLauncherBin); code != 1 || !strings.Contains(out, "out of date") {
		t.Errorf("a binary that is gone is stale: %d %q", code, out)
	}

	// An unreadable settings file is an error, not a pass.
	os.WriteFile(path, []byte("{not json"), 0o644)
	if code, _ := hookRun(t, "check", "--scope", "user", "--bin", newLauncherBin); code != 1 {
		t.Errorf("a broken settings file must fail: %d", code)
	}
}
