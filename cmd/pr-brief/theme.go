package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/gagoar/pr-brief/internal/config"
	"github.com/gagoar/pr-brief/internal/theme"
)

func runTheme(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: pr-brief theme list | show [NAME|FILE] | validate FILE")
		return 2
	}
	sub, rest := args[0], args[1:]
	fs := flag.NewFlagSet("theme "+sub, flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "print JSON")
	var pos []string
	for len(rest) > 0 {
		if err := fs.Parse(rest); err != nil {
			return 2
		}
		rest = fs.Args()
		if len(rest) > 0 {
			pos = append(pos, rest[0])
			rest = rest[1:]
		}
	}
	cwd, _ := os.Getwd()

	switch sub {
	case "list":
		for _, n := range theme.Names() {
			mark := ""
			if n == theme.DefaultName {
				mark = "  (default)"
			}
			fmt.Fprintf(stdout, "%s%s\n", n, mark)
		}
		return 0
	case "show":
		var th theme.Theme
		var err error
		if len(pos) > 0 {
			th, err = config.ThemeFromValue(pos[0], cwd)
		} else {
			th, _, err = resolveTheme("", cwd)
		}
		if err != nil {
			fmt.Fprintln(stderr, "pr-brief theme:", err)
			return 1
		}
		d := th.Derive()
		if *asJSON {
			data, _ := json.MarshalIndent(map[string]any{"id": th.ID(), "colors": th.Colors, "derived": d}, "", "  ")
			fmt.Fprintln(stdout, string(data))
			return 0
		}
		fmt.Fprintf(stdout, "theme        %s\n", th.ID())
		fmt.Fprintf(stdout, "background   %s   text %s   secondary text %s\n", d.BG, d.FG, d.TextSec)
		fmt.Fprintf(stdout, "node         fill %s   stroke %s\n", d.NodeFill, d.NodeStroke)
		fmt.Fprintf(stdout, "edges        %s\n", d.Line)
		fmt.Fprintf(stdout, "status       added %s   modified %s   removed %s\n", d.Added, d.Modified, d.Removed)
		fmt.Fprintf(stdout, "contrast     text on background %.1f:1   text on node %.1f:1\n", theme.Contrast(d.FG, d.BG), theme.Contrast(d.FG, d.NodeFill))
		return 0
	case "validate":
		if len(pos) != 1 {
			fmt.Fprintln(stderr, "usage: pr-brief theme validate FILE")
			return 2
		}
		th, err := config.ThemeFromValue(pos[0], cwd)
		if err != nil {
			fmt.Fprintln(stdout, "FAIL", err)
			return 1
		}
		fmt.Fprintf(stdout, "ok  %s  (%s)\n", pos[0], th.ID())
		return 0
	}
	fmt.Fprintf(stderr, "pr-brief theme: unknown subcommand %q\n", sub)
	return 2
}
