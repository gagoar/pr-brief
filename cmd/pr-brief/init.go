package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/gagoar/pr-brief/internal/config"
	"github.com/gagoar/pr-brief/internal/convention"
	"github.com/gagoar/pr-brief/internal/initfiles"
)

// writeNew writes a file only if it does not exist. It reports what it did.
func writeNew(root, rel, content string, stdout io.Writer) error {
	p := filepath.Join(root, rel)
	if _, err := os.Stat(p); err == nil {
		fmt.Fprintf(stdout, "kept   %s (already exists, not changed)\n", rel)
		return nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "wrote  %s\n", rel)
	return nil
}

func runInit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "a directory inside the repository")
	prTemplate := fs.Bool("pr-template", false, "also write .github/pull_request_template.md")
	workflow := fs.Bool("workflow", false, "also write .github/workflows/pr-brief.yml")
	style := fs.String("style", "", "prose style for the repo: ste+iceberg, ste or iceberg")
	themeFlag := fs.String("theme", "", "diagram theme for the repo: a built-in name or a path inside the repo")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfgPath := config.RepoPath(*dir)
	root := filepath.Dir(cfgPath)

	fail := func(err error) int {
		fmt.Fprintln(stderr, "pr-brief init:", err)
		return 1
	}

	created := false
	if _, err := os.Stat(cfgPath); errors.Is(err, os.ErrNotExist) {
		if err := config.Init(cfgPath); err != nil {
			return fail(err)
		}
		created = true
		fmt.Fprintf(stdout, "wrote  %s\n", filepath.Base(cfgPath))
		if *style != "" {
			if err := config.Set(cfgPath, "style", *style); err != nil {
				return fail(err)
			}
		}
		if *themeFlag != "" {
			if err := config.Set(cfgPath, "diagram.theme", *themeFlag); err != nil {
				return fail(err)
			}
		}
	} else {
		fmt.Fprintf(stdout, "kept   %s (already exists, not changed)\n", filepath.Base(cfgPath))
		if *style != "" || *themeFlag != "" {
			fmt.Fprintln(stderr, "pr-brief init: --style and --theme apply only to a new file; use `pr-brief config set` to change an existing one")
		}
	}

	// The files that follow use the repo's resolved style and theme.
	r, err := config.Resolve(config.Options{RepoPath: cfgPath})
	if err != nil {
		return fail(err)
	}
	th, err := r.LoadTheme()
	if err != nil {
		return fail(err)
	}
	if *prTemplate {
		if err := writeNew(root, initfiles.PRTemplatePath, initfiles.PRTemplate(r.Style, th.ID()), stdout); err != nil {
			return fail(err)
		}
	}
	if *workflow {
		if err := writeNew(root, initfiles.WorkflowPath, initfiles.Workflow(), stdout); err != nil {
			return fail(err)
		}
	}

	fmt.Fprintf(stdout, "\nstyle %s · theme %s · earlier description %s\n", r.Style, th.ID(), r.Previous)
	fmt.Fprintln(stdout, "\nNext:")
	if created {
		fmt.Fprintln(stdout, "  1. Review .pr-brief.json and commit it. It applies to everyone who works in this repo,")
		fmt.Fprintln(stdout, "     and it wins over personal settings. CI reads it from the base branch.")
	} else {
		fmt.Fprintln(stdout, "  1. .pr-brief.json already applies to everyone who works in this repo.")
	}
	if !*workflow {
		fmt.Fprintln(stdout, "  2. Add the CI check: pr-brief init --workflow")
	}
	if !*prTemplate {
		fmt.Fprintln(stdout, "  3. Show contributors the structure: pr-brief init --pr-template")
	}
	fmt.Fprintf(stdout, "  Limits and rules are fixed on purpose: %d diagrams per PR, %d nodes each. See the convention page.\n", convention.MaxDiagrams, convention.MaxNodes)
	return 0
}
