package links

import (
	"strings"
	"testing"

	"github.com/gagoar/pr-brief/internal/host"
)

func TestParseRemote(t *testing.T) {
	gh := Repo{Host: host.GitHub, Owner: "gagoar", Name: "pr-brief"}
	ado := Repo{Host: host.AzureDevOps, Owner: "planetdds", Project: "Planet.Denticon.Messaging", Name: "Apps"}
	cases := map[string]Repo{
		"https://github.com/gagoar/pr-brief.git":                                                   gh,
		"git@github.com:gagoar/pr-brief.git":                                                       gh,
		"ssh://git@github.com/gagoar/pr-brief":                                                     gh,
		"https://dev.azure.com/planetdds/Planet.Denticon.Messaging/_git/Apps":                      ado,
		"https://planetdds@dev.azure.com/planetdds/Planet.Denticon.Messaging/_git/Apps":            ado,
		"git@ssh.dev.azure.com:v3/planetdds/Planet.Denticon.Messaging/Apps":                        ado,
		"https://planetdds.visualstudio.com/Planet.Denticon.Messaging/_git/Apps":                   ado,
		"https://planetdds.visualstudio.com/DefaultCollection/Planet.Denticon.Messaging/_git/Apps": ado,
		"planetdds@vs-ssh.visualstudio.com:v3/planetdds/Planet.Denticon.Messaging/Apps":            ado,
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
	for _, u := range []string{
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
