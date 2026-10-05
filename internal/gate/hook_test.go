package gate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func bash(t *testing.T, cwd, cmd string) HookInput {
	t.Helper()
	return HookInput{ToolName: "Bash", Cwd: cwd, ToolInput: map[string]any{"command": cmd}}
}

func opts(string) Options { return Options{} }

func TestHookAllowsUnrelatedCommands(t *testing.T) {
	for _, cmd := range []string{
		"ls -la",
		"git push origin main",
		"gh pr view 12",
		"gh pr list --state open",
		"gh pr edit 12 --add-label bug",
		"gh pr edit 12 --title 'New title'",
		"az repos pr update --id 3 --auto-complete true",
		`git commit -m "docs: explain how to run gh pr create"`,
		"echo 'gh pr create --body x'",
		"cat > notes.md <<'EOF'\nrun gh pr create --fill\nEOF",
	} {
		if r := Evaluate(bash(t, t.TempDir(), cmd), opts); r != "" {
			t.Errorf("should allow %q, got: %s", cmd, r)
		}
	}
}

func TestHookDeniesBadBodies(t *testing.T) {
	dir := t.TempDir()
	cases := []struct{ name, cmd, want string }{
		{"quoted body", `gh pr create --title "x" --body "just text"`, "no pr-brief description"},
		{"single quoted", `gh pr create --title x --body 'just text'`, "no pr-brief description"},
		{"equals form", `gh pr create --title x --body="just text"`, "no pr-brief description"},
		{"short flag", `gh pr create -t x -b "just text"`, "no pr-brief description"},
		{"heredoc", "gh pr create --title x --body \"$(cat <<'EOF'\n## Summary\nplain body\nEOF\n)\"", "no pr-brief description"},
		{"edit with body", `gh pr edit 4 --body "text"`, "no pr-brief description"},
		{"az create", `az repos pr create --title x --description "text"`, "no pr-brief description"},
		{"az update", `az repos pr update --id 3 --description "text"`, "no pr-brief description"},
		{"chained", `cd repo && gh pr create --title x --body "text" && echo done`, "no pr-brief description"},
		{"no body", `gh pr create --title x`, "no description given"},
		{"fill", `gh pr create --fill`, "--fill / --web"},
		{"web", `gh pr create --web`, "--fill / --web"},
		{"variable", `gh pr create --title x --body "$BODY"`, "cannot evaluate"},
		{"unknown subst", `gh pr create --title x --body "$(make-body)"`, "cannot evaluate"},
		{"stdin", `gh pr create --title x --body-file -`, "stdin"},
		{"missing file", `gh pr create --title x --body-file nope.md`, "cannot read --body-file"},
		{"variable file", `gh pr create --title x --body-file "$F"`, "contains a variable"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := Evaluate(bash(t, dir, c.cmd), opts)
			if r == "" {
				t.Fatalf("expected a denial for %q", c.cmd)
			}
			if !strings.Contains(r, c.want) {
				t.Errorf("reason %q does not contain %q", r, c.want)
			}
		})
	}
}

func TestHookAllowsGoodBodyInEveryForm(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "body.md"), []byte(good), 0o644); err != nil {
		t.Fatal(err)
	}
	quoted := strings.ReplaceAll(good, `"`, `\"`)
	quoted = strings.ReplaceAll(quoted, "$", `\$`)
	quoted = strings.ReplaceAll(quoted, "`", "\\`")
	cmds := map[string]string{
		"body-file":      "gh pr create --title x --body-file body.md",
		"body-file eq":   "gh pr create --title x --body-file=body.md",
		"short -F":       "gh pr edit 3 -F body.md",
		"absolute path":  "gh pr create --body-file " + filepath.Join(dir, "body.md"),
		"cat file":       `gh pr create --title x --body "$(cat body.md)"`,
		"heredoc":        "gh pr create --title x --body \"$(cat <<'EOF'\n" + good + "EOF\n)\"",
		"heredoc dash":   "gh pr create --title x --body \"$(cat <<-'EOF'\n" + good + "EOF\n)\"",
		"double quoted":  `gh pr create --title x --body "` + quoted + `"`,
		"single quoted":  "gh pr create --title x --body '" + strings.ReplaceAll(good, "'", `'\''`) + "'",
		"after env":      "GH_TOKEN=abc gh pr create --body-file body.md",
		"in chain":       "cd . && gh pr create --body-file body.md | tee out.txt",
		"az description": "az repos pr create --title x --description \"$(cat <<'EOF'\n" + good + "EOF\n)\"",
	}
	for name, cmd := range cmds {
		t.Run(name, func(t *testing.T) {
			if r := Evaluate(bash(t, dir, cmd), opts); r != "" {
				t.Errorf("good body was denied: %s", r)
			}
		})
	}
}

func TestHookMCP(t *testing.T) {
	mk := func(tool string, input map[string]any) HookInput {
		return HookInput{ToolName: tool, ToolInput: input}
	}
	if r := Evaluate(mk("mcp__github__create_pull_request", map[string]any{"title": "x", "body": "plain"}), opts); r == "" {
		t.Error("plain MCP body should be denied")
	}
	if r := Evaluate(mk("mcp__github__create_pull_request", map[string]any{"title": "x"}), opts); !strings.Contains(r, "no body") {
		t.Errorf("MCP create without body: %q", r)
	}
	if r := Evaluate(mk("mcp__github__create_pull_request", map[string]any{"body": good}), opts); r != "" {
		t.Errorf("good MCP body denied: %s", r)
	}
	if r := Evaluate(mk("mcp__github__update_pull_request", map[string]any{"title": "new title", "pullNumber": 3}), opts); r != "" {
		t.Errorf("an update that leaves the body alone must pass: %s", r)
	}
	if r := Evaluate(mk("mcp__github__update_pull_request", map[string]any{"body": "plain"}), opts); r == "" {
		t.Error("update with a plain body should be denied")
	}
	if r := Evaluate(mk("mcp__azure-devops__repo_create_pull_request", map[string]any{"description": "plain"}), opts); r == "" {
		t.Error("an azure-devops description should be gated too")
	}
	if r := Evaluate(mk("mcp__github__get_file_contents", map[string]any{"body": "x"}), opts); r != "" {
		t.Errorf("unrelated MCP tool denied: %s", r)
	}
	if r := Evaluate(mk("Read", map[string]any{"file_path": "/x"}), opts); r != "" {
		t.Errorf("unrelated tool denied: %s", r)
	}
}

func TestHookSkipMarkerPasses(t *testing.T) {
	cmd := "gh pr create --title x --body \"$(cat <<'EOF'\n> pr-brief skipped: release PR, auto-generated\nEOF\n)\""
	if r := Evaluate(bash(t, t.TempDir(), cmd), opts); r != "" {
		t.Errorf("skip marker should pass: %s", r)
	}
}

func TestDenyJSONShape(t *testing.T) {
	var out map[string]map[string]string
	if err := json.Unmarshal(DenyJSON("because"), &out); err != nil {
		t.Fatal(err)
	}
	h := out["hookSpecificOutput"]
	if h["hookEventName"] != "PreToolUse" || h["permissionDecision"] != "deny" || h["permissionDecisionReason"] != "because" {
		t.Errorf("unexpected JSON: %v", out)
	}
}

func TestParseWordsStopsAtSeparators(t *testing.T) {
	w := parseWords(` --title "a b" --body x && echo hi`, newFileReader(""))
	var got []string
	for _, x := range w {
		got = append(got, x.text)
	}
	if strings.Join(got, "|") != "--title|a b|--body|x" {
		t.Errorf("words = %q", got)
	}
}
