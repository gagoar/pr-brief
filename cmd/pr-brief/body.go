package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"
	"unicode/utf8"

	"github.com/gagoar/pr-brief/internal/body"
	"github.com/gagoar/pr-brief/internal/config"
	"github.com/gagoar/pr-brief/internal/host"
	"github.com/gagoar/pr-brief/internal/tickets"
)

func runBody(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: pr-brief body improve|past|restore|uncomment")
		return 2
	}
	sub := args[0]
	fs := flag.NewFlagSet("body "+sub, flag.ContinueOnError)
	fs.SetOutput(stderr)
	current := fs.String("current", "-", "the PR's current description (file, or - for stdin)")
	managed := fs.String("managed", "", "file holding the new managed block")
	hostName := fs.String("host", "github.com", "host of the PR: github.com or dev.azure.com (sets the length limit)")
	owner := fs.String("owner", "", "repo owner or organisation")
	repo := fs.String("repo", "", "repo name")
	pr := fs.String("pr", "", "PR number")
	mode := fs.String("mode", "", "drop or comment (default: from config)")
	at := fs.String("at", "", "backup timestamp prefix for restore")
	var named_ listFlag
	fs.Var(&named_, "ticket", "a ticket the user named, as written, such as \"Closes ENG-45\" (repeat for more)")
	branch := fs.String("branch", "", "the PR's branch, for the Jira and Linear tickets in its name (default: the current branch)")
	asJSON := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	needPR := func() bool {
		if *owner == "" || *repo == "" || *pr == "" {
			fmt.Fprintf(stderr, "pr-brief body %s: --owner, --repo and --pr are required\n", sub)
			return false
		}
		return true
	}

	switch sub {
	case "past":
		text, err := readInput(*current, stdin)
		if err != nil {
			fmt.Fprintln(stderr, "pr-brief body:", err)
			return 2
		}
		past, has := body.FindPast(text)
		if *asJSON {
			data, _ := json.Marshal(map[string]any{"has": has, "text": past})
			fmt.Fprintln(stdout, string(data))
		} else if has {
			fmt.Fprint(stdout, past)
		}
		return 0

	case "improve":
		if !needPR() || *managed == "" {
			if *managed == "" {
				fmt.Fprintln(stderr, "pr-brief body improve: --managed is required")
			}
			return 2
		}
		cur, err := readInput(*current, stdin)
		if err != nil {
			fmt.Fprintln(stderr, "pr-brief body:", err)
			return 2
		}
		block, err := os.ReadFile(*managed)
		if err != nil {
			fmt.Fprintln(stderr, "pr-brief body:", err)
			return 2
		}
		m := *mode
		if m == "" {
			cwd, _ := os.Getwd()
			r, err := config.Resolve(config.Options{RepoPath: config.RepoPath(cwd), UserPath: config.UserPath()})
			if err != nil {
				fmt.Fprintln(stderr, "pr-brief config:", err)
				return 1
			}
			m = r.Previous
		}
		now := time.Now()
		backup := ""
		if cur != "" {
			backup, err = body.Backup(body.StateDir(), *hostName, *owner, *repo, *pr, cur, now)
			if err != nil {
				fmt.Fprintln(stderr, "pr-brief body: cannot write the backup, so nothing was changed:", err)
				return 1
			}
			fmt.Fprintln(stderr, "backup:", backup)
		}
		past, has := body.FindPast(cur)
		h, _ := host.Normalize(*hostName)
		if h == "" {
			h = host.Detect(*hostName)
		}
		// Jira and Linear link a PR by the ticket key in its description. Keep every ticket the old
		// description and the branch name carry. The line takes room, so the earlier text gets less.
		cwd, _ := os.Getwd()
		ts := tickets.Merge(tickets.Extract(cur), tickets.FromLine(string(block)), named(named_), tickets.FromBranch(branchOrCurrent(*branch, cwd)))
		room := host.Limit(h) - utf8.RuneCountInString(tickets.Line(ts)) - 2
		final := tickets.Ensure(body.Assemble(string(block), past, has, m, now, backup, room), ts)
		if lost := tickets.Missing(final, ts); len(lost) > 0 {
			fmt.Fprintf(stderr, "pr-brief body: refusing to write: the new description would lose %s. Nothing was changed.\n", ticketKeys(lost))
			return 1
		}
		if len(ts) > 0 {
			fmt.Fprintf(stderr, "kept %d ticket%s: %s\n", len(ts), map[bool]string{true: "", false: "s"}[len(ts) == 1], ticketKeys(ts))
		}
		fmt.Fprint(stdout, final)
		return 0

	case "restore":
		if !needPR() {
			return 2
		}
		text, err := body.Restore(body.StateDir(), *hostName, *owner, *repo, *pr, *at)
		if err != nil {
			fmt.Fprintln(stderr, "pr-brief body:", err)
			return 1
		}
		fmt.Fprint(stdout, text)
		return 0

	case "uncomment":
		text, err := readInput(*current, stdin)
		if err != nil {
			fmt.Fprintln(stderr, "pr-brief body:", err)
			return 2
		}
		out, ok := body.Uncomment(text)
		if !ok {
			fmt.Fprintln(stderr, "pr-brief body: no pr-brief:previous block found")
			return 1
		}
		fmt.Fprint(stdout, out)
		return 0
	}
	fmt.Fprintf(stderr, "pr-brief body: unknown subcommand %q\n", sub)
	return 2
}
