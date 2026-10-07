package gate

import (
	"regexp"
	"strings"
	"testing"

	"github.com/gagoar/pr-brief/internal/theme"
)

// bodyWith returns the good body with its diagram drawn in th and the marker naming markerTheme.
func bodyWith(t *testing.T, th theme.Theme, markerTheme string) string {
	t.Helper()
	b := strings.Replace(goodTemplate, "@@DIAGRAM@@", render(th), 1)
	return strings.Replace(b, "theme=github-dark", "theme="+markerTheme, 1)
}

func dropLines(body, contains string) string {
	var out []string
	for _, l := range strings.Split(body, "\n") {
		if !strings.Contains(l, contains) {
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n")
}

func TestEveryBuiltinThemePasses(t *testing.T) {
	for _, n := range theme.Names() {
		th, _ := theme.Get(n)
		if r := Check(bodyWith(t, th, n), Options{}); !r.OK() {
			t.Errorf("%s: %+v", n, r.Findings)
		} else if r.Theme != n {
			t.Errorf("%s: result theme = %q", n, r.Theme)
		}
	}
}

func TestTamperedStyleFails(t *testing.T) {
	cases := map[string]func(string) string{
		"one classDef property changed": func(b string) string {
			return regexp.MustCompile(`(classDef risk [^\n]*stroke-width:)2px`).ReplaceAllString(b, "${1}1px")
		},
		"a classDef line missing":     func(b string) string { return dropLines(b, "classDef context ") },
		"the ranker links deleted":    func(b string) string { return dropLines(b, "~~~") },
		"the ranker points elsewhere": func(b string) string { return strings.Replace(b, "F2 ~~~ O1", "F3 ~~~ O1", 1) },
		"a linkStyle index dropped":   func(b string) string { return strings.Replace(b, "linkStyle 0,3,4 ", "linkStyle 0,3 ", 1) },
		"the init line missing":       func(b string) string { return dropLines(b, "%%{init") },
		"an extra style line": func(b string) string {
			return strings.Replace(b, "  class IN,FN,OUT zone", "  class IN,FN,OUT zone\n  style F1 fill:#ffffff", 1)
		},
		"another theme's colours": func(b string) string {
			d, _ := theme.Get("dracula")
			return strings.Replace(b, render(theme.Default()), render(d), 1)
		},
		"the font changed": func(b string) string { return strings.Replace(b, "Inter, Helvetica, Arial", "Comic Sans MS", 1) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			r := Check(mutate(good), Options{})
			found := false
			for _, f := range r.Findings {
				if f.Rule == "style" || f.Rule == "diagram" {
					found = true
				}
			}
			if !found {
				t.Errorf("expected a style or diagram finding; got %+v", r.Findings)
			}
		})
	}
}

func TestStyleMessageSaysWhatToDo(t *testing.T) {
	r := Check(dropLines(good, "classDef context "), Options{})
	has(t, r, "style", "does not match theme github-dark")
	has(t, r, "style", "expected:")
	has(t, r, "style", "pr-brief diagram")
}

func TestMarkerThemeRules(t *testing.T) {
	noTheme := strings.Replace(good, " theme=github-dark", "", 1)
	has(t, Check(noTheme, Options{}), "markers", "has no theme")

	custom := strings.Replace(good, "theme=github-dark", "theme=custom:ab12cd34", 1)
	has(t, Check(custom, Options{}), "markers", "not built in")

	unknown := strings.Replace(good, "theme=github-dark", "theme=solarized", 1)
	has(t, Check(unknown, Options{}), "markers", "not built in")
}

func TestConfiguredThemeBeatsTheMarker(t *testing.T) {
	d, _ := theme.Get("dracula")
	// The author drew github-dark and says so in the marker, but the repo requires dracula.
	r := Check(good, Options{Theme: &d})
	has(t, r, "style", "theme dracula")
	// The author drew dracula but wrote github-dark in the marker: the repo's theme is what counts.
	if r := Check(bodyWith(t, d, "github-dark"), Options{Theme: &d}); !r.OK() {
		t.Errorf("the configured theme should win over the marker: %+v", r.Findings)
	}
	if r := Check(bodyWith(t, d, "dracula"), Options{}); !r.OK() {
		t.Errorf("a built-in theme named in the marker is enough without a config: %+v", r.Findings)
	}
}

func TestCustomTheme(t *testing.T) {
	custom, err := theme.Parse([]byte(`{"bg":"#10141c","fg":"#e4ecf7","accent":"#ff8a3d"}`), "mine")
	if err != nil {
		t.Fatal(err)
	}
	body := bodyWith(t, custom, custom.ID())
	if r := Check(body, Options{Theme: &custom}); !r.OK() {
		t.Errorf("a custom theme from the repo config must pass: %+v", r.Findings)
	}
	has(t, Check(body, Options{}), "markers", "not built in")
	// Same diagram, a different custom theme in the config.
	other, _ := theme.Parse([]byte(`{"bg":"#10141c","fg":"#e4ecf7","line":"#3dd6ff"}`), "mine")
	has(t, Check(body, Options{Theme: &other}), "style", "does not match")
}

func TestExtraDiagramsAreNotStyleChecked(t *testing.T) {
	extra := "### Extra: states\n```mermaid\nstateDiagram-v2\n  [*] --> Open\n  Open --> Closed\n```\n\ngreen added"
	body := strings.Replace(good, "green added", extra, 1)
	if r := Check(body, Options{}); !r.OK() {
		t.Errorf("a state diagram is not themed and must not fail the style check: %+v", r.Findings)
	}
}

func TestInvisibleLinksDoNotCountAsEdges(t *testing.T) {
	// good has 5 visible edges and 2 layout links; the limit is on visible edges only.
	if strings.Count(good, "~~~") != 2 {
		t.Fatalf("fixture changed: %d invisible links", strings.Count(good, "~~~"))
	}
	if r := Check(good, Options{}); !r.OK() {
		t.Errorf("%+v", r.Findings)
	}
}
