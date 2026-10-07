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
func (r Repo) DiffURL(pr int, path string) string {
	if r.Host == host.AzureDevOps {
		return fmt.Sprintf("%s/pullrequest/%d?_a=files&path=%s", r.adoBase(), pr, url.QueryEscape("/"+path))
	}
	return fmt.Sprintf("https://github.com/%s/%s/pull/%d/files#%s", r.Owner, r.Name, pr, DiffAnchor(path))
}

// BranchURL links to one file on a branch, so it shows the branch's latest commit. Use it
// before the PR exists, then replace it with DiffURL.
func (r Repo) BranchURL(branch, path string) string {
	if r.Host == host.AzureDevOps {
		return fmt.Sprintf("%s?path=%s&version=GB%s", r.adoBase(), url.QueryEscape("/"+path), url.QueryEscape(branch))
	}
	return fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s", r.Owner, r.Name, escapePath(branch), escapePath(path))
}

// Cell is a table cell: the path in code font, as a link.
func Cell(path, target string) string { return "[`" + path + "`](" + target + ")" }

var cellRe = regexp.MustCompile("^\\[`([^`]+)`\\]\\(([^)\\s]+)\\)$")

// Parse reads a first cell. It accepts `path` and [`path`](url). linked is true for the second form.
func Parse(cell string) (path, target string, linked bool) {
	c := strings.TrimSpace(cell)
	if m := cellRe.FindStringSubmatch(c); m != nil {
		return m[1], m[2], true
	}
	if len(c) > 2 && c[0] == '`' && c[len(c)-1] == '`' && !strings.Contains(c[1:len(c)-1], "`") {
		return c[1 : len(c)-1], "", false
	}
	return "", "", false
}

var commitRe = regexp.MustCompile(`^[0-9a-f]{7,40}$`)

// Valid reports whether target is a link that follows the PR and points at path: the diff of
// the path in a PR, or the path on a branch. It checks the file, not the host, so the gate
// needs no network and no repository. A link pinned to a commit is not valid.
func Valid(path, target string) bool {
	u, err := url.Parse(target)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return false
	}
	if u.Fragment == DiffAnchor(path) {
		return strings.Contains(u.Path, "/pull/") // GitHub: the diff in a PR
	}
	if p := u.Query().Get("path"); p != "" {
		if p != "/"+path {
			return false
		}
		if strings.Contains(u.Path, "/pullrequest/") {
			return true // Azure DevOps: the diff in a PR
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
		path, _, _ := Parse(first)
		if path == "" {
			unreadable = append(unreadable, strings.TrimSpace(first))
			continue
		}
		lines[i] = "| " + Cell(path, link(path)) + " |" + rest
		n++
	}
	return strings.Join(lines, "\n"), n, unreadable
}
