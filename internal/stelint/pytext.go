package stelint

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// This file reproduces the pieces of Python string and regex semantics that
// the upstream linter relies on and that Go does not offer directly.

// pyIsSpace reports whether r is whitespace by Python's str.isspace rules.
func pyIsSpace(r rune) bool {
	return unicode.IsSpace(r) || (r >= 0x1c && r <= 0x1f)
}

// pyIsWord reports whether r matches Python's Unicode-aware \w.
func pyIsWord(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r)
}

func pyStrip(s string) string  { return strings.TrimFunc(s, pyIsSpace) }
func pyLStrip(s string) string { return strings.TrimLeftFunc(s, pyIsSpace) }
func pyRStrip(s string) string { return strings.TrimRightFunc(s, pyIsSpace) }

// pyWordCount is len(s.split()).
func pyWordCount(s string) int {
	return len(strings.FieldsFunc(s, pyIsSpace))
}

// splitLines reproduces Python str.splitlines() (no keepends).
func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); {
		r, sz := utf8.DecodeRuneInString(s[i:])
		switch r {
		case '\n', '\v', '\f', 0x1c, 0x1d, 0x1e, 0x85, 0x2028, 0x2029:
			out = append(out, s[start:i])
			i += sz
			start = i
		case '\r':
			out = append(out, s[start:i])
			i += sz
			if i < len(s) && s[i] == '\n' {
				i++
			}
			start = i
		default:
			i += sz
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

// normalize maps a line to an equal-rune-count string that Go's ASCII-only
// \w, \b and \s treat exactly as Python treats the original:
//   - non-ASCII word runes become '0' (a word rune that no keyword contains);
//   - whitespace that Python's \s matches but Go's does not becomes '\t', so
//     it still fails to match the literal ' ' that upstream patterns use;
//   - U+017F and U+212A, which Python's IGNORECASE folds to s and k, become
//     's' and 'k'; U+0131 and U+0130 fold to 'i' likewise.
func normalize(rs []rune) string {
	var b strings.Builder
	b.Grow(len(rs))
	for _, r := range rs {
		switch {
		case r == 0x17f:
			b.WriteByte('s')
		case r == 0x212a:
			b.WriteByte('k')
		case r == 0x131 || r == 0x130:
			b.WriteByte('i')
		case r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\f':
			b.WriteRune(r)
		case pyIsSpace(r):
			b.WriteByte('\t')
		case r < 0x80:
			b.WriteRune(r)
		case pyIsWord(r):
			b.WriteByte('0')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// runeIndex converts a byte offset into s to a rune offset.
func runeIndex(s string, byteOff int) int {
	return utf8.RuneCountInString(s[:byteOff])
}

func isWordByte(c byte) bool {
	return c == '_' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
