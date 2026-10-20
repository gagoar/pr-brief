package gate

import (
	"os"
	"strings"
	"testing"
)

func TestProbe(t *testing.T) {
	template, err := os.ReadFile("../initfiles/templates/pull_request_template.md")
	if err != nil {
		t.Fatal(err)
	}
	tmpl := strings.NewReplacer("{{STYLE}}", "ste+iceberg", "{{THEME}}", "github-dark").Replace(string(template))
	cases := []struct {
		name    string
		text    string
		written bool
		reason  string
	}{
		{"a body that passes", good, true, ""},
		{"a body that passes, with CRLF", strings.ReplaceAll(good, "\n", "\r\n"), true, ""},
		{"markers but a failing body", mutate("## Brief\n", "## Summary\n"), false, "sections"},
		{"the bare PR template", tmpl, false, ""},
		{"a plain description", "Fixes the thing.", false, "no pr-brief markers"},
		{"an empty description", "", false, "no pr-brief markers"},
		{"a skip line", "> pr-brief skipped: release\n", false, "skip line"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			written, reason := Probe(c.text, Check(c.text, Options{}))
			if written != c.written {
				t.Fatalf("written = %v (reason %q), want %v", written, reason, c.written)
			}
			if c.written != (reason == "") {
				t.Errorf("reason = %q: it must be empty exactly when written is true", reason)
			}
			if !strings.Contains(reason, c.reason) {
				t.Errorf("reason = %q, want it to contain %q", reason, c.reason)
			}
		})
	}
}

const testSignature = "<sub>Written with [pr-brief](https://github.com/gagoar/pr-brief) v0.5.1</sub>\n"

// The signature is opt-in. The gate never requires it, and it never counts as prose.
func TestSignatureIsOptionalAndNotProse(t *testing.T) {
	if r := Check(good, Options{}); !r.OK() {
		t.Fatalf("a body with no signature must pass: %+v", r.Findings)
	}
	signed := strings.Replace(good, "<!-- pr-brief:end -->", testSignature+"<!-- pr-brief:end -->", 1)
	var prose string
	r := Check(signed, Options{Lint: func(p string) []string { prose = p; return nil }})
	if !r.OK() {
		t.Fatalf("a signed body must pass: %+v", r.Findings)
	}
	if prose == "" || strings.Contains(prose, "Written with") {
		t.Errorf("the signature must not reach the lint, and the prose must not be empty: %q", prose)
	}
	if written, reason := Probe(signed, r); !written {
		t.Errorf("a signed body that passes is written: %s", reason)
	}
}
