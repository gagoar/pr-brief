package theme

import (
	"fmt"
	"strings"
)

// Style constants, from beautiful-mermaid src/styles.ts and src/renderer.ts.
const (
	FontFamily    = "Inter, Helvetica, Arial" // no hyphen: GitHub blanks a theme value that has one
	FontSize      = "13px"
	NodeStroke    = "0.75px"
	ThinStroke    = "1px"
	ThickStroke   = "2px"
	DashArray     = "4 4"
	NodeSpacing   = 28
	RankSpacing   = 48
	DiagramMargin = 40
)

// Edge kinds, in the order linkStyle lines are written.
const (
	EdgeExisting = "existing" // -->
	EdgeNew      = "new"      // ==>
	EdgeRemoved  = "removed"  // -.->
)

// InitLine is the single %%{init}%% line for the theme.
func (t Theme) InitLine() string {
	d := t.Derive()
	q := func(s string) string { return `"` + s + `"` }
	vars := []string{
		`"darkMode":` + fmt.Sprint(d.Dark),
		`"background":` + q(d.BG),
		`"fontFamily":` + q(FontFamily),
		`"fontSize":` + q(FontSize),
		`"dropShadow":"none"`,
		`"primaryColor":` + q(d.NodeFill),
		`"primaryTextColor":` + q(d.FG),
		`"primaryBorderColor":` + q(d.NodeStroke),
		`"nodeTextColor":` + q(d.FG),
		`"textColor":` + q(d.FG),
		`"mainBkg":` + q(d.NodeFill),
		`"nodeBorder":` + q(d.NodeStroke),
		`"lineColor":` + q(d.Line),
		`"clusterBkg":` + q(d.BG),
		`"clusterBorder":` + q(d.NodeStroke),
		`"titleColor":` + q(d.TextSec),
		`"edgeLabelBackground":` + q(d.BG),
	}
	return fmt.Sprintf(`%%%%{init: {"theme":"base","flowchart":{"curve":"step","nodeSpacing":%d,"rankSpacing":%d,"diagramPadding":%d},"themeVariables":{%s}}}%%%%`,
		NodeSpacing, RankSpacing, DiagramMargin, strings.Join(vars, ","))
}

// Defs returns the classDef, class and linkStyle lines (indented two spaces).
// kinds has one entry per real edge, in the order the edges are written;
// invisible layout links are not part of it.
func (t Theme) Defs(kinds []string) []string {
	d := t.Derive()
	tint := func(c string) string { return Mix(c, d.BG, MIX.KeyBadge) }
	node := func(name, fill, stroke, width, dash, text string) string {
		s := fmt.Sprintf("  classDef %s fill:%s,stroke:%s,stroke-width:%s,", name, fill, stroke, width)
		if dash != "" {
			s += "stroke-dasharray:" + dash + ","
		}
		return s + "color:" + text + ",font-weight:500"
	}
	lines := []string{
		node("default", d.NodeFill, d.NodeStroke, NodeStroke, "", d.FG),
		node("added", tint(d.Added), d.Added, ThinStroke, "", d.FG),
		node("modified", tint(d.Modified), d.Modified, ThinStroke, "", d.FG),
		node("removed", tint(d.Removed), d.Removed, ThinStroke, DashArray, d.FG),
		node("context", d.NodeFill, d.NodeStroke, NodeStroke, "", d.TextSec),
		node("risk", d.NodeFill, d.Removed, ThickStroke, "", d.FG),
		node("riskadded", tint(d.Added), d.Removed, ThickStroke, "", d.FG),
		fmt.Sprintf("  classDef zone fill:%s,stroke:%s,stroke-width:%s,color:%s,font-weight:600,font-size:12px", d.BG, d.NodeStroke, ThinStroke, d.TextSec),
		"  class IN,FN,OUT zone",
	}
	idx := map[string][]string{}
	for i, k := range kinds {
		idx[k] = append(idx[k], fmt.Sprint(i))
	}
	link := func(kind, width string) {
		if len(idx[kind]) > 0 {
			lines = append(lines, fmt.Sprintf("  linkStyle %s stroke:%s,stroke-width:%s", strings.Join(idx[kind], ","), d.Line, width))
		}
	}
	link(EdgeExisting, ThinStroke)
	link(EdgeNew, ThickStroke)
	link(EdgeRemoved, ThinStroke+",stroke-dasharray:"+DashArray)
	return lines
}
