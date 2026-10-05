// Package diagram turns one flow (Input -> Functions -> Output) into the complete
// Mermaid source, with a theme applied. The gate recomputes the same text, so a
// diagram that did not come from this package fails the gate.
package diagram

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/gagoar/pr-brief/internal/convention"
)

// Port is an Input or an Output. Only the id appears in the diagram; the kind picks
// the shape of an Output node.
type Port struct {
	ID   string `json:"id"`
	Kind string `json:"kind,omitempty"`
}

// Node is a node in the Functions column.
type Node struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Status    string `json:"status"` // added | modified | removed | context
	RiskScore int    `json:"riskScore,omitempty"`
}

// Edge is a call or data flow. Status picks the arrow.
type Edge struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Status string `json:"status"` // new | existing | removed
}

// Flow is the input of Render. It is node-link shaped, and `pr-brief shape` emits it.
type Flow struct {
	Version int    `json:"version,omitempty"`
	Inputs  []Port `json:"inputs"`
	Nodes   []Node `json:"nodes"`
	Outputs []Port `json:"outputs"`
	Edges   []Edge `json:"edges"`
}

var (
	inputRe  = regexp.MustCompile(`^I\d+$`)
	outputRe = regexp.MustCompile(`^O\d+$`)
	nodeRe   = regexp.MustCompile(`^F\d+$`)
)

// Parse reads a flow file. Unknown keys are errors.
func Parse(data []byte) (Flow, error) {
	var f Flow
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return Flow{}, fmt.Errorf("invalid flow: %w", err)
	}
	if dec.More() {
		return Flow{}, errors.New("invalid flow: trailing data after the JSON value")
	}
	if f.Version != 0 && f.Version != 1 {
		return Flow{}, fmt.Errorf("invalid flow: version %d is not supported", f.Version)
	}
	return f, f.Validate()
}

// Validate checks ids, statuses, references and the limits of the convention.
func (f Flow) Validate() error {
	known := map[string]string{}
	add := func(id, kind string, re *regexp.Regexp, what string) error {
		if !re.MatchString(id) {
			return fmt.Errorf("invalid flow: %s id %q does not match %s", what, id, re)
		}
		if _, dup := known[id]; dup {
			return fmt.Errorf("invalid flow: id %q appears twice", id)
		}
		known[id] = kind
		return nil
	}
	if len(f.Inputs) == 0 || len(f.Outputs) == 0 || len(f.Nodes) == 0 {
		return errors.New("invalid flow: a flow needs at least one input, one node and one output")
	}
	for _, p := range f.Inputs {
		if err := add(p.ID, "in", inputRe, "input"); err != nil {
			return err
		}
	}
	for _, p := range f.Outputs {
		if err := add(p.ID, "out", outputRe, "output"); err != nil {
			return err
		}
	}
	for _, n := range f.Nodes {
		if err := add(n.ID, "fn", nodeRe, "node"); err != nil {
			return err
		}
		switch n.Status {
		case "added", "modified", "removed", "context":
		default:
			return fmt.Errorf("invalid flow: node %s has status %q (added, modified, removed or context)", n.ID, n.Status)
		}
		if utf8.RuneCountInString(Label(n)) > convention.MaxLabel {
			return fmt.Errorf("invalid flow: label of %s is %d characters; the limit is %d", n.ID, utf8.RuneCountInString(Label(n)), convention.MaxLabel)
		}
	}
	for _, e := range f.Edges {
		from, ok1 := known[e.From]
		to, ok2 := known[e.To]
		if !ok1 || !ok2 {
			return fmt.Errorf("invalid flow: edge %s -> %s names an unknown id", e.From, e.To)
		}
		if from == "out" || to == "in" {
			return fmt.Errorf("invalid flow: edge %s -> %s runs against Input -> Functions -> Output", e.From, e.To)
		}
		switch e.Status {
		case "new", "existing", "removed":
		default:
			return fmt.Errorf("invalid flow: edge %s -> %s has status %q (new, existing or removed)", e.From, e.To, e.Status)
		}
	}
	if n := len(f.Inputs) + len(f.Nodes) + len(f.Outputs); n > convention.MaxNodes {
		return fmt.Errorf("invalid flow: %d nodes; the limit is %d (collapse functions into modules first)", n, convention.MaxNodes)
	}
	if len(f.Edges) > convention.MaxEdges {
		return fmt.Errorf("invalid flow: %d edges; the limit is %d", len(f.Edges), convention.MaxEdges)
	}
	ctx := 0
	for _, n := range f.Nodes {
		if n.Status == "context" {
			ctx++
		}
	}
	if ctx > convention.MaxContext {
		return fmt.Errorf("invalid flow: %d context nodes; the limit is %d", ctx, convention.MaxContext)
	}
	return nil
}

// Risky reports whether a node gets the `!` prefix and a red border.
func (n Node) Risky() bool {
	return n.RiskScore >= convention.RiskClassThreshold && n.Status != "removed"
}

// Label is the node text as drawn: `!` for a risky node, quotes made safe.
func Label(n Node) string {
	l := strings.ReplaceAll(n.Label, `"`, `'`)
	if n.Risky() {
		return "!" + l
	}
	return l
}

// Class is the classDef name of a node.
func (n Node) Class() string {
	switch {
	case n.Risky() && n.Status == "added":
		return "riskadded"
	case n.Risky():
		return "risk"
	}
	return n.Status
}

// report is the part of `pr-brief shape` output that FromReport reads.
type report struct {
	Refs  map[string]struct{ Kind string } `json:"refs"`
	Flows []struct {
		Inputs    []string `json:"inputs"`
		Outputs   []string `json:"outputs"`
		Functions []struct {
			Node      string `json:"node"`
			Label     string `json:"label"`
			Status    string `json:"status"`
			RiskScore int    `json:"riskScore"`
		} `json:"functions"`
		Edges []Edge `json:"edges"`
	} `json:"flows"`
}

// FromReport converts flow n (1-based) of a `pr-brief shape` report.
func FromReport(data []byte, n int) (Flow, error) {
	var r report
	if err := json.Unmarshal(data, &r); err != nil {
		return Flow{}, fmt.Errorf("invalid report: %w", err)
	}
	if n < 1 || n > len(r.Flows) {
		return Flow{}, fmt.Errorf("the report has %d flow(s); flow %d does not exist", len(r.Flows), n)
	}
	rf := r.Flows[n-1]
	f := Flow{Version: 1, Edges: rf.Edges}
	for _, id := range rf.Inputs {
		f.Inputs = append(f.Inputs, Port{ID: id, Kind: r.Refs[id].Kind})
	}
	for _, id := range rf.Outputs {
		f.Outputs = append(f.Outputs, Port{ID: id, Kind: r.Refs[id].Kind})
	}
	for _, fn := range rf.Functions {
		f.Nodes = append(f.Nodes, Node{ID: fn.Node, Label: fn.Label, Status: fn.Status, RiskScore: fn.RiskScore})
	}
	return f, f.Validate()
}
