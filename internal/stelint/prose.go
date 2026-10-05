package stelint

import (
	"regexp"
	"strings"
)

var (
	urlRe = regexp.MustCompile(`(?i)(?:\b(?:https?|ftp)://|\bwww\.)[^\s<>]+`)
	// Trailing sentence punctuation is not part of a URL.
	urlTrailRe = regexp.MustCompile(`[.,;:!?)\]'"*_]+$`)
)

// LintProse lints prose with the default options. Fenced code blocks and
// inline code spans are already skipped by upstream's linter. Upstream does
// not skip URLs or HTML comments, so LintProse blanks them (with spaces, so
// line and column numbers stay put) before linting. This is a deviation from
// upstream; see UPSTREAM.md.
func LintProse(text string) Report {
	return Lint(blankNonProse(text), Options{})
}

func blankNonProse(text string) string {
	lines := splitLines(text)
	inFence, inComment := false, false
	for i, line := range lines {
		if !inComment {
			if codeFence.MatchString(pyStrip(line)) {
				inFence = !inFence
				continue
			}
			if inFence {
				continue
			}
		}
		rs := []rune(line)
		blank := func(from, to int) {
			for k := from; k < to; k++ {
				rs[k] = ' '
			}
		}
		pos := 0
		for pos < len(rs) {
			if inComment {
				end := indexRunes(rs, "-->", pos)
				if end < 0 {
					blank(pos, len(rs))
					pos = len(rs)
					break
				}
				blank(pos, end+3)
				pos = end + 3
				inComment = false
				continue
			}
			start := indexRunes(rs, "<!--", pos)
			for start >= 0 && insideCodeSpan(rs, start) {
				start = indexRunes(rs, "<!--", start+1)
			}
			if start < 0 {
				break
			}
			inComment = true
			blank(start, start+4)
			pos = start + 4
		}
		out := string(rs)
		if !inFence {
			out = blankURLs(out)
		}
		lines[i] = out
	}
	return strings.Join(lines, "\n")
}

func blankURLs(s string) string {
	// Skip URLs inside inline code spans: upstream removes those anyway.
	spans := inlineCode.FindAllStringIndex(s, -1)
	inSpan := func(b int) bool {
		for _, sp := range spans {
			if b >= sp[0] && b < sp[1] {
				return true
			}
		}
		return false
	}
	var out strings.Builder
	last := 0
	for _, loc := range urlRe.FindAllStringIndex(s, -1) {
		if inSpan(loc[0]) {
			continue
		}
		end := loc[1]
		if t := urlTrailRe.FindStringIndex(s[loc[0]:end]); t != nil {
			end = loc[0] + t[0]
		}
		out.WriteString(s[last:loc[0]])
		out.WriteString(strings.Repeat(" ", len([]rune(s[loc[0]:end]))))
		last = end
	}
	out.WriteString(s[last:])
	return out.String()
}

func indexRunes(rs []rune, sub string, from int) int {
	sr := []rune(sub)
outer:
	for i := from; i+len(sr) <= len(rs); i++ {
		for j, c := range sr {
			if rs[i+j] != c {
				continue outer
			}
		}
		return i
	}
	return -1
}

// insideCodeSpan reports whether rune offset p lies inside a backtick span.
func insideCodeSpan(rs []rune, p int) bool {
	s := string(rs)
	b := len(string(rs[:p]))
	for _, sp := range inlineCode.FindAllStringIndex(s, -1) {
		if b >= sp[0] && b < sp[1] {
			return true
		}
	}
	return false
}
