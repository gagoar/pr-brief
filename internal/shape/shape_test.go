package shape

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type repo struct {
	t   *testing.T
	dir string
}

func newRepo(t *testing.T) *repo {
	t.Helper()
	r := &repo{t: t, dir: t.TempDir()}
	r.git("init", "-q", "-b", "main")
	r.git("config", "user.email", "t@example.com")
	r.git("config", "user.name", "T")
	r.git("config", "commit.gpgsign", "false")
	return r
}

func (r *repo) git(args ...string) string {
	r.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = r.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func (r *repo) write(files map[string]string) {
	r.t.Helper()
	for p, c := range files {
		full := filepath.Join(r.dir, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			r.t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(c), 0o644); err != nil {
			r.t.Fatal(err)
		}
	}
}

func (r *repo) commit(msg string) {
	r.git("add", "-A")
	r.git("commit", "-q", "-m", msg)
}

// base commits files on main, then branches for the change.
func (r *repo) base(files map[string]string) {
	r.write(files)
	r.commit("base")
	r.git("checkout", "-q", "-b", "feature")
}

func (r *repo) analyze(title string) *Report {
	r.t.Helper()
	rep, err := Analyze(Options{Dir: r.dir, Base: "main", Title: title})
	if err != nil {
		r.t.Fatal(err)
	}
	return rep
}

func labels(fl FlowOut) map[string]string {
	m := map[string]string{}
	for _, f := range fl.Functions {
		m[f.Label] = f.Status
	}
	return m
}

func hasEdge(fl FlowOut, from, to string) bool {
	for _, e := range fl.Edges {
		if e.From == from && e.To == to {
			return true
		}
	}
	return false
}

func nodeOf(fl FlowOut, label string) string {
	for _, f := range fl.Functions {
		if f.Label == label {
			return f.Node
		}
	}
	return ""
}

const controllerCS = `namespace App.Invites
{
    [Route("api/[controller]")]
    public class InviteController
    {
        private readonly InviteService service;

        [HttpPost("create")]
        public async Task<IActionResult> Create(InviteRequest request)
        {
            var result = await service.Handle(request);
            return Ok(result);
        }
    }
}
`

const serviceBeforeCS = `namespace App.Invites
{
    public class InviteService
    {
        public async Task<Invite> Handle(InviteRequest request)
        {
            var invite = new Invite(request.Email);
            return invite;
        }
    }
}
`

const serviceAfterCS = `namespace App.Invites
{
    public class InviteService
    {
        public async Task<Invite> Handle(InviteRequest request)
        {
            var invite = new Invite(request.Email, NextCode());
            repository.Insert(invite);
            await bus.PublishAsync(new InviteCreated(invite.Id));
            return invite;
        }

        private string NextCode()
        {
            return Guid.NewGuid().ToString("N");
        }
    }
}
`

func TestSingleEndpointFeature(t *testing.T) {
	r := newRepo(t)
	r.base(map[string]string{"src/Invites/InviteController.cs": controllerCS, "src/Invites/InviteService.cs": serviceBeforeCS})
	r.write(map[string]string{"src/Invites/InviteService.cs": serviceAfterCS})
	r.commit("add invite code")

	rep := r.analyze("")
	if len(rep.Flows) != 1 {
		t.Fatalf("flows = %d, want 1: %+v", len(rep.Flows), rep.Flows)
	}
	fl := rep.Flows[0]
	if got := rep.Refs[fl.Inputs[0]]; got.What != "POST /api/invite/create" || got.Kind != "route" {
		t.Errorf("input = %+v", got)
	}
	want := map[string]string{
		"InviteService.Handle":    "modified",
		"InviteService.NextCode":  "added",
		"InviteController.Create": "context",
	}
	got := labels(fl)
	for l, st := range want {
		if got[l] != st {
			t.Errorf("%s = %q, want %q (all: %v)", l, got[l], st, got)
		}
	}
	var kinds []string
	for _, id := range fl.Outputs {
		kinds = append(kinds, rep.Refs[id].Kind+":"+rep.Refs[id].What)
	}
	joined := strings.Join(kinds, "|")
	if !strings.Contains(joined, "db:writes via repository") || !strings.Contains(joined, "event:publishes InviteCreated") {
		t.Errorf("outputs = %v", kinds)
	}
	create, handle, next := nodeOf(fl, "InviteController.Create"), nodeOf(fl, "InviteService.Handle"), nodeOf(fl, "InviteService.NextCode")
	if !hasEdge(fl, fl.Inputs[0], create) || !hasEdge(fl, create, handle) || !hasEdge(fl, handle, next) {
		t.Errorf("edges = %+v", fl.Edges)
	}
	if fl.Collapsed != "function" {
		t.Errorf("collapsed = %s", fl.Collapsed)
	}
	if len(rep.Unplaced) != 0 || len(rep.Dropped) != 0 {
		t.Errorf("unplaced=%v dropped=%v", rep.Unplaced, rep.Dropped)
	}
	if !rep.SmallPR {
		t.Error("2 code files is a small PR")
	}
}

func TestNoiseIsCountedNotDrawn(t *testing.T) {
	r := newRepo(t)
	r.base(map[string]string{"src/Invites/InviteController.cs": controllerCS, "src/Invites/InviteService.cs": serviceBeforeCS, "README.md": "x\n", "package-lock.json": "{}\n"})
	r.write(map[string]string{
		"src/Invites/InviteService.cs": serviceAfterCS,
		"README.md":                    "x\ny\n",
		"package-lock.json":            "{\"a\":1}\n",
		"tests/InviteTests.cs":         "class T {}\n",
		"docs/guide.md":                "# g\n",
		"assets/logo.png":              "\x89PNG\x00\x01",
		"config/settings.json":         "{}\n",
	})
	r.commit("work")
	rep := r.analyze("")
	n := rep.Noise
	if n.Tests != 1 || n.Docs != 2 || n.Generated != 1 || n.Other != 2 {
		t.Errorf("noise = %+v", n)
	}
	if rep.CodeFiles != 1 {
		t.Errorf("code files = %d", rep.CodeFiles)
	}
	for _, f := range rep.FilesRanked {
		if strings.Contains(f.File, "README") || strings.Contains(f.File, "lock") {
			t.Errorf("noise file ranked: %s", f.File)
		}
	}
}

func TestConfigOnlyPR(t *testing.T) {
	r := newRepo(t)
	r.base(map[string]string{"package.json": "{\"a\":1}\n"})
	r.write(map[string]string{"package.json": "{\"a\":2}\n"})
	r.commit("bump")
	rep := r.analyze("")
	if !rep.ConfigOnly || len(rep.Flows) != 0 || rep.CodeFiles != 0 {
		t.Errorf("config-only: %+v", rep)
	}
}

func TestTwoEndpointsShareAFunction(t *testing.T) {
	ctrl := `public class Api
{
    [HttpGet("a")]
    public string GetA()
    {
        return Shared();
    }

    [HttpGet("b")]
    public string GetB()
    {
        return Shared();
    }

    public string Shared()
    {
        return "x";
    }
}
`
	r := newRepo(t)
	r.base(map[string]string{"src/Api.cs": ctrl})
	r.write(map[string]string{"src/Api.cs": strings.Replace(ctrl, `return "x";`, `return "y";`, 1)})
	r.commit("change shared")
	rep := r.analyze("")
	if len(rep.Flows) != 2 {
		t.Fatalf("flows = %d, want 2", len(rep.Flows))
	}
	for _, fl := range rep.Flows {
		if labels(fl)["Api.Shared"] != "modified" {
			t.Errorf("each flow shows Api.Shared(): %v", labels(fl))
		}
	}
	if rep.Refs["I1"].What == rep.Refs["I2"].What {
		t.Errorf("two different inputs expected: %+v", rep.Refs)
	}
}

func TestFlowsBeyondTheCapAreDropped(t *testing.T) {
	var base, head strings.Builder
	base.WriteString("public class Api\n{\n")
	head.WriteString("public class Api\n{\n")
	for i := 1; i <= 5; i++ {
		fmt.Fprintf(&base, "    [HttpGet(\"r%d\")]\n    public string Get%d()\n    {\n        return \"a\";\n    }\n\n", i, i)
		fmt.Fprintf(&head, "    [HttpGet(\"r%d\")]\n    public string Get%d()\n    {\n        return \"b%d\";\n    }\n\n", i, i, i)
	}
	base.WriteString("}\n")
	head.WriteString("}\n")
	r := newRepo(t)
	r.base(map[string]string{"src/Api.cs": base.String()})
	r.write(map[string]string{"src/Api.cs": head.String()})
	r.commit("all five")
	rep := r.analyze("")
	if len(rep.Flows) != 3 || len(rep.Dropped) != 2 {
		t.Fatalf("flows=%d dropped=%d", len(rep.Flows), len(rep.Dropped))
	}
	if len(rep.Warnings) == 0 || !strings.Contains(rep.Warnings[0], "splitting") {
		t.Errorf("warnings = %v", rep.Warnings)
	}
	for _, id := range []string{"I4", "I5", "O4", "O5"} {
		if _, ok := rep.Refs[id]; ok {
			t.Errorf("a dropped flow must not consume reference %s: %v", id, rep.Refs)
		}
	}
	if len(rep.Refs) != 6 { // I1..I3 and O1..O3
		t.Errorf("want 3 inputs and 3 outputs for the kept flows, got %v", rep.Refs)
	}
}

func TestUnplacedFunctions(t *testing.T) {
	r := newRepo(t)
	r.base(map[string]string{"src/util.go": "package src\n\nfunc helper() string {\n\treturn \"a\"\n}\n"})
	r.write(map[string]string{"src/util.go": "package src\n\nfunc helper() string {\n\treturn \"b\"\n}\n"})
	r.commit("tweak")
	rep := r.analyze("")
	if len(rep.Flows) != 0 || len(rep.Unplaced) != 1 || rep.Unplaced[0].Label != "helper" {
		t.Errorf("flows=%d unplaced=%+v", len(rep.Flows), rep.Unplaced)
	}
}

func TestRemovedFunctions(t *testing.T) {
	r := newRepo(t)
	r.base(map[string]string{"src/util.go": "package src\n\nfunc keep() int { return 1 }\n\nfunc gone() int {\n\treturn 2\n}\n"})
	r.write(map[string]string{"src/util.go": "package src\n\nfunc keep() int { return 1 }\n"})
	r.commit("remove")
	rep := r.analyze("")
	if len(rep.Removed) != 1 || rep.Removed[0].Label != "gone" || rep.Removed[0].Status != "removed" {
		t.Errorf("removed = %+v", rep.Removed)
	}
}

func TestGoRegistrationBindsHandler(t *testing.T) {
	src := func(code string) string {
		return "package api\n\nimport \"net/http\"\n\nfunc Routes(mux *http.ServeMux) {\n\tmux.HandleFunc(\"GET /health\", health)\n}\n\nfunc health(w http.ResponseWriter, r *http.Request) {\n\tw.WriteHeader(" + code + ")\n}\n"
	}
	r := newRepo(t)
	r.base(map[string]string{"api/server.go": src("200")})
	r.write(map[string]string{"api/server.go": src("204")})
	r.commit("health 204")
	rep := r.analyze("")
	if len(rep.Flows) != 1 {
		t.Fatalf("flows = %d", len(rep.Flows))
	}
	fl := rep.Flows[0]
	if rep.Refs[fl.Inputs[0]].What != "GET /health" || labels(fl)["health"] != "modified" {
		t.Errorf("refs=%v labels=%v", rep.Refs, labels(fl))
	}
	if len(fl.Outputs) != 1 || rep.Refs[fl.Outputs[0]].Kind != "response" {
		t.Errorf("outputs = %v", fl.Outputs)
	}
}

func TestPythonRoute(t *testing.T) {
	src := func(v string) string {
		return "from flask import Flask\napp = Flask(__name__)\n\n@app.route(\"/items\", methods=[\"POST\"])\ndef create_item():\n    item = build()\n    return jsonify(item)\n\ndef build():\n    return {\"a\": " + v + "}\n"
	}
	r := newRepo(t)
	r.base(map[string]string{"app/views.py": src("1")})
	r.write(map[string]string{"app/views.py": src("2")})
	r.commit("change build")
	rep := r.analyze("")
	if len(rep.Flows) != 1 || rep.Refs[rep.Flows[0].Inputs[0]].What != "POST /items" {
		t.Fatalf("flows=%+v refs=%v", rep.Flows, rep.Refs)
	}
	if labels(rep.Flows[0])["build"] != "modified" || labels(rep.Flows[0])["create_item"] != "context" {
		t.Errorf("labels = %v", labels(rep.Flows[0]))
	}
}

func TestExpressInlineRoute(t *testing.T) {
	src := func(v string) string {
		return "const app = makeApp();\n\napp.get('/ping', (req, res) => {\n  res.json({ ok: " + v + " });\n});\n"
	}
	r := newRepo(t)
	r.base(map[string]string{"src/app.ts": src("true")})
	r.write(map[string]string{"src/app.ts": src("1")})
	r.commit("ping")
	rep := r.analyze("")
	if len(rep.Flows) != 1 || rep.Refs[rep.Flows[0].Inputs[0]].What != "GET /ping" {
		t.Fatalf("flows=%+v refs=%v", rep.Flows, rep.Refs)
	}
}

func TestTerraformFlow(t *testing.T) {
	before := "variable \"sku\" {\n  default = \"S1\"\n}\n\nresource \"azurerm_service_plan\" \"plan\" {\n  name = \"plan\"\n  sku_name = var.sku\n}\n\noutput \"plan_id\" {\n  value = azurerm_service_plan.plan.id\n}\n"
	after := strings.Replace(before, "sku_name = var.sku", "sku_name = var.sku\n  worker_count = 2", 1)
	r := newRepo(t)
	r.base(map[string]string{"infra/main.tf": before})
	r.write(map[string]string{"infra/main.tf": after})
	r.commit("scale")
	rep := r.analyze("")
	if len(rep.Flows) != 1 {
		t.Fatalf("flows = %d (%+v)", len(rep.Flows), rep.Warnings)
	}
	fl := rep.Flows[0]
	if labels(fl)["azurerm_service_plan.plan"] != "modified" {
		t.Errorf("labels = %v", labels(fl))
	}
	if !strings.Contains(rep.Refs[fl.Inputs[0]].What, "variable sku") || !strings.Contains(rep.Refs[fl.Outputs[0]].What, "plan_id") {
		t.Errorf("refs = %v", rep.Refs)
	}
}

func TestWorkflowJobs(t *testing.T) {
	before := "name: ci\non:\n  push:\n  pull_request:\njobs:\n  build:\n    runs-on: ubuntu-latest\n    steps:\n      - run: go build\n  deploy:\n    needs: build\n    runs-on: ubuntu-latest\n    steps:\n      - run: az webapp deploy\n"
	after := strings.Replace(before, "go build", "go build ./...", 1)
	r := newRepo(t)
	r.base(map[string]string{".github/workflows/ci.yml": before})
	r.write(map[string]string{".github/workflows/ci.yml": after})
	r.commit("build all")
	rep := r.analyze("")
	if len(rep.Flows) != 1 {
		t.Fatalf("flows = %d", len(rep.Flows))
	}
	fl := rep.Flows[0]
	if labels(fl)["build"] != "modified" || !strings.Contains(rep.Refs[fl.Inputs[0]].What, "push") {
		t.Errorf("labels=%v refs=%v", labels(fl), rep.Refs)
	}
}

func TestCollapseToModules(t *testing.T) {
	// One endpoint calls 12 functions spread over 3 classes: too many nodes for one diagram.
	var sb strings.Builder
	sb.WriteString("public class Api\n{\n    [HttpPost(\"run\")]\n    public string Run()\n    {\n")
	for c := 0; c < 3; c++ {
		for i := 0; i < 4; i++ {
			fmt.Fprintf(&sb, "        C%d.Step%d%d();\n", c, c, i)
		}
	}
	sb.WriteString("        return \"\";\n    }\n}\n")
	mk := func(version string) map[string]string {
		files := map[string]string{"src/Api.cs": sb.String()}
		for c := 0; c < 3; c++ {
			var b strings.Builder
			fmt.Fprintf(&b, "public class C%d\n{\n", c)
			for i := 0; i < 4; i++ {
				fmt.Fprintf(&b, "    public static void Step%d%d()\n    {\n        Console.Write(\"%s\");\n    }\n\n", c, i, version)
			}
			b.WriteString("}\n")
			files[fmt.Sprintf("src/C%d.cs", c)] = b.String()
		}
		return files
	}
	r := newRepo(t)
	r.base(mk("a"))
	r.write(mk("b"))
	r.commit("touch every step")
	rep := r.analyze("")
	if len(rep.Flows) != 1 {
		t.Fatalf("flows = %d", len(rep.Flows))
	}
	fl := rep.Flows[0]
	if fl.Collapsed != "module" {
		t.Errorf("12 changed functions in 3 classes must collapse to modules; got %s with %d functions", fl.Collapsed, len(fl.Functions))
	}
	if total := len(fl.Inputs) + len(fl.Functions) + len(fl.Outputs); total > 9 {
		t.Errorf("%d nodes after collapse; the limit is 9", total)
	}
}

func TestMoreFoldWhenModulesAreStillTooMany(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("public class Api\n{\n    [HttpPost(\"run\")]\n    public string Run()\n    {\n")
	for c := 0; c < 12; c++ {
		fmt.Fprintf(&sb, "        C%d.Go();\n", c)
	}
	sb.WriteString("        return \"\";\n    }\n}\n")
	mk := func(v string) map[string]string {
		files := map[string]string{"src/Api.cs": sb.String()}
		for c := 0; c < 12; c++ {
			files[fmt.Sprintf("src/C%d.cs", c)] = fmt.Sprintf("public class C%d\n{\n    public static void Go()\n    {\n        Console.Write(\"%s\");\n    }\n}\n", c, v)
		}
		return files
	}
	r := newRepo(t)
	r.base(mk("a"))
	r.write(mk("b"))
	r.commit("touch twelve classes")
	rep := r.analyze("")
	if len(rep.Flows) == 0 {
		b, _ := json.MarshalIndent(rep, "", " ")
		t.Fatalf("no flows: %s", b)
	}
	fl := rep.Flows[0]
	if fl.Collapsed != "more" {
		t.Fatalf("collapsed = %s", fl.Collapsed)
	}
	if total := len(fl.Inputs) + len(fl.Functions) + len(fl.Outputs); total > 9 {
		t.Errorf("%d nodes", total)
	}
	found := false
	for _, f := range fl.Functions {
		if strings.HasPrefix(f.Label, "+") && strings.HasSuffix(f.Label, " more") {
			found = true
		}
	}
	if !found {
		t.Errorf("no '+N more' node: %+v", fl.Functions)
	}
}

func TestExtras(t *testing.T) {
	r := newRepo(t)
	r.base(map[string]string{
		"src/OrderState.cs":     "public enum OrderState\n{\n    New,\n    Paid\n}\n",
		"db/migrations/001.sql": "CREATE TABLE a (id int);\n",
	})
	r.write(map[string]string{
		"src/OrderState.cs":     "public enum OrderState\n{\n    New,\n    Paid,\n    Shipped\n}\n",
		"db/migrations/001.sql": "CREATE TABLE a (id int);\nALTER TABLE a ADD COLUMN b int;\nALTER TABLE a ADD COLUMN c int;\n",
	})
	r.commit("states and schema")
	rep := r.analyze("fix: orders get stuck")
	types := map[string]bool{}
	for _, e := range rep.Extras {
		types[e.Type] = true
	}
	if !types["erDiagram"] || !types["stateDiagram-v2"] {
		t.Errorf("extras = %+v", rep.Extras)
	}
}

func TestFilesRankedByRisk(t *testing.T) {
	r := newRepo(t)
	r.base(map[string]string{"src/a.go": "package a\n\nfunc A() int { return 1 }\n", "src/auth/token.go": "package auth\n\nfunc Token() int { return 1 }\n"})
	r.write(map[string]string{
		"src/a.go":          "package a\n\nfunc A() int { return 2 }\n",
		"src/auth/token.go": "package auth\n\nfunc Token() int {\n\tmu.Lock()\n\treturn 2\n}\n",
	})
	r.commit("change")
	rep := r.analyze("")
	if len(rep.FilesRanked) != 2 || rep.FilesRanked[0].File != "src/auth/token.go" {
		t.Fatalf("ranking = %+v", rep.FilesRanked)
	}
	if len(rep.FilesRanked[0].Reasons) == 0 {
		t.Error("risky file needs reasons")
	}
}

func TestWorkingTreeIncludesUntracked(t *testing.T) {
	r := newRepo(t)
	r.base(map[string]string{"src/a.go": "package a\n\nfunc A() int { return 1 }\n"})
	r.write(map[string]string{"src/b.go": "package a\n\nfunc B() int {\n\treturn 2\n}\n"})
	rep, err := Analyze(Options{Dir: r.dir, Base: "main", WorkingTree: true})
	if err != nil {
		t.Fatal(err)
	}
	if rep.CodeFiles != 1 || len(rep.Unplaced) != 1 || rep.Unplaced[0].Label != "B" || rep.Unplaced[0].Status != "added" {
		t.Errorf("working tree: %+v", rep)
	}
}

func TestRiskIgnoresStringsAndCapsReasons(t *testing.T) {
	r := newRepo(t)
	r.base(map[string]string{
		"src/lint.go": "package src\n\nfunc Rules() int { return 1 }\n",
		"src/pay.go":  "package src\n\nfunc Pay() int { return 1 }\n",
	})
	r.write(map[string]string{
		// Talks about risk words only inside a string and a comment.
		"src/lint.go": "package src\n\n// retry lock mutex\nvar re = `lock|mutex|retry|timeout|secret|password`\n\nfunc Rules() int { return 2 }\n",
		// Really takes a lock and retries.
		"src/pay.go": "package src\n\nfunc Pay() int {\n\tmu.Lock()\n\tfor retry := 0; retry < 3; retry++ {\n\t}\n\treturn 2\n}\n",
	})
	r.commit("risk")
	rep := r.analyze("")
	rank := map[string]RankedFile{}
	for _, f := range rep.FilesRanked {
		rank[f.File] = f
	}
	if rank["src/lint.go"].Score != 0 {
		t.Errorf("words in strings and comments must not score: %+v", rank["src/lint.go"])
	}
	if rank["src/pay.go"].Score < 2 || rep.FilesRanked[0].File != "src/pay.go" {
		t.Errorf("real lock and retry logic must rank first: %+v", rep.FilesRanked)
	}
}

func TestProseIsNotSQL(t *testing.T) {
	r := newRepo(t)
	r.base(map[string]string{"src/a.go": "package src\n\nfunc A() string { return \"a\" }\n"})
	r.write(map[string]string{"src/a.go": "package src\n\n// A will update that table later.\nfunc A() string {\n\tmsg := \"update that\"\n\treturn msg\n}\n"})
	r.commit("prose")
	rep := r.analyze("")
	for id, ref := range rep.Refs {
		if ref.Kind == "db" {
			t.Errorf("%s: prose matched as SQL: %+v", id, ref)
		}
	}
}

func TestFlowWithoutEffectsGetsAReturnOutput(t *testing.T) {
	r := newRepo(t)
	r.base(map[string]string{"api/server.go": "package api\n\nimport \"net/http\"\n\nfunc Routes(mux *http.ServeMux) {\n\tmux.HandleFunc(\"GET /sum\", sum)\n}\n\nfunc sum(a, b int) int {\n\treturn a + b\n}\n"})
	r.write(map[string]string{"api/server.go": "package api\n\nimport \"net/http\"\n\nfunc Routes(mux *http.ServeMux) {\n\tmux.HandleFunc(\"GET /sum\", sum)\n}\n\nfunc sum(a, b int) int {\n\treturn a + b + 1\n}\n"})
	r.commit("sum")
	rep := r.analyze("")
	if len(rep.Flows) != 1 || len(rep.Flows[0].Outputs) != 1 {
		t.Fatalf("a flow always has at least one Output: %+v", rep.Flows)
	}
	o := rep.Refs[rep.Flows[0].Outputs[0]]
	if o.Kind != "return" || !strings.Contains(o.What, "result returned by") {
		t.Errorf("fallback output = %+v", o)
	}
	fl := rep.Flows[0]
	if !hasEdge(fl, nodeOf(fl, "sum"), fl.Outputs[0]) {
		t.Errorf("the output must be linked: %+v", fl.Edges)
	}
}

func TestWorkflowOutputDoesNotClaimADeploy(t *testing.T) {
	before := "name: ci\non: [push]\njobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - run: go test ./...\n"
	r := newRepo(t)
	r.base(map[string]string{".github/workflows/ci.yml": before})
	r.write(map[string]string{".github/workflows/ci.yml": strings.Replace(before, "go test ./...", "go test -race ./...", 1)})
	r.commit("race")
	rep := r.analyze("")
	if len(rep.Flows) != 1 {
		t.Fatalf("flows = %d", len(rep.Flows))
	}
	o := rep.Refs[rep.Flows[0].Outputs[0]]
	if strings.Contains(o.What, "deploy") || o.Kind != "result" {
		t.Errorf("a test job does not deploy: %+v", o)
	}
}

func TestScriptRunByWorkflowGetsItsOwnFlow(t *testing.T) {
	wf := "name: sync\non:\n  schedule:\n    - cron: \"*/15 * * * *\"\njobs:\n  sync:\n    runs-on: ubuntu-latest\n    steps:\n      - run: node scripts/sync.mjs\n"
	before := "async function main() {\n  console.log(1);\n}\nmain();\n"
	after := "function guard(x) {\n  return Array.isArray(x);\n}\nasync function main() {\n  console.log(guard([]));\n}\nmain();\n"
	r := newRepo(t)
	r.base(map[string]string{".github/workflows/sync.yml": wf, "scripts/sync.mjs": before})
	r.write(map[string]string{"scripts/sync.mjs": after, ".github/workflows/sync.yml": strings.Replace(wf, "sync.mjs", "sync.mjs --strict", 1)})
	r.commit("guard")
	rep := r.analyze("")
	if len(rep.Unplaced) != 0 {
		t.Fatalf("unplaced = %+v", rep.Unplaced)
	}
	var found bool
	for _, fl := range rep.Flows {
		if labels(fl)["guard"] == "added" && strings.Contains(rep.Refs[fl.Inputs[0]].What, "scripts/sync.mjs") {
			found = true
		}
	}
	if !found {
		t.Errorf("no flow for the script: %+v", rep.Flows)
	}
}
