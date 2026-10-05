package stelint

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// JSON renders the report exactly as upstream's `ste-lint.py --json` prints
// it (Python json.dumps with indent=2, ensure_ascii=True), without the
// trailing newline that print() adds.
func (r Report) JSON() ([]byte, error) {
	var b strings.Builder
	b.WriteString("{\n  \"violations\": ")
	if len(r.Violations) == 0 {
		b.WriteString("[]")
	} else {
		b.WriteString("[\n")
		for i, v := range r.Violations {
			b.WriteString("    {\n")
			fmt.Fprintf(&b, "      \"file\": %s,\n", pyJSONString(v.File))
			fmt.Fprintf(&b, "      \"line\": %d,\n", v.Line)
			fmt.Fprintf(&b, "      \"col\": %d,\n", v.Col)
			fmt.Fprintf(&b, "      \"rule\": %s,\n", pyJSONString(v.Rule))
			fmt.Fprintf(&b, "      \"level\": %s,\n", pyJSONString(v.Level))
			fmt.Fprintf(&b, "      \"match\": %s,\n", pyJSONString(v.Match))
			fmt.Fprintf(&b, "      \"message\": %s\n", pyJSONString(v.Message))
			b.WriteString("    }")
			if i < len(r.Violations)-1 {
				b.WriteString(",")
			}
			b.WriteString("\n")
		}
		b.WriteString("  ]")
	}
	fmt.Fprintf(&b, ",\n  \"count\": %d", r.Count)
	fmt.Fprintf(&b, ",\n  \"hard_count\": %d", r.HardCount)
	fmt.Fprintf(&b, ",\n  \"baseline\": %d", r.Baseline)
	fmt.Fprintf(&b, ",\n  \"words\": %d", r.Words)
	fmt.Fprintf(&b, ",\n  \"per_100_words\": %s\n}", pyFloat1(r.Per100Words))
	return []byte(b.String()), nil
}

// pyFloat1 formats a float already rounded to one decimal like Python repr.
func pyFloat1(v float64) string {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return s
}

// pyJSONString mirrors Python's json encoder with ensure_ascii=True.
func pyJSONString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); {
		r, sz := utf8.DecodeRuneInString(s[i:])
		i += sz
		switch {
		case r == '"':
			b.WriteString(`\"`)
		case r == '\\':
			b.WriteString(`\\`)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r == '\t':
			b.WriteString(`\t`)
		case r == '\b':
			b.WriteString(`\b`)
		case r == '\f':
			b.WriteString(`\f`)
		case r < 0x20:
			fmt.Fprintf(&b, `\u%04x`, r)
		case r < 0x7f || r == 0x7f:
			b.WriteRune(r)
		case r > 0xffff:
			r1, r2 := utf16.EncodeRune(r)
			fmt.Fprintf(&b, `\u%04x\u%04x`, r1, r2)
		default:
			fmt.Fprintf(&b, `\u%04x`, r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
