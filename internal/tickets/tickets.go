// Package tickets finds the Jira and Linear tickets a pull request refers to, in its
// description and in its branch name, and keeps them visible when pr-brief rewrites the
// description. Jira and Linear link a PR to a ticket by reading the key out of the
// description, so a rewrite that drops the key unlinks the PR. A ticket is never lost.
package tickets

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Ticket is one reference, kept as it was written.
type Ticket struct {
	Key    string // the key in upper case: ENG-123
	System string // "jira", "linear", or "" for a bare key
	Raw    string // the text to keep, as written: a link, a URL, or a key, with its magic word
	Magic  bool   // Raw starts with a word such as "Closes", which Jira and Linear act on
}

// Prefix of the line pr-brief writes above the begin marker.
const LinePrefix = "**Tickets:**"

var (
	keyPat = `[A-Za-z][A-Za-z0-9]{1,9}-[0-9]{1,7}`

	// [text](https://x.atlassian.net/browse/ABC-1)
	mdLinkRe = regexp.MustCompile(`\[[^\]\n]*\]\(\s*(https?://[^)\s]+)\s*\)`)
	urlRe    = regexp.MustCompile(`https?://[^\s)\]>"'<]+`)
	// https://acme.atlassian.net/browse/ABC-1, or a board URL with ?selectedIssue=ABC-1
	jiraURLRe = regexp.MustCompile(`(?i)/browse/(` + keyPat + `)(?:[/?#]|$)|[?&]selectedIssue=(` + keyPat + `)(?:&|#|$)`)
	// https://linear.app/acme/issue/ENG-123/the-title
	linearURLRe = regexp.MustCompile(`(?i)^https?://linear\.app/[^/\s]+/issue/(` + keyPat + `)(?:[/?#]|$)`)
	bareKeyRe   = regexp.MustCompile(`\b[A-Z][A-Z0-9]{1,9}-[0-9]{1,7}\b`)
	// Words Jira and Linear read, or people write, in front of a key: "Closes ENG-123".
	magicRe = regexp.MustCompile(`(?i)\b(?:close[sd]?|closing|fix(?:es|ed)?|fixing|resolve[sd]?|resolving|complete[sd]?|completing|part of|related to|relates to|contributes to|towards?|refs?|references?|implements?)[ \t]+$`)

	// Keys that look like a ticket but are not one.
	notTickets = map[string]bool{
		"UTF": true, "SHA": true, "ISO": true, "CVE": true, "CWE": true, "GHSA": true, "MD": true, "RFC": true,
		"TLS": true, "SSL": true, "AES": true, "RSA": true, "HTTP": true, "UTC": true, "GMT": true, "IPV": true,
		"GPT": true, "COVID": true, "STE": true, "ASD": true,
	}
	// Words a branch starts with, which are not a project key.
	branchWords = map[string]bool{
		"FEATURE": true, "FEAT": true, "FIX": true, "BUGFIX": true, "HOTFIX": true, "RELEASE": true, "REL": true,
		"CHORE": true, "DOCS": true, "DOC": true, "TEST": true, "TESTS": true, "REFACTOR": true, "PERF": true,
		"CI": true, "BUILD": true, "REVERT": true, "WIP": true, "RC": true, "BETA": true, "ALPHA": true,
		"PATCH": true, "ISSUE": true, "BUG": true, "TASK": true, "STORY": true, "EPIC": true, "PR": true,
		"PULL": true, "DEV": true, "MAIN": true, "MASTER": true, "PROD": true, "STAGING": true, "SPIKE": true,
		"POC": true, "MERGE": true, "SYNC": true, "UPDATE": true, "UPGRADE": true, "BUMP": true, "ADD": true,
		"REMOVE": true, "CLEANUP": true, "STYLE": true, "TMP": true, "TEMP": true, "USER": true, "USERS": true,
	}
	versionRe = regexp.MustCompile(`^V[0-9]+$`)
	branchRe  = regexp.MustCompile(`(?i)(?:^|[/_-])(` + keyPat + `)(?:$|[/_-])`)
)

type span struct{ start, end int }

func inside(s span, spans []span) bool {
	for _, o := range spans {
		if s.start >= o.start && s.end <= o.end {
			return true
		}
	}
	return false
}

// ticketURL reports the key and system when u is a Jira or Linear ticket URL.
func ticketURL(u string) (key, system string, ok bool) {
	if m := linearURLRe.FindStringSubmatch(u); m != nil {
		return strings.ToUpper(m[1]), "linear", true
	}
	if m := jiraURLRe.FindStringSubmatch(u); m != nil {
		k := m[1]
		if k == "" {
			k = m[2]
		}
		return strings.ToUpper(k), "jira", true
	}
	return "", "", false
}

// withMagic widens a match to include the word in front of it ("Closes ", "Part of ").
func withMagic(text string, start int) int {
	lineStart := strings.LastIndex(text[:start], "\n") + 1
	if m := magicRe.FindStringIndex(text[lineStart:start]); m != nil {
		return lineStart + m[0]
	}
	return start
}

// Finder finds tickets. Its zero value, Default, guesses Jira and Linear keys (ABC-123). A pattern
// from the config replaces that guess for bare keys and for the branch name, so a team can name
// its projects, or add another system such as Azure DevOps work items (AB#123). Jira and Linear
// links are always read.
type Finder struct {
	text   *regexp.Regexp // the team's pattern, for bare keys in text; nil means the built-in guess
	branch *regexp.Regexp // the same pattern, ignoring case, for a branch name
}

// Default is the Finder with no team pattern.
var Default = &Finder{}

// MaxPatternLen keeps a pattern readable.
const MaxPatternLen = 200

// ValidatePattern checks a team pattern: valid RE2, short, and never empty-matching, because a
// pattern that matches nothing in particular would match everything.
func ValidatePattern(p string) error {
	if p == "" {
		return nil
	}
	if len(p) > MaxPatternLen {
		return fmt.Errorf("the ticket pattern is %d characters; the limit is %d", len(p), MaxPatternLen)
	}
	re, err := regexp.Compile(p)
	if err != nil {
		return fmt.Errorf("the ticket pattern is not a valid regular expression: %w", err)
	}
	if re.MatchString("") {
		return fmt.Errorf("the ticket pattern %q matches the empty string; it must match a key such as (PAY|OPS)-[0-9]+", p)
	}
	return nil
}

// NewFinder builds a Finder from a team pattern. An empty pattern gives the built-in guess.
func NewFinder(pattern string) (*Finder, error) {
	if err := ValidatePattern(pattern); err != nil {
		return nil, err
	}
	if pattern == "" {
		return Default, nil
	}
	return &Finder{text: regexp.MustCompile(pattern), branch: regexp.MustCompile("(?i)" + pattern)}, nil
}

// wordByte reports whether b is a letter, a digit or an underscore.
func wordByte(b byte) bool {
	return b == '_' || b >= '0' && b <= '9' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z'
}

// whole keeps the matches that are not part of a longer word: PAY-1 inside XPAY-1 or PAY-12a is not a ticket.
func whole(s string, locs [][]int) [][]int {
	var out [][]int
	for _, l := range locs {
		if l[0] > 0 && wordByte(s[l[0]-1]) && wordByte(s[l[0]]) {
			continue
		}
		if l[1] < len(s) && wordByte(s[l[1]]) && wordByte(s[l[1]-1]) {
			continue
		}
		out = append(out, l)
	}
	return out
}

// bare finds the keys in text that are not links. Own pattern, or the built-in guess.
func (f *Finder) bare(text string) [][]int {
	if f.text != nil {
		return whole(text, f.text.FindAllStringIndex(text, -1))
	}
	var out [][]int
	for _, loc := range bareKeyRe.FindAllStringIndex(text, -1) {
		if prefix, _, _ := strings.Cut(text[loc[0]:loc[1]], "-"); notTickets[prefix] {
			continue
		}
		out = append(out, loc)
	}
	return out
}

// Extract finds every ticket in a description, in the order they appear. Each key is listed
// once. When a key appears more than once, the best form wins: one with a magic word, then
// a link, then a bare key.
func Extract(text string) []Ticket { return Default.Extract(text) }

// Extract finds every ticket in a description with this Finder.
func (f *Finder) Extract(text string) []Ticket {
	type hit struct {
		pos int
		t   Ticket
	}
	var found []hit
	var used []span
	add := func(start, end int, key, system string) {
		from := withMagic(text, start)
		t := Ticket{Key: key, System: system, Raw: strings.TrimSpace(text[from:end]), Magic: from != start}
		found = append(found, hit{from, t})
		used = append(used, span{start, end})
	}
	for _, loc := range mdLinkRe.FindAllStringSubmatchIndex(text, -1) {
		if key, system, ok := ticketURL(text[loc[2]:loc[3]]); ok {
			add(loc[0], loc[1], key, system)
		}
	}
	for _, loc := range urlRe.FindAllStringIndex(text, -1) {
		s := span{loc[0], loc[1]}
		if inside(s, used) {
			continue
		}
		u := strings.TrimRight(text[loc[0]:loc[1]], ".,;:")
		if key, system, ok := ticketURL(u); ok {
			add(loc[0], loc[0]+len(u), key, system)
		}
	}
	for _, loc := range f.bare(text) {
		s := span{loc[0], loc[1]}
		if inside(s, used) {
			continue
		}
		key := strings.ToUpper(text[loc[0]:loc[1]])
		// A key inside a URL that is not a ticket URL belongs to that URL.
		if i := strings.LastIndexAny(text[:loc[0]], " \t\n"); strings.Contains(text[i+1:loc[0]], "://") {
			continue
		}
		add(loc[0], loc[1], key, "")
	}
	// Links, URLs and bare keys were found one kind at a time. List them in text order.
	sort.SliceStable(found, func(i, j int) bool { return found[i].pos < found[j].pos })
	ts := make([]Ticket, len(found))
	for i, h := range found {
		ts[i] = h.t
	}
	return Merge(ts)
}

// FromBranch finds the ticket keys in a branch name: feature/ABC-123-add-x, or the lower
// case form Linear makes, gago/eng-45-fix-login. The key is returned in upper case.
func FromBranch(branch string) []Ticket { return Default.FromBranch(branch) }

// FromBranch finds the ticket keys in a branch name with this Finder. A team pattern is matched
// ignoring case, and is not filtered by the lists of branch words that only the built-in guess needs.
func (f *Finder) FromBranch(branch string) []Ticket {
	b := strings.TrimSpace(branch)
	b = strings.TrimPrefix(b, "refs/heads/")
	b = strings.TrimPrefix(b, "origin/")
	if f.branch != nil {
		var own []Ticket
		for _, l := range whole(b, f.branch.FindAllStringIndex(b, -1)) {
			key := strings.ToUpper(b[l[0]:l[1]])
			own = append(own, Ticket{Key: key, Raw: key})
		}
		return Merge(own)
	}
	var out []Ticket
	// Adjacent matches share a separator, so scan from just past each match's key.
	for rest := b; ; {
		loc := branchRe.FindStringSubmatchIndex(rest)
		if loc == nil {
			break
		}
		key := strings.ToUpper(rest[loc[2]:loc[3]])
		prefix, _, _ := strings.Cut(key, "-")
		if !branchWords[prefix] && !notTickets[prefix] && !versionRe.MatchString(prefix) {
			out = append(out, Ticket{Key: key, Raw: key})
		}
		rest = rest[loc[3]:]
	}
	return Merge(out)
}

// rank orders the forms of one ticket: a magic word is worth most, then a link, then a bare key.
func rank(t Ticket) int {
	switch {
	case t.Magic:
		return 3
	case strings.Contains(t.Raw, "://"):
		return 2
	}
	return 1
}

// Merge joins lists of tickets. Each key is listed once, at its first position, in its best form.
func Merge(lists ...[]Ticket) []Ticket {
	var out []Ticket
	at := map[string]int{}
	for _, list := range lists {
		for _, t := range list {
			i, seen := at[t.Key]
			switch {
			case !seen:
				at[t.Key] = len(out)
				out = append(out, t)
			case rank(t) > rank(out[i]):
				out[i] = t
			}
		}
	}
	return out
}

// FromLine reads the tickets in the Tickets line a body already holds, and nowhere else. The
// text of a new description is not a source: it is a summary of a diff, and a key it mentions
// as an example is not a ticket of this PR.
func FromLine(body string) []Ticket { return Default.FromLine(body) }

// FromLine reads the Tickets line with this Finder.
func (f *Finder) FromLine(body string) []Ticket {
	var out []Ticket
	for _, l := range strings.Split(body, "\n") {
		if t := strings.TrimSpace(l); strings.HasPrefix(t, LinePrefix) {
			out = append(out, f.Extract(strings.TrimPrefix(t, LinePrefix))...)
		}
	}
	return Merge(out)
}

// Missing lists the tickets whose key does not appear in text, in any case. ENG-1 is not found
// in ENG-12, and ENG-12 is not found in XENG-12.
func Missing(text string, ts []Ticket) []Ticket {
	var out []Ticket
	for _, t := range ts {
		re := regexp.MustCompile(`(?i)(?:^|[^A-Za-z0-9])` + regexp.QuoteMeta(t.Key) + `(?:[^0-9]|$)`)
		if !re.MatchString(text) {
			out = append(out, t)
		}
	}
	return out
}

// Line is the visible line that holds the tickets. It is empty when there are none.
func Line(ts []Ticket) string {
	if len(ts) == 0 {
		return ""
	}
	parts := make([]string, len(ts))
	for i, t := range ts {
		parts[i] = t.Raw
	}
	return LinePrefix + " " + strings.Join(parts, " · ")
}

const beginPrefix = "<!-- pr-brief:begin"

// Ensure puts the Tickets line above the begin marker, in place of any line pr-brief wrote
// before. Text that is not a Tickets line is left as it was. A body with no tickets comes back
// as it was.
func Ensure(body string, ts []Ticket) string {
	line := Line(ts)
	if line == "" {
		return body
	}
	lines := strings.Split(body, "\n")
	var kept []string
	for i := 0; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), LinePrefix) {
			if i+1 < len(lines) && strings.TrimSpace(lines[i+1]) == "" {
				i++
			}
			continue
		}
		kept = append(kept, lines[i])
	}
	at := len(kept)
	for i, l := range kept {
		if strings.HasPrefix(strings.TrimSpace(l), beginPrefix) {
			at = i
			break
		}
	}
	if at == len(kept) {
		at = 0 // no markers: the line leads the text
	}
	out := append(append(append([]string{}, kept[:at]...), line, ""), kept[at:]...)
	return strings.Join(out, "\n")
}
