package theme

import (
	"strings"
	"testing"
)

func TestMixMatchesHandComputation(t *testing.T) {
	// github-light: fg #1f2328 over #ffffff.
	d := Default().Derive()
	if d.BG != "#0d1117" {
		t.Fatalf("default theme is %s", d.BG)
	}
	gl, _ := Get("github-light")
	g := gl.Derive()
	if g.NodeFill != "#f8f8f9" || g.NodeStroke != "#d2d3d4" {
		t.Errorf("github-light node fill/stroke = %s/%s", g.NodeFill, g.NodeStroke)
	}
	if got := Mix("#000000", "#ffffff", 50); got != "#808080" {
		t.Errorf("50%% of black over white = %s (round half to even: 127.5 -> 128)", got)
	}
}

func TestContrast(t *testing.T) {
	if r := Contrast("#000000", "#ffffff"); r < 20.9 || r > 21.1 {
		t.Errorf("black on white = %.2f", r)
	}
	if r := Contrast("#777777", "#777777"); r != 1 {
		t.Errorf("same colour = %.2f", r)
	}
}

func TestBuiltinThemesAreReadable(t *testing.T) {
	for _, n := range Names() {
		th, err := Get(n)
		if err != nil {
			t.Fatal(err)
		}
		if err := th.checkContrast(); err != nil {
			t.Errorf("%s: %v", n, err)
		}
		if th.Status == nil {
			t.Errorf("%s has no status colours", n)
		}
	}
	if Names()[0] != "github-dark" {
		t.Errorf("default must come first: %v", Names())
	}
	if _, err := Get("nope"); err == nil || !strings.Contains(err.Error(), "github-dark") {
		t.Errorf("unknown theme error should list the built-ins: %v", err)
	}
}

func TestParseCustomTheme(t *testing.T) {
	th, err := Parse([]byte(`{"bg":"#FFF","fg":"#111","accent":"#ABCDEF"}`), "mine")
	if err != nil {
		t.Fatal(err)
	}
	if th.BG != "#ffffff" || th.FG != "#111111" || th.Accent != "#abcdef" {
		t.Errorf("not normalised: %+v", th.Colors)
	}
	d := th.Derive()
	if d.Dark || d.Added != "#1a7f37" {
		t.Errorf("a light custom theme gets the light status colours, got dark=%v added=%s", d.Dark, d.Added)
	}
	dark, _ := Parse([]byte(`{"bg":"#101010","fg":"#eeeeee"}`), "d")
	if dd := dark.Derive(); !dd.Dark || dd.Added != "#3fb950" {
		t.Errorf("a dark custom theme gets the dark status colours: %+v", dd)
	}
	withStatus, err := Parse([]byte(`{"bg":"#ffffff","fg":"#000000","status":{"added":"#0a0","modified":"#a60","removed":"#a00"}}`), "s")
	if err != nil || withStatus.Derive().Added != "#00aa00" {
		t.Errorf("status override: %v %+v", err, withStatus.Status)
	}
}

func TestParseRejects(t *testing.T) {
	cases := map[string]string{
		"unknown key":     `{"bg":"#fff","fg":"#000","font":"Inter"}`,
		"missing bg":      `{"fg":"#000000"}`,
		"named colour":    `{"bg":"white","fg":"#000000"}`,
		"rgb() colour":    `{"bg":"#ffffff","fg":"rgb(0,0,0)"}`,
		"low contrast":    `{"bg":"#888888","fg":"#777777"}`,
		"partial status":  `{"bg":"#ffffff","fg":"#000000","status":{"added":"#0a0"}}`,
		"bad status":      `{"bg":"#ffffff","fg":"#000000","status":{"added":"green","modified":"#a60","removed":"#a00"}}`,
		"trailing data":   `{"bg":"#ffffff","fg":"#000000"} {}`,
		"not json":        `bg: white`,
		"empty":           ``,
		"fg on node fill": `{"bg":"#ffffff","fg":"#000000","surface":"#111111"}`,
	}
	for name, body := range cases {
		if _, err := Parse([]byte(body), "x"); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
	if _, err := Parse([]byte(strings.Repeat(" ", MaxFileBytes+1)), "x"); err == nil {
		t.Error("an oversize file must be rejected")
	}
}

func TestContrastErrorSaysTheNumbers(t *testing.T) {
	_, err := Parse([]byte(`{"bg":"#888888","fg":"#777777"}`), "x")
	if err == nil || !strings.Contains(err.Error(), "contrast") || !strings.Contains(err.Error(), "4.5") {
		t.Errorf("error should explain the contrast: %v", err)
	}
}

func TestID(t *testing.T) {
	if gd, _ := Get("dracula"); gd.ID() != "dracula" {
		t.Errorf("built-in id = %s", gd.ID())
	}
	a, _ := Parse([]byte(`{"bg":"#ffffff","fg":"#000000"}`), "a")
	b, _ := Parse([]byte(`{ "fg": "#000", "bg": "#FFF" }`), "other-name")
	c, _ := Parse([]byte(`{"bg":"#fffffe","fg":"#000000"}`), "a")
	if a.ID() != b.ID() {
		t.Errorf("same colours, different spelling or file name must give one id: %s %s", a.ID(), b.ID())
	}
	if a.ID() == c.ID() || !strings.HasPrefix(a.ID(), "custom:") || len(a.ID()) != len("custom:")+8 {
		t.Errorf("custom ids: %s %s", a.ID(), c.ID())
	}
}

func TestInitLineAndDefs(t *testing.T) {
	th := Default()
	init := th.InitLine()
	for _, want := range []string{`"darkMode":true`, `"background":"#0d1117"`, `"fontFamily":"Inter, Helvetica, Arial"`, `"dropShadow":"none"`, `"curve":"step"`} {
		if !strings.Contains(init, want) {
			t.Errorf("init line lacks %s:\n%s", want, init)
		}
	}
	if strings.Contains(init, "-") && strings.Contains(strings.SplitN(init, `"fontFamily"`, 2)[1][:40], "-") {
		t.Errorf("the font value must have no hyphen (GitHub blanks it): %s", init)
	}
	defs := strings.Join(th.Defs([]string{"existing", "new", "new", "removed", "existing"}), "\n")
	for _, want := range []string{"classDef default ", "classDef zone ", "class IN,FN,OUT zone",
		"linkStyle 0,4 ", "linkStyle 1,2 ", "linkStyle 3 stroke:#3d444d,stroke-width:1px,stroke-dasharray:4 4"} {
		if !strings.Contains(defs, want) {
			t.Errorf("defs lack %q:\n%s", want, defs)
		}
	}
	if strings.Contains(strings.Join(th.Defs(nil), "\n"), "linkStyle") {
		t.Error("no edges, no linkStyle")
	}
}

// accent is part of beautiful-mermaid's format, where it colours arrowheads. Mermaid
// colours arrowheads like the edge, so accent is accepted but cannot change a diagram.
// This test pins that down so the README stays true.
func TestAccentHasNoEffectOnTheStyle(t *testing.T) {
	a, _ := Parse([]byte(`{"bg":"#10141c","fg":"#e4ecf7","accent":"#ff8a3d"}`), "a")
	b, _ := Parse([]byte(`{"bg":"#10141c","fg":"#e4ecf7","accent":"#3dd6ff"}`), "b")
	kinds := []string{"new", "existing"}
	if a.InitLine() != b.InitLine() || strings.Join(a.Defs(kinds), "\n") != strings.Join(b.Defs(kinds), "\n") {
		t.Error("accent must not change the emitted style")
	}
	if a.ID() == b.ID() {
		t.Error("but the two files are still different themes")
	}
}
