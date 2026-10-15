package gate

import (
	"strings"

	"github.com/gagoar/pr-brief/internal/body"
)

// Probe says whether pr-brief already wrote the description: it has valid markers and passes
// every check. A skip line, a body without markers and a body that only holds the repo's PR
// template (markers, but no brief) are not written. reason names the first thing that is
// missing, and is empty when written is true.
func Probe(text string, res Result) (written bool, reason string) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	switch {
	case res.Skipped:
		return false, "the description has a pr-brief skip line"
	case !body.FindMarkers(text).Found:
		return false, "no pr-brief markers"
	case len(res.Findings) > 0:
		return false, res.Findings[0].Rule + ": " + res.Findings[0].Message
	}
	return true, ""
}
