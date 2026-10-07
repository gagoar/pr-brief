package shape

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/gagoar/pr-brief/internal/convention"
)

func TestFuncLabelsHaveNoParentheses(t *testing.T) {
	for _, f := range []*Func{
		{Kind: "func", Name: "Handle", Class: "InviteCommand"},
		{Kind: "func", Name: "helper"},
		{Kind: "func", Name: "GET /health"},
	} {
		if l := f.Label(); strings.ContainsAny(l, "()") {
			t.Errorf("label %q has a parenthesis", l)
		}
	}
}

func TestShortLabelOnlyDropsAClassPrefix(t *testing.T) {
	cases := map[string]string{
		"VeryLongClassNameForTestingOnly.SomeMethodName": "SomeMethodName",
		"short.Name": "short.Name",
	}
	for in, want := range cases {
		if got := shortLabel(in); got != want {
			t.Errorf("shortLabel(%q) = %q, want %q", in, got, want)
		}
	}
	// A dot inside a module name or a route is not a class separator.
	for _, in := range []string{
		"GET /api/v1.2/very/long/path/that/keeps/going",
		"some.module.with.dots.in.the.name · 3 fns",
	} {
		got := shortLabel(in)
		if utf8.RuneCountInString(got) > convention.MaxLabel {
			t.Errorf("shortLabel(%q) = %q is longer than %d", in, got, convention.MaxLabel)
		}
		if !strings.HasPrefix(got, in[:10]) {
			t.Errorf("shortLabel(%q) = %q lost the start of the name", in, got)
		}
	}
}
