// Package initfiles holds the files that `pr-brief init` writes into a repository.
package initfiles

import (
	"embed"
	"strings"
)

//go:embed templates/*
var fs embed.FS

func read(name string) string {
	b, err := fs.ReadFile("templates/" + name)
	if err != nil {
		panic(err) // the templates are embedded; a missing one is a build error
	}
	return string(b)
}

// PRTemplate is .github/pull_request_template.md: the description skeleton, so a
// contributor without the plugin sees what the gate expects.
func PRTemplate(style, themeID string) string {
	return strings.NewReplacer("{{STYLE}}", style, "{{THEME}}", themeID).Replace(read("pull_request_template.md"))
}

// Workflow is .github/workflows/pr-brief.yml. It checks out the base commit, so a PR
// cannot edit the rules that check it.
func Workflow() string { return read("workflow.yml") }

// PRTemplatePath and WorkflowPath are where the files go, relative to the repo root.
const (
	PRTemplatePath = ".github/pull_request_template.md"
	WorkflowPath   = ".github/workflows/pr-brief.yml"
)
