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
	"strings"
)

// hookMatcher selects the tool calls the gate reads: every Bash call, and the MCP tools that create or
// update a pull request or a merge request.
const hookMatcher = "Bash|mcp__.*(create|update)_(pull|merge)_request"

// hookMarker is the part of the command that tells our entry from any other in settings.json.
const hookMarker = "gate --hook"

const hookUsage = `usage: pr-brief hook <show|install|remove> [flags]

The plugin declares no hook. This command lets you opt in yourself. It adds a PreToolUse hook to your
own Claude Code settings.json, or to your Codex hooks.json, so the harness runs the gate before a PR
command (gh pr create|edit, az repos pr, glab mr, the MCP PR tools) and blocks a description that
fails it. Claude Code and Codex send the same payload, so one entry serves both.

  show      print the hook entry, and the file it would go in
  install   add the hook to the file (--scope user|repo), or update its path
  remove    take the hook out again; nothing else in the file changes

flags:
  --harness claude|codex  claude (default) writes settings.json; codex writes hooks.json
  --scope user|repo       user is ~/.claude/settings.json or ~/.codex/hooks.json ($CODEX_HOME is honored);
                          repo is .claude/settings.json or .codex/hooks.json in this repository
  --dir DIR               a directory inside the repository (default: .)
  --bin PATH              the pr-brief binary the hook runs (default: this binary)
`

func hookEntry(bin string) map[string]any {
	return map[string]any{
		"matcher": hookMatcher,
		"hooks": []any{map[string]any{
			"type":    "command",
			"command": fmt.Sprintf("%q %s", bin, hookMarker),
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

func installHook(m map[string]any, bin string) (changed string, err error) {
	hooks, list, err := preToolUse(m, true)
	if err != nil {
		return "", err
	}
	entry := hookEntry(bin)
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
	if sub != "show" && sub != "install" && sub != "remove" {
		fmt.Fprintf(stderr, "pr-brief hook: unknown command %q\n\n%s", sub, hookUsage)
		return 2
	}
	if *scope == "" {
		if sub == "show" {
			*scope = "user"
		} else {
			fmt.Fprintf(stderr, "pr-brief hook %s: choose --scope user or --scope repo\n", sub)
			return 2
		}
	}
	path, err := settingsPath(*harness, *scope, *dir)
	if err != nil {
		return fail(err)
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
	switch sub {
	case "show":
		out, _ := json.MarshalIndent(map[string]any{"hooks": map[string]any{"PreToolUse": []any{hookEntry(binary)}}}, "", "  ")
		fmt.Fprintf(stdout, "%s\n\nfile: %s\n", out, path)
	case "install":
		m, err := readSettings(path)
		if err != nil {
			return fail(err)
		}
		changed, err := installHook(m, binary)
		if err != nil {
			return fail(err)
		}
		if changed != "unchanged" {
			if err := writeSettings(path, m); err != nil {
				return fail(err)
			}
		}
		fmt.Fprintf(stdout, "%s the pr-brief hook in %s\nit runs: %s\n", changed, path, binary)
		if *harness == "codex" {
			fmt.Fprintln(stdout, "Codex runs a hook only after you trust it: start Codex, open /hooks and trust the pr-brief entry. Restart the session. Codex hooks are a guardrail, not a boundary: they do not see hosted tools. After a plugin update the binary path changes: run this command again.")
		} else {
			fmt.Fprintln(stdout, "Claude Code reads settings.json when a session starts: restart the session. After a plugin update the binary path changes: run this command again.")
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
