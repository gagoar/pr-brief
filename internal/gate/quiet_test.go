package gate

import (
	"strings"
	"testing"

	"github.com/gagoar/pr-brief/internal/body"
)

// quietOf folds the good body's managed block the way `pr-brief body wrap` does.
func quietOf(t *testing.T) string {
	t.Helper()
	m := body.FindMarkers(good)
	if !m.Found {
		t.Fatal("good body has no markers")
	}
	wrapped, err := body.WrapQuiet(good[m.BeginIdx:m.EndIdx], "Change map and review guide · pr-brief, Claude")
	if err != nil {
		t.Fatal(err)
	}
	return good[:m.BeginIdx] + wrapped + good[m.EndIdx:]
}

func TestQuietBodyPasses(t *testing.T) {
	if r := Check(quietOf(t), Options{}); !r.OK() {
		t.Fatalf("a quiet body failed: %+v", r.Findings)
	}
}

func TestQuietFoldMustClose(t *testing.T) {
	q := quietOf(t)
	i := strings.LastIndex(q, "</details>") // the fold's, not the Other-files block's
	q = q[:i] + q[i+len("</details>"):]
	has(t, Check(q, Options{}), "quiet", "must close with `</details>`")
}

func TestQuietNeedsBlankLineAfterSummary(t *testing.T) {
	q := strings.Replace(quietOf(t), "</summary>\n\n", "</summary>\n", 1)
	has(t, Check(q, Options{}), "quiet", "blank line after `</summary>`")
}

func TestQuietBriefStillCounts(t *testing.T) {
	q := strings.Replace(quietOf(t), "## Brief\n", "## Brief\nOne. Two. Three. Four. Five. Six.\n", 1)
	has(t, Check(q, Options{}), "brief", "sentences")
}

func TestPlainBodyHasNoQuietFindings(t *testing.T) {
	for _, f := range Check(good, Options{}).Findings {
		if f.Rule == "quiet" {
			t.Errorf("unexpected quiet finding: %+v", f)
		}
	}
}
