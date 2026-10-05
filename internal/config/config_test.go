package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestParseRejectsUnknownKeys(t *testing.T) {
	for _, body := range []string{
		`{"version":1,"diagram":{"maxNodes":7}}`,
		`{"version":1,"style":"ste","gate":"warn"}`,
		`{"version":1,"improve":{"previous":"drop","extra":1}}`,
	} {
		if _, err := Parse([]byte(body)); err == nil {
			t.Errorf("expected error for %s", body)
		}
	}
}

func TestParseRejectsBadValues(t *testing.T) {
	for _, body := range []string{
		`{"version":2}`,
		`{"version":1,"style":"hemingway"}`,
		`{"version":1,"improve":{"previous":"keep"}}`,
		`{"version":1} {"version":1}`,
	} {
		if _, err := Parse([]byte(body)); err == nil {
			t.Errorf("expected error for %s", body)
		}
	}
}

func TestResolvePrecedence(t *testing.T) {
	d := t.TempDir()
	user := writeFile(t, d, "user.json", `{"version":1,"style":"iceberg","improve":{"previous":"comment"}}`)
	repo := writeFile(t, d, "repo.json", `{"version":1,"style":"ste"}`)

	r, err := Resolve(Options{RepoPath: repo, UserPath: user})
	if err != nil {
		t.Fatal(err)
	}
	if r.Style != "ste" || r.StyleSource != SourceRepo {
		t.Errorf("style = %s from %s", r.Style, r.StyleSource)
	}
	if r.Previous != "comment" || r.PreviousSource != SourceUser {
		t.Errorf("previous = %s from %s", r.Previous, r.PreviousSource)
	}

	r, _ = Resolve(Options{RepoPath: repo, UserPath: user, FlagStyle: "iceberg"})
	if r.Style != "iceberg" || r.StyleSource != SourceFlag {
		t.Errorf("flag should win: %s from %s", r.Style, r.StyleSource)
	}

	r, _ = Resolve(Options{})
	if r.Style != "ste+iceberg" || r.StyleSource != SourceDefault || r.Previous != "drop" {
		t.Errorf("defaults wrong: %+v", r)
	}
}

func TestResolveCIIgnoresUserAndFlag(t *testing.T) {
	d := t.TempDir()
	user := writeFile(t, d, "user.json", `{"version":1,"style":"iceberg","improve":{"previous":"comment"}}`)
	repo := writeFile(t, d, "repo.json", `{"version":1,"style":"ste"}`)
	r, err := Resolve(Options{RepoPath: repo, UserPath: user, FlagStyle: "iceberg", CI: true})
	if err != nil {
		t.Fatal(err)
	}
	if r.Style != "ste" || r.Previous != "drop" || r.PreviousSource != SourceDefault {
		t.Errorf("CI leaked a personal setting: %+v", r)
	}
}

func TestResolveBadFlag(t *testing.T) {
	if _, err := Resolve(Options{FlagStyle: "nope"}); err == nil {
		t.Error("expected error for bad --style")
	}
}

func TestSetAndGet(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "c.json")
	if err := Set(p, "style", "ste"); err != nil {
		t.Fatal(err)
	}
	if err := Set(p, "improve.previous", "comment"); err != nil {
		t.Fatal(err)
	}
	f, ok, err := Load(p)
	if err != nil || !ok {
		t.Fatalf("load: %v ok=%v", err, ok)
	}
	if f.Style != "ste" || f.Improve.Previous != "comment" || f.Schema != SchemaURL {
		t.Errorf("unexpected file: %+v", f)
	}
	if err := Set(p, "style", "bad"); err == nil {
		t.Error("bad style accepted")
	}
	if err := Set(p, "diagram.maxNodes", "7"); err == nil || !strings.Contains(err.Error(), "unknown key") {
		t.Errorf("unknown key accepted: %v", err)
	}
	// A rejected set must not damage the file.
	f2, _, _ := Load(p)
	if f2.Style != "ste" {
		t.Error("file changed by a rejected set")
	}
}

func TestInitRefusesOverwrite(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.json")
	if err := Init(p); err != nil {
		t.Fatal(err)
	}
	if err := Init(p); err == nil {
		t.Error("second init should fail")
	}
}

const goodTheme = `{"bg":"#10141c","fg":"#e4ecf7","accent":"#ff8a3d"}`

func TestThemeDefaultAndBuiltins(t *testing.T) {
	r, err := Resolve(Options{})
	if err != nil {
		t.Fatal(err)
	}
	if r.Theme != "github-dark" || r.ThemeSource != SourceDefault {
		t.Errorf("default theme = %s from %s", r.Theme, r.ThemeSource)
	}
	th, err := r.LoadTheme()
	if err != nil || th.Name != "github-dark" {
		t.Errorf("load default: %v %+v", err, th.Name)
	}

	d := t.TempDir()
	repo := writeFile(t, d, ".pr-brief.json", `{"version":1,"diagram":{"theme":"dracula"}}`)
	r, _ = Resolve(Options{RepoPath: repo})
	if th, err := r.LoadTheme(); err != nil || th.Name != "dracula" || r.ThemeSource != SourceRepo {
		t.Errorf("repo theme: %v %s from %s", err, th.Name, r.ThemeSource)
	}
}

func TestThemeFileIsRelativeToTheConfigFile(t *testing.T) {
	d := t.TempDir()
	if err := os.MkdirAll(filepath.Join(d, "design"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, d, "design/mine.json", goodTheme)
	repo := writeFile(t, d, ".pr-brief.json", `{"version":1,"diagram":{"theme":"./design/mine.json"}}`)
	r, err := Resolve(Options{RepoPath: repo})
	if err != nil {
		t.Fatal(err)
	}
	th, err := r.LoadTheme()
	if err != nil {
		t.Fatal(err)
	}
	if th.Name != "mine" || th.BG != "#10141c" || !strings.HasPrefix(th.ID(), "custom:") {
		t.Errorf("custom theme: %+v id=%s", th, th.ID())
	}
}

func TestRepoThemeCannotLeaveTheRepo(t *testing.T) {
	outer := t.TempDir()
	repoDir := filepath.Join(outer, "repo")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, outer, "outside.json", goodTheme)
	writeFile(t, repoDir, "inside.json", goodTheme)

	try := func(value string) error {
		cfg := writeFile(t, repoDir, ".pr-brief.json", `{"version":1,"diagram":{"theme":"`+value+`"}}`)
		r, err := Resolve(Options{RepoPath: cfg})
		if err != nil {
			return err
		}
		_, err = r.LoadTheme()
		return err
	}
	if err := try("./inside.json"); err != nil {
		t.Errorf("a file inside the repo must load: %v", err)
	}
	if err := try("../outside.json"); err == nil || !strings.Contains(err.Error(), "outside the repo") {
		t.Errorf(".. must be rejected: %v", err)
	}
	if err := try(filepath.Join(outer, "outside.json")); err == nil || !strings.Contains(err.Error(), "relative") {
		t.Errorf("an absolute path in a repo config must be rejected: %v", err)
	}
	// A symlink inside the repo that points outside is the same escape.
	if err := os.Symlink(filepath.Join(outer, "outside.json"), filepath.Join(repoDir, "link.json")); err != nil {
		t.Fatal(err)
	}
	if err := try("./link.json"); err == nil || !strings.Contains(err.Error(), "outside the repo") {
		t.Errorf("a symlink out of the repo must be rejected: %v", err)
	}
}

func TestUserThemeMayBeAbsolute(t *testing.T) {
	d := t.TempDir()
	abs := writeFile(t, d, "mine.json", goodTheme)
	user := writeFile(t, d, "user.json", `{"version":1,"diagram":{"theme":"`+abs+`"}}`)
	r, err := Resolve(Options{UserPath: user})
	if err != nil {
		t.Fatal(err)
	}
	if th, err := r.LoadTheme(); err != nil || th.BG != "#10141c" {
		t.Errorf("user absolute path: %v %+v", err, th.Colors)
	}
}

func TestBadThemeFilesAreRejectedWithAReason(t *testing.T) {
	d := t.TempDir()
	cases := map[string]string{
		"low contrast":  `{"bg":"#888888","fg":"#777777"}`,
		"unknown key":   `{"bg":"#ffffff","fg":"#000000","font":"x"}`,
		"not hex":       `{"bg":"white","fg":"#000000"}`,
		"not json":      `bg = white`,
		"oversize file": strings.Repeat(" ", 9000) + goodTheme,
	}
	for name, body := range cases {
		cfg := writeFile(t, d, ".pr-brief.json", `{"version":1,"diagram":{"theme":"./bad.json"}}`)
		writeFile(t, d, "bad.json", body)
		r, err := Resolve(Options{RepoPath: cfg})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := r.LoadTheme(); err == nil || !strings.Contains(err.Error(), "bad.json") {
			t.Errorf("%s: want an error that names the file, got %v", name, err)
		}
	}
	cfg := writeFile(t, d, ".pr-brief.json", `{"version":1,"diagram":{"theme":"./missing.json"}}`)
	r, _ := Resolve(Options{RepoPath: cfg})
	if _, err := r.LoadTheme(); err == nil {
		t.Error("a missing theme file must be an error")
	}
}

func TestSetTheme(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, ".pr-brief.json")
	if err := Set(p, "diagram.theme", "alucard"); err != nil {
		t.Fatal(err)
	}
	writeFile(t, d, "mine.json", goodTheme)
	if err := Set(p, "diagram.theme", "./mine.json"); err != nil {
		t.Fatalf("a path to a good file: %v", err)
	}
	before, _ := os.ReadFile(p)
	if err := Set(p, "diagram.theme", "./gone.json"); err == nil {
		t.Error("a path to a missing file must be refused")
	}
	if err := Set(p, "diagram.theme", "solarized"); err == nil || !strings.Contains(err.Error(), "github-dark") {
		t.Errorf("an unknown name should list the built-ins: %v", err)
	}
	if after, _ := os.ReadFile(p); string(after) != string(before) {
		t.Error("a refused set must not change the file")
	}
	r, _ := Resolve(Options{RepoPath: p})
	if v, err := r.Get("diagram.theme"); err != nil || v != "./mine.json" {
		t.Errorf("get: %v %q", err, v)
	}
}

func TestCIIgnoresTheUserTheme(t *testing.T) {
	d := t.TempDir()
	user := writeFile(t, d, "user.json", `{"version":1,"diagram":{"theme":"dracula"}}`)
	r, _ := Resolve(Options{UserPath: user, CI: true})
	if r.Theme != "github-dark" || r.ThemeSource != SourceDefault {
		t.Errorf("CI must not read the user theme: %s from %s", r.Theme, r.ThemeSource)
	}
}

func TestDiagramRejectsUnknownKeys(t *testing.T) {
	if _, err := Parse([]byte(`{"version":1,"diagram":{"colors":{}}}`)); err == nil {
		t.Error("an unknown key under diagram must be rejected")
	}
}
