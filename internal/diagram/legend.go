package diagram

import "fmt"

// LegendStyle gives the Legend box the same look as the other three boxes.
const LegendStyle = "  class LEG zone"

// legendOrder is the order the legend lists classes in. Every class the nodes of a
// diagram use gets one sample node, so a reader never has to guess what a colour means.
var legendOrder = []string{"added", "modified", "removed", "context", "risk", "riskadded"}

var legendText = map[string]string{
	"added":     "added",
	"modified":  "modified",
	"removed":   "removed",
	"context":   "unchanged context",
	"risk":      "! risk",
	"riskadded": "! risk, added",
}

// LegendEntry is one sample node in the legend.
type LegendEntry struct{ ID, Text, Class string }

// LegendSet is the legend of one diagram.
type LegendSet []LegendEntry

// Legend builds the legend for the node classes a diagram uses. Unknown and empty
// classes are ignored. The result is empty when no class is in use.
func Legend(classes []string) LegendSet {
	used := map[string]bool{}
	for _, c := range classes {
		used[c] = true
	}
	var set LegendSet
	for _, c := range legendOrder {
		if used[c] {
			set = append(set, LegendEntry{ID: fmt.Sprintf("L%d", len(set)+1), Text: legendText[c], Class: c})
		}
	}
	return set
}

// Lines is the mermaid source of the legend: one subgraph and its sample nodes. The nodes
// have no links. Mermaid then lays them out in a row, and links would stack them in a column.
func (s LegendSet) Lines() []string {
	if len(s) == 0 {
		return nil
	}
	lines := []string{`  subgraph LEG["Legend"]`}
	for _, e := range s {
		lines = append(lines, fmt.Sprintf(`    %s["%s"]:::%s`, e.ID, e.Text, e.Class))
	}
	return append(lines, "  end")
}
