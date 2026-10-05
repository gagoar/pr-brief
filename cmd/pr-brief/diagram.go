package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/gagoar/pr-brief/internal/config"
	"github.com/gagoar/pr-brief/internal/diagram"
	"github.com/gagoar/pr-brief/internal/theme"
)

// resolveTheme returns the theme to draw with: the --theme flag if given, else the config.
func resolveTheme(flagValue, cwd string) (theme.Theme, string, error) {
	if flagValue != "" {
		th, err := config.ThemeFromValue(flagValue, cwd)
		return th, "flag", err
	}
	r, err := config.Resolve(config.Options{RepoPath: config.RepoPath(cwd), UserPath: config.UserPath()})
	if err != nil {
		return theme.Theme{}, "", err
	}
	th, err := r.LoadTheme()
	return th, r.ThemeSource, err
}

func runDiagram(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("diagram", flag.ContinueOnError)
	fs.SetOutput(stderr)
	flowFile := fs.String("flow", "", "a flow JSON file, or - for stdin (see schema/pr-brief-flow.schema.json)")
	report := fs.String("report", "", "a `pr-brief shape` report; use with --flow-number")
	number := fs.Int("flow-number", 1, "which flow of the report to draw (1-based)")
	themeFlag := fs.String("theme", "", "a built-in theme name or a theme JSON file (default: from the config)")
	raw := fs.Bool("raw", false, "print the Mermaid source without the ```mermaid fence")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	var flow diagram.Flow
	switch {
	case *report != "":
		data, err := os.ReadFile(*report)
		if err == nil {
			flow, err = diagram.FromReport(data, *number)
		}
		if err != nil {
			fmt.Fprintln(stderr, "pr-brief diagram:", err)
			return 1
		}
	case *flowFile != "":
		text, err := readInput(*flowFile, stdin)
		if err == nil {
			flow, err = diagram.Parse([]byte(text))
		}
		if err != nil {
			fmt.Fprintln(stderr, "pr-brief diagram:", err)
			return 1
		}
	default:
		fmt.Fprintln(stderr, "pr-brief diagram: give --flow FILE or --report FILE")
		return 2
	}
	cwd, _ := os.Getwd()
	th, source, err := resolveTheme(*themeFlag, cwd)
	if err != nil {
		fmt.Fprintln(stderr, "pr-brief diagram:", err)
		return 1
	}
	src, err := diagram.Render(flow, th)
	if err != nil {
		fmt.Fprintln(stderr, "pr-brief diagram:", err)
		return 1
	}
	fmt.Fprintf(stderr, "theme: %s (%s)\n", th.ID(), source)
	if *raw {
		fmt.Fprintln(stdout, src)
	} else {
		fmt.Fprintf(stdout, "```mermaid\n%s\n```\n", src)
	}
	return 0
}
