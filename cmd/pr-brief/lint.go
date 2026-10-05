package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/gagoar/pr-brief/internal/stelint"
)

// steHard returns the hard STE100 violations in prose, formatted for the gate.
func steHard(prose string) []string {
	var out []string
	for _, v := range stelint.LintProse(prose).Violations {
		if v.Hard() {
			out = append(out, fmt.Sprintf("STE100 %s at line %d: %q: %s", v.Rule, v.Line, v.Match, v.Message))
		}
	}
	return out
}

func runLint(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("lint", flag.ContinueOnError)
	fs.SetOutput(stderr)
	file := fs.String("file", "-", "file to lint, or - for stdin")
	asJSON := fs.Bool("json", false, "print the report as JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	text, err := readInput(*file, stdin)
	if err != nil {
		fmt.Fprintln(stderr, "pr-brief lint:", err)
		return 2
	}
	rep := stelint.LintProse(text)
	if *asJSON {
		data, err := rep.JSON()
		if err != nil {
			fmt.Fprintln(stderr, "pr-brief lint:", err)
			return 2
		}
		fmt.Fprintln(stdout, string(data))
	} else {
		for _, v := range rep.Violations {
			fmt.Fprintf(stdout, "%d:%d %s [%s] %q: %s\n", v.Line, v.Col, v.Level, v.Rule, v.Match, v.Message)
		}
		fmt.Fprintf(stdout, "%d hard, %d total\n", rep.HardCount, rep.Count)
	}
	if rep.Failed() {
		return 1
	}
	return 0
}
