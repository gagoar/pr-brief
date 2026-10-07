package shape

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/gagoar/pr-brief/internal/convention"
)

// ---- Output types

// RefOut is an Input or Output the diagram refers to by id.
type RefOut struct {
	Kind string `json:"kind"`
	What string `json:"what"`
	File string `json:"file,omitempty"`
	Line int    `json:"line,omitempty"`
}

// FuncOut is a node in the Functions column.
type FuncOut struct {
	Node      string   `json:"node"` // short diagram id: F1, F2, ...
	ID        string   `json:"id"`
	Label     string   `json:"label"`
	Status    string   `json:"status"` // added | modified | removed | context
	Module    string   `json:"module,omitempty"`
	File      string   `json:"file,omitempty"`
	Line      int      `json:"line,omitempty"`
	Risk      []string `json:"risk,omitempty"`
	RiskScore int      `json:"riskScore,omitempty"`
	Members   []string `json:"members,omitempty"` // collapsed functions
}

// EdgeOut is a diagram edge.
type EdgeOut struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Status string `json:"status"` // new | existing | removed
}

// FlowOut is one Input -> Functions -> Output diagram.
type FlowOut struct {
	Inputs    []string  `json:"inputs"`
	Functions []FuncOut `json:"functions"`
	Outputs   []string  `json:"outputs"`
	Edges     []EdgeOut `json:"edges"`
	Collapsed string    `json:"collapsedLevel"` // function | module | more
	Warnings  []string  `json:"warnings,omitempty"`
	Score     int       `json:"score"`

	// internal: candidates before ids are assigned
	inCands  []RefCand
	outCands []RefCand
	outEdges map[string][]int // function id -> indexes into outCands
	inEdges  map[string][]int // function id -> indexes into inCands
}

type graph struct {
	byName map[string][]*Func
	all    []*Func
	out    map[*Func][]*Func
	in     map[*Func][]*Func
}

func dirOf(f *Func) string { return path.Dir(f.File) }

func buildGraph(index []*Func) *graph {
	g := &graph{byName: map[string][]*Func{}, out: map[*Func][]*Func{}, in: map[*Func][]*Func{}}
	for _, f := range index {
		if f.Kind == "func" {
			g.byName[f.Name] = append(g.byName[f.Name], f)
			g.all = append(g.all, f)
		}
	}
	for _, f := range g.all {
		seen := map[*Func]bool{}
		for _, name := range f.Calls {
			t := g.resolve(f, name)
			if t == nil || t == f || seen[t] {
				continue
			}
			seen[t] = true
			g.out[f] = append(g.out[f], t)
			g.in[t] = append(g.in[t], f)
		}
	}
	return g
}

// resolve picks the callee by closeness: same class and file, same file, same
// directory, anywhere. An ambiguous bucket resolves to nothing.
func (g *graph) resolve(from *Func, call string) *Func {
	qual, name := "", call
	if i := strings.LastIndex(call, "."); i >= 0 {
		qual, name = call[:i], call[i+1:]
	}
	cands := g.byName[name]
	if len(cands) == 0 {
		return nil
	}
	if q := strings.ToLower(strings.TrimLeft(qual, "_")); q != "" && len(cands) > 1 {
		// A receiver such as C0 or inviteService points at the class that owns the method.
		var match []*Func
		for _, c := range cands {
			cl := strings.ToLower(c.Class)
			if cl == q || (len(q) >= 3 && strings.Contains(cl, q)) {
				match = append(match, c)
			}
		}
		if len(match) > 0 {
			cands = match
		}
	}
	buckets := [][]*Func{nil, nil, nil, cands}
	for _, c := range cands {
		switch {
		case c.File == from.File && c.Class == from.Class:
			buckets[0] = append(buckets[0], c)
		case c.File == from.File:
			buckets[1] = append(buckets[1], c)
		case dirOf(c) == dirOf(from):
			buckets[2] = append(buckets[2], c)
		}
	}
	for _, b := range buckets {
		if len(b) == 1 {
			return b[0]
		}
		if len(b) > 1 {
			return nil
		}
	}
	return nil
}

// bindRegistrations gives each handler the Input of the route that names it and
// returns the index without registration pseudo-functions.
func bindRegistrations(index []*Func) []*Func {
	g := &graph{byName: map[string][]*Func{}}
	for _, f := range index {
		if f.Kind == "func" {
			g.byName[f.Name] = append(g.byName[f.Name], f)
		}
	}
	var out []*Func
	for _, f := range index {
		if f.Kind != "registration" {
			out = append(out, f)
			continue
		}
		if len(f.Calls) == 0 {
			continue
		}
		if h := g.resolve(f, f.Calls[0]); h != nil && h.Entry == nil {
			e := *f.Entry
			h.Entry = &e
		}
	}
	return out
}

// bindScripts gives the main function of a script the Input of the workflow step
// that runs it. Without it a script started only from a workflow has no entry,
// so its changed functions end up unplaced.
func bindScripts(index []*Func) {
	mains := map[string]*Func{}
	for _, f := range index {
		if f.Kind == "func" && f.Name == "main" && f.Class == "" && f.Entry == nil {
			mains[f.File] = f
		}
	}
	for _, j := range index {
		for _, script := range j.Runs {
			if m := mains[script]; m != nil {
				m.Entry = &RefCand{Kind: "cli", What: "workflow step runs " + script, File: j.File, Line: j.Start}
			}
		}
	}
}

// ---- Flow construction for application code

func isChanged(f *Func) bool { return f.Status != "" }

func (g *graph) buildFlows(maxDepth int) []*FlowOut {
	var flows []*FlowOut
	var entries []*Func
	for _, f := range g.all {
		if f.Entry != nil {
			entries = append(entries, f)
		}
	}
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].ID() < entries[j].ID() })

	for _, e := range entries {
		// Breadth-first search with predecessors, so each reachable function has one path.
		pred := map[*Func]*Func{e: nil}
		depth := map[*Func]int{e: 0}
		queue := []*Func{e}
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			if depth[cur] >= maxDepth {
				continue
			}
			for _, nx := range g.out[cur] {
				if _, ok := pred[nx]; !ok {
					pred[nx] = cur
					depth[nx] = depth[cur] + 1
					queue = append(queue, nx)
				}
			}
		}
		var changed []*Func
		for f := range pred {
			if isChanged(f) {
				changed = append(changed, f)
			}
		}
		if len(changed) == 0 {
			continue
		}
		sort.Slice(changed, func(i, j int) bool { return changed[i].ID() < changed[j].ID() })

		// Show changed functions, plus the unchanged neighbour next to each on its path.
		show := map[*Func]bool{}
		order := []*Func{}
		add := func(f *Func) {
			if !show[f] {
				show[f] = true
				order = append(order, f)
			}
		}
		var contexts []*Func
		for _, c := range changed {
			var p []*Func
			for n := c; n != nil; n = pred[n] {
				p = append([]*Func{n}, p...)
			}
			for i, n := range p {
				switch {
				case isChanged(n):
					add(n)
				case (i > 0 && isChanged(p[i-1])) || (i+1 < len(p) && isChanged(p[i+1])):
					contexts = append(contexts, n)
				}
			}
		}
		for _, c := range contexts {
			count := 0
			for _, s := range order {
				if !isChanged(s) {
					count++
				}
			}
			if count < convention.MaxContext {
				add(c)
			}
		}
		flows = append(flows, g.flowFor(e, pred, order, show))
	}
	return flows
}

// flowFor makes the edges between displayed functions. When a displayed function
// reaches another only through hidden ones, the hidden ones are skipped.
func (g *graph) flowFor(entry *Func, pred map[*Func]*Func, order []*Func, show map[*Func]bool) *FlowOut {
	fl := &FlowOut{Collapsed: "function", outEdges: map[string][]int{}, inEdges: map[string][]int{}}
	fl.inCands = []RefCand{*entry.Entry}

	// first displayed node on the path from the entry
	first := entry
	if !show[first] {
		best := -1
		for _, f := range order {
			d := 0
			for n := f; n != nil && n != entry; n = pred[n] {
				d++
			}
			if best < 0 || d < best {
				best, first = d, f
			}
		}
	}

	seen := map[[2]string]bool{}
	addEdge := func(a, b *Func) {
		k := [2]string{a.ID(), b.ID()}
		if a == b || seen[k] {
			return
		}
		seen[k] = true
		st := "existing"
		if a.Status == "A" || b.Status == "A" {
			st = "new"
		}
		if a.Status == "D" || b.Status == "D" {
			st = "removed"
		}
		fl.Edges = append(fl.Edges, EdgeOut{From: a.ID(), To: b.ID(), Status: st})
	}
	for _, a := range order {
		for _, b := range g.out[a] {
			if show[b] {
				addEdge(a, b)
			}
		}
	}
	// compressed edges: a -> hidden... -> b
	for _, b := range order {
		for n := pred[b]; n != nil; n = pred[n] {
			if show[n] {
				addEdge(n, b)
				break
			}
		}
	}
	fl.inEdges[first.ID()] = []int{0}

	outIdx := map[string]int{}
	for _, f := range order {
		if !isChanged(f) {
			continue
		}
		for _, o := range f.Outputs {
			k := o.Kind + "|" + o.What
			i, ok := outIdx[k]
			if !ok {
				i = len(fl.outCands)
				outIdx[k] = i
				fl.outCands = append(fl.outCands, o)
			}
			fl.outEdges[f.ID()] = append(fl.outEdges[f.ID()], i)
		}
	}
	if len(fl.outCands) == 0 {
		// No stored record, message or call was found. The visible effect is what the caller gets back.
		what := "result returned by " + entry.Label()
		if entry.Entry.Kind == "cli" {
			what = "exit code and printed output of " + entry.Entry.What
		}
		fl.outCands = []RefCand{{Kind: "return", What: what, File: entry.File, Line: entry.Start}}
		fl.outEdges[first.ID()] = append(fl.outEdges[first.ID()], 0)
	}
	for _, f := range order {
		fl.Functions = append(fl.Functions, toFuncOut(f))
		fl.Score += f.Risk.Score*2 + f.ChangedLines
	}
	return fl
}

func toFuncOut(f *Func) FuncOut {
	st := "context"
	switch f.Status {
	case "A":
		st = "added"
	case "M":
		st = "modified"
	case "D":
		st = "removed"
	}
	return FuncOut{ID: f.ID(), Label: f.Label(), Status: st, Module: f.Module(), File: f.File, Line: f.Start,
		Risk: f.Risk.Reasons, RiskScore: f.Risk.Score}
}

// ---- Flow construction for infrastructure files

func iacFlows(index []*Func) []*FlowOut {
	byFile := map[string][]*Func{}
	var files []string
	for _, f := range index {
		switch f.Kind {
		case "resource", "module", "job", "param", "output", "trigger":
			if _, ok := byFile[f.File]; !ok {
				files = append(files, f.File)
			}
			byFile[f.File] = append(byFile[f.File], f)
		}
	}
	sort.Strings(files)
	var flows []*FlowOut
	for _, file := range files {
		blocks := byFile[file]
		var shown []*Func
		for _, b := range blocks {
			if isChanged(b) && (b.Kind == "resource" || b.Kind == "module" || b.Kind == "job") {
				shown = append(shown, b)
			}
		}
		if len(shown) == 0 {
			continue
		}
		fl := &FlowOut{Collapsed: "function", outEdges: map[string][]int{}, inEdges: map[string][]int{}}
		isShown := map[string]bool{}
		for _, s := range shown {
			isShown[s.Name] = true
			fl.Functions = append(fl.Functions, toFuncOut(s))
			fl.Score += s.Risk.Score*2 + s.ChangedLines
		}
		refs := func(b *Func, names map[string]bool) bool {
			for _, r := range b.Refs {
				if names[r] {
					return true
				}
			}
			return false
		}
		for _, b := range blocks {
			switch b.Kind {
			case "param":
				var users []*Func
				for _, s := range shown {
					for _, r := range s.Refs {
						if r == b.Name {
							users = append(users, s)
						}
					}
				}
				if len(users) > 0 || isChanged(b) {
					i := len(fl.inCands)
					fl.inCands = append(fl.inCands, *b.Entry)
					for _, u := range users {
						fl.inEdges[u.ID()] = append(fl.inEdges[u.ID()], i)
					}
				}
			case "trigger":
				i := len(fl.inCands)
				fl.inCands = append(fl.inCands, *b.Entry)
				for _, s := range shown {
					if s.Kind == "job" && len(s.Refs) == 0 {
						fl.inEdges[s.ID()] = append(fl.inEdges[s.ID()], i)
					}
				}
			case "output":
				if refs(b, isShown) {
					i := len(fl.outCands)
					fl.outCands = append(fl.outCands, RefCand{Kind: "output", What: "output " + strings.TrimPrefix(b.Name, "output."), File: file, Line: b.Start})
					for _, s := range shown {
						for _, r := range b.Refs {
							if r == s.Name {
								fl.outEdges[s.ID()] = append(fl.outEdges[s.ID()], i)
							}
						}
					}
				}
			}
		}
		for _, s := range shown {
			for _, o := range s.Outputs {
				i := len(fl.outCands)
				fl.outCands = append(fl.outCands, o)
				fl.outEdges[s.ID()] = append(fl.outEdges[s.ID()], i)
			}
			for _, r := range s.Refs {
				for _, t := range shown {
					if t.Name == r && t != s {
						st := "existing"
						if s.Status == "A" || t.Status == "A" {
							st = "new"
						}
						fl.Edges = append(fl.Edges, EdgeOut{From: t.ID(), To: s.ID(), Status: st})
					}
				}
			}
		}
		if len(fl.inCands) == 0 {
			fl.inCands = append(fl.inCands, RefCand{Kind: "trigger", What: "apply of " + file, File: file, Line: 1})
			for _, s := range shown {
				fl.inEdges[s.ID()] = append(fl.inEdges[s.ID()], 0)
			}
		}
		if len(fl.outCands) == 0 {
			kind, what := "deploy", fmt.Sprintf("applies %d changed block(s) from %s", len(shown), file)
			if shown[0].Kind == "job" {
				kind, what = "result", fmt.Sprintf("pass or fail result of %d changed job(s) in %s", len(shown), file)
			}
			fl.outCands = append(fl.outCands, RefCand{Kind: kind, What: what, File: file, Line: 1})
			for _, s := range shown {
				fl.outEdges[s.ID()] = append(fl.outEdges[s.ID()], 0)
			}
		}
		flows = append(flows, fl)
	}
	return flows
}

// ---- Collapse and numbering

func nodeCount(fl *FlowOut) int { return len(fl.inCands) + len(fl.Functions) + len(fl.outCands) }

// fit shrinks a flow to the node limit: outputs merge first, then functions group
// into modules, then the rest folds into one "+N more" node.
func (fl *FlowOut) fit() {
	if nodeCount(fl) > convention.MaxNodes && len(fl.outCands) > convention.MaxNodes-len(fl.inCands)-2 {
		keep := convention.MaxNodes - len(fl.inCands) - 2
		if keep < 1 {
			keep = 1
		}
		var names []string
		for _, o := range fl.outCands[keep:] {
			names = append(names, o.What)
		}
		merged := RefCand{Kind: "multiple", What: fmt.Sprintf("%d more outputs: %s", len(names), strings.Join(names, "; "))}
		remap := func(i int) int {
			if i >= keep {
				return keep
			}
			return i
		}
		for id, idxs := range fl.outEdges {
			seen := map[int]bool{}
			var n []int
			for _, i := range idxs {
				if r := remap(i); !seen[r] {
					seen[r] = true
					n = append(n, r)
				}
			}
			fl.outEdges[id] = n
		}
		fl.outCands = append(append([]RefCand{}, fl.outCands[:keep]...), merged)
	}
	if nodeCount(fl) <= convention.MaxNodes {
		return
	}

	// Group by module.
	groups := map[string][]FuncOut{}
	var order []string
	for _, f := range fl.Functions {
		if _, ok := groups[f.Module]; !ok {
			order = append(order, f.Module)
		}
		groups[f.Module] = append(groups[f.Module], f)
	}
	remap := map[string]string{}
	var merged []FuncOut
	for _, m := range order {
		g := groups[m]
		if len(g) == 1 {
			merged = append(merged, g[0])
			continue
		}
		mo := mergeFuncs(fmt.Sprintf("%s · %d fns", m, len(g)), "mod:"+m, g)
		for _, f := range g {
			remap[f.ID] = mo.ID
		}
		merged = append(merged, mo)
	}
	fl.apply(merged, remap)
	fl.Collapsed = "module"
	if nodeCount(fl) <= convention.MaxNodes {
		return
	}

	// Keep the most important nodes, fold the rest.
	room := convention.MaxNodes - len(fl.inCands) - len(fl.outCands) - 1
	if room < 1 {
		room = 1
	}
	sorted := append([]FuncOut{}, fl.Functions...)
	sort.SliceStable(sorted, func(i, j int) bool {
		ci, cj := sorted[i].Status == "context", sorted[j].Status == "context"
		if ci != cj {
			return !ci
		}
		return sorted[i].RiskScore > sorted[j].RiskScore
	})
	keepSet := map[string]bool{}
	for i := 0; i < room && i < len(sorted); i++ {
		keepSet[sorted[i].ID] = true
	}
	var kept, folded []FuncOut
	for _, f := range fl.Functions {
		if keepSet[f.ID] {
			kept = append(kept, f)
		} else {
			folded = append(folded, f)
		}
	}
	if len(folded) > 0 {
		more := mergeFuncs(fmt.Sprintf("+%d more", len(folded)), "more", folded)
		more.Status = "modified"
		remap := map[string]string{}
		for _, f := range folded {
			remap[f.ID] = more.ID
		}
		fl.apply(append(kept, more), remap)
	}
	fl.Collapsed = "more"
}

func mergeFuncs(label, id string, g []FuncOut) FuncOut {
	mo := FuncOut{ID: id, Label: label, Status: "context", Module: g[0].Module, File: g[0].File, Line: g[0].Line}
	allA, allD, anyChanged := true, true, false
	for _, f := range g {
		if f.Status != "added" {
			allA = false
		}
		if f.Status != "removed" {
			allD = false
		}
		if f.Status != "context" {
			anyChanged = true
		}
		if f.RiskScore > mo.RiskScore {
			mo.RiskScore = f.RiskScore
		}
		mo.Risk = unionStrings(mo.Risk, f.Risk)
		mo.Members = append(mo.Members, f.Label)
	}
	switch {
	case allA:
		mo.Status = "added"
	case allD:
		mo.Status = "removed"
	case anyChanged:
		mo.Status = "modified"
	}
	return mo
}

func unionStrings(a, b []string) []string {
	seen := map[string]bool{}
	for _, s := range a {
		seen[s] = true
	}
	for _, s := range b {
		if !seen[s] {
			a = append(a, s)
		}
	}
	return a
}

// apply swaps in new function nodes and rewrites edges and I/O links.
func (fl *FlowOut) apply(funcs []FuncOut, remap map[string]string) {
	fl.Functions = funcs
	re := func(id string) string {
		if n, ok := remap[id]; ok {
			return n
		}
		return id
	}
	var edges []EdgeOut
	seen := map[[2]string]bool{}
	for _, e := range fl.Edges {
		e.From, e.To = re(e.From), re(e.To)
		k := [2]string{e.From, e.To}
		if e.From == e.To || seen[k] {
			continue
		}
		seen[k] = true
		edges = append(edges, e)
	}
	fl.Edges = edges
	rewrite := func(m map[string][]int) map[string][]int {
		out := map[string][]int{}
		for id, idxs := range m {
			n := re(id)
			for _, i := range idxs {
				dup := false
				for _, x := range out[n] {
					if x == i {
						dup = true
					}
				}
				if !dup {
					out[n] = append(out[n], i)
				}
			}
		}
		return out
	}
	fl.inEdges, fl.outEdges = rewrite(fl.inEdges), rewrite(fl.outEdges)
}

var qualifiedRe = regexp.MustCompile(`^(\w+\.)+\w+$`)

// shortLabel keeps a label within the convention.
func shortLabel(s string) string {
	if utf8.RuneCountInString(s) <= convention.MaxLabel {
		return s
	}
	// Drop the class prefix of Class.Name. Dots inside a module or a route are not separators.
	if qualifiedRe.MatchString(s) {
		s = s[strings.LastIndex(s, ".")+1:]
	}
	r := []rune(s)
	if len(r) > convention.MaxLabel {
		r = append(r[:convention.MaxLabel-1], '…')
	}
	return string(r)
}
