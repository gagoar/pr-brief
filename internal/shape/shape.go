package shape

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/gagoar/pr-brief/internal/convention"
)

// Options controls Analyze.
type Options struct {
	Dir         string
	Base        string // default: merge base with the first existing of origin/HEAD, origin/main, ...
	Head        string // default: HEAD
	WorkingTree bool   // compare the base with the working tree (adds untracked files)
	Title       string // optional PR title, used to spot fixes
}

// Report is the JSON the skill reads.
type Report struct {
	Base        string            `json:"base"`
	Head        string            `json:"head"`
	CodeFiles   int               `json:"codeFiles"`
	SmallPR     bool              `json:"smallPR"`
	ConfigOnly  bool              `json:"configOnly"`
	Noise       Noise             `json:"noiseCounts"`
	Refs        map[string]RefOut `json:"refs"`
	Flows       []FlowOut         `json:"flows"`
	Extras      []Extra           `json:"extras"`
	Unplaced    []FuncOut         `json:"unplacedFunctions"`
	Removed     []FuncOut         `json:"removedFunctions"`
	Dropped     []Dropped         `json:"droppedFlows"`
	FilesRanked []RankedFile      `json:"filesRanked"`
	Warnings    []string          `json:"warnings,omitempty"`
}

// Noise counts files that never become diagram nodes.
type Noise struct {
	Tests     int `json:"tests"`
	Docs      int `json:"docs"`
	Generated int `json:"generated"`
	Other     int `json:"other"`
}

// Extra is a diagram type to add besides the Input -> Functions -> Output flows.
type Extra struct {
	Type   string `json:"type"` // stateDiagram-v2 | erDiagram | sequenceDiagram
	Reason string `json:"reason"`
}

// Dropped is a flow that did not get a diagram.
type Dropped struct {
	Input     string `json:"input"`
	Functions int    `json:"functions"`
	Score     int    `json:"score"`
}

// RankedFile is a file to read, with the reasons it is delicate.
type RankedFile struct {
	File      string   `json:"file"`
	Status    string   `json:"status"`
	Added     int      `json:"added"`
	Deleted   int      `json:"deleted"`
	Score     int      `json:"score"`
	Reasons   []string `json:"reasons,omitempty"`
	Functions []string `json:"changedFunctions,omitempty"`
}

type span struct{ lo, hi int }

// Analyze reads the diff and builds the report.
func Analyze(o Options) (*Report, error) {
	if o.Dir == "" {
		o.Dir = "."
	}
	g := git{dir: o.Dir}
	head := o.Head
	if head == "" {
		head = "HEAD"
	}
	base, err := g.mergeBase(o.Base, head)
	if err != nil {
		return nil, err
	}
	diffHead, contentRef := head, head
	if o.WorkingTree {
		diffHead, contentRef = "", ""
	}
	files, err := g.changes(base, diffHead)
	if err != nil {
		return nil, err
	}
	if o.WorkingTree {
		files = append(files, g.untracked(files)...)
	}

	rep := &Report{Base: base, Head: head, Refs: map[string]RefOut{}, Flows: []FlowOut{}, Extras: []Extra{},
		Unplaced: []FuncOut{}, Removed: []FuncOut{}, Dropped: []Dropped{}, FilesRanked: []RankedFile{}}
	if o.WorkingTree {
		rep.Head = "working tree"
	}

	var code []FileChange
	manifests := 0
	for _, f := range files {
		switch c := Classify(f.Path); {
		case f.Binary:
			rep.Noise.Other++
		case c == ClassTest:
			rep.Noise.Tests++
		case c == ClassDoc:
			rep.Noise.Docs++
		case c == ClassGenerated:
			rep.Noise.Generated++
		case c == ClassCode:
			code = append(code, f)
		default:
			rep.Noise.Other++
			if IsManifest(f.Path) {
				manifests++
			}
		}
	}
	rep.CodeFiles = len(code)
	rep.SmallPR = len(code) <= convention.SmallPRFiles
	rep.ConfigOnly = len(code) == 0 && rep.Noise.Other > 0

	changedSpans := map[string][]span{}
	var index, changedFuncs []*Func
	inChangedFile := map[string]bool{}
	fileFuncs := map[string][]*Func{}

	for i := range code {
		f := &code[i]
		if f.Hunks == nil {
			if f.Status == "A" && f.Hunks == nil && o.WorkingTree && f.Added > 0 {
				f.Hunks = []Hunk{{NewStart: 1, NewCount: f.Added}}
			} else if hs, err := g.hunks(base, diffHead, *f); err == nil {
				f.Hunks = hs
			}
		}
		var newF, oldF []*Func
		if f.Status != "D" {
			if c, err := g.show(contentRef, f.Path); err == nil {
				newF = Extract(f.Path, c)
			}
		}
		if f.Status != "A" {
			op := f.OldPath
			if op == "" {
				op = f.Path
			}
			if c, err := g.show(base, op); err == nil {
				oldF = Extract(op, c)
			}
		}
		markStatus(f, newF, oldF, rep, changedSpans)
		inChangedFile[f.Path] = true
		fileFuncs[f.Path] = newF
		index = append(index, newF...)
	}
	for _, f := range index {
		if isChanged(f) {
			changedFuncs = append(changedFuncs, f)
		}
	}

	// Neighbours: callers found by name, and siblings that may be callees.
	for _, nf := range g.neighbours(contentRef, base, changedFuncs, code, inChangedFile) {
		index = append(index, nf)
	}
	index = bindRegistrations(index)

	graph := buildGraph(index)
	flows := graph.buildFlows(6)
	flows = append(flows, iacFlows(index)...)

	placed := map[string]bool{}
	for _, fl := range flows {
		for _, f := range fl.Functions {
			placed[f.ID] = true
		}
	}
	for _, f := range changedFuncs {
		if (f.Kind == "func") && f.Status != "D" && !placed[f.ID()] {
			rep.Unplaced = append(rep.Unplaced, toFuncOut(f))
		}
	}

	for _, fl := range flows {
		fl.fit()
	}
	sort.SliceStable(flows, func(i, j int) bool { return flows[i].Score > flows[j].Score })
	for i, fl := range flows {
		if i >= convention.MaxDiagrams {
			what := ""
			if len(fl.inCands) > 0 {
				what = fl.inCands[0].What
			}
			rep.Dropped = append(rep.Dropped, Dropped{Input: what, Functions: len(fl.Functions), Score: fl.Score})
			continue
		}
		rep.Flows = append(rep.Flows, number(fl, rep.Refs, changedSpans))
	}
	if len(rep.Dropped) > 0 {
		rep.Warnings = append(rep.Warnings, fmt.Sprintf("%d flow(s) have no diagram; consider splitting this PR", len(rep.Dropped)))
	}

	rep.Extras = detectExtras(code, o.Title, len(rep.Flows))
	rep.FilesRanked = rankFiles(code, changedFuncs)
	return rep, nil
}

// markStatus sets Status, ChangedLines and Risk on the new functions of a file
// and records removed functions.
func markStatus(f *FileChange, newF, oldF []*Func, rep *Report, spans map[string][]span) {
	key := func(x *Func) string { return x.Class + "." + x.Name }
	oldKeys := map[string]bool{}
	for _, o := range oldF {
		oldKeys[key(o)] = true
	}
	newKeys := map[string]bool{}
	for _, n := range newF {
		newKeys[key(n)] = true
	}

	var ns, os []span
	for _, h := range f.Hunks {
		if h.NewCount > 0 {
			ns = append(ns, span{h.NewStart, h.NewStart + h.NewCount - 1})
		} else {
			ns = append(ns, span{h.NewStart, h.NewStart + 1})
		}
		if h.OldCount > 0 {
			os = append(os, span{h.OldStart, h.OldStart + h.OldCount - 1})
		}
	}
	spans[f.Path] = ns
	hit := func(lo, hi int, ss []span) bool {
		for _, s := range ss {
			if s.lo <= hi && lo <= s.hi {
				return true
			}
		}
		return false
	}

	pr := pathRisk(f.Path)
	for _, n := range newF {
		switch {
		case f.Status == "A":
			n.Status = "A"
		case hit(n.Start, n.End, ns):
			n.Status = "M"
			if !oldKeys[key(n)] {
				n.Status = "A"
			}
		default:
			continue
		}
		var added []string
		for _, h := range f.Hunks {
			lo, hi := h.NewStart, h.NewStart+max(h.NewCount, 1)-1
			if lo <= n.End && n.Start <= hi {
				added = append(added, h.Added...)
				n.ChangedLines += len(h.Added) + len(h.Removed)
			}
		}
		if f.Status == "A" {
			n.ChangedLines = n.End - n.Start + 1
		}
		n.Risk = merge(pr, diffRisk(added), sizeRisk(n.ChangedLines))
	}
	for _, o := range oldF {
		if newKeys[key(o)] {
			continue
		}
		if f.Status == "D" || hit(o.Start, o.End, os) {
			o.Status = "D"
			rep.Removed = append(rep.Removed, toFuncOut(o))
		}
	}
}

func (g git) untracked(have []FileChange) []FileChange {
	out, err := g.run("ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	for _, f := range have {
		seen[f.Path] = true
	}
	var fs []FileChange
	for _, p := range strings.Split(strings.TrimSpace(out), "\n") {
		if p == "" || seen[p] {
			continue
		}
		data, err := os.ReadFile(g.dir + "/" + p)
		if err != nil {
			continue
		}
		fs = append(fs, FileChange{Path: p, Status: "A", Added: strings.Count(string(data), "\n") + 1})
	}
	return fs
}

var grepGlobs = []string{"*.go", "*.cs", "*.java", "*.ts", "*.tsx", "*.js", "*.jsx", "*.mjs", "*.py"}

// neighbours parses unchanged files that call, or sit next to, the changed code.
func (g git) neighbours(ref, base string, changed []*Func, code []FileChange, skip map[string]bool) []*Func {
	const maxFiles = 40
	paths := map[string]bool{}
	var names []string
	seen := map[string]bool{}
	for _, f := range changed {
		if f.Kind == "func" && f.Status != "D" && !seen[f.Name] && len(f.Name) > 3 {
			seen[f.Name] = true
			names = append(names, f.Name)
		}
	}
	if len(names) > 30 {
		names = names[:30]
	}
	if len(names) > 0 {
		args := []string{"grep", "-l", "-w"}
		for _, n := range names {
			args = append(args, "-e", n)
		}
		if ref != "" {
			args = append(args, ref)
		}
		args = append(args, "--")
		args = append(args, grepGlobs...)
		if out, err := g.run(args...); err == nil || out != "" {
			for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
				l = strings.TrimPrefix(l, ref+":")
				if l != "" && !skip[l] {
					paths[l] = true
				}
			}
		}
	}
	dirs := map[string]bool{}
	for _, f := range code {
		dirs[path.Dir(f.Path)] = true
	}
	var dl []string
	for d := range dirs {
		dl = append(dl, d)
	}
	sort.Strings(dl)
	if len(dl) > 10 {
		dl = dl[:10]
	}
	r := ref
	if r == "" {
		r = "HEAD"
	}
	for _, d := range dl {
		spec := d + "/"
		if d == "." {
			spec = ""
		}
		args := []string{"ls-tree", "--name-only", r}
		if spec != "" {
			args = append(args, spec)
		}
		if out, err := g.run(args...); err == nil {
			for _, p := range strings.Split(strings.TrimSpace(out), "\n") {
				if p != "" && !skip[p] && Classify(p) == ClassCode && path.Dir(p) == d {
					paths[p] = true
				}
			}
		}
	}

	var list []string
	for p := range paths {
		if Classify(p) == ClassCode {
			list = append(list, p)
		}
	}
	sort.Strings(list)
	if len(list) > maxFiles {
		list = list[:maxFiles]
	}
	var out []*Func
	for _, p := range list {
		c, err := g.show(ref, p)
		if err != nil || len(c) > 200_000 {
			continue
		}
		out = append(out, Extract(p, c)...)
	}
	return out
}

// number assigns I/O ids, builds the final edges, and fixes the node ids.
func number(fl *FlowOut, refs map[string]RefOut, spans map[string][]span) FlowOut {
	idFor := func(prefix string, c RefCand) string {
		for id, r := range refs {
			if strings.HasPrefix(id, prefix) && r.Kind == c.Kind && r.What == c.What {
				return id
			}
		}
		n := 0
		for id := range refs {
			if strings.HasPrefix(id, prefix) {
				n++
			}
		}
		id := fmt.Sprintf("%s%d", prefix, n+1)
		refs[id] = RefOut{Kind: c.Kind, What: c.What, File: c.File, Line: c.Line}
		return id
	}
	node := map[string]string{}
	for i := range fl.Functions {
		n := fmt.Sprintf("F%d", i+1)
		node[fl.Functions[i].ID] = n
		fl.Functions[i].Node = n
		fl.Functions[i].Label = shortLabel(fl.Functions[i].Label)
	}
	statusOf := map[string]string{}
	for _, f := range fl.Functions {
		statusOf[f.ID] = f.Status
	}

	var edges []EdgeOut
	for _, e := range fl.Edges {
		edges = append(edges, EdgeOut{From: node[e.From], To: node[e.To], Status: e.Status})
	}
	inIDs := make([]string, len(fl.inCands))
	for i, c := range fl.inCands {
		inIDs[i] = idFor("I", c)
	}
	outIDs := make([]string, len(fl.outCands))
	for i, c := range fl.outCands {
		outIDs[i] = idFor("O", c)
	}
	var fids []string
	for _, f := range fl.Functions {
		fids = append(fids, f.ID)
	}
	for _, id := range fids {
		for _, i := range fl.inEdges[id] {
			st := "existing"
			if statusOf[id] == "added" {
				st = "new"
			}
			edges = append(edges, EdgeOut{From: inIDs[i], To: node[id], Status: st})
		}
		for _, i := range fl.outEdges[id] {
			st := "existing"
			c := fl.outCands[i]
			if statusOf[id] == "added" || inSpans(spans[c.File], c.Line) {
				st = "new"
			}
			edges = append(edges, EdgeOut{From: node[id], To: outIDs[i], Status: st})
		}
	}
	fl.Edges = dedupeEdges(edges)
	fl.Inputs, fl.Outputs = uniq(inIDs), uniq(outIDs)
	if len(fl.Edges) > convention.MaxEdges {
		fl.Warnings = append(fl.Warnings, fmt.Sprintf("%d edges; the limit is %d", len(fl.Edges), convention.MaxEdges))
	}
	return *fl
}

func inSpans(ss []span, line int) bool {
	for _, s := range ss {
		if s.lo <= line && line <= s.hi {
			return true
		}
	}
	return false
}

func dedupeEdges(es []EdgeOut) []EdgeOut {
	seen := map[[2]string]bool{}
	var out []EdgeOut
	for _, e := range es {
		k := [2]string{e.From, e.To}
		if e.From == "" || e.To == "" || e.From == e.To || seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, e)
	}
	return out
}

func uniq(s []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

var (
	titleFixRe  = regexp.MustCompile(`(?i)^(fix|bugfix|hotfix)(\([^)]*\))?[:!]`)
	schemaPath  = regexp.MustCompile(`(?i)(^|/)migrations?/|\.sql$|\.proto$|(^|/)schema\.|(^|/)models?/|entit`)
	statePath   = regexp.MustCompile(`(?i)state|status|lifecycle|workflow|saga`)
	stateLineRe = regexp.MustCompile(`(?i)\b(enum|transition\w*|state|status)\b|^\s*case\s|^\s*[A-Z]\w*\s*(=\s*\d+)?\s*,?\s*$|->`)
)

func detectExtras(code []FileChange, title string, flows int) []Extra {
	extras := []Extra{}
	total, schema := 0, 0
	stateHits := 0
	for _, f := range code {
		total += f.Lines()
		if schemaPath.MatchString(f.Path) {
			schema += f.Lines()
		}
		if statePath.MatchString(path.Base(f.Path)) {
			for _, h := range f.Hunks {
				for _, l := range append(append([]string{}, h.Added...), h.Removed...) {
					if stateLineRe.MatchString(l) {
						stateHits++
					}
				}
			}
		}
	}
	if total > 0 && schema*100 >= total*40 {
		extras = append(extras, Extra{"erDiagram", fmt.Sprintf("%d%% of the changed lines are in schema or migration files", schema*100/total)})
	}
	if stateHits >= 2 {
		extras = append(extras, Extra{"stateDiagram-v2", "the diff adds or removes states or transitions in a state/status/workflow file"})
	}
	if titleFixRe.MatchString(strings.TrimSpace(title)) && flows == 1 {
		extras = append(extras, Extra{"sequenceDiagram", "a fix in a single call chain; show the call order with an alt before/after block"})
	}
	return extras
}

func rankFiles(code []FileChange, changed []*Func) []RankedFile {
	byFile := map[string][]string{}
	for _, f := range changed {
		if f.Status != "D" {
			byFile[f.File] = append(byFile[f.File], f.Label())
		}
	}
	var out []RankedFile
	for _, f := range code {
		var added []string
		for _, h := range f.Hunks {
			added = append(added, h.Added...)
		}
		r := merge(pathRisk(f.Path), diffRisk(added), sizeRisk(f.Lines()))
		out = append(out, RankedFile{File: f.Path, Status: f.Status, Added: f.Added, Deleted: f.Deleted,
			Score: r.Score, Reasons: r.Reasons, Functions: byFile[f.Path]})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		li, lj := out[i].Added+out[i].Deleted, out[j].Added+out[j].Deleted
		if li != lj {
			return li > lj
		}
		return out[i].File < out[j].File
	})
	return out
}
