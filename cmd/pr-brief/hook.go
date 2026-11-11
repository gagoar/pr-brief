package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// hookMatcher selects the tool calls the gate reads: every Bash call, and the MCP tools that create or
// update a pull request or a merge request.
const hookMatcher = "Bash|mcp__.*(create|update)_(pull|merge)_request"

// hookMarker is the part of the command that tells our entry from any other in settings.json.
const hookMarker = "gate --hook"

const hookUsage = `usage: pr-brief hook <show|install|check|remove|status|asked> [flags]

The plugin declares no hook. This command lets you opt in yourself. It adds a PreToolUse hook to your
own Claude Code settings.json, or to your Codex hooks.json, so the harness runs the gate before a PR
command (gh pr create|edit, az repos pr, glab mr, the MCP PR tools) and blocks a description that
fails it. Claude Code and Codex send the same payload, so one entry serves both.

  show      print the hook entry, and the file it would go in
  install   add the hook to the file (--scope user|repo), or update it
  check     say whether the hook is out of date (exit 1 when it is)
  remove    take the hook out again; nothing else in the file changes
  status    print installed (user and repo scope) and asked, one key=value per line
  asked     record that the user was offered the hook, so the skill does not offer it again

flags:
  --harness claude|codex  claude (default) writes settings.json; codex writes hooks.json
  --scope user|repo       user is ~/.claude/settings.json or ~/.codex/hooks.json ($CODEX_HOME is honored);
                          repo is .claude/settings.json or .codex/hooks.json in this repository
                          (check looks in both when you leave it out)
  --dir DIR               a directory inside the repository (default: .)
  --bin PATH              the pr-brief binary the hook runs (default: this binary)
`

// launcherCacheRe matches a binary that the plugin launcher downloaded, such as
// ~/.cache/pr-brief/v0.8.0/pr-brief-darwin-arm64. Its path holds the version, so a hook that names it
// keeps running that version after a plugin update.
var launcherCacheRe = regexp.MustCompile(`[/\\]pr-brief[/\\]v\d+\.\d+\.\d+[^/\\]*[/\\](pr-brief-(?:darwin|linux)-(?:amd64|arm64))$`)

// launcherBinary reports whether bin is a binary that the plugin launcher downloaded, and gives its file name.
func launcherBinary(bin string) (name string, ok bool) {
	m := launcherCacheRe.FindStringSubmatch(bin)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// stableTag marks a command that finds the newest installed pr-brief when it runs.
const stableTag = ": newest installed pr-brief;"

// stableCommand is a hook command that does not name a version. It runs the launcher of the newest
// installed plugin (the launcher downloads and checks the release binary for that version), or else the
// newest binary in the launcher cache. A plugin update then needs no change to settings.json. It looks only
// in the two marketplaces that the README names, gago-plugins and pr-brief, so a plugin of the same name from
// another marketplace is never run.
func stableCommand(name string) string {
	return `sh -c '` + stableTag + ` b=$(ls "$HOME"/.claude/plugins/cache/gago-plugins/pr-brief/*/bin/pr-brief "$HOME"/.claude/plugins/cache/pr-brief/pr-brief/*/bin/pr-brief 2>/dev/null | sort -V | tail -1);` +
		` [ -n "$b" ] || b=$(ls "${XDG_CACHE_HOME:-$HOME/.cache}"/pr-brief/v*/` + name + ` 2>/dev/null | sort -V | tail -1);` +
		` [ -n "$b" ] || { echo "pr-brief: no installed binary found. Run: pr-brief hook install" >&2; exit 1; };` +
		` exec "$b" ` + hookMarker + `'`
}

// plainCommand is the hook command that names one binary. The path is double-quoted as is, with no Go
// escapes: %q doubles every backslash, which some Windows shells read literally. On Windows the separators
// become forward slashes, which Git Bash, cmd and PowerShell all accept.
func plainCommand(bin, goos string) string {
	if goos == "windows" {
		bin = strings.ReplaceAll(bin, `\`, "/")
	}
	return `"` + bin + `" ` + hookMarker
}

// hookCommand picks the command for the hook. A binary the plugin launcher downloaded gets the command that
// follows the plugin; --bin (explicit), Windows, Codex and any other binary keep the plain path.
func hookCommand(bin string, explicit bool, goos string) (command string, stable bool) {
	if !explicit && goos != "windows" {
		if name, ok := launcherBinary(bin); ok {
			return stableCommand(name), true
		}
	}
	return plainCommand(bin, goos), false
}

func hookEntry(command string) map[string]any {
	return map[string]any{
		"matcher": hookMatcher,
		"hooks": []any{map[string]any{
			"type":    "command",
			"command": command,
		}},
	}
}

// settingsPath is the file the hook goes in. Claude Code keeps hooks in settings.json. Codex keeps them in
// hooks.json, under CODEX_HOME (default ~/.codex) for the user and in .codex/ for a repository.
func settingsPath(harness, scope, dir string) (string, error) {
	folder, file := ".claude", "settings.json"
	switch harness {
	case "claude":
	case "codex":
		folder, file = ".codex", "hooks.json"
	default:
		return "", fmt.Errorf("--harness must be claude or codex, not %q", harness)
	}
	switch scope {
	case "user":
		if harness == "codex" {
			if h := os.Getenv("CODEX_HOME"); h != "" {
				return filepath.Join(h, file), nil
			}
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, folder, file), nil
	case "repo":
		root, err := repoRoot(dir)
		if err != nil {
			return "", err
		}
		return filepath.Join(root, folder, file), nil
	}
	return "", fmt.Errorf("--scope must be user or repo, not %q", scope)
}

func repoRoot(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for d := abs; ; d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, ".git")); err == nil {
			return d, nil
		}
		if filepath.Dir(d) == d {
			return "", fmt.Errorf("%s is not inside a git repository; use --scope user", abs)
		}
	}
}

// readSettings parses settings.json. A missing file is an empty object. A file that is not a JSON object
// is an error, so that we never overwrite something we could not read.
func readSettings(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return map[string]any{}, nil
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("%s is not valid JSON, so it is left alone: %v", path, err)
	}
	return m, nil
}

func writeSettings(path string, m map[string]any) error {
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	// Keep a copy of the file as it was, once. A rewrite of settings.json is never silent.
	if old, err := os.ReadFile(path); err == nil {
		if _, err := os.Stat(path + ".pr-brief.bak"); errors.Is(err, fs.ErrNotExist) {
			if err := os.WriteFile(path+".pr-brief.bak", old, 0o600); err != nil {
				return err
			}
		}
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(out, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// isOurs reports whether a PreToolUse entry runs `gate --hook`.
func isOurs(entry any) bool {
	e, ok := entry.(map[string]any)
	if !ok {
		return false
	}
	hs, _ := e["hooks"].([]any)
	for _, h := range hs {
		if hm, ok := h.(map[string]any); ok {
			if c, _ := hm["command"].(string); strings.Contains(c, "pr-brief") && strings.Contains(c, hookMarker) {
				return true
			}
		}
	}
	return false
}

// preToolUse returns settings.hooks.PreToolUse, creating the path when create is set.
func preToolUse(m map[string]any, create bool) (hooks map[string]any, list []any, err error) {
	raw, has := m["hooks"]
	if !has {
		if !create {
			return nil, nil, nil
		}
		hooks = map[string]any{}
		m["hooks"] = hooks
	} else if hooks, _ = raw.(map[string]any); hooks == nil {
		return nil, nil, errors.New(`"hooks" in settings.json is not an object, so it is left alone`)
	}
	if l, has := hooks["PreToolUse"]; has {
		if list, _ = l.([]any); list == nil {
			return nil, nil, errors.New(`"hooks.PreToolUse" in settings.json is not a list, so it is left alone`)
		}
	}
	return hooks, list, nil
}

func installHook(m map[string]any, command string) (changed string, err error) {
	hooks, list, err := preToolUse(m, true)
	if err != nil {
		return "", err
	}
	entry := hookEntry(command)
	for i, e := range list {
		if isOurs(e) {
			if fmt.Sprint(e) == fmt.Sprint(entry) {
				return "unchanged", nil
			}
			list[i] = entry
			hooks["PreToolUse"] = list
			return "updated", nil
		}
	}
	hooks["PreToolUse"] = append(list, entry)
	return "added", nil
}

// ourCommand returns the command of our PreToolUse entry, if settings has one.
func ourCommand(m map[string]any) (string, bool) {
	_, list, err := preToolUse(m, false)
	if err != nil {
		return "", false
	}
	for _, e := range list {
		if isOurs(e) {
			hs, _ := e.(map[string]any)["hooks"].([]any)
			for _, h := range hs {
				if hm, ok := h.(map[string]any); ok {
					if c, _ := hm["command"].(string); strings.Contains(c, hookMarker) {
						return c, true
					}
				}
			}
		}
	}
	return "", false
}

// hookState says what the hook in settings does after a plugin update. current is the binary that runs now.
//
//	none     settings has no pr-brief hook
//	follows  the hook finds the newest installed version when it runs
//	current  the hook names the binary that runs now
//	stale    the hook names another downloaded version, or a file that is gone
//	custom   the hook names a binary that the launcher did not download: left to its owner
func hookState(m map[string]any, current string) (state, command string) {
	c, ok := ourCommand(m)
	if !ok {
		return "none", ""
	}
	if strings.Contains(c, stableTag) {
		return "follows", c
	}
	named, err := strconv.Unquote(strings.TrimSuffix(c, " "+hookMarker))
	if err != nil {
		return "custom", c
	}
	if named == current {
		return "current", c
	}
	if _, err := os.Stat(named); err != nil {
		return "stale", c
	}
	_, namedManaged := launcherBinary(named)
	_, currentManaged := launcherBinary(current)
	if namedManaged && currentManaged {
		return "stale", c
	}
	return "custom", c
}

// runHookCheck says whether the hook in settings.json is out of date. It exits 1 when it is, and 0 in every
// other case, also when there is no hook.
func runHookCheck(harness, scope, dir, current string, stdout, stderr io.Writer) int {
	scopes := []string{scope}
	if scope == "" {
		scopes = []string{"user", "repo"}
	}
	stale := false
	for _, sc := range scopes {
		path, err := settingsPath(harness, sc, dir)
		if err != nil {
			if scope == "" { // not inside a repository: only the user file counts
				continue
			}
			fmt.Fprintln(stderr, "pr-brief hook:", err)
			return 1
		}
		m, err := readSettings(path)
		if err != nil {
			fmt.Fprintln(stderr, "pr-brief hook:", err)
			return 1
		}
		state, _ := hookState(m, current)
		switch state {
		case "none":
			if scope != "" {
				fmt.Fprintf(stdout, "no pr-brief hook in %s\n", path)
			}
		case "follows":
			fmt.Fprintf(stdout, "ok: the pr-brief hook in %s follows the newest installed version\n", path)
		case "current":
			fmt.Fprintf(stdout, "ok: the pr-brief hook in %s runs this binary\n", path)
		case "custom":
			fmt.Fprintf(stdout, "the pr-brief hook in %s runs a binary you chose, so it is not checked\n", path)
		case "stale":
			stale = true
			fmt.Fprintf(stdout, "out of date: the pr-brief hook in %s runs an older version. Run: pr-brief hook install --scope %s\n", path, sc)
		}
	}
	if stale {
		return 1
	}
	return 0
}

func removeHook(m map[string]any) (removed int, err error) {
	hooks, list, err := preToolUse(m, false)
	if err != nil || hooks == nil {
		return 0, err
	}
	var keep []any
	for _, e := range list {
		if isOurs(e) {
			removed++
		} else {
			keep = append(keep, e)
		}
	}
	switch {
	case removed == 0:
	case len(keep) > 0:
		hooks["PreToolUse"] = keep
	default:
		delete(hooks, "PreToolUse")
		if len(hooks) == 0 {
			delete(m, "hooks")
		}
	}
	return removed, nil
}

// askedMarker is the file that records the offer. It lives in the same cache folder as the launcher's
// downloads, and the user can delete it to be asked again. It is state, not a setting.
func askedMarker() (string, error) {
	dir := os.Getenv("PR_BRIEF_CACHE")
	if dir == "" {
		base := os.Getenv("XDG_CACHE_HOME")
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			base = filepath.Join(home, ".cache")
		}
		dir = filepath.Join(base, "pr-brief")
	}
	return filepath.Join(dir, "hook-asked"), nil
}

func markAsked() error {
	p, err := askedMarker()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte("the pr-brief hook was offered once\n"), 0o644)
}

// hasHook reports whether the file at path holds our entry. A missing or unreadable file has none.
func hasHook(path string) bool {
	m, err := readSettings(path)
	if err != nil {
		return false
	}
	_, list, err := preToolUse(m, false)
	if err != nil {
		return false
	}
	for _, e := range list {
		if isOurs(e) {
			return true
		}
	}
	return false
}

func hookStatus(harness, dir string, stdout io.Writer) int {
	state := func(scope string) string {
		p, err := settingsPath(harness, scope, dir)
		if err != nil {
			return "false" // not inside a repository, or an unknown harness
		}
		return fmt.Sprint(hasHook(p))
	}
	asked := false
	if p, err := askedMarker(); err == nil {
		_, statErr := os.Stat(p)
		asked = statErr == nil
	}
	fmt.Fprintf(stdout, "user=%s\nrepo=%s\nasked=%t\n", state("user"), state("repo"), asked)
	return 0
}

func runHook(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		fmt.Fprint(stdout, hookUsage)
		if len(args) == 0 {
			return 2
		}
		return 0
	}
	sub := args[0]
	fs := flag.NewFlagSet("hook "+sub, flag.ContinueOnError)
	fs.SetOutput(stderr)
	harness := fs.String("harness", "claude", "claude or codex")
	scope := fs.String("scope", "", "user or repo")
	dir := fs.String("dir", ".", "a directory inside the repository")
	bin := fs.String("bin", "", "the pr-brief binary the hook runs (default: this binary)")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	fail := func(err error) int {
		fmt.Fprintln(stderr, "pr-brief hook:", err)
		return 1
	}
	if sub != "show" && sub != "install" && sub != "remove" && sub != "check" && sub != "status" && sub != "asked" {
		fmt.Fprintf(stderr, "pr-brief hook: unknown command %q\n\n%s", sub, hookUsage)
		return 2
	}
	switch sub {
	case "status":
		return hookStatus(*harness, *dir, stdout)
	case "asked":
		if err := markAsked(); err != nil {
			return fail(err)
		}
		return 0
	}
	if *scope == "" && sub != "check" { // check, with no scope, looks in both files

		if sub == "show" {
			*scope = "user"
		} else {
			fmt.Fprintf(stderr, "pr-brief hook %s: choose --scope user or --scope repo\n", sub)
			return 2
		}
	}
	binary := *bin
	if binary == "" {
		exe, err := os.Executable()
		if err != nil {
			return fail(err)
		}
		if r, err := filepath.EvalSymlinks(exe); err == nil {
			exe = r
		}
		binary = exe
	}
	if sub == "check" {
		return runHookCheck(*harness, *scope, *dir, binary, stdout, stderr)
	}
	path, err := settingsPath(*harness, *scope, *dir)
	if err != nil {
		return fail(err)
	}
	// Only Claude Code has the plugin folders that the stable command searches.
	command, stable := hookCommand(binary, *bin != "" || *harness != "claude", runtime.GOOS)
	switch sub {
	case "show":
		out, _ := json.MarshalIndent(map[string]any{"hooks": map[string]any{"PreToolUse": []any{hookEntry(command)}}}, "", "  ")
		fmt.Fprintf(stdout, "%s\n\nfile: %s\n", out, path)
	case "install":
		m, err := readSettings(path)
		if err != nil {
			return fail(err)
		}
		changed, err := installHook(m, command)
		if err != nil {
			return fail(err)
		}
		if changed != "unchanged" {
			if err := writeSettings(path, m); err != nil {
				return fail(err)
			}
		}
		if stable {
			fmt.Fprintf(stdout, "%s the pr-brief hook in %s\nit runs the newest installed pr-brief, so a plugin update needs no change\n", changed, path)
		} else {
			fmt.Fprintf(stdout, "%s the pr-brief hook in %s\nit runs: %s\n", changed, path, binary)
		}
		if *harness == "codex" {
			fmt.Fprintln(stdout, "Codex runs a hook only after you trust it: start Codex, open /hooks and trust the pr-brief entry. Restart the session. Codex hooks are a guardrail, not a boundary: they do not see hosted tools. This path names one version: after a plugin update, run `pr-brief hook check --harness codex`, and this command again when it says the hook is out of date.")
		} else if stable {
			fmt.Fprintln(stdout, "Claude Code reads settings.json when a session starts: restart the session.")
		} else {
			fmt.Fprintln(stdout, "Claude Code reads settings.json when a session starts: restart the session. This path names one version: after a plugin update, run `pr-brief hook check`, and this command again when it says the hook is out of date.")
		}
	case "remove":
		m, err := readSettings(path)
		if err != nil {
			return fail(err)
		}
		n, err := removeHook(m)
		if err != nil {
			return fail(err)
		}
		if n > 0 {
			if err := writeSettings(path, m); err != nil {
				return fail(err)
			}
		}
		fmt.Fprintf(stdout, "removed %d pr-brief hook(s) from %s\n", n, path)
	}
	return 0
}
