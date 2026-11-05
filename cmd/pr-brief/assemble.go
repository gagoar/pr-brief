package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// diagramPlaceholder is a line that stands for the output of `pr-brief diagram --flow FILE`, or
// `--report FILE --flow-number N` for `{{diagram FILE N}}`.
// The model writes the line. It does not copy the chart, so it spends no output tokens on it.
var diagramPlaceholder = regexp.MustCompile(`(?m)^[ \t]*\{\{diagram[ \t]+([^}\n]+?)[ \t]*\}\}[ \t]*$`)

// runAssemble finishes a description in one call: it replaces each {{diagram FILE}} line with the
// diagram, links the Read-these-first files, and keeps the tickets. It runs the same code as
// `diagram`, `links` and `tickets`, in that order, so the result is what the three commands give.
// It never draws plain text: the model draws that, so a placeholder with --format ascii is an error.
func runAssemble(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("assemble", flag.ContinueOnError)
	fs.SetOutput(stderr)
	bodyFile := fs.String("body", "-", "the description with {{diagram FILE}} lines (file, or - for stdin)")
	pr := fs.Int("pr", 0, "the PR number: link each file to its diff in the PR (default: link to the file on --branch)")
	branch := fs.String("branch", "", "the PR's branch (default: the current branch)")
	remote := fs.String("remote", "", "the repository URL (default: the origin remote)")
	current := fs.String("current", "", "the description being replaced, when there is one (file)")
	themeFlag := fs.String("theme", "", "a built-in theme name or a theme JSON file (default: from the config)")
	formatFlag := fs.String("format", "", "mermaid, ascii or auto (default: the diagram.format setting)")
	hostFlag := fs.String("host", "", "github, gitlab or azure-devops; decides what auto draws")
	noTickets := fs.Bool("no-tickets", false, "skip the tickets step (for improve: `body improve` keeps the tickets itself)")
	var named_ listFlag
	fs.Var(&named_, "ticket", "a ticket the user named, as written, such as \"Closes ENG-45\" (repeat for more)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	text, err := readInput(*bodyFile, stdin)
	if err != nil {
		fmt.Fprintln(stderr, "pr-brief assemble:", err)
		return 2
	}

	// 1. Diagrams.
	var failed bool
	text = diagramPlaceholder.ReplaceAllStringFunc(text, func(line string) string {
		file := strings.TrimSpace(diagramPlaceholder.FindStringSubmatch(line)[1])
		// "{{diagram FILE}}" is a flow file. "{{diagram REPORT N}}" is flow N of a `shape` report.
		dargs := []string{"--flow", file}
		if f := strings.Fields(file); len(f) == 2 {
			if _, err := strconv.Atoi(f[1]); err == nil {
				file, dargs = f[0], []string{"--report", f[0], "--flow-number", f[1]}
			}
		}
		for flagName, v := range map[string]string{"--theme": *themeFlag, "--format": *formatFlag, "--host": *hostFlag} {
			if v != "" {
				dargs = append(dargs, flagName, v)
			}
		}
		var out, errb bytes.Buffer
		if code := runDiagram(dargs, nil, &out, &errb); code != 0 {
			fmt.Fprintf(stderr, "pr-brief assemble: flow %s: %s", file, errb.String())
			failed = true
			return line
		}
		if strings.Contains(out.String(), "```text") {
			fmt.Fprintf(stderr, "pr-brief assemble: flow %s is drawn in plain text, which you draw by hand. Run `pr-brief diagram --flow %s`, draw it, and paste it. Assemble does not expand it.\n", file, file)
			failed = true
			return line
		}
		return strings.TrimRight(out.String(), "\n") + "\n"
	})
	if failed {
		return 1
	}

	// 2. Links.
	largs := []string{"--body", "-"}
	if *pr > 0 {
		largs = append(largs, "--pr", strconv.Itoa(*pr))
	}
	if *branch != "" {
		largs = append(largs, "--branch", *branch)
	}
	if *remote != "" {
		largs = append(largs, "--remote", *remote)
	}
	var linked, lerr bytes.Buffer
	if code := runLinks(largs, strings.NewReader(text), &linked, &lerr); code != 0 {
		fmt.Fprint(stderr, strings.ReplaceAll(lerr.String(), "pr-brief links:", "pr-brief assemble (links):"))
		return code
	}
	fmt.Fprint(stderr, lerr.String())

	if *noTickets {
		fmt.Fprint(stdout, linked.String())
		return 0
	}

	// 3. Tickets.
	targs := []string{"--body", "-"}
	if *current != "" {
		targs = append(targs, "--current", *current)
	}
	if *branch != "" {
		targs = append(targs, "--branch", *branch)
	}
	for _, t := range named_ {
		targs = append(targs, "--ticket", t)
	}
	var ticketed, terr bytes.Buffer
	if code := runTickets(targs, strings.NewReader(linked.String()), &ticketed, &terr); code != 0 {
		fmt.Fprint(stderr, strings.ReplaceAll(terr.String(), "pr-brief tickets:", "pr-brief assemble (tickets):"))
		return code
	}
	fmt.Fprint(stderr, terr.String())
	fmt.Fprint(stdout, ticketed.String())
	return 0
}
