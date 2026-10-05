package stelint

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// UpstreamSHA is the asd-ste100-skill commit this port tracks.
const upstreamSHA = "32511c6992ecb5f1971e46a2943f2e6adceedafe"

// upstreamScript returns the path of upstream's ste-lint.py. It uses
// STE_UPSTREAM_SCRIPT when set, otherwise downloads the pinned commit with gh.
func upstreamScript(t *testing.T) string {
	t.Helper()
	if p := os.Getenv("STE_UPSTREAM_SCRIPT"); p != "" {
		return p
	}
	if _, err := exec.LookPath("gh"); err != nil {
		t.Skip("gh not available and STE_UPSTREAM_SCRIPT unset")
	}
	out, err := exec.Command("gh", "api",
		"repos/danyuchn/asd-ste100-skill/contents/scripts/ste-lint.py?ref="+upstreamSHA,
		"--jq", ".content").Output()
	if err != nil {
		t.Skipf("cannot download upstream script: %v", err)
	}
	dec := exec.Command("base64", "-d")
	dec.Stdin = bytes.NewReader(out)
	src, err := dec.Output()
	if err != nil {
		t.Skipf("cannot decode upstream script: %v", err)
	}
	p := filepath.Join(t.TempDir(), "ste-lint.py")
	if err := os.WriteFile(p, src, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func runUpstream(t *testing.T, script string, stdin string, args ...string) string {
	t.Helper()
	cmd := exec.Command("python3", append([]string{script, "--json"}, args...)...)
	cmd.Env = append(os.Environ(), "PYTHONUTF8=1", "PYTHONIOENCODING=utf-8")
	if stdin != "" || len(args) == 0 {
		cmd.Stdin = strings.NewReader(stdin)
	}
	out, err := cmd.Output()
	var ee *exec.ExitError
	if err != nil && !errors.As(err, &ee) {
		t.Fatalf("python3: %v", err)
	}
	if ee != nil && ee.ExitCode() != 1 {
		t.Fatalf("upstream exit %d: %s", ee.ExitCode(), ee.Stderr)
	}
	return strings.TrimSuffix(string(out), "\n")
}

// TestParity compares this port's JSON byte for byte with upstream's output.
// It is skipped unless STE_PARITY=1 and python3 is installed, so a plain
// `go test ./...` needs no Python. See parity/README.md.
func TestParity(t *testing.T) {
	if os.Getenv("STE_PARITY") != "1" {
		t.Skip("set STE_PARITY=1 to compare against upstream ste-lint.py")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}
	script := upstreamScript(t)

	files, _ := filepath.Glob("testdata/*.md")
	more, _ := filepath.Glob("testdata/*.txt")
	files = append(files, more...)
	for _, f := range files {
		t.Run("file/"+filepath.Base(f), func(t *testing.T) {
			data, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			got, _ := LintNamed(string(data), f, Options{}).JSON()
			if want := runUpstream(t, script, "", f); string(got) != want {
				t.Errorf("mismatch\n got: %s\nwant: %s", got, want)
			}
		})
	}

	t.Run("options", func(t *testing.T) {
		data, _ := os.ReadFile("testdata/parity-extra.md")
		f := "testdata/parity-extra.md"
		got, _ := LintNamed(string(data), f, Options{Disable: []string{"passive-voice", "semicolon"}, Baseline: 5}).JSON()
		want := runUpstream(t, script, "", "--baseline", "5", "--disable", "passive-voice,semicolon", f)
		if string(got) != want {
			t.Errorf("mismatch\n got: %s\nwant: %s", got, want)
		}
	})

	snippets := []string{
		"", "a", "a; b", "x is used for parsing and was sent for running.",
		"It was done by hand. It was sent to running.", "The file is is set to making.",
		"may have run, has been had, have had gone", "He MIGHT NOT have run; she could’nt have.",
		"Ünïcode wörd was removed. é seamless", "éseamless seamlessé _seamless_ seamless_",
		"Check. verify this. confirm", "blazing fast blazing-fast blazing fast",
		"spin up spin up", "Perform an analysis of x. Perform an éanalysis.",
		"Ends with and ", "- a and \n- b or　", "word second; line third",
		"a | b\n--- | ---\nc \\| d | e;", "|a|b|\n|---|---|\n|x\\|y|z; q|",
		"KKeep ſtart the sſtop. İs ıs", "Long " + strings.Repeat("w ", 40) + "end. Short.",
		"Tab\there\ttext;\u0000nul", "\ufeffBOM; line", "emoji \U0001F600; \U0001F600 seamless",
		"> - quote and\n> - x or", "- a\n\n  and\n- b and\n  ```\n  x and\n  ```\n",
		"1) x\n   - y and\n2) z or",
	}
	for i, s := range snippets {
		t.Run(fmt.Sprintf("snippet/%d", i), func(t *testing.T) {
			got, _ := Lint(s, Options{}).JSON()
			if want := runUpstream(t, script, s); string(got) != want {
				t.Errorf("input %q\n got: %s\nwant: %s", s, got, want)
			}
		})
	}

	// Deterministic random corpus built from tricky tokens.
	toks := []string{"is", "are", "was", "were", "been", "being", "has", "have", "had", "may", "might",
		"could", "not", "n’t", "used", "removed", "run", "set", "gone", "begun", "to", "for", "by",
		"parsing", "running", "and", "or", ";", ".", "!", "?", "|", "`code`", "`", "```", "~~~",
		"-", "*", "1.", "2)", "perform", "an", "analysis", "carry out", "seamless", "robust", "check",
		"verify", "confirm", "use", "utilize", "spin up", "kick off", "é", "ü", "日本", " ", "　",
		"\t", "  ", "\n", "\n", "\r\n", "\r", "\v", " ", "\x1c", "\x1f", "\u0085", "|---|---|", "---",
		"a | b", "ſ", "K", "ı", "Straße", "x"}
	rng := rand.New(rand.NewSource(100))
	count := 300
	if v, err := strconv.Atoi(os.Getenv("STE_PARITY_N")); err == nil {
		count = v
	}
	for n := 0; n < count; n++ {
		var b strings.Builder
		for k := 0; k < 3+rng.Intn(40); k++ {
			b.WriteString(toks[rng.Intn(len(toks))])
			if rng.Intn(5) > 0 {
				b.WriteByte(' ')
			}
		}
		s := b.String()
		got, _ := Lint(s, Options{}).JSON()
		if want := runUpstream(t, script, s); string(got) != want {
			t.Fatalf("random input %d %q\n got: %s\nwant: %s", n, s, got, want)
		}
	}
}
