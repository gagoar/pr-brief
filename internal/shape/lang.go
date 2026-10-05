package shape

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

// RefCand is a possible Input or Output found in code.
type RefCand struct {
	Kind string // route, cli, queue, timer, trigger, param | db, event, call, file, response, deploy, output
	What string
	File string
	Line int
}

// Func is one function, method, or IaC block.
type Func struct {
	File, Class, Name string
	Start, End        int    // 1-based, inclusive
	Kind              string // func | resource | module | job | param | output | trigger
	Entry             *RefCand
	Outputs           []RefCand
	Calls             []string
	Refs              []string // IaC: names of other symbols this block mentions
	Status            string   // A, M, D, or "" when unchanged
	ChangedLines      int
	Risk              Risk
	body              []string
}

// ID is stable within one analysis.
func (f *Func) ID() string {
	q := f.Name
	if f.Class != "" {
		q = f.Class + "." + f.Name
	}
	return f.File + "::" + q
}

// Module is the group a function collapses into.
func (f *Func) Module() string {
	if f.Class != "" {
		return f.Class
	}
	base := path.Base(f.File)
	return strings.TrimSuffix(base, path.Ext(base))
}

// Label is the node text for a function.
func (f *Func) Label() string {
	switch f.Kind {
	case "func":
		if f.Class != "" {
			return f.Class + "." + f.Name + "()"
		}
		return f.Name + "()"
	}
	return f.Name
}

var keywords = map[string]bool{
	"if": true, "for": true, "foreach": true, "while": true, "switch": true, "catch": true, "using": true,
	"lock": true, "return": true, "new": true, "await": true, "typeof": true, "nameof": true, "sizeof": true,
	"function": true, "func": true, "def": true, "print": true, "len": true, "make": true, "append": true,
	"range": true, "select": true, "defer": true, "panic": true, "string": true, "int": true,
	"sizeof_": true, "when": true, "fixed": true, "checked": true, "unchecked": true, "base": true, "this": true,
	"super": true, "throw": true, "assert": true, "not": true, "and": true, "or": true, "in": true, "is": true,
}

var (
	commentRe = regexp.MustCompile(`//.*$|#.*$`)
	strRe     = regexp.MustCompile("\"(?:[^\"\\\\]|\\\\.)*\"|'(?:[^'\\\\]|\\\\.)*'|`[^`]*`")
	callRe    = regexp.MustCompile(`(?:([A-Za-z_][A-Za-z0-9_]*)\s*\.\s*)?([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
)

// stripLine removes strings and trailing comments so braces count correctly.
func stripLine(s string, hashComments bool) string {
	s = strRe.ReplaceAllString(s, `""`)
	if i := strings.Index(s, "//"); i >= 0 {
		s = s[:i]
	}
	if hashComments {
		if i := strings.Index(s, "#"); i >= 0 {
			s = s[:i]
		}
	}
	return s
}

// braceEnd returns the 0-based index of the line that closes the first "{" at or
// after line `from`. If no brace opens within 6 lines (an expression body or an
// abstract member), the member ends on the line that ends with ";".
func braceEnd(lines []string, from int) int {
	depth, opened := 0, false
	for i := from; i < len(lines); i++ {
		l := stripLine(lines[i], false)
		for _, c := range l {
			switch c {
			case '{':
				depth++
				opened = true
			case '}':
				depth--
				if opened && depth == 0 {
					return i
				}
			}
		}
		if !opened && i-from >= 6 {
			return from
		}
		if !opened && strings.HasSuffix(strings.TrimSpace(l), ";") {
			return i
		}
	}
	return len(lines) - 1
}

type classRange struct {
	name       string
	start, end int
	attrs      []string
}

func innermost(cs []classRange, line int) *classRange {
	var best *classRange
	for i := range cs {
		c := &cs[i]
		if c.start <= line && line <= c.end && (best == nil || c.start > best.start) {
			best = c
		}
	}
	return best
}

// attrsAbove returns the attribute/decorator lines directly above 0-based index i.
func attrsAbove(lines []string, i int, prefixes ...string) []string {
	var out []string
	for j := i - 1; j >= 0 && j >= i-8; j-- {
		t := strings.TrimSpace(lines[j])
		if t == "" {
			break
		}
		ok := false
		for _, p := range prefixes {
			if strings.HasPrefix(t, p) {
				ok = true
			}
		}
		if !ok {
			break
		}
		out = append(out, t)
	}
	return out
}

// Extract parses the functions of one file.
func Extract(file, content string) []*Func {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	var fs []*Func
	switch strings.ToLower(path.Ext(file)) {
	case ".go":
		fs = goFuncs(file, lines)
	case ".cs":
		fs = braceFuncs(file, lines, csMethodRe, csClassRe, csEntry)
	case ".java":
		fs = braceFuncs(file, lines, javaMethodRe, javaClassRe, javaEntry)
	case ".ts", ".tsx", ".js", ".jsx", ".mjs":
		fs = jsFuncs(file, lines)
	case ".py":
		fs = pyFuncs(file, lines)
	case ".tf":
		fs = tfBlocks(file, lines)
	case ".bicep":
		fs = bicepBlocks(file, lines)
	case ".yml", ".yaml":
		if workflowRe.MatchString(file) {
			fs = workflowJobs(file, lines)
		}
	}
	switch strings.ToLower(path.Ext(file)) {
	case ".go", ".cs":
		fs = append(fs, registrationFuncs(file, lines)...)
	}
	for _, f := range fs {
		if f.Start < 1 {
			f.Start = 1
		}
		if f.End > len(lines) {
			f.End = len(lines)
		}
		if f.End < f.Start {
			f.End = f.Start
		}
		f.body = lines[f.Start-1 : f.End]
		if f.Kind == "func" {
			f.Calls = findCalls(f.body)
			detectOutputs(f)
		}
	}
	return fs
}

func findCalls(body []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, l := range body {
		l = stripLine(l, false)
		for _, m := range callRe.FindAllStringSubmatch(l, -1) {
			n := m[2]
			if keywords[strings.ToLower(n)] {
				continue
			}
			if m[1] != "" {
				n = m[1] + "." + n // keep the receiver: it picks between same-named methods
			}
			if seen[n] {
				continue
			}
			seen[n] = true
			out = append(out, n)
		}
	}
	return out
}

// ---- Go

var goFuncRe = regexp.MustCompile(`^func\s+(?:\(\s*\w*\s*\*?\s*(\w+)(?:\[[^\]]*\])?\s*\)\s*)?(\w+)\s*(?:\[[^\]]*\])?\(`)

func goFuncs(file string, lines []string) []*Func {
	var out []*Func
	for i, l := range lines {
		m := goFuncRe.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		f := &Func{File: file, Class: m[1], Name: m[2], Kind: "func", Start: i + 1, End: braceEnd(lines, i) + 1}
		if m[2] == "main" && m[1] == "" {
			f.Entry = &RefCand{Kind: "cli", What: "run " + path.Base(path.Dir(file)), File: file, Line: i + 1}
		}
		out = append(out, f)
	}
	return out
}

// ---- C# and Java

var (
	mods          = `(?:(?:public|private|protected|internal|static|async|override|virtual|abstract|sealed|partial|new|extern|final|synchronized|default)\s+)`
	csMethodRe    = regexp.MustCompile(`^\s*` + mods + `+(?:[\w<>\[\],.?]+\s+)+(\w+)\s*(?:<[^>]*>)?\s*\(`)
	javaMethodRe  = csMethodRe
	csClassRe     = regexp.MustCompile(`^\s*` + mods + `*(?:class|record|struct|interface|enum)\s+(\w+)`)
	javaClassRe   = csClassRe
	httpAttrRe    = regexp.MustCompile(`^\[Http(Get|Post|Put|Delete|Patch)(?:\(\s*"([^"]*)")?`)
	routeAttrRe   = regexp.MustCompile(`^\[Route\(\s*"([^"]*)"`)
	triggerAttrRe = regexp.MustCompile(`\[(ServiceBusTrigger|QueueTrigger|TimerTrigger|EventHubTrigger|BlobTrigger|HttpTrigger|CosmosDBTrigger|EventGridTrigger)\(([^)]*)\)`)
	springRe      = regexp.MustCompile(`^@(Get|Post|Put|Delete|Patch|Request)Mapping(?:\(\s*(?:value\s*=\s*|path\s*=\s*)?\{?"([^"]*)")?`)
	listenerRe    = regexp.MustCompile(`^@(Scheduled|KafkaListener|RabbitListener|SqsListener|JmsListener|EventListener|StreamListener)(?:\(([^)]*)\))?`)
)

type entryFn func(lines []string, i int, class *classRange) *RefCand

func braceFuncs(file string, lines []string, methodRe, classRe *regexp.Regexp, entry entryFn) []*Func {
	var classes []classRange
	for i, l := range lines {
		if m := classRe.FindStringSubmatch(l); m != nil {
			classes = append(classes, classRange{name: m[1], start: i, end: braceEnd(lines, i), attrs: attrsAbove(lines, i, "[", "@")})
		}
	}
	var out []*Func
	for i, l := range lines {
		m := methodRe.FindStringSubmatch(l)
		if m == nil || classRe.MatchString(l) {
			continue
		}
		name := m[1] // a modifier precedes it, so it cannot be a control keyword
		c := innermost(classes, i)
		f := &Func{File: file, Name: name, Kind: "func", Start: i + 1, End: braceEnd(lines, i) + 1}
		if c != nil {
			f.Class = c.name
		}
		// Attributes belong to the member, so the span starts at the first one.
		if a := attrsAbove(lines, i, "[", "@"); len(a) > 0 {
			f.Start = i + 1 - len(a)
		}
		f.Entry = entry(lines, i, c)
		if f.Entry != nil {
			f.Entry.File, f.Entry.Line = file, i+1
		}
		out = append(out, f)
	}
	return out
}

func csEntry(lines []string, i int, c *classRange) *RefCand {
	method, sub := "", ""
	for _, a := range attrsAbove(lines, i, "[") {
		if m := httpAttrRe.FindStringSubmatch(a); m != nil {
			method, sub = strings.ToUpper(m[1]), m[2]
		}
	}
	if method != "" {
		prefix := ""
		if c != nil {
			for _, a := range c.attrs {
				if m := routeAttrRe.FindStringSubmatch(a); m != nil {
					prefix = strings.ReplaceAll(m[1], "[controller]", strings.ToLower(strings.TrimSuffix(c.name, "Controller")))
				}
			}
		}
		p := "/" + strings.Trim(strings.Trim(prefix, "/")+"/"+strings.Trim(sub, "/"), "/")
		return &RefCand{Kind: "route", What: method + " " + strings.TrimRight(p, "/")}
	}
	end := i + 8
	if end > len(lines) {
		end = len(lines)
	}
	sig := strings.Join(lines[max(0, i-2):end], " ")
	if m := triggerAttrRe.FindStringSubmatch(sig); m != nil {
		kind := "queue"
		if m[1] == "TimerTrigger" {
			kind = "timer"
		}
		if m[1] == "HttpTrigger" {
			kind = "route"
		}
		return &RefCand{Kind: kind, What: m[1] + " " + strings.TrimSpace(m[2])}
	}
	return nil
}

func javaEntry(lines []string, i int, c *classRange) *RefCand {
	method, sub := "", ""
	for _, a := range attrsAbove(lines, i, "@") {
		if m := springRe.FindStringSubmatch(a); m != nil {
			method, sub = strings.ToUpper(m[1]), m[2]
			if method == "REQUEST" {
				method = "ANY"
			}
		}
		if m := listenerRe.FindStringSubmatch(a); m != nil {
			kind := "queue"
			if m[1] == "Scheduled" {
				kind = "timer"
			}
			return &RefCand{Kind: kind, What: m[1] + " " + strings.TrimSpace(m[2])}
		}
	}
	if method == "" {
		return nil
	}
	prefix := ""
	if c != nil {
		for _, a := range c.attrs {
			if m := springRe.FindStringSubmatch(a); m != nil {
				prefix = m[2]
			}
		}
	}
	p := "/" + strings.Trim(strings.Trim(prefix, "/")+"/"+strings.Trim(sub, "/"), "/")
	return &RefCand{Kind: "route", What: method + " " + strings.TrimRight(p, "/")}
}

// ---- JS / TS

var (
	jsFuncRe   = regexp.MustCompile(`^\s*(?:export\s+)?(?:default\s+)?(?:async\s+)?function\*?\s+(\w+)\s*\(`)
	jsArrowRe  = regexp.MustCompile(`^\s*(?:export\s+)?(?:const|let|var)\s+(\w+)\s*(?::[^=]+)?=\s*(?:async\s+)?(?:function\b|\([^)]*\)\s*(?::[^=]+)?=>|\w+\s*=>)`)
	jsMethodRe = regexp.MustCompile(`^\s*(?:(?:public|private|protected|static|async|override|readonly)\s+)*(\w+)\s*(?:<[^>]*>)?\s*\([^)]*\)\s*(?::\s*[^{]+)?\{\s*$`)
	jsClassRe  = regexp.MustCompile(`^\s*(?:export\s+)?(?:default\s+)?(?:abstract\s+)?class\s+(\w+)`)
	jsRouteRe  = regexp.MustCompile("\\b(?:app|router|server|fastify|api)\\.(get|post|put|delete|patch)\\(\\s*['\"`]([^'\"`]+)['\"`]\\s*,\\s*(.*)$")
)

func jsFuncs(file string, lines []string) []*Func {
	var classes []classRange
	for i, l := range lines {
		if m := jsClassRe.FindStringSubmatch(l); m != nil {
			classes = append(classes, classRange{name: m[1], start: i, end: braceEnd(lines, i)})
		}
	}
	var out []*Func
	for i, l := range lines {
		if m := jsRouteRe.FindStringSubmatch(l); m != nil {
			handler := lastIdent(m[3])
			f := &Func{File: file, Name: strings.ToUpper(m[1]) + " " + m[2], Kind: "func", Start: i + 1, End: braceEnd(lines, i) + 1}
			f.Entry = &RefCand{Kind: "route", What: strings.ToUpper(m[1]) + " " + m[2], File: file, Line: i + 1}
			if handler != "" && !strings.Contains(m[3], "=>") && !strings.Contains(m[3], "function") {
				f.Kind = "registration"
				f.Calls = []string{handler}
			}
			out = append(out, f)
			continue
		}
		var name string
		switch {
		case jsFuncRe.MatchString(l):
			name = jsFuncRe.FindStringSubmatch(l)[1]
		case jsArrowRe.MatchString(l):
			name = jsArrowRe.FindStringSubmatch(l)[1]
		case jsMethodRe.MatchString(l):
			name = jsMethodRe.FindStringSubmatch(l)[1]
			if keywords[strings.ToLower(name)] || name == "constructor" {
				continue
			}
		default:
			continue
		}
		f := &Func{File: file, Name: name, Kind: "func", Start: i + 1, End: braceEnd(lines, i) + 1}
		if c := innermost(classes, i); c != nil {
			f.Class = c.name
		}
		out = append(out, f)
	}
	return out
}

var identRe = regexp.MustCompile(`([A-Za-z_$][\w$]*)\s*\)?\s*;?\s*$`)

func lastIdent(s string) string {
	s = strings.TrimSpace(strings.TrimRight(strings.TrimSpace(s), ");"))
	if m := identRe.FindStringSubmatch(s); m != nil && !keywords[m[1]] {
		return m[1]
	}
	return ""
}

// ---- Python

var (
	pyDefRe   = regexp.MustCompile(`^(\s*)(?:async\s+)?def\s+(\w+)\s*\(`)
	pyClassRe = regexp.MustCompile(`^(\s*)class\s+(\w+)`)
	pyRouteRe = regexp.MustCompile(`^@\w+\.(route|get|post|put|delete|patch)\(\s*["']([^"']+)["'](?:.*methods\s*=\s*\[\s*["'](\w+))?`)
	pyCronRe  = regexp.MustCompile(`^@(?:\w+\.)?(task|shared_task|periodic_task|scheduled|cron)\b`)
)

func indentOf(s string) int { return len(s) - len(strings.TrimLeft(s, " \t")) }

func pyBlockEnd(lines []string, i, indent int) int {
	end := i
	for j := i + 1; j < len(lines); j++ {
		if strings.TrimSpace(lines[j]) == "" {
			continue
		}
		if indentOf(lines[j]) <= indent {
			break
		}
		end = j
	}
	return end
}

func pyFuncs(file string, lines []string) []*Func {
	var classes []classRange
	for i, l := range lines {
		if m := pyClassRe.FindStringSubmatch(l); m != nil {
			classes = append(classes, classRange{name: m[2], start: i, end: pyBlockEnd(lines, i, len(m[1]))})
		}
	}
	var out []*Func
	for i, l := range lines {
		m := pyDefRe.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		f := &Func{File: file, Name: m[2], Kind: "func", Start: i + 1, End: pyBlockEnd(lines, i, len(m[1])) + 1}
		if c := innermost(classes, i); c != nil {
			f.Class = c.name
		}
		decos := attrsAbove(lines, i, "@")
		if len(decos) > 0 {
			f.Start = i + 1 - len(decos)
		}
		for _, d := range decos {
			if rm := pyRouteRe.FindStringSubmatch(d); rm != nil {
				method := strings.ToUpper(rm[1])
				if method == "ROUTE" {
					method = "ANY"
					if rm[3] != "" {
						method = strings.ToUpper(rm[3])
					}
				}
				f.Entry = &RefCand{Kind: "route", What: method + " " + rm[2], File: file, Line: i + 1}
			} else if cm := pyCronRe.FindStringSubmatch(d); cm != nil {
				f.Entry = &RefCand{Kind: "timer", What: "task " + m[2], File: file, Line: i + 1}
			}
		}
		out = append(out, f)
	}
	return out
}

// ---- Terraform, Bicep, workflows

var (
	tfBlockRe = regexp.MustCompile(`^(resource|module|data|variable|output)\s+"([^"]+)"(?:\s+"([^"]+)")?\s*\{`)
	tfVarRef  = regexp.MustCompile(`\b(var|module|local)\.(\w+)`)
	tfResRef  = regexp.MustCompile(`\b([a-z][a-z0-9]*_[a-z0-9_]+)\.(\w+)\b`)
)

func tfBlocks(file string, lines []string) []*Func {
	var out []*Func
	for i, l := range lines {
		m := tfBlockRe.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		f := &Func{File: file, Start: i + 1, End: braceEnd(lines, i) + 1}
		switch m[1] {
		case "variable":
			f.Kind, f.Name = "param", "var."+m[2]
			f.Entry = &RefCand{Kind: "param", What: "variable " + m[2], File: file, Line: i + 1}
		case "output":
			f.Kind, f.Name = "output", "output."+m[2]
		case "module":
			f.Kind, f.Name = "module", "module."+m[2]
		case "data":
			f.Kind, f.Name = "resource", "data."+m[2]+"."+m[3]
		default:
			f.Kind, f.Name = "resource", m[2]+"."+m[3]
		}
		out = append(out, f)
	}
	// References, computed after bodies are known.
	for _, f := range out {
		body := strings.Join(lines[f.Start-1:min(f.End, len(lines))], "\n")
		seen := map[string]bool{}
		for _, m := range tfVarRef.FindAllStringSubmatch(body, -1) {
			n := m[1] + "." + m[2]
			if !seen[n] && n != f.Name {
				seen[n] = true
				f.Refs = append(f.Refs, n)
			}
		}
		for _, m := range tfResRef.FindAllStringSubmatch(body, -1) {
			n := m[1] + "." + m[2]
			if !seen[n] && n != f.Name {
				seen[n] = true
				f.Refs = append(f.Refs, n)
			}
		}
	}
	return out
}

var (
	bicepParamRe  = regexp.MustCompile(`^param\s+(\w+)`)
	bicepResRe    = regexp.MustCompile(`^(resource|module)\s+(\w+)\s+'([^']+)'`)
	bicepOutputRe = regexp.MustCompile(`^output\s+(\w+)`)
	wordRe        = regexp.MustCompile(`[A-Za-z_]\w*`)
)

func bicepBlocks(file string, lines []string) []*Func {
	var out []*Func
	for i, l := range lines {
		var f *Func
		switch {
		case bicepParamRe.MatchString(l):
			n := bicepParamRe.FindStringSubmatch(l)[1]
			f = &Func{Kind: "param", Name: n, Start: i + 1, End: braceEndOrSelf(lines, i) + 1,
				Entry: &RefCand{Kind: "param", What: "parameter " + n, File: file, Line: i + 1}}
		case bicepResRe.MatchString(l):
			m := bicepResRe.FindStringSubmatch(l)
			kind := "resource"
			if m[1] == "module" {
				kind = "module"
			}
			f = &Func{Kind: kind, Name: m[2], Start: i + 1, End: braceEnd(lines, i) + 1}
		case bicepOutputRe.MatchString(l):
			f = &Func{Kind: "output", Name: "output." + bicepOutputRe.FindStringSubmatch(l)[1], Start: i + 1, End: i + 1}
		}
		if f != nil {
			f.File = file
			out = append(out, f)
		}
	}
	names := map[string]bool{}
	for _, f := range out {
		names[f.Name] = true
	}
	for _, f := range out {
		seen := map[string]bool{}
		for _, w := range wordRe.FindAllString(strings.Join(lines[f.Start-1:min(f.End, len(lines))], "\n"), -1) {
			if names[w] && w != f.Name && !seen[w] {
				seen[w] = true
				f.Refs = append(f.Refs, w)
			}
		}
	}
	return out
}

func braceEndOrSelf(lines []string, i int) int {
	if strings.Contains(lines[i], "{") {
		return braceEnd(lines, i)
	}
	return i
}

var (
	wfJobRe    = regexp.MustCompile(`^  ([A-Za-z0-9_][\w-]*):\s*$`)
	wfOnRe     = regexp.MustCompile(`^on:\s*(.*)$`)
	wfNeedsRe  = regexp.MustCompile(`^\s+needs:\s*\[?([\w\s,-]+)\]?\s*$`)
	wfDeployRe = regexp.MustCompile(`(?i)uses:\s*\S*(deploy|publish|upload-artifact|release)\S*|run:.*\b(az (webapp|functionapp|deployment)|kubectl apply|helm (upgrade|install)|terraform apply|docker push|npm publish|dotnet nuget push)\b`)
)

func workflowJobs(file string, lines []string) []*Func {
	var out []*Func
	inJobs := false
	var trig []string
	for i, l := range lines {
		if m := wfOnRe.FindStringSubmatch(l); m != nil {
			rest := strings.Trim(strings.TrimSpace(m[1]), "[]")
			if rest != "" {
				trig = append(trig, strings.Fields(strings.ReplaceAll(rest, ",", " "))...)
			} else {
				for j := i + 1; j < len(lines) && (strings.HasPrefix(lines[j], " ") || strings.TrimSpace(lines[j]) == ""); j++ {
					if m2 := wfJobRe.FindStringSubmatch(lines[j]); m2 != nil {
						trig = append(trig, m2[1])
					}
				}
			}
			continue
		}
		if strings.HasPrefix(l, "jobs:") {
			inJobs = true
			continue
		}
		if inJobs && len(l) > 0 && l[0] != ' ' && l[0] != '#' {
			inJobs = false
		}
		if inJobs {
			if m := wfJobRe.FindStringSubmatch(l); m != nil {
				out = append(out, &Func{File: file, Kind: "job", Name: m[1], Start: i + 1})
			}
		}
	}
	for k, f := range out {
		if k+1 < len(out) {
			f.End = out[k+1].Start - 1
		} else {
			f.End = len(lines)
		}
		for _, l := range lines[f.Start-1 : f.End] {
			if m := wfNeedsRe.FindStringSubmatch(l); m != nil {
				for _, n := range strings.Fields(strings.ReplaceAll(m[1], ",", " ")) {
					f.Refs = append(f.Refs, n)
				}
			}
			if wfDeployRe.MatchString(l) {
				f.Outputs = append(f.Outputs, RefCand{Kind: "deploy", What: "deploys or publishes from job " + f.Name, File: file, Line: f.Start})
				break
			}
		}
	}
	if len(trig) > 0 && len(out) > 0 {
		t := &Func{File: file, Kind: "trigger", Name: "on: " + strings.Join(trig, ", "), Start: 1, End: 1,
			Entry: &RefCand{Kind: "trigger", What: "workflow trigger on " + strings.Join(trig, ", "), File: file, Line: 1}}
		out = append(out, t)
	}
	return out
}

// ---- Outputs found in function bodies

type outPattern struct {
	re   *regexp.Regexp
	kind string
	what func(m []string) string
}

var outPatterns = []outPattern{
	{regexp.MustCompile(`\b(INSERT\s+INTO|UPDATE|DELETE\s+FROM|MERGE\s+INTO)\s+([\w."\[\]]+)`), "db", func(m []string) string {
		return "writes table " + strings.Trim(m[2], `"[]`)
	}},
	{regexp.MustCompile(`\.SaveChanges(?:Async)?\(`), "db", func([]string) string { return "writes the database (SaveChanges)" }},
	{regexp.MustCompile(`\b\w*(?:[Rr]epo(?:sitory)?|[Dd]b|DB|[Cc]ontext|[Cc]ollection|[Ss]tore|[Tt]able)\w*\.(Insert|Update|Delete|Upsert|Save|Add|Remove|Put)\w*\(`), "db", func(m []string) string {
		return "writes via " + strings.TrimSuffix(strings.Fields(strings.ReplaceAll(m[0], ".", " "))[0], "(")
	}},
	{regexp.MustCompile(`\.(Exec|ExecContext|ExecuteNonQuery|ExecuteNonQueryAsync)\(`), "db", func([]string) string { return "runs a SQL statement" }},
	{regexp.MustCompile(`\.(Publish|PublishAsync|SendAsync|SendMessage|SendMessageAsync|Emit|emit|Produce|produce|Enqueue|EnqueueAsync|PutRecord|publish)\(\s*(?:new\s+)?([\w.<>]+)?`), "event", func(m []string) string {
		if m[2] != "" {
			return "publishes " + m[2]
		}
		return "publishes a message"
	}},
	{regexp.MustCompile(`\b(?:http\.(?:Get|Post|NewRequest\w*)|HttpClient|httpClient\.\w+|fetch|axios\.\w+|requests\.(?:get|post|put|delete|patch)|RestTemplate|WebClient|PostAsJsonAsync|GetAsync|PostAsync|PutAsync|DeleteAsync)\b`), "call", func([]string) string { return "calls an external HTTP service" }},
	{regexp.MustCompile(`\b(?:os\.WriteFile|ioutil\.WriteFile|File\.Write\w*|fs\.writeFile\w*|Files\.write|WriteAllText|WriteAllBytes)\b`), "file", func([]string) string { return "writes a file" }},
	{regexp.MustCompile(`\breturn\s+(?:Ok|Created|Accepted|NoContent|BadRequest|NotFound|Conflict|StatusCode|Results\.\w+|new\s+JsonResult|ResponseEntity|Response)\w*\(|\bres\.(?:json|send|status)\(|\bc\.JSON\(|\bw\.WriteHeader\(|json\.NewEncoder\(w\)|\bjsonify\(`), "response", func([]string) string { return "returns an HTTP response" }},
}

func detectOutputs(f *Func) {
	seen := map[string]bool{}
	for i, l := range f.body {
		l = stripCommentOnly(l)
		for _, p := range outPatterns {
			if m := p.re.FindStringSubmatch(l); m != nil {
				what := p.what(m)
				k := p.kind + "|" + what
				if seen[k] {
					continue
				}
				seen[k] = true
				f.Outputs = append(f.Outputs, RefCand{Kind: p.kind, What: what, File: f.File, Line: f.Start + i})
			}
		}
	}
}

func stripCommentOnly(s string) string {
	t := strings.TrimSpace(s)
	if strings.HasPrefix(t, "//") || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "*") || strings.HasPrefix(t, "/*") {
		return ""
	}
	return s
}

func describe(f *Func) string { return fmt.Sprintf("%s:%d %s", f.File, f.Start, f.Label()) }

// ---- Route and command registrations (Go, C# minimal APIs, cobra)

var (
	goRegRe    = regexp.MustCompile(`\.(GET|POST|PUT|DELETE|PATCH|Get|Post|Put|Delete|Patch|Handle|HandleFunc)\(\s*"([^"]+)"\s*,\s*(.+)$`)
	csRegRe    = regexp.MustCompile(`\.Map(Get|Post|Put|Delete|Patch)\(\s*"([^"]+)"\s*,\s*(.+)$`)
	cobraUseRe = regexp.MustCompile(`Use:\s*"([^"]+)"`)
	cobraRunRe = regexp.MustCompile(`Run[E]?:\s*([\w.]+)\s*,?\s*$`)
)

func registrationFuncs(file string, lines []string) []*Func {
	var out []*Func
	add := func(i int, what, rest string) {
		f := &Func{File: file, Name: what, Kind: "func", Start: i + 1, End: braceEnd(lines, i) + 1,
			Entry: &RefCand{Kind: "route", What: what, File: file, Line: i + 1}}
		inline := strings.Contains(rest, "=>") || strings.Contains(rest, "func(") || strings.Contains(rest, "function")
		if !inline {
			h := lastIdent(rest)
			if h == "" {
				return
			}
			f.Kind, f.Calls, f.End = "registration", []string{h}, i+1
		}
		out = append(out, f)
	}
	for i, l := range lines {
		if m := goRegRe.FindStringSubmatch(l); m != nil {
			method, p := strings.ToUpper(m[1]), m[2]
			if method == "HANDLE" || method == "HANDLEFUNC" {
				method = "ANY"
				if parts := strings.Fields(p); len(parts) == 2 {
					method, p = parts[0], parts[1]
				}
			}
			add(i, method+" "+p, m[3])
		} else if m := csRegRe.FindStringSubmatch(l); m != nil {
			add(i, strings.ToUpper(m[1])+" "+m[2], m[3])
		}
		if m := cobraUseRe.FindStringSubmatch(l); m != nil {
			for j := i; j < len(lines) && j < i+12; j++ {
				if r := cobraRunRe.FindStringSubmatch(lines[j]); r != nil && !strings.HasPrefix(r[1], "func") {
					cmd := strings.Fields(m[1])[0]
					out = append(out, &Func{File: file, Name: "cmd " + cmd, Kind: "registration", Start: i + 1, End: i + 1,
						Calls: []string{lastSegment(r[1])}, Entry: &RefCand{Kind: "cli", What: "command " + cmd, File: file, Line: i + 1}})
					break
				}
			}
		}
	}
	return out
}

func lastSegment(s string) string {
	if i := strings.LastIndex(s, "."); i >= 0 {
		return s[i+1:]
	}
	return s
}
