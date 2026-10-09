package tickets

import (
	"reflect"
	"strings"
	"testing"
)

func keys(ts []Ticket) []string {
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = t.Key
	}
	return out
}

func TestExtractFindsJiraAndLinearInEveryForm(t *testing.T) {
	text := `Fixes the login bug. See [ABC-101](https://acme.atlassian.net/browse/ABC-101) first.
Linear: https://linear.app/acme/issue/eng-42/fix-the-thing and the board
https://acme.atlassian.net/jira/software/projects/XYZ/boards/1?selectedIssue=XYZ-7.
A bare key: PAY-9, and "Closes OPS-3".`
	got := Extract(text)
	want := []Ticket{
		{Key: "ABC-101", System: "jira", Raw: "[ABC-101](https://acme.atlassian.net/browse/ABC-101)"},
		{Key: "ENG-42", System: "linear", Raw: "https://linear.app/acme/issue/eng-42/fix-the-thing"},
		{Key: "XYZ-7", System: "jira", Raw: "https://acme.atlassian.net/jira/software/projects/XYZ/boards/1?selectedIssue=XYZ-7"},
		{Key: "PAY-9", Raw: "PAY-9"},
		{Key: "OPS-3", Raw: "Closes OPS-3", Magic: true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %+v\nwant %+v", got, want)
	}
}

func TestExtractKeepsTheMagicWordBecauseLinearActsOnIt(t *testing.T) {
	for in, raw := range map[string]string{
		"Closes ENG-123":         "Closes ENG-123",
		"fixes ENG-123 and more": "fixes ENG-123",
		"Part of ENG-5":          "Part of ENG-5",
		"Resolves [ENG-9](https://linear.app/a/issue/ENG-9/x)": "Resolves [ENG-9](https://linear.app/a/issue/ENG-9/x)",
		"See ENG-123": "ENG-123",
	} {
		got := Extract(in)
		if len(got) != 1 || got[0].Raw != raw {
			t.Errorf("Extract(%q) = %+v, want raw %q", in, got, raw)
		}
	}
}

func TestExtractSkipsWhatLooksLikeATicketButIsNot(t *testing.T) {
	text := "Use UTF-8 and SHA-256. See CVE-2024-1234, RFC-2616 and TLS-1. Docs: https://example.com/guides/ABC-12 and ASD-STE100."
	if got := Extract(text); len(got) != 0 {
		t.Errorf("found tickets that are not: %+v", got)
	}
}

func TestExtractListsEachKeyOnceInItsBestForm(t *testing.T) {
	text := "ENG-1 is the plan. Later: https://linear.app/a/issue/ENG-1/x and then Closes ENG-1."
	got := Extract(text)
	if len(got) != 1 || got[0].Raw != "Closes ENG-1" {
		t.Errorf("want one ticket, the one with the magic word: %+v", got)
	}
	got = Extract("ENG-1 first, then https://linear.app/a/issue/ENG-1/x")
	if len(got) != 1 || got[0].System != "linear" || !strings.Contains(got[0].Raw, "linear.app") {
		t.Errorf("a link beats a bare key: %+v", got)
	}
}

func TestFromBranch(t *testing.T) {
	cases := map[string][]string{
		"feature/ABC-123-add-login":              {"ABC-123"},
		"gago/eng-45-fix-login":                  {"ENG-45"},
		"refs/heads/ENG-5-x":                     {"ENG-5"},
		"origin/bugfix/PAY-9":                    {"PAY-9"},
		"feature/ABC-1-and-ABC-2":                {"ABC-1", "ABC-2"},
		"main":                                   nil,
		"hotfix-2":                               nil,
		"fix-123-typo":                           nil,
		"v2-123":                                 nil,
		"release/2026-10-08":                     nil,
		"dependabot/npm_and_yarn/lodash-4.17.21": nil,
		"renovate/net-0.x":                       nil,
		"feature/no-ticket-here":                 nil,
	}
	for branch, want := range cases {
		if got := keys(FromBranch(branch)); !reflect.DeepEqual(got, append([]string(nil), want...)) && !(len(got) == 0 && len(want) == 0) {
			t.Errorf("FromBranch(%q) = %v, want %v", branch, got, want)
		}
	}
}

func TestEnsurePutsTheLineAboveTheMarkerAndIsIdempotent(t *testing.T) {
	body := "<!-- pr-brief:begin v1 style=iceberg theme=github-dark -->\n## Brief\nText.\n<!-- pr-brief:end -->\n"
	ts := []Ticket{{Key: "ENG-1", Raw: "Closes ENG-1"}, {Key: "ABC-2", Raw: "ABC-2"}}
	got := Ensure(body, ts)
	want := "**Tickets:** Closes ENG-1 · ABC-2\n\n" + body
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	if again := Ensure(got, ts); again != got {
		t.Error("running it again changes the body")
	}
	// A new set replaces the old line and leaves other text alone.
	next := Ensure("Intro text\n"+got, []Ticket{{Key: "ENG-1", Raw: "Closes ENG-1"}})
	if strings.Count(next, LinePrefix) != 1 || strings.Contains(next, "ABC-2") || !strings.HasPrefix(next, "Intro text\n") {
		t.Errorf("the line must be replaced, not repeated:\n%s", next)
	}
	// What was written is read back whole.
	if got := keys(Extract(Ensure(body, ts))); !reflect.DeepEqual(got, []string{"ENG-1", "ABC-2"}) {
		t.Errorf("Extract(Ensure(..)) = %v", got)
	}
	if Ensure(body, nil) != body {
		t.Error("no tickets, no change")
	}
	if got := Ensure("plain text", ts); !strings.HasPrefix(got, "**Tickets:** Closes ENG-1 · ABC-2\n\nplain text") {
		t.Errorf("no markers: the line leads the text:\n%s", got)
	}
}

func TestMissing(t *testing.T) {
	ts := []Ticket{{Key: "ENG-1"}, {Key: "ABC-2"}}
	if got := keys(Missing("closes eng-1 only", ts)); !reflect.DeepEqual(got, []string{"ABC-2"}) {
		t.Errorf("Missing = %v", got)
	}
	if len(Missing("ENG-1 and ABC-2", ts)) != 0 {
		t.Error("nothing is missing")
	}
}

func TestMissingMatchesWholeKeys(t *testing.T) {
	ts := []Ticket{{Key: "ENG-1"}}
	for text, missing := range map[string]bool{
		"ENG-12 only":   true,
		"XENG-1 only":   true,
		"eng-1":         false,
		"(ENG-1)":       false,
		"Closes ENG-1.": false,
		"x/ENG-1/y":     false,
		"ENG-1":         false,
		"":              true,
	} {
		if got := len(Missing(text, ts)) == 1; got != missing {
			t.Errorf("Missing(%q) = %v, want %v", text, got, missing)
		}
	}
}

func TestFromLineReadsOnlyTheTicketsLine(t *testing.T) {
	body := "**Tickets:** Closes ENG-1 · [ABC-2](https://acme.atlassian.net/browse/ABC-2)\n\n<!-- pr-brief:begin v1 -->\n## Brief\nThe branch feature/XYZ-9-x is an example in the text.\n<!-- pr-brief:end -->\n"
	got := FromLine(body)
	if k := keys(got); !reflect.DeepEqual(k, []string{"ENG-1", "ABC-2"}) {
		t.Errorf("FromLine = %v; the Brief's example key must not count", k)
	}
	if FromLine("no line here, ENG-5") != nil {
		t.Error("text outside a Tickets line is not read")
	}
}

func finder(t *testing.T, pattern string) *Finder {
	t.Helper()
	f, err := NewFinder(pattern)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestAPatternNamesTheProjectsAndSilencesTheGuess(t *testing.T) {
	f := finder(t, `(PAY|OPS)-[0-9]+`)
	// The built-in guess would also take ENG-5 and LODASH-4. The team's pattern takes only its projects.
	got := keys(f.Extract("PAY-1 and ENG-5, OPS-22 and LODASH-4. Closes PAY-3."))
	if !reflect.DeepEqual(got, []string{"PAY-1", "OPS-22", "PAY-3"}) {
		t.Errorf("Extract = %v", got)
	}
	for in, want := range map[string][]string{
		"feature/pay-12-login":                   {"PAY-12"},
		"gago/OPS-7":                             {"OPS-7"},
		"dependabot/npm_and_yarn/lodash-4.17.21": nil,
		"feature/ENG-9-x":                        nil,
	} {
		got := keys(f.FromBranch(in))
		if len(got) != len(want) || (len(want) > 0 && !reflect.DeepEqual(got, want)) {
			t.Errorf("FromBranch(%q) = %v, want %v", in, got, want)
		}
	}
	// A match inside a longer word is not a ticket.
	if got := f.Extract("XPAY-1 and PAY-12a"); len(got) != 0 {
		t.Errorf("a key inside a longer word is not a ticket: %+v", got)
	}
}

func TestAPatternMayNameAnotherSystem(t *testing.T) {
	f := finder(t, `AB#[0-9]+|#[0-9]+`)
	text := "Fixes AB#1234, see #56 and (#7). Closes https://linear.app/a/issue/ENG-2/x."
	got := f.Extract(text)
	if k := keys(got); !reflect.DeepEqual(k, []string{"AB#1234", "#56", "#7", "ENG-2"}) {
		t.Fatalf("Extract = %v", k)
	}
	if got[0].Raw != "Fixes AB#1234" || !got[0].Magic {
		t.Errorf("the magic word stays with the work item: %+v", got[0])
	}
	// What was written is read back, so a second run keeps every item.
	body := Ensure("<!-- pr-brief:begin v1 -->\nx\n<!-- pr-brief:end -->\n", got)
	if k := keys(f.FromLine(body)); !reflect.DeepEqual(k, []string{"AB#1234", "#56", "#7", "ENG-2"}) {
		t.Errorf("FromLine = %v", k)
	}
	if k := keys(Default.FromLine(body)); reflect.DeepEqual(k, []string{"AB#1234", "#56", "#7", "ENG-2"}) {
		t.Error("the built-in guess cannot read AB#1234, which is why the line must be read with the team's pattern")
	}
	if len(Missing("Fixes AB#12345", []Ticket{{Key: "AB#1234"}})) != 1 {
		t.Error("AB#1234 is not in AB#12345")
	}
}

func TestValidatePattern(t *testing.T) {
	for _, ok := range []string{"", `[A-Z]+-[0-9]+`, `(PAY|OPS)-[0-9]+`, `AB#[0-9]+`} {
		if err := ValidatePattern(ok); err != nil {
			t.Errorf("%q must be valid: %v", ok, err)
		}
	}
	for bad, want := range map[string]string{
		"(":                      "not a valid regular expression",
		"[0-9]*":                 "matches the empty string",
		"a?":                     "matches the empty string",
		strings.Repeat("a", 201): "the limit is 200",
		`(?=PAY)PAY-[0-9]+`:      "not a valid regular expression", // RE2 has no lookahead
	} {
		if err := ValidatePattern(bad); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("ValidatePattern(%q) = %v, want %q", bad, err, want)
		}
	}
	if f, err := NewFinder(""); err != nil || f != Default {
		t.Error("an empty pattern gives the built-in Finder")
	}
}
