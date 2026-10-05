package gate

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/gagoar/pr-brief/internal/convention"
)

// Diagram kinds the convention accepts.
const (
	kindFlow     = "graph"
	kindState    = "stateDiagram-v2"
	kindER       = "erDiagram"
	kindSequence = "sequenceDiagram"
)

var (
	idRe        = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	pieceRe     = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)(?::::([A-Za-z_][A-Za-z0-9_]*))?$`)
	inputRefRe  = regexp.MustCompile(`^I\d+$`)
	outputRefRe = regexp.MustCompile(`^O\d+$`)
	subgraphRe  = regexp.MustCompile(`^subgraph\s+([A-Za-z_][A-Za-z0-9_]*)(?:\s*\[\s*"?([^"\]]*?)"?\s*\])?\s*$`)
	classStmtRe = regexp.MustCompile(`^class\s+(\S+)\s+([A-Za-z_][A-Za-z0-9_]*)\s*$`)
	// id, open token, quoted label, close token, optional :::class
	declRe   = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*)(\(\[|\[\(|\[|>)"([^"]*)"(\]\)|\)\]|\])(?::::([A-Za-z_][A-Za-z0-9_]*))?`)
	arrowRe  = regexp.MustCompile(`\s*(==>|-\.->|-->)(?:\|[^|]*\|)?\s*`)
	anyArrow = regexp.MustCompile(`==>|-\.->|-->|---|===|-\.-|--[ox>]`)
)

type node struct {
	id, label, open, column, class string
}

type edge struct{ from, to string }

type diagram struct {
	kind     string
	nodes    map[string]*node
	order    []string
	edges    []edge
	problems []string
	inputs   []string // labels, I<n>
	outputs  []string // labels, O<n>
}

func (d *diagram) problem(format string, a ...any) {
	d.problems = append(d.problems, fmt.Sprintf(format, a...))
}

// parseDiagram reads one mermaid source and records every convention problem.
func parseDiagram(src string) *diagram {
	d := &diagram{nodes: map[string]*node{}}
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")

	header := ""
	start := 0
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if t == "" || strings.HasPrefix(t, "%%") {
			continue
		}
		header, start = t, i+1
		break
	}

	switch {
	case header == "graph LR":
		d.kind = kindFlow
	case header == kindState || header == kindER || header == kindSequence:
		d.kind = header
	case strings.HasPrefix(header, "flowchart"):
		d.problem("uses the `flowchart` keyword; Azure DevOps rejects it. Write `graph LR`")
		return d
	case strings.HasPrefix(header, "graph"):
		d.problem("%q is not allowed; Input -> Functions -> Output diagrams are `graph LR`", header)
		return d
	default:
		d.problem("unsupported diagram type %q (allowed: graph LR, stateDiagram-v2, erDiagram, sequenceDiagram)", header)
		return d
	}

	// Rules for every diagram kind.
	for _, l := range lines[start:] {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "click ") || t == "click" {
			d.problem("uses `click`; GitHub blocks it")
		}
		if strings.Contains(t, "href") || strings.Contains(t, "<a ") {
			d.problem("contains a link; links are blocked inside diagrams")
		}
	}
	if d.kind != kindFlow {
		return d
	}
	d.parseFlow(lines[start:])
	return d
}

func (d *diagram) parseFlow(lines []string) {
	var subLabels []string
	current := "" // label of the open subgraph
	open := false
	classOf := map[string]string{}

	for _, raw := range lines {
		t := strings.TrimSpace(raw)
		switch {
		case t == "" || strings.HasPrefix(t, "%%"):
			continue
		case strings.HasPrefix(t, "classDef "), strings.HasPrefix(t, "linkStyle "),
			strings.HasPrefix(t, "style "), strings.HasPrefix(t, "direction "):
			continue
		}
		if m := classStmtRe.FindStringSubmatch(t); m != nil {
			for _, id := range strings.Split(m[1], ",") {
				classOf[strings.TrimSpace(id)] = m[2]
			}
			continue
		}
		if m := subgraphRe.FindStringSubmatch(t); m != nil {
			if open {
				d.problem("nested subgraph %q; use exactly Input, Functions, Output", m[1])
			}
			label := m[2]
			if label == "" {
				label = m[1]
			}
			subLabels = append(subLabels, label)
			current, open = label, true
			continue
		}
		if t == "end" {
			open, current = false, ""
			continue
		}

		// Replace each declaration by its bare id, recording the node.
		line := declRe.ReplaceAllStringFunc(t, func(s string) string {
			m := declRe.FindStringSubmatch(s)
			id, openTok, label, cls := m[1], m[2], m[3], m[5]
			if _, dup := d.nodes[id]; dup {
				d.problem("node %q is declared twice", id)
			} else {
				d.nodes[id] = &node{id: id, label: label, open: openTok, column: current, class: cls}
				d.order = append(d.order, id)
				if !open {
					d.problem("node %q is declared outside Input, Functions and Output", id)
				}
			}
			return id
		})

		if !arrowRe.MatchString(line) {
			if anyArrow.MatchString(line) {
				d.problem("unsupported edge syntax in %q (use ==>, --> or -.->)", t)
				continue
			}
			if m := pieceRe.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
				if m[2] != "" {
					classOf[m[1]] = m[2]
				}
				continue
			}
			d.problem("cannot read line %q (quote every label; one declaration or one edge chain per line)", t)
			continue
		}

		parts := arrowRe.Split(line, -1)
		var ids []string
		for _, p := range parts {
			p = strings.TrimSpace(p)
			m := pieceRe.FindStringSubmatch(p)
			if m == nil {
				d.problem("cannot read edge endpoint %q in %q (use one plain node id per endpoint)", p, t)
				ids = nil
				break
			}
			if m[2] != "" {
				classOf[m[1]] = m[2]
			}
			ids = append(ids, m[1])
		}
		for i := 0; i+1 < len(ids); i++ {
			d.edges = append(d.edges, edge{ids[i], ids[i+1]})
		}
	}
	if open {
		d.problem("subgraph %q is never closed with `end`", current)
	}

	for id, c := range classOf {
		if n, ok := d.nodes[id]; ok && n.class == "" {
			n.class = c
		}
	}

	want := []string{"Input", "Functions", "Output"}
	if strings.Join(subLabels, ",") != strings.Join(want, ",") {
		d.problem("subgraphs must be exactly Input, Functions, Output in that order (found: %s)", orNone(subLabels))
	}

	for _, id := range d.order {
		n := d.nodes[id]
		if utf8.RuneCountInString(n.label) > convention.MaxLabel {
			d.problem("label of %s is %d characters; the limit is %d", id, utf8.RuneCountInString(n.label), convention.MaxLabel)
		}
		switch n.column {
		case "Input":
			if !inputRefRe.MatchString(n.label) {
				d.problem("Input node %s is labelled %q; Input nodes show only a reference such as I1 (describe it in the References table)", id, n.label)
			} else {
				d.inputs = append(d.inputs, n.label)
			}
		case "Output":
			if !outputRefRe.MatchString(n.label) {
				d.problem("Output node %s is labelled %q; Output nodes show only a reference such as O1 (describe it in the References table)", id, n.label)
			} else {
				d.outputs = append(d.outputs, n.label)
			}
		}
	}

	if len(d.inputs) == 0 {
		d.problem("no Input node; every flow starts at an Input (I1, I2, ...)")
	}
	if len(d.outputs) == 0 {
		d.problem("no Output node; every flow ends at an Output (O1, O2, ...)")
	}

	context := 0
	functions := 0
	for _, id := range d.order {
		n := d.nodes[id]
		if n.class == "context" {
			context++
		}
		if n.column == "Functions" {
			functions++
		}
	}
	if functions == 0 {
		d.problem("no Functions node")
	}
	if len(d.nodes) > convention.MaxNodes {
		d.problem("%d nodes; the limit is %d (collapse functions into modules or split the flow)", len(d.nodes), convention.MaxNodes)
	}
	if len(d.edges) > convention.MaxEdges {
		d.problem("%d edges; the limit is %d", len(d.edges), convention.MaxEdges)
	}
	if context > convention.MaxContext {
		d.problem("%d context nodes; the limit is %d", context, convention.MaxContext)
	}

	for _, e := range d.edges {
		from, okf := d.nodes[e.from]
		to, okt := d.nodes[e.to]
		if !okf {
			d.problem("edge starts at undeclared node %q", e.from)
			continue
		}
		if !okt {
			d.problem("edge ends at undeclared node %q", e.to)
			continue
		}
		if from.column == "Output" {
			d.problem("edge %s -> %s leaves an Output; edges only run Input -> Functions -> Output", e.from, e.to)
		} else if to.column == "Input" {
			d.problem("edge %s -> %s enters an Input; edges only run Input -> Functions -> Output", e.from, e.to)
		}
	}
}

func orNone(s []string) string {
	if len(s) == 0 {
		return "none"
	}
	return strings.Join(s, ", ")
}
