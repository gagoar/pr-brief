package gate

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// word is one shell word. literal is false when part of it depends on
// something the hook cannot evaluate (a variable, an unknown substitution).
type word struct {
	text    string
	literal bool
}

var (
	heredocStartRe = regexp.MustCompile(`^cat\s+<<(-?)\s*(?:'(\w+)'|"(\w+)"|(\w+))[ \t]*\n`)
	catFileRe      = regexp.MustCompile(`^cat\s+(\S+)\s*$`)
	heredocMarkRe  = regexp.MustCompile(`<<(-?)\s*(?:'(\w+)'|"(\w+)"|(\w+))`)
)

// fileReader reads a file named in a command, relative to the command's cwd.
type fileReader func(path string) (string, error)

func newFileReader(cwd string) fileReader {
	return func(p string) (string, error) {
		if strings.HasPrefix(p, "~/") {
			if home, err := os.UserHomeDir(); err == nil {
				p = filepath.Join(home, p[2:])
			}
		}
		if !filepath.IsAbs(p) && cwd != "" {
			p = filepath.Join(cwd, p)
		}
		data, err := os.ReadFile(p)
		return string(data), err
	}
}

func isSpace(c byte) bool { return c == ' ' || c == '\t' }

// parseWords reads shell words from s until an unquoted command separator.
func parseWords(s string, read fileReader) []word {
	var words []word
	i := 0
	for i < len(s) {
		for i < len(s) && isSpace(s[i]) {
			i++
		}
		if i >= len(s) {
			break
		}
		if s[i] == '\\' && i+1 < len(s) && s[i+1] == '\n' { // line continuation
			i += 2
			continue
		}
		if stopsAt(s, i) {
			break
		}
		var w word
		w.literal = true
		var b strings.Builder
	word:
		for i < len(s) {
			c := s[i]
			switch {
			case isSpace(c), c == '\n':
				break word
			case stopsAt(s, i):
				break word
			case c == '\'':
				j := strings.IndexByte(s[i+1:], '\'')
				if j < 0 {
					b.WriteString(s[i+1:])
					i = len(s)
					break word
				}
				b.WriteString(s[i+1 : i+1+j])
				i += j + 2
			case c == '"':
				i++
				for i < len(s) && s[i] != '"' {
					switch {
					case s[i] == '\\' && i+1 < len(s):
						if strings.IndexByte("\"\\$`\n", s[i+1]) >= 0 {
							if s[i+1] != '\n' {
								b.WriteByte(s[i+1])
							}
						} else {
							b.WriteByte('\\')
							b.WriteByte(s[i+1])
						}
						i += 2
					case s[i] == '$' && i+1 < len(s) && s[i+1] == '(':
						text, lit, n := substitution(s, i, read)
						b.WriteString(text)
						w.literal = w.literal && lit
						i += n
					case s[i] == '$' || s[i] == '`':
						w.literal = false
						b.WriteByte(s[i])
						i++
					default:
						b.WriteByte(s[i])
						i++
					}
				}
				i++ // closing quote
			case c == '\\' && i+1 < len(s):
				b.WriteByte(s[i+1])
				i += 2
			case c == '$' && i+1 < len(s) && s[i+1] == '(':
				text, lit, n := substitution(s, i, read)
				b.WriteString(text)
				w.literal = w.literal && lit
				i += n
			case c == '$' || c == '`':
				w.literal = false
				b.WriteByte(c)
				i++
			default:
				b.WriteByte(c)
				i++
			}
		}
		w.text = b.String()
		words = append(words, w)
	}
	return words
}

// stopsAt reports an unquoted command separator at s[i].
func stopsAt(s string, i int) bool {
	switch s[i] {
	case ';', '\n', ')':
		return true
	case '&':
		return true // && and a lone & both end the command
	case '|':
		return true
	case '#':
		return i == 0 || isSpace(s[i-1])
	}
	return false
}

// substitution evaluates $( ... ) that starts at s[i]. It returns the text,
// whether it is fully known, and how many bytes it consumed.
func substitution(s string, i int, read fileReader) (string, bool, int) {
	end := matchParen(s, i+1)
	if end < 0 {
		return s[i:], false, len(s) - i
	}
	inner := s[i+2 : end]
	n := end + 1 - i
	raw := s[i : end+1]

	trimmed := strings.TrimSpace(inner)
	if m := heredocStartRe.FindStringSubmatch(trimmed); m != nil {
		delim, quoted := m[2], true
		if delim == "" {
			delim = m[3]
		}
		if delim == "" {
			delim, quoted = m[4], false
		}
		rest := trimmed[len(m[0]):]
		var out []string
		found := false
		for _, line := range strings.Split(rest, "\n") {
			check := line
			if m[1] == "-" {
				check = strings.TrimLeft(line, "\t")
			}
			if check == delim {
				found = true
				break
			}
			out = append(out, line)
		}
		if found {
			text := strings.TrimRight(strings.Join(out, "\n"), "\n")
			lit := quoted || !strings.ContainsAny(text, "$`")
			return text, lit, n
		}
	}
	if m := catFileRe.FindStringSubmatch(trimmed); m != nil && !strings.ContainsAny(m[1], "$`*?") {
		if data, err := read(strings.Trim(m[1], `'"`)); err == nil {
			return strings.TrimRight(data, "\n"), true, n
		}
	}
	return raw, false, n
}

// matchParen returns the index of the ")" that closes the "(" at s[open].
func matchParen(s string, open int) int {
	depth := 0
	for i := open; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case '\'':
			if j := strings.IndexByte(s[i+1:], '\''); j >= 0 {
				i += j + 1
			}
		case '"':
			for i++; i < len(s) && s[i] != '"'; i++ {
				if s[i] == '\\' {
					i++
				}
			}
		case '<':
			if strings.HasPrefix(s[i:], "<<") {
				if m := heredocMarkRe.FindStringSubmatch(s[i:]); m != nil && strings.Index(s[i:], m[0]) == 0 {
					delim := firstNonEmpty(m[2], m[3], m[4])
					i = skipHeredoc(s, i+len(m[0]), delim, m[1] == "-") - 1
				}
			}
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// skipHeredoc returns the index after the heredoc body that starts on the line
// after i. If the terminator is missing it returns len(s).
func skipHeredoc(s string, i int, delim string, dash bool) int {
	nl := strings.IndexByte(s[i:], '\n')
	if nl < 0 {
		return len(s)
	}
	pos := i + nl + 1
	for pos <= len(s) {
		eol := strings.IndexByte(s[pos:], '\n')
		line := s[pos:]
		next := len(s)
		if eol >= 0 {
			line = s[pos : pos+eol]
			next = pos + eol + 1
		}
		if dash {
			line = strings.TrimLeft(line, "\t")
		}
		if line == delim {
			return next
		}
		if eol < 0 {
			return len(s)
		}
		pos = next
	}
	return len(s)
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

// atCommandPosition reports whether s[idx] is outside any quote and outside any
// unquoted heredoc body, so a match there is a real command and not text.
func atCommandPosition(s string, idx int) bool {
	for i := 0; i < idx; i++ {
		switch s[i] {
		case '\\':
			i++
		case '\'':
			j := strings.IndexByte(s[i+1:], '\'')
			if j < 0 || i+1+j >= idx {
				return false
			}
			i += j + 1
		case '"':
			i++
			for i < len(s) && s[i] != '"' {
				if s[i] == '\\' {
					i++
				}
				i++
			}
			if i >= idx {
				return false
			}
		case '<':
			if m := heredocMarkRe.FindStringSubmatch(s[i:]); m != nil && strings.HasPrefix(s[i:], m[0]) {
				delim := firstNonEmpty(m[2], m[3], m[4])
				after := skipHeredoc(s, i+len(m[0]), delim, m[1] == "-")
				if after > idx {
					return false
				}
				i = after - 1
			}
		}
	}
	return true
}
