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
