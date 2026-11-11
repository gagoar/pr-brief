package body

import (
	"errors"
	"html"
	"regexp"
	"strings"

	"github.com/gagoar/pr-brief/internal/convention"
)

// A quiet description keeps the Brief visible and folds the rest. The fold sits inside the
// markers, between the Brief and the Change map, and closes just before the end marker:
//
//	<!-- pr-brief:begin ... -->
//	## Brief
//	<one paragraph>
//
//	<details>
//	<summary>Change map and review guide · pr-brief, <model> <effort></summary>
//
//	## Change map
//	...
//	## Review guide
//	...
//
//	</details>
//	<!-- pr-brief:end -->

// DefaultQuietSummary is the summary text when the caller gives none.
const DefaultQuietSummary = "Change map and review guide"

// Quiet describes the fold found in a managed block.
type Quiet struct {
	Summary string // the summary text, decoded
	Spaced  bool   // a blank line follows </summary>, so the host renders the Markdown inside
	Closed  bool   // </details> is the last line before the end marker
}

var summaryRe = regexp.MustCompile(`^<summary>(.*)</summary>$`)

// WrapQuiet folds everything from the Change map heading to the end marker. managed is the block
// from the begin marker to the end marker. summary is plain text: it is escaped here.
func WrapQuiet(managed, summary string) (string, error) {
	if _, ok := FindQuiet(managed); ok {
		return "", errors.New("the description is quiet already")
	}
	m := FindMarkers(managed)
	if !m.Found {
		return "", errors.New("the block has no begin and end markers")
	}
	lines := strings.Split(strings.TrimRight(managed, "\n"), "\n")
	heading, end := -1, -1
	var fs fenceFlag
	for i, l := range lines {
		fs.next(l)
		switch {
		case fs.in:
		case heading < 0 && l == convention.Sections[1]:
			heading = i
		case strings.TrimSpace(l) == convention.End:
			end = i
		}
	}
	if heading < 0 || end < heading {
		return "", errors.New("the block needs a `" + convention.Sections[1] + "` section before the end marker")
	}
	summary = strings.Join(strings.Fields(summary), " ")
	if summary == "" {
		summary = DefaultQuietSummary
	}
	var out []string
	out = append(out, lines[:heading]...)
	out = append(out, "<details>", "<summary>"+html.EscapeString(summary)+"</summary>", "")
	out = append(out, lines[heading:end]...)
	for len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
		out = out[:len(out)-1]
	}
	out = append(out, "", "</details>")
	out = append(out, lines[end:]...)
	return strings.Join(out, "\n") + "\n", nil
}

// FindQuiet returns the fold in the managed block, if there is one: a `<details>` line and its
// `<summary>` line between the Brief heading and the Change map heading. It reads the body with
// Windows line endings turned into "\n".
func FindQuiet(text string) (Quiet, bool) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	m := FindMarkers(WithoutFold(text))
	if !m.Found {
		return Quiet{}, false
	}
	desc := WithoutFold(text)
	lines := strings.Split(desc[m.BeginIdx:m.EndIdx], "\n")
	var fs fenceFlag
	inBrief := false
	for i, l := range lines {
		fs.next(l)
		if fs.in {
			continue
		}
		switch {
		case l == convention.Sections[0]:
			inBrief = true
		case l == convention.Sections[1]:
			return Quiet{}, false
		case inBrief && strings.TrimSpace(l) == "<details>":
			if i+1 >= len(lines) {
				return Quiet{}, false
			}
			sm := summaryRe.FindStringSubmatch(strings.TrimSpace(lines[i+1]))
			if sm == nil {
				return Quiet{}, false
			}
			q := Quiet{Summary: html.UnescapeString(sm[1])}
			q.Spaced = i+2 < len(lines) && strings.TrimSpace(lines[i+2]) == ""
			q.Closed = closesBeforeEnd(lines)
			return q, true
		}
	}
	return Quiet{}, false
}

// closesBeforeEnd reports whether every <details> opened in the block is closed, and the last
// non-blank line before the end marker is </details>. The count matters: the Other-files block in
// the Review guide closes its own <details>, and that must not pass for the fold's.
func closesBeforeEnd(lines []string) bool {
	var fs fenceFlag
	balance := 0
	for _, l := range lines {
		fs.next(l)
		if fs.in {
			continue
		}
		balance += strings.Count(l, "<details") - strings.Count(l, "</details>")
	}
	if balance != 0 {
		return false
	}
	i := len(lines) - 1
	for i >= 0 && strings.TrimSpace(lines[i]) != convention.End {
		i--
	}
	for i--; i >= 0 && strings.TrimSpace(lines[i]) == ""; i-- {
	}
	return i >= 0 && strings.TrimSpace(lines[i]) == "</details>"
}

// fenceFlag follows code fences, so a heading inside a diagram or a code sample is not read.
type fenceFlag struct {
	in    bool
	token string
}

func (f *fenceFlag) next(l string) {
	t := strings.TrimSpace(l)
	for _, tok := range []string{"```", "~~~"} {
		if strings.HasPrefix(t, tok) {
			if !f.in {
				f.in, f.token = true, tok
			} else if tok == f.token {
				f.in = false
			}
			return
		}
	}
}
