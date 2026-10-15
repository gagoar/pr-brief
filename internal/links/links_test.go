package links

import (
	"strconv"
	"strings"
	"testing"

	"github.com/gagoar/pr-brief/internal/host"
)

func TestParseRemote(t *testing.T) {
	gh := Repo{Host: host.GitHub, Owner: "gagoar", Name: "pr-brief"}
	ado := Repo{Host: host.AzureDevOps, Owner: "org", Project: "proj", Name: "repo"}
	cases := map[string]Repo{
		"https://github.com/gagoar/pr-brief.git":                        gh,
		"git@github.com:gagoar/pr-brief.git":                            gh,
		"ssh://git@github.com/gagoar/pr-brief":                          gh,
		"https://dev.azure.com/org/proj/_git/repo":                      ado,
		"https://org@dev.azure.com/org/proj/_git/repo":                  ado,
		"git@ssh.dev.azure.com:v3/org/proj/repo":                        ado,
		"https://org.visualstudio.com/proj/_git/repo":                   ado,
		"https://org.visualstudio.com/DefaultCollection/proj/_git/repo": ado,
		"org@vs-ssh.visualstudio.com:v3/org/proj/repo":                  ado,
	}
	for in, want := range cases {
		if got, ok := ParseRemote(in); !ok || got != want {
			t.Errorf("ParseRemote(%q) = %+v, %v; want %+v", in, got, ok, want)
		}
	}
	if _, ok := ParseRemote("https://gitlab.com/a/b.git"); ok {
		t.Error("another host must not parse")
	}
}

func TestDiffAnchorIsTheSHA256OfThePath(t *testing.T) {
	// The anchor GitHub gives README.md on the Files changed tab.
	if got := DiffAnchor("README.md"); got != "diff-b335630551682c19a781afebcf4d07bf978fb1f8ac04c6bf87428ed5106870f5" {
		t.Errorf("DiffAnchor(README.md) = %s", got)
	}
}

func TestURLsAndTheirValidity(t *testing.T) {
	gh := Repo{Host: host.GitHub, Owner: "o", Name: "r"}
	ado := Repo{Host: host.AzureDevOps, Owner: "org", Project: "My Project", Name: "repo"}
	for _, path := range []string{"cmd/main.go", "docs/a file.md", "internal/x/y_test.go"} {
		for name, u := range map[string]string{
			"github diff":                gh.DiffURL(7, path),
			"github branch":              gh.BranchURL("main", path),
			"github branch with a slash": gh.BranchURL("feature/x", path),
			"ado diff":                   ado.DiffURL(7, path),
			"ado branch":                 ado.BranchURL("feature/x", path),
		} {
			if !Valid(path, u) {
				t.Errorf("%s: %q must be valid for %q", name, u, path)
			}
			if Valid(path+"2", u) || Valid("other/"+path, u) {
				t.Errorf("%s: %q must not be valid for another path", name, u)
			}
		}
	}
	if got := gh.DiffURL(13, "README.md"); got != "https://github.com/o/r/pull/13/files#diff-b335630551682c19a781afebcf4d07bf978fb1f8ac04c6bf87428ed5106870f5" {
		t.Errorf("github diff url: %s", got)
	}
	if got := gh.BranchURL("feature/x", "a b/c.go"); got != "https://github.com/o/r/blob/feature/x/a%20b/c.go" {
		t.Errorf("github branch url: %s", got)
	}
	if got := ado.DiffURL(13, "a b/c.go"); got != "https://dev.azure.com/org/My%20Project/_git/repo/pullrequest/13?_a=files&path=%2Fa+b%2Fc.go" {
		t.Errorf("ado diff url: %s", got)
	}
	if got := ado.BranchURL("feature/x", "a.go"); got != "https://dev.azure.com/org/My%20Project/_git/repo?path=%2Fa.go&version=GBfeature%2Fx" {
		t.Errorf("ado branch url: %s", got)
	}
}

// A link must follow the PR. One pinned to a commit shows the file as it was, not as it is.
func TestLinksPinnedToACommitAreNotValid(t *testing.T) {
	sha := "13d9af75098e7c84242e8b5f7d6bdbd59c426658"
	anchor := "#" + DiffAnchor("cmd/main.go")
	for _, u := range []string{
		// Pasted from the browser while looking at one commit, or a range of commits.
		"https://github.com/o/r/pull/14/commits/" + sha + anchor,
		"https://github.com/o/r/pull/14/files/" + sha + ".." + sha + anchor,
		"https://github.com/o/r/pull/14/files/aaaaaaa..bbbbbbb" + anchor,
		"https://github.com/o/r/commit/" + sha + anchor,
		// An older iteration of an Azure DevOps PR.
		"https://dev.azure.com/org/p/_git/repo/pullrequest/14?_a=files&path=%2Fcmd%2Fmain.go&iteration=2",
		"https://dev.azure.com/org/p/_git/repo/pullrequest/14?_a=files&path=%2Fcmd%2Fmain.go&base=1&iteration=3",
		"https://github.com/o/r/blob/" + sha + "/cmd/main.go",
		"https://github.com/o/r/blob/13d9af7/cmd/main.go",
		"https://dev.azure.com/org/p/_git/repo?path=%2Fcmd%2Fmain.go&version=GC" + sha,
		"https://dev.azure.com/org/p/_git/repo?path=%2Fcmd%2Fmain.go",
	} {
		if Valid("cmd/main.go", u) {
			t.Errorf("%q is pinned to a commit, or to no ref, and must not be valid", u)
		}
	}
	for _, bad := range []string{"", "not a url", "ftp://github.com/o/r/blob/x/a.go", "https://github.com/o/r/blob/x", "https://github.com/o/r/pull/1/files", "https://github.com/o/r/tree/main/a.go#diff-00", "javascript:alert(1)"} {
		if Valid("a.go", bad) {
			t.Errorf("%q must not be valid", bad)
		}
	}
}

func TestParseCell(t *testing.T) {
	for _, tc := range []struct {
		in, path string
		linked   bool
	}{
		{"`a/b.go`", "a/b.go", false},
		{" `a/b.go` ", "a/b.go", false},
		{"[`a/b.go`](https://x/y)", "a/b.go", true},
		{"a/b.go", "", false},
		{"[a/b.go](https://x/y)", "", false},
	} {
		p, _, linked := Parse(tc.in)
		if p != tc.path || linked != tc.linked {
			t.Errorf("Parse(%q) = %q, %v; want %q, %v", tc.in, p, linked, tc.path, tc.linked)
		}
	}
}

func TestRewriteReadFirst(t *testing.T) {
	body := "intro\n**Read these first**\n| File | Why | Check |\n|---|---|---|\n| `a.go` | x | y |\n| [`b.go`](https://old/b.go) | x | y |\n| nonsense | x | y |\n\n**Review order**: a.go\n"
	repo := Repo{Host: host.GitHub, Owner: "o", Name: "r"}
	out, n, bad := RewriteReadFirst(body, func(p string) string { return repo.DiffURL(3, p) })
	if n != 2 || len(bad) != 1 || bad[0] != "nonsense" {
		t.Fatalf("n=%d unreadable=%v\n%s", n, bad, out)
	}
	for _, p := range []string{"a.go", "b.go"} {
		if !strings.Contains(out, Cell(p, repo.DiffURL(3, p))) {
			t.Errorf("row for %s is not linked:\n%s", p, out)
		}
	}
	if strings.Contains(out, "https://old/b.go") {
		t.Error("an old link must be replaced")
	}
	if !strings.HasSuffix(out, "**Review order**: a.go\n") || !strings.Contains(out, "| nonsense | x | y |") {
		t.Errorf("everything else stays as it was:\n%s", out)
	}
	// Running it again changes nothing.
	again, _, _ := RewriteReadFirst(out, func(p string) string { return repo.DiffURL(3, p) })
	if again != out {
		t.Error("rewriting is not idempotent")
	}
	// A body with no table comes back as it was.
	if same, n, _ := RewriteReadFirst("no table here", func(string) string { return "x" }); same != "no table here" || n != 0 {
		t.Errorf("no table: %q %d", same, n)
	}
}

func TestLinksToALine(t *testing.T) {
	gh := Repo{Host: host.GitHub, Owner: "o", Name: "r"}
	ado := Repo{Host: host.AzureDevOps, Owner: "org", Project: "proj", Name: "repo"}
	const path = "src/Invite/CodeGenerator.cs"

	pr := gh.DiffURLAt(7, path, 42)
	if want := "https://github.com/o/r/pull/7/files#" + DiffAnchor(path) + "R42"; pr != want {
		t.Errorf("GitHub PR link = %s, want %s", pr, want)
	}
	if got := gh.DiffURLAt(7, path, 0); got != gh.DiffURL(7, path) || strings.HasSuffix(got, "R0") {
		t.Errorf("no line must link to the file: %s", got)
	}
	if got := gh.BranchURLAt("main", path, 42); !strings.HasSuffix(got, "/blob/main/"+path+"#L42") {
		t.Errorf("GitHub branch link = %s", got)
	}
	if got := ado.BranchURLAt("main", path, 42); !strings.Contains(got, "&line=42&lineEnd=43") {
		t.Errorf("Azure DevOps branch link = %s", got)
	}
	if got := ado.DiffURLAt(7, path, 42); got != ado.DiffURL(7, path) {
		t.Errorf("Azure DevOps has no line link in a PR diff; it must open the file: %s", got)
	}
	for _, target := range []string{pr, gh.DiffURL(7, path), gh.BranchURLAt("main", path, 42), ado.BranchURLAt("main", path, 42), ado.DiffURLAt(7, path, 42)} {
		if !Valid(path, target) {
			t.Errorf("a link to a line must be valid: %s", target)
		}
	}
	// The old side of the diff, another file's anchor and a pinned commit are still refused.
	for _, bad := range []string{
		"https://github.com/o/r/pull/7/files#" + DiffAnchor(path) + "L42",
		"https://github.com/o/r/pull/7/files#" + DiffAnchor("other.go") + "R42",
		"https://github.com/o/r/pull/7/files#" + DiffAnchor(path) + "R0",
		"https://github.com/o/r/pull/7/commits/0123456789abcdef#" + DiffAnchor(path) + "R42",
	} {
		if Valid(path, bad) {
			t.Errorf("must not be valid: %s", bad)
		}
	}
}

func TestParseCellWithALine(t *testing.T) {
	for cell, want := range map[string]struct {
		path   string
		line   int
		linked bool
	}{
		"`src/a.cs`":                        {"src/a.cs", 0, false},
		"`src/a.cs:42`":                     {"src/a.cs", 42, false},
		"[`src/a.cs:42`](https://x.test/a)": {"src/a.cs", 42, true},
		"[`src/a.cs`](https://x.test/a)":    {"src/a.cs", 0, true},
		"`src/a.cs:0`":                      {"src/a.cs:0", 0, false},
		"`src/a.cs:x`":                      {"src/a.cs:x", 0, false},
	} {
		path, line, _, linked := ParseAt(cell)
		if path != want.path || line != want.line || linked != want.linked {
			t.Errorf("ParseAt(%q) = %q, %d, %v; want %q, %d, %v", cell, path, line, linked, want.path, want.line, want.linked)
		}
		if p, _, l := Parse(cell); p != want.path || l != want.linked {
			t.Errorf("Parse(%q) = %q, %v", cell, p, l)
		}
	}
}

func TestRewriteReadFirstKeepsTheLine(t *testing.T) {
	body := "**Read these first**\n| File | Why | Check |\n|---|---|---|\n| `src/a.cs:42` | delicate | check |\n| `src/b.cs` | delicate | check |\n"
	out, n, bad := RewriteReadFirstAt(body, func(path string, line int) string {
		return "https://x.test/" + path + "?line=" + strconv.Itoa(line)
	})
	if n != 2 || len(bad) != 0 {
		t.Fatalf("n=%d bad=%v", n, bad)
	}
	if !strings.Contains(out, "| [`src/a.cs:42`](https://x.test/src/a.cs?line=42) |") || !strings.Contains(out, "| [`src/b.cs`](https://x.test/src/b.cs?line=0) |") {
		t.Errorf("rows wrong:\n%s", out)
	}
	// Running it again on the linked rows changes nothing.
	again, _, _ := RewriteReadFirstAt(out, func(path string, line int) string {
		return "https://x.test/" + path + "?line=" + strconv.Itoa(line)
	})
	if again != out {
		t.Errorf("a second run changed the table:\n%s", again)
	}
}
