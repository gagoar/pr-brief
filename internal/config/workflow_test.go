package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkflowValuesFillGapsAndTheRepoFileWins(t *testing.T) {
	dir := t.TempDir()
	repo := filepath.Join(dir, ".pr-brief.json")
	wf := File{Style: "iceberg", Diagram: &Diagram{Theme: "dracula"}}

	// No repo file: the workflow values apply, and say where they came from.
	r, err := Resolve(Options{RepoPath: repo, CI: true, Workflow: wf})
	if err != nil {
		t.Fatal(err)
	}
	if r.Style != "iceberg" || r.StyleSource != SourceWorkflow || r.Theme != "dracula" || r.ThemeSource != SourceWorkflow {
		t.Errorf("workflow values must apply without a repo file: %+v", r)
	}

	// A repo file that sets the style wins; the theme still comes from the workflow.
	if err := os.WriteFile(repo, []byte(`{"version":1,"style":"ste"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err = Resolve(Options{RepoPath: repo, CI: true, Workflow: wf})
	if err != nil {
		t.Fatal(err)
	}
	if r.Style != "ste" || r.StyleSource != SourceRepo || r.Theme != "dracula" {
		t.Errorf("the repo file must win over the workflow: %+v", r)
	}
	if strings.Join(r.Shadowed, ",") != "style" {
		t.Errorf("the overridden input must be reported, got %v", r.Shadowed)
	}

	// Outside CI the workflow values are ignored.
	r, _ = Resolve(Options{RepoPath: filepath.Join(dir, "none.json"), Workflow: wf})
	if r.StyleSource != SourceDefault {
		t.Errorf("workflow values apply only in CI: %+v", r)
	}
}

func TestWorkflowValuesAreValidated(t *testing.T) {
	for _, wf := range []File{
		{Style: "loud"},
		{Diagram: &Diagram{Theme: "./theme.json"}},
	} {
		if _, err := Resolve(Options{CI: true, Workflow: wf}); err == nil {
			t.Errorf("%+v must be rejected", wf)
		}
	}
}

func TestTicketsPatternIsTheFourthSetting(t *testing.T) {
	dir := t.TempDir()
	repo, user := filepath.Join(dir, ".pr-brief.json"), filepath.Join(dir, "user.json")

	r, err := Resolve(Options{RepoPath: repo, UserPath: user})
	if err != nil || r.TicketsPattern != "" || r.TicketsPatternSource != SourceDefault {
		t.Fatalf("the default is the built-in guess: %+v, %v", r, err)
	}
	if err := Set(user, "tickets.pattern", `ENG-[0-9]+`); err != nil {
		t.Fatal(err)
	}
	r, _ = Resolve(Options{RepoPath: repo, UserPath: user})
	if r.TicketsPattern != `ENG-[0-9]+` || r.TicketsPatternSource != SourceUser {
		t.Errorf("the user file sets it: %+v", r)
	}
	if err := Set(repo, "tickets.pattern", `(PAY|OPS)-[0-9]+`); err != nil {
		t.Fatal(err)
	}
	r, _ = Resolve(Options{RepoPath: repo, UserPath: user})
	if r.TicketsPattern != `(PAY|OPS)-[0-9]+` || r.TicketsPatternSource != SourceRepo {
		t.Errorf("the repo file wins over the user file: %+v", r)
	}
	// CI reads the repo file only.
	if r, _ = Resolve(Options{RepoPath: filepath.Join(dir, "none.json"), UserPath: user, CI: true}); r.TicketsPattern != "" {
		t.Errorf("CI ignores the user file: %+v", r)
	}
	if v, err := r.Get("tickets.pattern"); err != nil || v != "" {
		t.Errorf("Get: %q, %v", v, err)
	}
	// An empty value goes back to the built-in guess.
	if err := Set(repo, "tickets.pattern", ""); err != nil {
		t.Fatal(err)
	}
	if r, _ = Resolve(Options{RepoPath: repo}); r.TicketsPattern != "" || r.TicketsPatternSource != SourceDefault {
		t.Errorf("an empty pattern clears it: %+v", r)
	}
}

func TestTicketsPatternIsValidated(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".pr-brief.json")
	for _, bad := range []string{"(", "[0-9]*", strings.Repeat("a", 201)} {
		if err := Set(path, "tickets.pattern", bad); err == nil {
			t.Errorf("%q must be rejected", bad)
		}
	}
	if _, err := os.Stat(path); err == nil {
		t.Error("a rejected value must not create the file")
	}
	// A file with a bad pattern is an error naming the key, not a silent fallback.
	os.WriteFile(path, []byte(`{"version":1,"tickets":{"pattern":"("}}`), 0o644)
	if _, err := Resolve(Options{RepoPath: path}); err == nil || !strings.Contains(err.Error(), "tickets.pattern") {
		t.Errorf("a bad pattern in the file must name tickets.pattern: %v", err)
	}
	// Unknown keys under tickets are still errors.
	os.WriteFile(path, []byte(`{"version":1,"tickets":{"patern":"x"}}`), 0o644)
	if _, err := Resolve(Options{RepoPath: path}); err == nil {
		t.Error("an unknown key under tickets must be rejected")
	}
}
