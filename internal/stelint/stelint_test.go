package stelint

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func rulesOf(r Report) map[string]int {
	m := map[string]int{}
	for _, v := range r.Violations {
		m[v.Rule]++
	}
	return m
}

func lintOne(text string) Report { return Lint(text, Options{}) }

func ofRule(r Report, rule string) []Violation {
	var out []Violation
	for _, v := range r.Violations {
		if v.Rule == rule {
			out = append(out, v)
		}
	}
	return out
}

// TestSelftest ports upstream's --selftest assertions one for one.
func TestSelftest(t *testing.T) {
	t.Run("all rules fire", func(t *testing.T) {
		r := lintOne("The panel is removed; spin up the job. Perform an analysis of the seamless log. We have received the report.")
		for _, want := range []string{"semicolon", "phrasal-verb", "nominalization", "marketing-adjective", "passive-voice", "present-perfect"} {
			if rulesOf(r)[want] == 0 {
				t.Errorf("missing %s: %+v", want, r.Violations)
			}
		}
	})
	t.Run("hedges never flagged", func(t *testing.T) {
		r := lintOne("The request may have failed. It could be a timeout. The disk might have filled.")
		if len(r.Violations) != 0 {
			t.Errorf("%+v", r.Violations)
		}
	})
	t.Run("irregular participles", func(t *testing.T) {
		r := lintOne("The task has run. The job has set the flag. We have begun.")
		if n := rulesOf(r)["present-perfect"]; n != 3 {
			t.Errorf("got %d: %+v", n, r.Violations)
		}
		r = lintOne("The job may have run.")
		if rulesOf(r)["present-perfect"] != 0 {
			t.Errorf("%+v", r.Violations)
		}
	})
	t.Run("modal perfects across negation and whitespace", func(t *testing.T) {
		for _, modal := range []string{"may", "might", "could", "should", "would", "must"} {
			for _, gap := range []string{" ", "  ", "\t", " not "} {
				r := lintOne(fmt.Sprintf("The task %s%shave run.", modal, gap))
				if rulesOf(r)["present-perfect"] != 0 {
					t.Errorf("%q %q: %+v", modal, gap, r.Violations)
				}
			}
		}
		r := lintOne("The task couldn't have run. The task MAY NOT HAVE RUN.")
		if rulesOf(r)["present-perfect"] != 0 {
			t.Errorf("%+v", r.Violations)
		}
		r = lintOne("The task has run. We have begun. The flag is set.")
		if n := rulesOf(r)["present-perfect"]; n != 2 {
			t.Errorf("got %d: %+v", n, r.Violations)
		}
		if rulesOf(r)["passive-voice"] == 0 {
			t.Errorf("want passive: %+v", r.Violations)
		}
		r = lintOne("The task is gone.")
		if rulesOf(r)["passive-voice"] != 0 {
			t.Errorf("%+v", r.Violations)
		}
	})
	t.Run("code fence skipped", func(t *testing.T) {
		if r := lintOne("```\nx = a; y = b\n```"); len(r.Violations) != 0 {
			t.Errorf("%+v", r.Violations)
		}
	})
	t.Run("dangling conjunction markers", func(t *testing.T) {
		r := lintOne("- Confirm the target and\n* Record the result OR  \n+ Close the panel\n1. Start the task and\n2) Stop the task OR")
		d := ofRule(r, "dangling-conjunction")
		if len(d) != 4 {
			t.Fatalf("%+v", d)
		}
		for i, line := range []int{1, 2, 4, 5} {
			if d[i].Line != line || d[i].Col != 1 || d[i].Level != LevelHard {
				t.Errorf("%d: %+v", i, d[i])
			}
		}
	})
	t.Run("dangling conjunction continuation and ignores", func(t *testing.T) {
		none := func(text string) {
			t.Helper()
			if r := lintOne(text); len(ofRule(r, "dangling-conjunction")) != 0 {
				t.Errorf("%q: %+v", text, r.Violations)
			}
		}
		none("  - Confirm the target and\n    record the result.")
		none("    - code and")
		none("> - Confirm the target and\n> - Record the result or")
		none("```text\n- code and\n```")
		none(" - Start the task and\n   record the result.")
		none("-  Start the task and\n   record the result.")
		none("-\tStart the task and")
		none("- Start the task and.\n- Stop the task or,")
		none("- Start the task and\n\n  record the result.")
		none("The process may include steps and")
		none("- Use `and` as a label")
		none("- Combine `left` and `right`")
		none("~~~\n- code and\n~~~")

		d := ofRule(lintOne("- Confirm the target\n  and"), "dangling-conjunction")
		if len(d) != 1 || d[0].Line != 2 || d[0].Col != 3 {
			t.Errorf("%+v", d)
		}
		d = ofRule(lintOne("- Do this and\n~~~\ncode and\n~~~"), "dangling-conjunction")
		if len(d) != 1 || d[0].Line != 1 {
			t.Errorf("%+v", d)
		}
		d = ofRule(LintNamed("   - Start the task and", "fixture.md", Options{}), "dangling-conjunction")
		if len(d) != 1 || d[0].Col != 4 || d[0].File != "fixture.md" ||
			!strings.HasSuffix(d[0].Match, "and") || !strings.Contains(d[0].Message, "Complete the item") {
			t.Errorf("%+v", d)
		}
		d = ofRule(lintOne("100. Start the task and\n  unrelated text"), "dangling-conjunction")
		if len(d) != 1 {
			t.Errorf("%+v", d)
		}
		d = ofRule(lintOne("- Parent item and\n  - Nested item or"), "dangling-conjunction")
		if len(d) != 2 || d[0].Line != 1 || d[1].Line != 2 {
			t.Errorf("%+v", d)
		}
	})
	t.Run("long sentence", func(t *testing.T) {
		r := lintOne(strings.TrimSpace(strings.Repeat("word ", 30)) + ".")
		if rulesOf(r)["long-sentence"] == 0 {
			t.Errorf("%+v", r.Violations)
		}
	})
	t.Run("tables", func(t *testing.T) {
		var terms []string
		for i := 1; i <= 24; i++ {
			terms = append(terms, fmt.Sprintf("term%d", i))
		}
		short := strings.Join(terms, " ") + "."
		for _, table := range []string{
			"| Label | Detail |\n| --- | --- |\n| Clear | " + short + " |",
			"Label | Detail\n--- | ---\nClear | " + short,
		} {
			r := lintOne(table)
			if rulesOf(r)["long-sentence"] != 0 || r.Words != 27 {
				t.Errorf("words=%d %+v", r.Words, r.Violations)
			}
		}
		terms = append(terms, "term25", "term26")
		long := strings.Join(terms[:26], " ") + "."
		r := lintOne("| Label | Detail |\n| --- | --- |\n| Clear | " + long + " |")
		ls := ofRule(r, "long-sentence")
		if len(ls) != 1 || ls[0].Match != "26 words" {
			t.Errorf("%+v", ls)
		}
	})
	t.Run("synonym rotation", func(t *testing.T) {
		r := lintOne("Check the config file. Then verify the output. Verify twice.")
		rot := ofRule(r, "synonym-rotation")
		if len(rot) != 1 || !strings.Contains(rot[0].Message, "'verify' and 'check'") {
			t.Errorf("%+v", rot)
		}
		if r := lintOne("Check the config. Check the output."); rulesOf(r)["synonym-rotation"] != 0 {
			t.Errorf("%+v", r.Violations)
		}
	})
	t.Run("file label", func(t *testing.T) {
		r := LintNamed("a; b", "x.md", Options{})
		if len(r.Violations) == 0 || r.Violations[0].File != "x.md" {
			t.Errorf("%+v", r.Violations)
		}
	})
}

func TestPassiveLookahead(t *testing.T) {
	// "is used for parsing" is rejected by the lookahead; the later match stays.
	r := lintOne("The log is used for parsing. The file was sent by hand. It is set to running.")
	var got []string
	for _, v := range ofRule(r, "passive-voice") {
		got = append(got, v.Match)
	}
	want := "was sent"
	if len(got) != 1 || got[0] != want {
		t.Errorf("got %q want [%q]", got, want)
	}
}

func TestUnicode(t *testing.T) {
	r := lintOne("Café; naïve ünï was removed.")
	if len(r.Violations) != 2 {
		t.Fatalf("%+v", r.Violations)
	}
	if r.Violations[0].Col != 5 {
		t.Errorf("semicolon col %d, want 5 (code points)", r.Violations[0].Col)
	}
	// A non-ASCII letter must continue a word: "éseamless" is not a keyword hit.
	if r := lintOne("It is éseamless and seamlessé."); rulesOf(r)["marketing-adjective"] != 0 {
		t.Errorf("%+v", r.Violations)
	}
	// Every Python splitlines separator starts a new line.
	r = lintOne("a\r\nb;\rc;\vd;\fe;\x1cf;\x1dg;\x1eh;\u0085i; j; k;")
	for i, v := range r.Violations {
		if v.Line != i+2 {
			t.Errorf("violation %d on line %d", i, v.Line)
		}
	}
	if len(r.Violations) != 10 {
		t.Errorf("got %d violations", len(r.Violations))
	}
}

func TestSplitLines(t *testing.T) {
	got := splitLines("a\n\nb\r\nc\r")
	if fmt.Sprint(got) != "[a  b c]" || len(got) != 4 {
		t.Errorf("%q", got)
	}
	if len(splitLines("")) != 0 || len(splitLines("\n")) != 1 {
		t.Error("empty handling")
	}
}

func TestOptions(t *testing.T) {
	text := "Use it; it was run."
	r := Lint(text, Options{Disable: []string{"passive-voice"}, Baseline: 1})
	if rulesOf(r)["passive-voice"] != 0 || r.Failed() {
		t.Errorf("%+v", r)
	}
	if r := Lint(text, Options{}); !r.Failed() || r.HardCount != 1 || r.Count != 2 {
		t.Errorf("%+v", r)
	}
}

func TestGolden(t *testing.T) {
	files, _ := filepath.Glob("testdata/golden/*.json")
	if len(files) == 0 {
		t.Fatal("no golden files")
	}
	for _, g := range files {
		name := strings.TrimSuffix(filepath.Base(g), ".json")
		t.Run(name, func(t *testing.T) {
			src := filepath.Join("testdata", name)
			data, err := os.ReadFile(src)
			if err != nil {
				t.Fatal(err)
			}
			want, _ := os.ReadFile(g)
			got, _ := LintNamed(string(data), src, Options{}).JSON()
			if string(got) != strings.TrimSuffix(string(want), "\n") {
				t.Errorf("JSON differs from upstream golden output\n got: %s\nwant: %s", got, want)
			}
		})
	}
}

func TestJSONEmpty(t *testing.T) {
	b, _ := lintOne("Clean text.").JSON()
	want := "{\n  \"violations\": [],\n  \"count\": 0,\n  \"hard_count\": 0,\n  \"baseline\": 0,\n  \"words\": 2,\n  \"per_100_words\": 0.0\n}"
	if string(b) != want {
		t.Errorf("%s", b)
	}
}

func TestLintProse(t *testing.T) {
	text := "See https://example.com/a;b?c=1 for more.\n<!-- hidden; comment\nstill; hidden -->\nVisible; text.\n```\ncode; <!-- x\n```\nAfter; fence."
	r := LintProse(text)
	var lines []int
	for _, v := range r.Violations {
		lines = append(lines, v.Line)
	}
	if fmt.Sprint(lines) != "[4 8]" {
		t.Errorf("lines %v: %+v", lines, r.Violations)
	}
	if up := lintOne(text); len(up.Violations) <= len(r.Violations) {
		t.Errorf("Lint should find more than LintProse")
	}
	// Inline code and fences are skipped by upstream itself.
	if r := LintProse("Use `a; b` here.\n"); len(r.Violations) != 0 {
		t.Errorf("%+v", r.Violations)
	}
	// A comment opener inside a code span is not a comment.
	if r := LintProse("Write `<!--` now; ok."); len(r.Violations) != 1 {
		t.Errorf("%+v", r.Violations)
	}
}
