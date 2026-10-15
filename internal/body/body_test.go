package body

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/gagoar/pr-brief/internal/convention"
)

const managed = "<!-- pr-brief:begin v1 style=ste+iceberg -->\n## Brief\nText.\n<!-- pr-brief:end -->"

var when = time.Date(2026, 10, 5, 14, 2, 0, 0, time.UTC)

func TestEncodeRoundTrip(t *testing.T) {
	for _, s := range []string{
		"plain",
		"has -- double dash and --> arrow and <!-- open",
		"----- many -----",
		"literal -&#45; and &amp; and &#45; entities",
		"![img](https://x/y.png)\n- [ ] task\n- [x] done\r\nwindows line",
		"ends with dash -",
		"--!> bogus close",
		"",
	} {
		enc := Encode(s)
		if strings.Contains(enc, "--") {
			t.Errorf("encoded text still has --: %q", enc)
		}
		if got := Decode(enc); got != s {
			t.Errorf("round trip failed:\n got %q\nwant %q", got, s)
		}
	}
}

func TestFindPast(t *testing.T) {
	if p, ok := FindPast("Fixes the thing.\n![shot](a.png)"); !ok || !strings.Contains(p, "shot") {
		t.Error("a body with no markers is the past writing")
	}
	if _, ok := FindPast(managed); ok {
		t.Error("a body with markers and no previous block has no past writing")
	}
	if _, ok := FindPast("  \n"); ok {
		t.Error("blank body has no past writing")
	}
	b := Assemble(managed, "original -- text", true, convention.PreviousComment, when, "/tmp/x.md", convention.MaxBodyCharsGitHub)
	p, ok := FindPast(b)
	if !ok || p != "original -- text" {
		t.Errorf("past from previous block = %q, %v", p, ok)
	}
}

func TestAssembleDropAndComment(t *testing.T) {
	d := Assemble(managed, "old", true, convention.PreviousDrop, when, "", convention.MaxBodyCharsGitHub)
	if strings.Contains(d, "old") || strings.Contains(d, "pr-brief:previous") {
		t.Errorf("drop must not keep the past: %q", d)
	}
	c := Assemble(managed, "old", true, convention.PreviousComment, when, "", convention.MaxBodyCharsGitHub)
	if !strings.HasPrefix(c, managed) || !strings.Contains(c, "pr-brief:previous v1 saved=2026-10-05T14:02Z") {
		t.Errorf("comment output wrong: %q", c)
	}
	if n := strings.Count(c, "-->"); n != 3 { // begin marker, end marker, previous end
		t.Errorf("expected 3 comment closers, got %d in %q", n, c)
	}
	if Assemble(managed, "", false, convention.PreviousComment, when, "", convention.MaxBodyCharsGitHub) != managed+"\n" {
		t.Error("comment with no past writing must emit only the managed block")
	}
}

func TestSecondAndThirdRunDoNotNest(t *testing.T) {
	human := "My notes\n- [ ] one\n![a](b.png)"
	run1 := Assemble(managed, human, true, convention.PreviousComment, when, "", convention.MaxBodyCharsGitHub)

	past2, ok := FindPast(run1)
	if !ok || past2 != human {
		t.Fatalf("run 2 past = %q", past2)
	}
	run2 := Assemble(managed, past2, true, convention.PreviousComment, when.Add(time.Hour), "", convention.MaxBodyCharsGitHub)
	past3, _ := FindPast(run2)
	run3 := Assemble(managed, past3, true, convention.PreviousComment, when.Add(2*time.Hour), "", convention.MaxBodyCharsGitHub)

	if strings.Count(run3, convention.PreviousOpen) != 1 {
		t.Errorf("blocks nested: %q", run3)
	}
	if p, _ := FindPast(run3); p != human {
		t.Errorf("original text drifted: %q", p)
	}
	if len(run3) > len(run1)+40 { // only the timestamp may differ
		t.Errorf("block grew: %d -> %d", len(run1), len(run3))
	}
}

func TestTruncation(t *testing.T) {
	past := strings.Repeat("word -- ", 20000)
	got := Assemble(managed, past, true, convention.PreviousComment, when, "/state/1.md", convention.MaxBodyCharsGitHub)
	if len(got) > convention.MaxBodyCharsGitHub {
		t.Errorf("body is %d chars, limit %d", len(got), convention.MaxBodyCharsGitHub)
	}
	p, ok := FindPrevious(got)
	if !ok || !strings.Contains(p.Text, "[truncated, full copy at /state/1.md]") {
		t.Error("truncated block must point at the backup")
	}
}

func TestUncomment(t *testing.T) {
	human := "keep -- this\n![a](b.png)"
	b := Assemble(managed, human, true, convention.PreviousComment, when, "", convention.MaxBodyCharsGitHub)
	out, ok := Uncomment(b)
	if !ok || !strings.Contains(out, "## Previous description") || !strings.Contains(out, human) {
		t.Errorf("uncomment output: %q", out)
	}
	if strings.Contains(out, "pr-brief:previous") {
		t.Error("block should be gone")
	}
	if _, ok := Uncomment(managed); ok {
		t.Error("nothing to uncomment")
	}
}

func TestFindMarkers(t *testing.T) {
	m := FindMarkers("intro\n" + managed + "\ntail")
	if !m.Found || m.Style != "ste+iceberg" || m.Version != "1" {
		t.Errorf("markers: %+v", m)
	}
	if FindMarkers("<!-- pr-brief:begin v1 style=ste -->\nno end").Found {
		t.Error("begin without end is not a block")
	}
	if !HasAnyMarker("<!-- pr-brief:begin v1 style=ste -->\nno end") {
		t.Error("broken pair still counts as a marker")
	}
}

func TestBackupAndRestore(t *testing.T) {
	dir := t.TempDir()
	p1, err := Backup(dir, "github.com", "o", "r", "7", "first", when)
	if err != nil {
		t.Fatal(err)
	}
	p2, err := Backup(dir, "github.com", "o", "r", "7", "second", when) // same second
	if err != nil || p1 == p2 {
		t.Fatalf("same-second backup must not overwrite: %v %s %s", err, p1, p2)
	}
	if got, _ := Restore(dir, "github.com", "o", "r", "7", ""); got != "first" {
		t.Errorf("default restore should be the oldest, got %q", got)
	}
	if got, _ := Restore(dir, "github.com", "o", "r", "7", "20261005T140200Z-01"); got != "second" {
		t.Errorf("restore by prefix got %q", got)
	}
	if _, err := Restore(dir, "github.com", "o", "r", "8", ""); err == nil {
		t.Error("restore of unknown PR should fail")
	}
}

// A host that accepts fewer characters gets a shorter previous block, counted in characters.
func TestAssembleFitsTheHostLimit(t *testing.T) {
	managed := "<!-- pr-brief:begin v1 -->\nbrief\n<!-- pr-brief:end -->"
	past := strings.Repeat("é", 6000) // 6,000 characters, 12,000 bytes
	got := Assemble(managed, past, true, convention.PreviousComment, when, "/state/1.md", convention.MaxBodyCharsAzureDevOps)
	if n := utf8.RuneCountInString(got); n > convention.MaxBodyCharsAzureDevOps {
		t.Errorf("body is %d characters, limit %d", n, convention.MaxBodyCharsAzureDevOps)
	}
	if !strings.Contains(got, "[truncated, full copy at /state/1.md]") {
		t.Errorf("a cut previous block must point at the backup")
	}
}

func TestFoldEncodeRoundTrip(t *testing.T) {
	for _, s := range []string{
		"plain",
		"a <!-- pr-brief:end --> marker and <!-- pr-brief:begin v1 style=ste -->",
		"<details><summary>Logs</summary>\n\nerr\n</details>\n\nAfter logs",
		"literal &lt;!-- pr-brief and &amp;lt;!-- pr-brief",
		"&amp;amp;lt;!-- pr-brief and &lt;!-- other",
		"> pr-brief skipped: old note",
		"![img](https://x/y.png)\n- [ ] task\n```go\nx := 1 -- 2\n```",
		"",
	} {
		enc := EncodeFold(s)
		if strings.Contains(enc, "<!-- pr-brief") {
			t.Errorf("encoded text still has a marker: %q", enc)
		}
		if got := DecodeFold(enc); got != s {
			t.Errorf("round trip %q -> %q -> %q", s, enc, got)
		}
	}
}

func TestAssembleFold(t *testing.T) {
	human := "My notes\n- [ ] one\n![a](b.png)\n\n> pr-brief skipped: ignore me"
	f := Assemble(managed, human, true, convention.PreviousFold, when, "", convention.MaxBodyCharsGitHub)
	if !strings.HasPrefix(f, managed) || !strings.Contains(f, "<details>\n<summary>Previous description</summary>") {
		t.Fatalf("fold output wrong: %q", f)
	}
	if !strings.Contains(f, "My notes\n- [ ] one") {
		t.Errorf("the earlier text must stay readable in the fold: %q", f)
	}
	if past, ok := FindPast(f); !ok || past != human {
		t.Errorf("past from fold = %q, %v", past, ok)
	}
	if got := WithoutFold(f); strings.Contains(got, "ignore me") || strings.Contains(got, "details") {
		t.Errorf("WithoutFold left the fold: %q", got)
	}
	if Assemble(managed, "", false, convention.PreviousFold, when, "", convention.MaxBodyCharsGitHub) != managed+"\n" {
		t.Error("fold with no past writing must emit only the managed block")
	}
}

func TestFoldRunsDoNotNestAndNeverSaveGeneratedText(t *testing.T) {
	human := "My notes"
	run1 := Assemble(managed, human, true, convention.PreviousFold, when, "", convention.MaxBodyCharsGitHub)
	past2, ok := FindPast(run1)
	if !ok || past2 != human {
		t.Fatalf("run 2 past = %q, %v", past2, ok)
	}
	run2 := Assemble(managed, past2, true, convention.PreviousFold, when, "", convention.MaxBodyCharsGitHub)
	if run2 != run1 || strings.Count(run2, "<details>") != 1 {
		t.Errorf("a second run must give the same description:\n%q\n%q", run1, run2)
	}
	if _, ok := FindPast(managed + "\n"); ok {
		t.Error("an old generated brief must never be kept")
	}
}

func TestFoldTruncatesToTheLimit(t *testing.T) {
	long := strings.Repeat("word ", 400)
	limit := utf8.RuneCountInString(managed) + 300
	f := Assemble(managed, long, true, convention.PreviousFold, when, "/tmp/b.md", limit)
	if n := utf8.RuneCountInString(f); n > limit {
		t.Errorf("fold output has %d characters, limit %d", n, limit)
	}
	if !strings.Contains(f, "[truncated, full copy at /tmp/b.md]") {
		t.Errorf("a cut fold must point at the backup: %q", f)
	}
}

func TestFoldKeepsTheAuthorsOwnDetailsBlocks(t *testing.T) {
	human := "Intro\n<details><summary>Logs</summary>\n\nerr\n</details>\n\nAfter logs"
	f := Assemble(managed, human, true, convention.PreviousFold, when, "", convention.MaxBodyCharsGitHub)
	if !strings.Contains(f, "err\n</details>\n\nAfter logs") {
		t.Errorf("the author's closing tag must stay: %q", f)
	}
	if past, ok := FindPast(f); !ok || past != human {
		t.Errorf("past = %q, %v", past, ok)
	}
}

func TestPastSurvivesWindowsLineEndings(t *testing.T) {
	human := "My notes\nsecond line"
	for _, mode := range []string{convention.PreviousFold, convention.PreviousComment} {
		stored := strings.ReplaceAll(Assemble(managed, human, true, mode, when, "/tmp/x.md", convention.MaxBodyCharsGitHub), "\n", "\r\n")
		if past, ok := FindPast(stored); !ok || past != human {
			t.Errorf("%s: after the web editor stored CRLF, past = %q, %v", mode, past, ok)
		}
		if strings.Contains(WithoutFold(stored), "details") && mode == convention.PreviousFold {
			t.Errorf("WithoutFold must find a CRLF fold")
		}
	}
}

func TestNeverPassesTheLimit(t *testing.T) {
	long := strings.Repeat("word ", 400)
	for _, mode := range []string{convention.PreviousFold, convention.PreviousComment} {
		for _, extra := range []int{0, 20, 60, 100, 150, 200, 300, 2000} {
			limit := utf8.RuneCountInString(managed) + 2 + extra
			out := Assemble(managed, long, true, mode, when, "/tmp/a/long/backup/path.md", limit)
			if n := utf8.RuneCountInString(out); n > limit && out != managed+"\n" {
				t.Errorf("%s with room for %d: %d characters, limit %d", mode, extra, n, limit)
			}
		}
	}
	tight := utf8.RuneCountInString(managed) + 5
	if out := Assemble(managed, long, true, convention.PreviousFold, when, "/tmp/b.md", tight); out != managed+"\n" {
		t.Errorf("with no room the fold is left out, got %q", out)
	}
}
