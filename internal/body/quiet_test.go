package body

import (
	"strings"
	"testing"
	"time"
)

const quietManaged = "<!-- pr-brief:begin v1 style=ste+iceberg theme=github-dark -->\n## Brief\nText.\n<!-- pr-brief:end -->\n"

func TestWrapQuietAndFind(t *testing.T) {
	out, err := WrapQuiet(quietManaged, "Admins can invite people by email.", "pr-brief, Claude Opus 5.5 <medium>")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "<summary>pr-brief, Claude Opus 5.5 &lt;medium&gt;</summary>\n\n<!-- pr-brief:begin") {
		t.Errorf("summary is escaped and followed by a blank line:\n%s", out)
	}
	q, ok := FindQuiet(out)
	if !ok || q.Lead != "Admins can invite people by email." || q.Summary != "pr-brief, Claude Opus 5.5 <medium>" || !q.Spaced || q.Close < 0 {
		t.Errorf("FindQuiet = %+v, %v", q, ok)
	}
	if m := FindMarkers(out); !m.Found {
		t.Error("the wrapper must not hide the markers")
	}
}

func TestWrapQuietRefusals(t *testing.T) {
	for name, lead := range map[string]string{
		"empty":       "  ",
		"two parts":   "One.\n\nTwo.",
		"tickets row": "**Tickets:** ENG-1",
	} {
		if _, err := WrapQuiet(quietManaged, lead, ""); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	out, err := WrapQuiet(quietManaged, "Lead.", "  ")
	if err != nil || !strings.Contains(out, "<summary>pr-brief</summary>") {
		t.Errorf("an empty summary reads pr-brief: %v\n%s", err, out)
	}
}

func TestFindQuietIgnoresTicketsAndFold(t *testing.T) {
	wrapped, _ := WrapQuiet(quietManaged, "Lead.", "pr-brief")
	saved := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	full := Assemble(wrapped, "old text", true, "fold", saved, "/tmp/b.md", 65536)
	full = "**Tickets:** ENG-45\n\n" + full
	q, ok := FindQuiet(full)
	if !ok || q.Lead != "Lead." {
		t.Fatalf("FindQuiet = %+v, %v", q, ok)
	}
	f, hasFold := FindFold(full)
	if !hasFold || f.Start < q.Close {
		t.Errorf("the fold must sit after the wrapper: fold at %d, wrapper closes at %d", f.Start, q.Close)
	}
	if strings.Count(full, "<details>\n<summary>pr-brief") != 1 {
		t.Errorf("one wrapper only:\n%s", full)
	}
}

func TestFindQuietNotQuiet(t *testing.T) {
	if _, ok := FindQuiet(quietManaged); ok {
		t.Error("a plain description has no wrapper")
	}
	other := "<details>\n<summary>Notes</summary>\n\n" + quietManaged + "</details>\n"
	if q, ok := FindQuiet(other); !ok || q.Close < 0 {
		t.Errorf("any details/summary opening the block counts as the wrapper: %+v %v", q, ok)
	}
}

func TestFindQuietUnclosedAndUnspaced(t *testing.T) {
	bad := "Lead.\n\n<details>\n<summary>pr-brief</summary>\n" + quietManaged
	q, ok := FindQuiet(bad)
	if !ok || q.Close != -1 || q.Spaced {
		t.Errorf("FindQuiet = %+v, %v", q, ok)
	}
}
