package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/gagoar/pr-brief/internal/config"
	"github.com/gagoar/pr-brief/internal/convention"
	"github.com/gagoar/pr-brief/internal/diagram"
	"github.com/gagoar/pr-brief/internal/host"
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

// resolveFormat returns the format to draw in: the --format flag if given, else the
// diagram.format setting; auto follows the host (--host, else the git remote).
func resolveFormat(flagValue, hostFlag, cwd string) (format, source string, err error) {
	setting, source := flagValue, "flag"
	if setting == "" {
		r, err := config.Resolve(config.Options{RepoPath: config.RepoPath(cwd), UserPath: config.UserPath()})
		if err != nil {
			return "", "", err
		}
		setting, source = r.Format, r.FormatSource
	}
	if !slices.Contains(convention.Formats, setting) {
		return "", "", fmt.Errorf("--format %q must be one of %s", setting, strings.Join(convention.Formats, ", "))
	}
	h, ok := host.Normalize(hostFlag)
	if !ok {
		return "", "", fmt.Errorf("unknown host %q; use github or azure-devops", hostFlag)
	}
	if h == "" {
		h = host.FromGit(cwd)
	}
	if setting == convention.FormatAuto && h != "" {
		source += ", auto on " + host.Name(h)
	}
	return host.ResolveFormat(setting, h), source, nil
}

func runDiagram(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("diagram", flag.ContinueOnError)
	fs.SetOutput(stderr)
	flowFile := fs.String("flow", "", "a flow JSON file, or - for stdin (see schema/pr-brief-flow.schema.json)")
	report := fs.String("report", "", "a `pr-brief shape` report; use with --flow-number")
	number := fs.Int("flow-number", 1, "which flow of the report to draw (1-based)")
	themeFlag := fs.String("theme", "", "a built-in theme name or a theme JSON file (default: from the config)")
	formatFlag := fs.String("format", "", "mermaid, ascii or auto (default: the diagram.format setting; auto follows the host)")
	hostFlag := fs.String("host", "", "github or azure-devops; decides what auto draws (default: from the git remote)")
	raw := fs.Bool("raw", false, "print the diagram source without the code fence")
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
	format, formatSource, err := resolveFormat(*formatFlag, *hostFlag, cwd)
	if err != nil {
		fmt.Fprintln(stderr, "pr-brief diagram:", err)
		return 1
	}
	if format == convention.FormatASCII {
		if err := flow.Validate(); err != nil {
			fmt.Fprintln(stderr, "pr-brief diagram:", err)
			return 1
		}
		fmt.Fprintf(stderr, "format: ascii (%s)\n", formatSource)
		// The model draws the boxes. This prints what it needs: the spec, then a skeleton to
		// fill in. The comment and the Legend line stay as printed, and the placeholder line
		// is replaced by the drawing. The gate fails until it is.
		spec := diagram.Spec(flow)
		if *raw {
			fmt.Fprint(stdout, spec)
			return 0
		}
		fmt.Fprint(stderr, "\n"+spec+"\n")
		fmt.Fprintf(stdout, "%s\n```%s\n%s\n```\n", diagram.FlowComment(flow), convention.ASCIIFence, diagram.Placeholder)
		if l := diagram.FlowASCIILegend(flow); l != "" {
			fmt.Fprintf(stdout, "%s\n\n", l)
		}
		return 0
	}
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
	fmt.Fprintf(stderr, "format: mermaid (%s)\ntheme: %s (%s)\n", formatSource, th.ID(), source)
	if *raw {
		fmt.Fprintln(stdout, src)
	} else {
		fmt.Fprintf(stdout, "```mermaid\n%s\n```\n", src)
		// The legend line goes directly under the chart. The gate requires it there.
		if l := diagram.FlowLegend(flow); l != "" {
			fmt.Fprintf(stdout, "%s\n\n", l)
		}
	}
	return 0
}
