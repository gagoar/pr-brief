package body

import (
	"strings"
	"testing"
	"time"
)

const quietManaged = "<!-- pr-brief:begin v1 style=ste+iceberg theme=github-dark -->\n" +
	"## Brief\nText.\n\n## Change map\n### Flow 1: Input 1 -> x\n```mermaid\n## not a heading\n```\n\n" +
	"## Review guide\n**Review order**: Start here.\n<!-- pr-brief:end -->\n"

func TestWrapQuietFoldsEverythingAfterTheBrief(t *testing.T) {
	out, err := WrapQuiet(quietManaged, "Change map and review guide · pr-brief, Claude <x>")
	if err != nil {
		t.Fatal(err)
	}
	want := "<!-- pr-brief:begin v1 style=ste+iceberg theme=github-dark -->\n## Brief\nText.\n\n" +
		"<details>\n<summary>Change map and review guide · pr-brief, Claude &lt;x&gt;</summary>\n\n## Change map\n"
	if !strings.HasPrefix(out, want) {
		t.Errorf("the Brief stays outside the fold:\n%s", out)
	}
	if !strings.HasSuffix(out, "**Review order**: Start here.\n\n</details>\n<!-- pr-brief:end -->\n") {
		t.Errorf("the fold closes just before the end marker:\n%s", out)
	}
	q, ok := FindQuiet(out)
	if !ok || !q.Spaced || !q.Closed || q.Summary != "Change map and review guide · pr-brief, Claude <x>" {
		t.Errorf("FindQuiet = %+v, %v", q, ok)
	}
	if m := FindMarkers(out); !m.Found {
		t.Error("the fold must not hide the markers")
	}
}

func TestWrapQuietDefaultsAndRefusals(t *testing.T) {
	out, err := WrapQuiet(quietManaged, "  ")
	if err != nil || !strings.Contains(out, "<summary>"+DefaultQuietSummary+"</summary>") {
		t.Errorf("an empty summary reads %q: %v", DefaultQuietSummary, err)
	}
	if _, err := WrapQuiet(out, "again"); err == nil {
		t.Error("a quiet block must not be wrapped twice")
	}
	if _, err := WrapQuiet("no markers", ""); err == nil {
		t.Error("a block with no markers must fail")
	}
	if _, err := WrapQuiet("<!-- pr-brief:begin v1 style=ste -->\n## Brief\nx.\n<!-- pr-brief:end -->\n", ""); err == nil {
		t.Error("a block with no Change map must fail")
	}
}

func TestFindQuietIgnoresPlainAndOtherFolds(t *testing.T) {
	if _, ok := FindQuiet(quietManaged); ok {
		t.Error("a plain description has no fold")
	}
	// The Other-files block sits in the Review guide: it is not the quiet fold.
	other := strings.Replace(quietManaged, "**Review order**: Start here.\n", "**Review order**: Start here.\n<details><summary>Other changed files (1)</summary>\n\n- `a`\n\n</details>\n", 1)
	if _, ok := FindQuiet(other); ok {
		t.Error("the Other-files block is not a quiet fold")
	}
}

func TestFindQuietUnclosedAndUnspaced(t *testing.T) {
	bad := strings.Replace(quietManaged, "## Change map", "<details>\n<summary>S</summary>\n## Change map", 1)
	q, ok := FindQuiet(bad)
	if !ok || q.Spaced || q.Closed {
		t.Errorf("FindQuiet = %+v, %v", q, ok)
	}
}

func TestQuietBodySurvivesTwoImproveRuns(t *testing.T) {
	wrapped, _ := WrapQuiet(quietManaged, "")
	saved := time.Date(2026, 11, 11, 9, 0, 0, 0, time.UTC)
	first := Assemble(wrapped, "old human text", true, "fold", saved, "/tmp/b.md", 65536)
	second := Assemble(wrapped, mustPast(t, first), true, "fold", saved, "/tmp/b.md", 65536)
	for _, s := range []string{first, second} {
		if strings.Count(s, "<summary>"+DefaultQuietSummary+"</summary>") != 1 {
			t.Errorf("one fold only:\n%s", s)
		}
		if f, ok := FindFold(s); !ok || f.Start < strings.Index(s, "<!-- pr-brief:end -->") {
			t.Errorf("the earlier text folds after the end marker:\n%s", s)
		}
	}
}

func mustPast(t *testing.T, s string) string {
	t.Helper()
	p, ok := FindPast(s)
	if !ok {
		t.Fatal("no past text")
	}
	return p
}
