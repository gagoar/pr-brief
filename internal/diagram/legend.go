package diagram

import "strings"

// legendOrder is the order the legend lists colours in.
var legendOrder = []string{"added", "modified", "removed", "context", "risk"}

var legendText = map[string]string{
	"added":    "\U0001F7E9 added",
	"modified": "\U0001F7E7 modified",
	"removed":  "\U0001F7E5 removed (dashed)",
	"context":  "\u2B1C unchanged context",
	"risk":     "\U0001F534 risk (! and a thick red border)",
}

// LegendLine is the line that goes directly under a chart. It names each colour the nodes
// use, in one row, so a reader never has to guess what a colour means. It sits outside the
// Mermaid source: a legend drawn inside the chart is laid out by the renderer, and GitHub
// and the local renderer disagree on where it goes. A line of text is the same everywhere.
// classes holds Class() of each Functions node. It returns "" when no class is in use.
func LegendLine(classes []string) string {
	used := map[string]bool{}
	for _, c := range classes {
		switch c {
		case "riskadded":
			used["added"], used["risk"] = true, true
		case "added", "modified", "removed", "context", "risk":
			used[c] = true
		}
	}
	var parts []string
	for _, c := range legendOrder {
		if used[c] {
			parts = append(parts, legendText[c])
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "**Legend:** " + strings.Join(parts, " \u00b7 ")
}

// FlowLegend is the legend line for a flow.
func FlowLegend(f Flow) string {
	var classes []string
	for _, n := range f.Nodes {
		classes = append(classes, n.Class())
	}
	return LegendLine(classes)
}
