// Package links turns the files in the "Read these first" table into links, so a reviewer
// opens a file from the description instead of hunting for it in the Files changed tab.
//
// A link must follow the PR, never a single commit: new commits land on the branch, and the
// description is not rewritten each time. Once the PR exists, a row links to that file's diff
// in the PR, which always shows the latest commit. Before the PR exists there is no number,
// so a row links to the file on the branch. A link pinned to a commit is not accepted.
package links

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/gagoar/pr-brief/internal/host"
)

// Repo names a repository on one of the two hosts.
type Repo struct {
	Host    string // host.GitHub or host.AzureDevOps
	Owner   string // GitHub owner, or the Azure DevOps organisation
	Project string // Azure DevOps project; empty on GitHub
	Name    string // repository name
}

var (
	githubRe = regexp.MustCompile(`github\.com[:/]+([^/\s]+)/([^/\s]+?)(?:\.git)?/?$`)
	// https://dev.azure.com/org/project/_git/repo, with an optional user@ in front.
	adoHTTPSRe = regexp.MustCompile(`dev\.azure\.com/([^/]+)/([^/]+)/_git/([^/\s]+?)/?$`)
	// git@ssh.dev.azure.com:v3/org/project/repo
	adoSSHRe = regexp.MustCompile(`ssh\.dev\.azure\.com[:/]+v3/([^/]+)/([^/]+)/([^/\s]+?)/?$`)
	// https://org.visualstudio.com/[DefaultCollection/]project/_git/repo
	adoVSRe = regexp.MustCompile(`([^/@.]+)\.visualstudio\.com/(?:DefaultCollection/)?([^/]+)/_git/([^/\s]+?)/?$`)
	// org@vs-ssh.visualstudio.com:v3/org/project/repo
	adoVSSSHRe = regexp.MustCompile(`vs-ssh\.visualstudio\.com[:/]+v3/([^/]+)/([^/]+)/([^/\s]+?)/?$`)
)

// ParseRemote reads a git remote URL. It reports false for a remote on another host.
func ParseRemote(remote string) (Repo, bool) {
	r := strings.TrimSpace(remote)
	for _, re := range []*regexp.Regexp{adoHTTPSRe, adoSSHRe, adoVSSSHRe} {
		if m := re.FindStringSubmatch(r); m != nil {
			return Repo{Host: host.AzureDevOps, Owner: m[1], Project: m[2], Name: m[3]}, true
		}
	}
	if m := adoVSRe.FindStringSubmatch(r); m != nil {
		return Repo{Host: host.AzureDevOps, Owner: m[1], Project: m[2], Name: m[3]}, true
	}
	if m := githubRe.FindStringSubmatch(r); m != nil {
		return Repo{Host: host.GitHub, Owner: m[1], Name: m[2]}, true
	}
	return Repo{}, false
}

// DiffAnchor is the fragment GitHub gives a file's diff on the Files changed tab:
// "diff-" and the SHA-256 of the path.
func DiffAnchor(path string) string {
	sum := sha256.Sum256([]byte(path))
	return "diff-" + hex.EncodeToString(sum[:])
}

func escapePath(path string) string {
	parts := strings.Split(path, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/")
}

func (r Repo) adoBase() string {
	return fmt.Sprintf("https://dev.azure.com/%s/%s/_git/%s", url.PathEscape(r.Owner), url.PathEscape(r.Project), url.PathEscape(r.Name))
}

// DiffURL links to the diff of one file in pull request number pr.
func (r Repo) DiffURL(pr int, path string) string { return r.DiffURLAt(pr, path, 0) }

// DiffURLAt links to the diff of one file in pull request number pr, at a line of the new file.
// A line of 0 or less links to the file. GitHub scrolls to the line and highlights it. Azure DevOps
// has no link to a line of a pull request diff, so it opens the file.
func (r Repo) DiffURLAt(pr int, path string, line int) string {
	if r.Host == host.AzureDevOps {
		return fmt.Sprintf("%s/pullrequest/%d?_a=files&path=%s", r.adoBase(), pr, url.QueryEscape("/"+path))
	}
	anchor := DiffAnchor(path)
	if line > 0 {
		anchor += fmt.Sprintf("R%d", line)
	}
	return fmt.Sprintf("https://github.com/%s/%s/pull/%d/files#%s", r.Owner, r.Name, pr, anchor)
}

// BranchURL links to one file on a branch, so it shows the branch's latest commit. Use it
// before the PR exists, then replace it with DiffURL.
func (r Repo) BranchURL(branch, path string) string { return r.BranchURLAt(branch, path, 0) }

// BranchURLAt links to one file on a branch, at a line. A line of 0 or less links to the file.
func (r Repo) BranchURLAt(branch, path string, line int) string {
	if r.Host == host.AzureDevOps {
		u := fmt.Sprintf("%s?path=%s&version=GB%s", r.adoBase(), url.QueryEscape("/"+path), url.QueryEscape(branch))
		if line > 0 {
			u += fmt.Sprintf("&line=%d&lineEnd=%d&lineStartColumn=1&lineEndColumn=1&_a=contents", line, line+1)
		}
		return u
	}
	u := fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s", r.Owner, r.Name, escapePath(branch), escapePath(path))
	if line > 0 {
		u += fmt.Sprintf("#L%d", line)
	}
	return u
}

// Cell is a table cell: the path in code font, as a link.
func Cell(path, target string) string { return CellAt(path, 0, target) }

// CellAt is a table cell: the path and the line in code font, as a link. A line of 0 or less is left out.
func CellAt(path string, line int, target string) string {
	label := path
	if line > 0 {
		label = fmt.Sprintf("%s:%d", path, line)
	}
	return "[`" + label + "`](" + target + ")"
}

var cellRe = regexp.MustCompile("^\\[`([^`]+)`\\]\\(([^)\\s]+)\\)$")

// Matches `src/Invite/CodeGenerator.cs:42` as the path and 42; rejects `src/Invite/CodeGenerator.cs`, `a.cs:0` and `a.cs:x`.
var lineSuffixRe = regexp.MustCompile(`^(.+):([1-9][0-9]*)$`)

// Parse reads a first cell. It accepts `path`, `path:line` and a link on either. linked is true for a link.
// The line is dropped: use ParseAt to read it.
func Parse(cell string) (path, target string, linked bool) {
	path, _, target, linked = ParseAt(cell)
	return path, target, linked
}

// ParseAt reads a first cell like Parse, and returns the line of `path:line`, or 0 when there is none.
func ParseAt(cell string) (path string, line int, target string, linked bool) {
	c := strings.TrimSpace(cell)
	label := ""
	if m := cellRe.FindStringSubmatch(c); m != nil {
		label, target, linked = m[1], m[2], true
	} else if len(c) > 2 && c[0] == '`' && c[len(c)-1] == '`' && !strings.Contains(c[1:len(c)-1], "`") {
		label = c[1 : len(c)-1]
	} else {
		return "", 0, "", false
	}
	if m := lineSuffixRe.FindStringSubmatch(label); m != nil {
		n, err := strconv.Atoi(m[2])
		if err == nil {
			return m[1], n, target, linked
		}
	}
	return label, 0, target, linked
}

var (
	commitRe = regexp.MustCompile(`^[0-9a-f]{7,40}$`)
	// The whole PR, which shows the latest commit: /owner/repo/pull/N/files. A single commit
	// (/pull/N/commits/<sha>) or a range (/pull/N/files/<a>..<b>) shows an older state.
	githubPRFilesRe = regexp.MustCompile(`^/[^/]+/[^/]+/pull/[0-9]+/files/?$`)
	// /org/project/_git/repo/pullrequest/N, with no iteration or base to pin an older state.
	adoPRRe = regexp.MustCompile(`/pullrequest/[0-9]+/?$`)
)

// anchorRe matches the diff anchor of a path, with an optional line.
// Matches `diff-<hash>` and `diff-<hash>R42`; rejects `diff-<hash>L42` (the old side) and another file's anchor.
func anchorRe(path string) *regexp.Regexp {
	return regexp.MustCompile("^" + regexp.QuoteMeta(DiffAnchor(path)) + "(?:R[1-9][0-9]*)?$")
}

// Valid reports whether target is a link that follows the PR and points at path: the diff of
// the path in a PR, or the path on a branch. It checks the file, not the host, so the gate
// needs no network and no repository. A link pinned to a commit is not valid.
func Valid(path, target string) bool {
	u, err := url.Parse(target)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return false
	}
	if anchorRe(path).MatchString(u.Fragment) {
		return githubPRFilesRe.MatchString(u.EscapedPath()) // GitHub: the diff in the PR, all commits
	}
	if p := u.Query().Get("path"); p != "" {
		if p != "/"+path {
			return false
		}
		if adoPRRe.MatchString(u.EscapedPath()) {
			q := u.Query()
			return q.Get("iteration") == "" && q.Get("base") == "" // the latest iteration, not an older one
		}
		return strings.HasPrefix(u.Query().Get("version"), "GB") // a branch, not GC (a commit)
	}
	if _, rest, ok := strings.Cut(u.EscapedPath(), "/blob/"); ok {
		unescaped, err := url.PathUnescape(rest)
		if err != nil || !strings.HasSuffix(unescaped, "/"+path) {
			return false
		}
		ref := strings.TrimSuffix(unescaped, "/"+path)
		return ref != "" && !commitRe.MatchString(ref)
	}
	return false
}

// RewriteReadFirst rewrites the first cell of each row of the "Read these first" table.
// link returns the URL for a path. It returns the new body, the number of rows linked, and
// the first cells it could not read (they are left as they were). A body with no such
// table comes back unchanged.
func RewriteReadFirst(body string, link func(path string) string) (out string, n int, unreadable []string) {
	return RewriteReadFirstAt(body, func(path string, _ int) string { return link(path) })
}

// RewriteReadFirstAt is RewriteReadFirst for a link that points at a line. A first cell written as
// `path:line` keeps its line, and a cell with no line links to the file.
func RewriteReadFirstAt(body string, link func(path string, line int) string) (out string, n int, unreadable []string) {
	lines := strings.Split(body, "\n")
	i := 0
	for i < len(lines) && !strings.Contains(lines[i], "**Read these first**") {
		i++
	}
	for i++; i < len(lines) && strings.TrimSpace(lines[i]) == ""; i++ {
	}
	// The header row and the separator row come first.
	i += 2
	for ; i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), "|"); i++ {
		row := strings.TrimSpace(lines[i])
		first, rest, ok := strings.Cut(row[1:], "|")
		if !ok {
			continue
		}
		path, line, _, _ := ParseAt(first)
		if path == "" {
			unreadable = append(unreadable, strings.TrimSpace(first))
			continue
		}
		lines[i] = "| " + CellAt(path, line, link(path, line)) + " |" + rest
		n++
	}
	return strings.Join(lines, "\n"), n, unreadable
}
