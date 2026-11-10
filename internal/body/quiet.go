package body

import (
	"errors"
	"html"
	"regexp"
	"strings"

	"github.com/gagoar/pr-brief/internal/tickets"
)

// A quiet description keeps the managed block in a collapsed <details> element, under one
// visible lead sentence. The wrapper sits outside the markers, so the gate still reads the
// same block it always reads.
//
//	**Tickets:** ...            (optional, kept by `tickets`)
//
//	<lead>
//
//	<details>
//	<summary>pr-brief, <model> <effort></summary>
//
//	<!-- pr-brief:begin ... -->
//	...
//	<!-- pr-brief:end -->
//
//	</details>

// Quiet describes the wrapper found around the managed block.
type Quiet struct {
	Lead    string // the visible text above the wrapper, without the Tickets line
	Summary string // the summary text, decoded
	Start   int    // index of the opening <details> tag
	Close   int    // index just after the closing </details>; -1 when it is missing
	Spaced  bool   // a blank line follows </summary>, so the host renders the Markdown inside
}

var quietHeadRe = regexp.MustCompile(`<details>\n<summary>([^\n]*)</summary>\n*$`)

// WrapQuiet puts managed (the block from the begin marker to the end marker) in the quiet
// wrapper. lead is one visible paragraph. summary is plain text: it is escaped here.
func WrapQuiet(managed, lead, summary string) (string, error) {
	lead = strings.TrimSpace(strings.ReplaceAll(lead, "\r\n", "\n"))
	if lead == "" {
		return "", errors.New("a quiet description needs a lead: one visible sentence on what the PR is for")
	}
	if strings.Contains(lead, "\n\n") {
		return "", errors.New("the lead is one paragraph")
	}
	if strings.HasPrefix(lead, tickets.LinePrefix) {
		return "", errors.New("the lead is not the Tickets line: pr-brief adds that line itself")
	}
	summary = strings.Join(strings.Fields(summary), " ")
	if summary == "" {
		summary = "pr-brief"
	}
	return lead + "\n\n<details>\n<summary>" + html.EscapeString(summary) + "</summary>\n\n" +
		strings.Trim(managed, "\n") + "\n\n</details>\n", nil
}

// FindQuiet returns the quiet wrapper around the managed block, if the body has one. It reads the
// body with Windows line endings turned into "\n", so the indexes refer to that text.
func FindQuiet(text string) (Quiet, bool) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	m := FindMarkers(WithoutFold(text))
	if !m.Found {
		return Quiet{}, false
	}
	loc := quietHeadRe.FindStringSubmatchIndex(text[:m.BeginIdx])
	if loc == nil {
		return Quiet{}, false
	}
	q := Quiet{
		Summary: html.UnescapeString(text[loc[2]:loc[3]]),
		Start:   loc[0],
		Close:   -1,
		Spaced:  strings.Count(text[loc[3]:m.BeginIdx], "\n") >= 2,
		Lead:    leadOf(text[:loc[0]]),
	}
	rest := text[m.EndIdx:]
	if trimmed := strings.TrimLeft(rest, "\n"); strings.HasPrefix(trimmed, "</details>") {
		q.Close = m.EndIdx + (len(rest) - len(trimmed)) + len("</details>")
	}
	return q, true
}

// leadOf drops the Tickets line from the text above the wrapper.
func leadOf(above string) string {
	var kept []string
	for _, l := range strings.Split(above, "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), tickets.LinePrefix) {
			continue
		}
		kept = append(kept, l)
	}
	return strings.TrimSpace(strings.Join(kept, "\n"))
}
