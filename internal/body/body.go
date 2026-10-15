// Package body parses pr-brief markers in a PR description, finds the past
// (human-written) description, and builds the new description with the past
// writing dropped or kept as a hidden HTML comment.
package body

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

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

// Fold is a decoded pr-brief:fold block: the earlier description inside a
// collapsed <details> element that follows the managed block.
type Fold struct {
	Text  string
	Saved string
	Start int // index of the opening <details> tag
	End   int // index just after </details>
}

// Matches `<details>\n<summary>Previous description</summary>\n<!-- pr-brief:fold v1 saved=2026-10-05T14:02Z -->\n`.
var foldHeadRe = regexp.MustCompile(`<details>\n<summary>Previous description</summary>\n<!-- pr-brief:fold v1 saved=(\S+) -->\n`)

const foldTail = "\n" + convention.FoldEnd + "\n</details>"

// FindFold returns the decoded fold block, if the body has one.
// It reads the body with Windows line endings turned into "\n", as the web editor stores them, so the
// indexes refer to that text.
func FindFold(body string) (Fold, bool) {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	loc := foldHeadRe.FindStringSubmatchIndex(body)
	if loc == nil {
		return Fold{}, false
	}
	rest := body[loc[1]:]
	i := strings.Index(rest, foldTail)
	if i < 0 {
		return Fold{}, false
	}
	return Fold{
		Text:  DecodeFold(strings.TrimPrefix(rest[:i], "\n")),
		Saved: body[loc[2]:loc[3]],
		Start: loc[0],
		End:   loc[1] + i + len(foldTail),
	}, true
}

// WithoutFold returns the body with the fold block removed, so a check never
// reads the earlier author's text as part of the description.
func WithoutFold(body string) string {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	f, ok := FindFold(body)
	if !ok {
		return body
	}
	return body[:f.Start] + body[f.End:]
}

// foldTokens pair each text a fold must not contain raw with the entity form that replaces it. A `</details>`
// in the earlier text is left alone: the fold ends at its own marker, and the author's own <details> blocks
// must keep working.
var foldTokens = []struct {
	raw, entity string
	escaped     *regexp.Regexp // the entity form with any number of `amp;` inserted: already-escaped text
	unescaped   *regexp.Regexp // the same, with at least one `amp;`
}{
	// Matches `&lt;!-- pr-brief` and `&amp;amp;lt;!-- pr-brief`; rejects `&lt;!-- other`.
	{"<!-- pr-brief", "&lt;!-- pr-brief", regexp.MustCompile(`&((?:amp;)*)lt;!-- pr-brief`), regexp.MustCompile(`&amp;((?:amp;)*)lt;!-- pr-brief`)},
}

// EncodeFold keeps the earlier text from forging a pr-brief marker. The text
// stays readable: the change is one HTML entity, and only where a marker
// would otherwise start. An entity form already in
// the text gets one more `amp;`, so DecodeFold reverses the change exactly.
func EncodeFold(s string) string {
	for _, t := range foldTokens {
		s = t.escaped.ReplaceAllString(s, "&amp;${1}"+t.entity[len("&"):])
		s = strings.ReplaceAll(s, t.raw, t.entity)
	}
	return s
}

// DecodeFold reverses EncodeFold.
func DecodeFold(s string) string {
	for i := len(foldTokens) - 1; i >= 0; i-- {
		t := foldTokens[i]
		s = strings.ReplaceAll(s, t.entity, t.raw)
		s = t.unescaped.ReplaceAllString(s, "&${1}"+t.entity[len("&"):])
	}
	return s
}

// FindPast returns the description a human wrote.
//   - A pr-brief:fold or pr-brief:previous block holds it from an earlier run.
//   - A body with no pr-brief markers is the past writing itself.
//   - A body with markers but no previous block has none: the old generated
//     brief is never saved.
func FindPast(body string) (string, bool) {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	if f, ok := FindFold(body); ok {
		return f.Text, true
	}
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

// BuildPrevious renders the hidden block for the given past text. It returns "" when not even a
// truncated block fits under the limit. When the
// whole description would pass maxTotal characters, the text is cut and a note
// points at the local backup. managedLen is the length of the managed block
// plus the separator that precedes the previous block.
func BuildPrevious(past string, saved time.Time, managedLen, maxTotal int, backupPath string) string {
	head := fmt.Sprintf("%s v1 saved=%s\n", convention.PreviousOpen, saved.UTC().Format("2006-01-02T15:04Z"))
	tail := "\n" + convention.PreviousEnd
	render := func(text string) string { return head + Encode(text) + tail }

	if managedLen+utf8.RuneCountInString(render(past)) <= maxTotal {
		return render(past)
	}
	note := fmt.Sprintf("\n[truncated, full copy at %s]", backupPath)
	if managedLen+utf8.RuneCountInString(render(note)) > maxTotal {
		return "" // not even the note fits: keep nothing, the backup holds the text
	}
	runes := []rune(past)
	lo, hi := 0, len(runes)
	for lo < hi { // largest prefix that fits
		mid := (lo + hi + 1) / 2
		if managedLen+utf8.RuneCountInString(render(string(runes[:mid])+note)) <= maxTotal {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return render(string(runes[:lo]) + note)
}

// BuildFold renders the collapsed block for the given past text. It returns "" when not even a
// truncated block fits under the limit. When the whole
// description would pass maxTotal characters, the text is cut and a note points
// at the local backup. managedLen is the length of the managed block plus the
// separators around the fold.
func BuildFold(past string, saved time.Time, managedLen, maxTotal int, backupPath string) string {
	head := fmt.Sprintf("<details>\n<summary>Previous description</summary>\n%s v1 saved=%s -->\n\n", convention.FoldOpen, saved.UTC().Format("2006-01-02T15:04Z"))
	render := func(text string) string { return head + EncodeFold(strings.TrimRight(text, "\n")) + foldTail }

	if managedLen+utf8.RuneCountInString(render(past)) <= maxTotal {
		return render(past)
	}
	note := fmt.Sprintf("\n\n[truncated, full copy at %s]", backupPath)
	if managedLen+utf8.RuneCountInString(render(note)) > maxTotal {
		return "" // not even the note fits: keep nothing, the backup holds the text
	}
	runes := []rune(past)
	lo, hi := 0, len(runes)
	for lo < hi { // largest prefix that fits
		mid := (lo + hi + 1) / 2
		if managedLen+utf8.RuneCountInString(render(string(runes[:mid])+note)) <= maxTotal {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return render(string(runes[:lo]) + note)
}

// Assemble builds the new description. managed is the whole block from the
// begin marker to the end marker. With a past text, mode "fold" puts it in a
// collapsed block after the managed block and "comment" in a hidden comment;
// with "drop" the managed block is the description.
// maxTotal is the longest description the host accepts, in characters.
func Assemble(managed, past string, hasPast bool, mode string, saved time.Time, backupPath string, maxTotal int) string {
	out := strings.TrimRight(managed, "\n") + "\n"
	if !hasPast || (mode != convention.PreviousComment && mode != convention.PreviousFold) {
		return out
	}
	const sep, trailer = "\n", "\n"
	if mode == convention.PreviousFold {
		fold := BuildFold(past, saved, utf8.RuneCountInString(out)+len(sep)+len(trailer), maxTotal, backupPath)
		if fold == "" {
			return out
		}
		return out + sep + fold + trailer
	}
	prev := BuildPrevious(past, saved, utf8.RuneCountInString(out)+len(sep)+len(trailer), maxTotal, backupPath)
	if prev == "" {
		return out
	}
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
