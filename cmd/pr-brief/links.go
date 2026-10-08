package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/gagoar/pr-brief/internal/host"
	"github.com/gagoar/pr-brief/internal/links"
)

// runLinks makes each file in the "Read these first" table a link. Before the PR exists
// it links to the file on the branch. With --pr it links to the file's diff in that PR, which
// always shows the latest commit. A link never pins a commit.
func runLinks(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("links", flag.ContinueOnError)
	fs.SetOutput(stderr)
	bodyFile := fs.String("body", "-", "the description (file, or - for stdin)")
	pr := fs.Int("pr", 0, "the PR number: link each file to its diff in the PR (default: link to the file on --branch)")
	branch := fs.String("branch", "", "the branch to link to when there is no PR (default: the current branch)")
	remote := fs.String("remote", "", "the repository URL (default: the origin remote)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	text, err := readInput(*bodyFile, stdin)
	if err != nil {
		fmt.Fprintln(stderr, "pr-brief links:", err)
		return 2
	}
	cwd, _ := os.Getwd()
	url := *remote
	if url == "" {
		url = host.Remote(cwd)
	}
	repo, ok := links.ParseRemote(url)
	if !ok {
		fmt.Fprintf(stderr, "pr-brief links: cannot tell the host from the remote %q; it must be on GitHub or Azure DevOps. Pass --remote.\n", url)
		return 1
	}
	link := func(path string) string { return repo.DiffURL(*pr, path) }
	if *pr <= 0 {
		name := *branch
		if name == "" {
			out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
			name = strings.TrimSpace(string(out))
			if err != nil || name == "" || name == "HEAD" {
				fmt.Fprintln(stderr, "pr-brief links: no --pr and no --branch, and git is not on a branch. Pass one of them.")
				return 1
			}
		}
		link = func(path string) string { return repo.BranchURL(name, path) }
	}
	out, n, bad := links.RewriteReadFirst(text, link)
	if len(bad) > 0 {
		fmt.Fprintf(stderr, "pr-brief links: cannot read these cells in the Read-these-first table; write each file as `path/to/file`: %s\n", strings.Join(bad, ", "))
		return 1
	}
	if n == 0 {
		fmt.Fprintln(stderr, "pr-brief links: no Read-these-first table found; nothing linked")
		return 1
	}
	fmt.Fprint(stdout, out)
	target := "the file on the branch"
	if *pr > 0 {
		target = fmt.Sprintf("its diff in PR %d", *pr)
	}
	fmt.Fprintf(stderr, "linked %d file%s to %s\n", n, map[bool]string{true: "", false: "s"}[n == 1], target)
	return 0
}
