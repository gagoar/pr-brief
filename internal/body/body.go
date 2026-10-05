// Package body parses pr-brief markers in a PR description, finds the past
// (human-written) description, and builds the new description with the past
// writing dropped or kept as a hidden HTML comment.
package body

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gagoar/pr-brief/internal/convention"
)

var beginRe = regexp.MustCompile(`<!-- pr-brief:begin v(\d+) style=(\S+)(?: theme=(\S+))? -->`)

// Markers describes the managed block found in a body.
type Markers struct {
	Found    bool
	Version  string
	Style    string
	Theme    string // theme=... in the begin marker; empty when absent
	BeginIdx int    // index of the begin marker
	EndIdx   int    // index just after the end marker
}

// FindMarkers locates the managed block. Found is true only when both the
// begin marker and a later end marker exist.
func FindMarkers(body string) Markers {
	loc := beginRe.FindStringSubmatchIndex(body)
	if loc == nil {
		return Markers{}
	}
	end := strings.Index(body[loc[1]:], convention.End)
	if end < 0 {
		return Markers{}
	}
	return Markers{
		Found:    true,
		Version:  body[loc[2]:loc[3]],
		Style:    body[loc[4]:loc[5]],
		Theme:    themeOf(body, loc),
		BeginIdx: loc[0],
		EndIdx:   loc[1] + end + len(convention.End),
	}
}

// HasAnyMarker reports whether the body mentions pr-brief markers at all,
// including a broken pair.
func HasAnyMarker(body string) bool {
	return strings.Contains(body, convention.BeginPrefix) || strings.Contains(body, convention.End)
}

// Previous is a decoded pr-brief:previous block.
type Previous struct {
	Text  string
	Raw   string // encoded text exactly as stored in the body
	Saved string // timestamp text from the header
	Start int    // index of the block start in the body
	End   int    // index just after the block
}

var prevHeadRe = regexp.MustCompile(`<!-- pr-brief:previous v1 saved=(\S+)\n`)

// FindPrevious returns the decoded previous block, if the body has one.
func FindPrevious(body string) (Previous, bool) {
	loc := prevHeadRe.FindStringSubmatchIndex(body)
	if loc == nil {
		return Previous{}, false
	}
	rest := body[loc[1]:]
	endTok := "\n" + convention.PreviousEnd
	i := strings.Index(rest, endTok)
	if i < 0 {
		return Previous{}, false
	}
	return Previous{
		Text:  Decode(rest[:i]),
		Raw:   rest[:i],
		Saved: body[loc[2]:loc[3]],
		Start: loc[0],
		End:   loc[1] + i + len(endTok),
	}, true
}

// FindPast returns the description a human wrote.
//   - A pr-brief:previous block holds it from an earlier run.
//   - A body with no pr-brief markers is the past writing itself.
//   - A body with markers but no previous block has none: the old generated
//     brief is never saved.
func FindPast(body string) (string, bool) {
	if p, ok := FindPrevious(body); ok {
		return p.Text, true
	}
	if HasAnyMarker(body) {
		return "", false
	}
	if strings.TrimSpace(body) == "" {
		return "", false
	}
	return body, true
}

// Encode makes text safe inside an HTML comment: no "--" can appear, so the
// comment cannot close early. Decode reverses it exactly.
func Encode(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	return strings.ReplaceAll(s, "--", "-&#45;")
}

// Decode reverses Encode.
func Decode(s string) string {
	s = strings.ReplaceAll(s, "-&#45;", "--")
	return strings.ReplaceAll(s, "&amp;", "&")
}

// BuildPrevious renders the hidden block for the given past text. When the
// whole description would pass maxTotal characters, the text is cut and a note
// points at the local backup. managedLen is the length of the managed block
// plus the separator that precedes the previous block.
func BuildPrevious(past string, saved time.Time, managedLen, maxTotal int, backupPath string) string {
	head := fmt.Sprintf("%s v1 saved=%s\n", convention.PreviousOpen, saved.UTC().Format("2006-01-02T15:04Z"))
	tail := "\n" + convention.PreviousEnd
	render := func(text string) string { return head + Encode(text) + tail }

	if managedLen+len(render(past)) <= maxTotal {
		return render(past)
	}
	note := fmt.Sprintf("\n[truncated, full copy at %s]", backupPath)
	runes := []rune(past)
	lo, hi := 0, len(runes)
	for lo < hi { // largest prefix that fits
		mid := (lo + hi + 1) / 2
		if managedLen+len(render(string(runes[:mid])+note)) <= maxTotal {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return render(string(runes[:lo]) + note)
}

// Assemble builds the new description. managed is the whole block from the
// begin marker to the end marker. With mode "comment" and a past text, the
// hidden block follows it; with "drop" the managed block is the description.
func Assemble(managed, past string, hasPast bool, mode string, saved time.Time, backupPath string) string {
	out := strings.TrimRight(managed, "\n") + "\n"
	if mode != convention.PreviousComment || !hasPast {
		return out
	}
	const sep, trailer = "\n", "\n"
	prev := BuildPrevious(past, saved, len(out)+len(sep)+len(trailer), convention.MaxBodyChars, backupPath)
	return out + sep + prev + trailer
}

// Uncomment replaces the previous block with its decoded text under a heading,
// so an author can read it in the rendered PR.
func Uncomment(body string) (string, bool) {
	p, ok := FindPrevious(body)
	if !ok {
		return body, false
	}
	visible := "## Previous description\n\n" + strings.TrimRight(p.Text, "\n") + "\n"
	return body[:p.Start] + visible + body[p.End:], true
}

func themeOf(body string, loc []int) string {
	if len(loc) >= 8 && loc[6] >= 0 {
		return body[loc[6]:loc[7]]
	}
	return ""
}
