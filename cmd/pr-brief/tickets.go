package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/gagoar/pr-brief/internal/config"
	"github.com/gagoar/pr-brief/internal/tickets"
)

// branchOrCurrent is the branch to read tickets from: the flag, or the branch checked out in dir.
func branchOrCurrent(flagValue, dir string) string {
	if flagValue != "" {
		return flagValue
	}
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if name := strings.TrimSpace(string(out)); err == nil && name != "HEAD" {
		return name
	}
	return ""
}

// listFlag is a flag that may be given more than once.
type listFlag []string

func (l *listFlag) String() string     { return strings.Join(*l, ", ") }
func (l *listFlag) Set(v string) error { *l = append(*l, v); return nil }

// named reads the tickets the user named with --ticket, as written: "Closes ENG-45" or a link.
func named(f *tickets.Finder, values []string) []tickets.Ticket {
	var out []tickets.Ticket
	for _, v := range values {
		out = append(out, f.Extract(v)...)
	}
	return tickets.Merge(out)
}

// ticketFinder is the Finder for the repo at dir: the team's tickets.pattern, or the built-in guess.
// In CI only the repo file counts, like every other setting.
func ticketFinder(dir string, ci bool) (*tickets.Finder, error) {
	r, err := config.Resolve(config.Options{RepoPath: config.RepoPath(dir), UserPath: config.UserPath(), CI: ci})
	if err != nil {
		return nil, err
	}
	return tickets.NewFinder(r.TicketsPattern)
}

func ticketKeys(ts []tickets.Ticket) string {
	keys := make([]string, len(ts))
	for i, t := range ts {
		keys[i] = t.Key
	}
	return strings.Join(keys, ", ")
}

// runTickets keeps the Jira and Linear tickets visible in a description: it writes a Tickets line
// above the begin marker with every ticket in --current (the description being replaced), in a
// Tickets line the description already has, in each --ticket, and in the branch name. Use it for a new PR. `body improve` does the same
// for an existing one.
func runTickets(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("tickets", flag.ContinueOnError)
	fs.SetOutput(stderr)
	bodyFile := fs.String("body", "-", "the new description (file, or - for stdin)")
	current := fs.String("current", "", "the description being replaced, when there is one (file)")
	branch := fs.String("branch", "", "the PR's branch (default: the current branch)")
	var named_ listFlag
	fs.Var(&named_, "ticket", "a ticket the user named, as written, such as \"Closes ENG-45\" (repeat for more)")
	list := fs.Bool("list", false, "print the tickets found, one per line, and do not change the description")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	text, err := readInput(*bodyFile, stdin)
	if err != nil {
		fmt.Fprintln(stderr, "pr-brief tickets:", err)
		return 2
	}
	var old string
	if *current != "" {
		data, err := os.ReadFile(*current)
		if err != nil {
			fmt.Fprintln(stderr, "pr-brief tickets:", err)
			return 2
		}
		old = string(data)
	}
	cwd, _ := os.Getwd()
	find, err := ticketFinder(cwd, false)
	if err != nil {
		fmt.Fprintln(stderr, "pr-brief tickets:", err)
		return 1
	}
	// The sources are the old description, a Tickets line already in this one, and the branch. The
	// text of this description is not one: it is a summary of the diff.
	ts := tickets.Merge(find.Extract(old), find.FromLine(text), named(find, named_), find.FromBranch(branchOrCurrent(*branch, cwd)))
	if *list {
		for _, t := range ts {
			fmt.Fprintf(stdout, "%s\t%s\n", t.Key, t.Raw)
		}
		return 0
	}
	out := tickets.Ensure(text, ts)
	if lost := tickets.Missing(out, ts); len(lost) > 0 {
		fmt.Fprintf(stderr, "pr-brief tickets: refusing to write: the description would lose %s.\n", ticketKeys(lost))
		return 1
	}
	fmt.Fprint(stdout, out)
	if len(ts) == 0 {
		fmt.Fprintln(stderr, "no tickets found in the description or the branch name")
	} else {
		fmt.Fprintf(stderr, "kept %d ticket%s: %s\n", len(ts), map[bool]string{true: "", false: "s"}[len(ts) == 1], ticketKeys(ts))
	}
	return 0
}
