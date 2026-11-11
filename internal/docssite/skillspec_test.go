package docssite

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// The skill follows the open Agent Skills spec (agentskills.io/specification), so any harness that
// reads that spec can load it. This test checks the rules of the spec on the SKILL.md in this repo.
// It is not a full YAML parser: the front matter here is flat, with one nested map (metadata).

const skillDir = "../../plugin/skills/pr-brief"

// Fields the spec defines, plus the Claude Code extensions we keep on purpose.
var allowedKeys = map[string]bool{
	"name": true, "description": true, "license": true, "compatibility": true,
	"metadata": true, "allowed-tools": true,
	"argument-hint": true, // Claude Code extension
}

type skillFile struct {
	top      map[string]string // top-level key -> value (folded text joined with spaces)
	metadata map[string]string // nested under metadata:
	body     string
}

func parseSkill(t *testing.T) skillFile {
	t.Helper()
	text := read(t, filepath.Join(skillDir, "SKILL.md"))
	parts := strings.SplitN(text, "\n---\n", 2)
	if !strings.HasPrefix(parts[0], "---") || len(parts) != 2 {
		t.Fatal("SKILL.md must start with a --- front matter block")
	}
	sf := skillFile{top: map[string]string{}, metadata: map[string]string{}, body: parts[1]}
	key := regexp.MustCompile(`^([a-z][a-z-]*):\s*(.*)$`)
	nested := regexp.MustCompile(`^  ([a-z][a-z-]*):\s*(.*)$`)
	cur := ""
	for _, line := range strings.Split(strings.TrimPrefix(parts[0], "---\n"), "\n") {
		if m := key.FindStringSubmatch(line); m != nil {
			cur = m[1]
			v := strings.Trim(strings.TrimSpace(m[2]), `"`)
			if v == ">" || v == "|" {
				v = ""
			}
			sf.top[cur] = v
			continue
		}
		if cur == "metadata" {
			if m := nested.FindStringSubmatch(line); m != nil {
				sf.metadata[m[1]] = strings.Trim(strings.TrimSpace(m[2]), `"`)
			}
			continue
		}
		if cur != "" && strings.TrimSpace(line) != "" {
			sf.top[cur] = strings.TrimSpace(sf.top[cur] + " " + strings.TrimSpace(line))
		}
	}
	return sf
}

func TestSkillFrontMatterFollowsTheAgentSkillsSpec(t *testing.T) {
	sf := parseSkill(t)

	for k := range sf.top {
		if !allowedKeys[k] {
			t.Errorf("front matter key %q is not in the spec or the Claude allowlist; a strict harness may reject it", k)
		}
	}

	name := sf.top["name"]
	if len(name) < 1 || len(name) > 64 || !regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`).MatchString(name) {
		t.Errorf("name %q must be 1-64 chars of a-z, 0-9 and single hyphens, not starting or ending with one", name)
	}
	if dir := filepath.Base(filepath.Clean(skillDir)); name != dir {
		t.Errorf("name %q must equal the skill directory %q", name, dir)
	}

	desc := strings.Join(strings.Fields(sf.top["description"]), " ")
	if n := len([]rune(desc)); n < 1 || n > 1024 {
		t.Errorf("description is %d characters; the spec allows 1-1024", n)
	}
	if n := len([]rune(sf.top["compatibility"])); n > 500 {
		t.Errorf("compatibility is %d characters; the spec allows at most 500", n)
	}
	if lines := strings.Count(sf.body, "\n"); lines > 500 {
		t.Errorf("SKILL.md body is %d lines; the spec advises under 500. Move detail to references/", lines)
	}
}

// metadata.version is what the launcher reads when the skill runs without the plugin around it.
func TestSkillVersionMatchesThePlugin(t *testing.T) {
	sf := parseSkill(t)
	if got, want := sf.metadata["version"], version(t); got != want {
		t.Errorf("SKILL.md metadata.version is %q, plugin.json says %q; bump both together", got, want)
	}
}

// The skill must work copied alone (npx skills add, ~/.agents/skills): the launcher is inside it.
func TestSkillShipsItsOwnLauncher(t *testing.T) {
	launcher := filepath.Join(skillDir, "scripts", "pr-brief")
	st, err := os.Stat(launcher)
	if err != nil {
		t.Fatalf("the skill needs its launcher: %v", err)
	}
	if st.Mode()&0o111 == 0 {
		t.Errorf("%s is not executable", launcher)
	}
	if b := read(t, launcher); strings.Contains(b, ".claude-plugin") {
		t.Error("the launcher must not read anything outside the skill directory (found .claude-plugin)")
	}
	shim := read(t, "../../plugin/bin/pr-brief")
	if !strings.Contains(shim, "skills/pr-brief/scripts/pr-brief") {
		t.Error("plugin/bin/pr-brief must be a shim to the launcher in the skill")
	}
}

// The launcher finds its version in SKILL.md. Run the real script, with no binary beside it and a
// cache that already holds a stand-in for the release binary, so no network is used. The stand-in
// runs only if the launcher parsed the version and built the right cache path.
func TestLauncherRunsFromACopyOfTheSkillAlone(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the launcher is a POSIX shell script")
	}
	osName := map[string]string{"darwin": "darwin", "linux": "linux"}[runtime.GOOS]
	arch := map[string]string{"arm64": "arm64", "amd64": "amd64"}[runtime.GOARCH]
	if osName == "" || arch == "" {
		t.Skipf("no release binary name for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	want := parseSkill(t).metadata["version"]
	original := read(t, filepath.Join(skillDir, "SKILL.md"))
	line := regexp.MustCompile(`(?m)^  version: .*$`)
	for name, replacement := range map[string]string{
		"quoted":   `  version: "` + want + `"`,
		"unquoted": `  version: ` + want,
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			skill := filepath.Join(dir, "pr-brief") // a copy of the skill alone: no plugin, no repository
			if err := os.MkdirAll(filepath.Join(skill, "scripts"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte(line.ReplaceAllString(original, replacement)), 0o644); err != nil {
				t.Fatal(err)
			}
			script := read(t, filepath.Join(skillDir, "scripts", "pr-brief"))
			if err := os.WriteFile(filepath.Join(skill, "scripts", "pr-brief"), []byte(script), 0o755); err != nil {
				t.Fatal(err)
			}
			cache := filepath.Join(dir, "cache", "v"+want)
			if err := os.MkdirAll(cache, 0o755); err != nil {
				t.Fatal(err)
			}
			stand := "#!/bin/sh\necho STAND-IN \"$@\"\n"
			if err := os.WriteFile(filepath.Join(cache, "pr-brief-"+osName+"-"+arch), []byte(stand), 0o755); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("sh", filepath.Join(skill, "scripts", "pr-brief"), "version")
			cmd.Env = append(os.Environ(), "PR_BRIEF_CACHE="+filepath.Join(dir, "cache"), "PR_BRIEF_BIN=")
			out, err := cmd.CombinedOutput()
			if err != nil || strings.TrimSpace(string(out)) != "STAND-IN version" {
				t.Errorf("the launcher did not reach the cached binary (err %v): %s", err, out)
			}
		})
	}
}
