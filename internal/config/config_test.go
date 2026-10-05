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
