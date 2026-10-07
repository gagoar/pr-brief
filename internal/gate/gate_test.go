package gate

import (
	"strings"
	"testing"

	"github.com/gagoar/pr-brief/internal/convention"
	dg "github.com/gagoar/pr-brief/internal/diagram"
	"github.com/gagoar/pr-brief/internal/host"
	"github.com/gagoar/pr-brief/internal/theme"
)

const goodTemplate = "<!-- pr-brief:begin v1 style=ste+iceberg theme=github-dark -->\n" +
	"## Brief\n" +
	"Admins can now invite people by email. The service creates a one-time code and tells the mailer. The code expires after seven days.\n" +
	"\n" +
	"## Change map\n" +
	"### Flow 1: Input 1 -> invite created\n" +
	"```mermaid\n" +
	"@@DIAGRAM@@\n" +
	"```\n" +
	"| Ref | What | Detail |\n" +
	"|---|---|---|\n" +
	"| Input 1 | `POST /invite-code` (new) | The body holds an email and a role. |\n" +
	"| Output 1 | `invites` table | The service adds one row. |\n" +
	"| O2 | `InviteCreated` event | The mailer reads it. |\n" +
	"\n" +
	"green added, amber modified, grey context, red border delicate\n" +
	"\n" +
	"## Review guide\n" +
	"**What changed**:\n" +
	"- `InviteCommand.Handle()` creates the invite and sends the event.\n" +
	"\n" +
	"**Read these first**\n" +
	"| File | Why it is delicate | What to check |\n" +
	"|---|---|---|\n" +
	"| `src/Invite/CodeGenerator.cs` | It makes the secret code. | Check the random source. |\n" +
	"\n" +
	"**Review order**: Start at `InviteCommand.Handle()`, then follow the arrows.\n" +
	"<details><summary>Other changed files (4) · tests: 3 · docs: 1 · generated: 0</summary>\n" +
	"\n- tests/InviteTests.cs\n\n</details>\n" +
	"<!-- pr-brief:end -->\n"

var testFlow = dg.Flow{
	Inputs: []dg.Port{{ID: "I1"}},
	Nodes: []dg.Node{
		{ID: "F1", Label: "InviteCommand.Handle", Status: "added"},
		{ID: "F2", Label: "CodeGenerator.Next", Status: "modified", RiskScore: 3},
		{ID: "F3", Label: "UserQuery.Get", Status: "context"},
	},
	Outputs: []dg.Port{{ID: "O1", Kind: "db"}, {ID: "O2", Kind: "event"}},
	Edges: []dg.Edge{
		{From: "I1", To: "F1", Status: "new"}, {From: "F1", To: "F2", Status: "existing"},
		{From: "F1", To: "F3", Status: "existing"}, {From: "F1", To: "O1", Status: "new"},
		{From: "F1", To: "O2", Status: "new"},
	},
}

func render(th theme.Theme) string {
	src, err := dg.Render(testFlow, th)
	if err != nil {
		panic(err)
	}
	return src
}

var good = strings.Replace(goodTemplate, "@@DIAGRAM@@", render(theme.Default()), 1)

func has(t *testing.T, r Result, rule, substr string) {
	t.Helper()
	for _, f := range r.Findings {
		if f.Rule == rule && strings.Contains(f.Message, substr) {
			return
		}
	}
	t.Errorf("expected a %q finding containing %q; got %+v", rule, substr, r.Findings)
}

func mutate(old, new string) string {
	if !strings.Contains(good, old) {
		panic("mutate: " + old + " not in good body")
	}
	return strings.Replace(good, old, new, 1)
}

func TestGoodBodyPasses(t *testing.T) {
	r := Check(good, Options{})
	if !r.OK() {
		t.Fatalf("good body failed: %+v", r.Findings)
	}
	if r.Style != "ste+iceberg" {
		t.Errorf("style = %q", r.Style)
	}
}

func TestSkipMarker(t *testing.T) {
	r := Check("> pr-brief skipped: hotfix, no time\n", Options{})
	if !r.OK() || !r.Skipped || r.SkipReason != "hotfix, no time" {
		t.Errorf("skip: %+v", r)
	}
	has(t, Check("> pr-brief skipped:\n", Options{}), "skip", "needs a reason")
}

func TestMissingAndBrokenMarkers(t *testing.T) {
	has(t, Check("just a description", Options{}), "markers", "no pr-brief description")
	has(t, Check("<!-- pr-brief:begin v1 style=ste -->\nno end", Options{}), "markers", "broken")
	has(t, Check(mutate("style=ste+iceberg", "style=hemingway"), Options{}), "markers", "hemingway")
}

func TestRepoStyleOverridesMarker(t *testing.T) {
	r := Check(mutate("style=ste+iceberg", "style=iceberg"), Options{Style: "ste"})
	if r.Style != "ste" {
		t.Errorf("style = %q", r.Style)
	}
}

func TestSectionsMissingOrMisordered(t *testing.T) {
	has(t, Check(mutate("## Review guide", "## Notes"), Options{}), "sections", "exactly")
	swapped := strings.Replace(good, "## Brief", "## Change map X", 1)
	has(t, Check(swapped, Options{}), "sections", "exactly")
}

func TestBrief(t *testing.T) {
	two := mutate("The code expires after seven days.\n", "The code expires after seven days.\n\nA second paragraph.\n")
	has(t, Check(two, Options{}), "brief", "second one")
	long := mutate("Admins can now invite people by email.", "One. Two. Three. Four. Five. Six.")
	has(t, Check(long, Options{}), "brief", "sentences")
}

func TestDiagramRules(t *testing.T) {
	cases := []struct{ name, old, new, rule, want string }{
		{"flowchart", "graph LR\n", "flowchart LR\n", "diagram", "flowchart"},
		{"graph TD", "graph LR\n", "graph TD\n", "diagram", "graph LR"},
		{"click", "  classDef added", "  click F1 \"https://x\"\n  classDef added", "diagram", "click"},
		{"descriptive input", "I1([\"Input 1\"])", "I1([\"POST /invite\"])", "diagram", "word and the number"},
		{"descriptive output", "O1[(\"Output 1\")]", "O1[(\"invites table\")]", "diagram", "word and the number"},
		{"back edge", "  F1 ==> O2\n", "  F1 ==> O2\n  O2 --> F1\n", "diagram", "leaves an Output"},
		{"edge into input", "  F1 ==> O2\n", "  F1 ==> O2\n  F2 --> I1\n", "diagram", "enters an Input"},
		{"missing column", "  subgraph IN[\"Input\"]\n    I1([\"Input 1\"])\n  end\n", "", "diagram", "exactly Input"},
		{"long label", "UserQuery.Get", "AVeryLongFunctionNameThatKeepsGoing.Handle", "diagram", "characters"},
		{"too many context", "F1[\"InviteCommand.Handle\"]:::added\n    F2[\"!CodeGenerator.Next\"]:::risk", "F1[\"InviteCommand.Handle\"]:::context\n    F2[\"!CodeGenerator.Next\"]:::context", "diagram", "3 context nodes"},
		{"no heading", "### Flow 1: Input 1 -> invite created\n", "", "diagram", "heading"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			has(t, Check(mutate(c.old, c.new), Options{}), c.rule, c.want)
		})
	}
}

func TestTooManyNodesAndEdges(t *testing.T) {
	var extra strings.Builder
	for i := 4; i <= 9; i++ {
		extra.WriteString("    F" + string(rune('0'+i)) + "[\"Fn" + string(rune('0'+i)) + "\"]\n")
	}
	body := mutate("  end\n  subgraph OUT", extra.String()+"  end\n  subgraph OUT")
	has(t, Check(body, Options{}), "diagram", "nodes; the limit is 9")

	var edges strings.Builder
	for i := 0; i < 12; i++ {
		edges.WriteString("  F1 --> F3\n")
	}
	has(t, Check(mutate("  F1 ==> O2\n", "  F1 ==> O2\n"+edges.String()), Options{}), "diagram", "edges; the limit is 14")
}

func TestTooManyDiagrams(t *testing.T) {
	start := strings.Index(good, "### Flow 1")
	end := strings.Index(good, "green added")
	flow := good[start:end]
	// Flows 2..4 reuse the same refs through "see Flow 1" rows.
	see := strings.NewReplacer(
		"| Input 1 | `POST /invite-code` (new) | The body holds an email and a role. |", "| Input 1 | see Flow 1 | |",
		"| Output 1 | `invites` table | The service adds one row. |", "| Output 1 | see Flow 1 | |",
		"| O2 | `InviteCreated` event | The mailer reads it. |", "| O2 | see Flow 1 | |",
	)
	var more string
	for n := 2; n <= 4; n++ {
		f := see.Replace(flow)
		f = strings.Replace(f, "### Flow 1", "### Flow "+string(rune('0'+n)), 1)
		more += f
	}
	body := strings.Replace(good, "green added", more+"green added", 1)
	has(t, Check(body, Options{}), "diagram", "4 diagrams; the limit is 3")
}

func TestSeeFlowPointers(t *testing.T) {
	start := strings.Index(good, "### Flow 1")
	end := strings.Index(good, "green added")
	flow2 := strings.Replace(good[start:end], "### Flow 1", "### Flow 2", 1)
	// Redefining instead of pointing is an error.
	has(t, Check(strings.Replace(good, "green added", flow2+"green added", 1), Options{}), "references", "already defined in Flow 1")

	ptr := strings.NewReplacer(
		"| Input 1 | `POST /invite-code` (new) | The body holds an email and a role. |", "| Input 1 | see Flow 1 | |",
		"| Output 1 | `invites` table | The service adds one row. |", "| Output 1 | see Flow 1 | |",
		"| O2 | `InviteCreated` event | The mailer reads it. |", "| O2 | see Flow 1 | |",
	).Replace(flow2)
	r := Check(strings.Replace(good, "green added", ptr+"green added", 1), Options{})
	if !r.OK() {
		t.Errorf("pointer rows should pass: %+v", r.Findings)
	}
	// Flow 2 pointing at itself or the future is wrong.
	bad := strings.Replace(ptr, "see Flow 1 | |\n| Output 1", "see Flow 2 | |\n| Output 1", 1)
	has(t, Check(strings.Replace(good, "green added", bad+"green added", 1), Options{}), "references", "does not define it")
}

func TestReferenceRules(t *testing.T) {
	has(t, Check(mutate("| O2 | `InviteCreated` event | The mailer reads it. |\n", ""), Options{}), "references", "O2 is in the diagram but has no References row")
	has(t, Check(mutate("| O2 |", "| O9 |"), Options{}), "references", "O9 has a row but is not used")
	has(t, Check(mutate("| Output 1 | `invites` table | The service adds one row. |", "| Output 1 | `invites` table | |"), Options{}), "references", "needs both What and Detail")
	has(t, Check(mutate("| Ref | What | Detail |\n|---|---|---|\n", ""), Options{}), "references", "no References table")
}

func TestNoDiagramMarker(t *testing.T) {
	start := strings.Index(good, "### Flow 1")
	end := strings.Index(good, "green added")
	none := strings.Replace(good, good[start:end], "<!-- pr-brief:no-diagram: two files, one config key -->\n", 1)
	if r := Check(none, Options{}); !r.OK() {
		t.Errorf("no-diagram with a reason should pass: %+v", r.Findings)
	}
	has(t, Check(strings.Replace(none, "two files, one config key", "", 1), Options{}), "diagram", "needs a reason")
	empty := strings.Replace(good, good[start:end], "", 1)
	has(t, Check(empty, Options{}), "diagram", "no mermaid diagram")
}

func TestReviewGuide(t *testing.T) {
	has(t, Check(mutate("**Read these first**", "**Read first**"), Options{}), "review", "Read these first")
	has(t, Check(mutate("**Review order**", "**Order**"), Options{}), "review", "Review order")
	has(t, Check(mutate("**What changed**", "**Changes**"), Options{}), "review", "What changed")

	row := "| `src/Invite/CodeGenerator.cs` | It makes the secret code. | Check the random source. |\n"
	has(t, Check(mutate(row, ""), Options{}), "review", "0 rows")
	var many strings.Builder
	for i := 0; i < 8; i++ {
		many.WriteString(row)
	}
	has(t, Check(mutate(row, many.String()), Options{}), "review", "8 rows")
}

func TestPreviousBlock(t *testing.T) {
	ok := good + "\n<!-- pr-brief:previous v1 saved=2026-10-05T14:02Z\nold text\npr-brief:previous:end -->\n"
	if r := Check(ok, Options{}); !r.OK() {
		t.Errorf("valid previous block failed: %+v", r.Findings)
	}
	dash := good + "\n<!-- pr-brief:previous v1 saved=2026-10-05T14:02Z\nold -- text\npr-brief:previous:end -->\n"
	has(t, Check(dash, Options{}), "previous", "close the HTML comment")
	before := "<!-- pr-brief:previous v1 saved=2026-10-05T14:02Z\nold\npr-brief:previous:end -->\n" + good
	has(t, Check(before, Options{}), "previous", "come after")
	has(t, Check(good+"\n<!-- pr-brief:previous v1 saved=x\nno terminator", Options{}), "previous", "malformed")
}

func TestLength(t *testing.T) {
	big := good + strings.Repeat("x", convention.MaxBodyCharsGitHub)
	has(t, Check(big, Options{}), "length", "GitHub rejects")
}

func TestSTELintByStyle(t *testing.T) {
	var got string
	lint := func(p string) []string { got = p; return []string{"line 1: passive voice"} }

	r := Check(good, Options{Lint: lint})
	has(t, r, "ste", "passive voice")
	for _, want := range []string{"Admins can now invite", "The body holds an email", "It makes the secret code", "Check the random source", "creates the invite", "Start at"} {
		if !strings.Contains(got, want) {
			t.Errorf("prose sent to lint misses %q", want)
		}
	}
	for _, not := range []string{"graph LR", "InviteCommand.Handle() ==>", "tests/InviteTests.cs", "Ref | What"} {
		if strings.Contains(got, not) {
			t.Errorf("prose sent to lint must not include %q", not)
		}
	}

	if r := Check(mutate("style=ste+iceberg", "style=iceberg"), Options{Lint: lint}); !r.OK() {
		t.Errorf("iceberg style must not run the STE lint: %+v", r.Findings)
	}
	if r := Check(mutate("style=ste+iceberg", "style=iceberg"), Options{Style: "ste", Lint: lint}); r.OK() {
		t.Error("a repo style of ste must beat an iceberg marker")
	}
	if r := Check(good, Options{}); len(r.Warnings) == 0 {
		t.Error("missing lint should warn")
	}
}

func TestLengthLimitFollowsTheHost(t *testing.T) {
	pad := good + strings.Repeat("x", 5000)
	if r := Check(pad, Options{Host: host.GitHub}); hasRule(r, "length") {
		t.Errorf("5,000 more characters fit on GitHub: %+v", r.Findings)
	}
	has(t, Check(pad, Options{Host: host.AzureDevOps}), "length", "Azure DevOps rejects more than 4000")
	// The limit counts characters, not bytes: 3,000 two-byte characters fit in 4,000.
	accents := strings.Replace(good, "<!-- pr-brief:end -->", strings.Repeat("é", 3000)+"\n<!-- pr-brief:end -->", 1)
	if r := Check(accents, Options{Host: host.AzureDevOps}); hasRule(r, "length") && len([]rune(accents)) <= convention.MaxBodyCharsAzureDevOps {
		t.Errorf("the limit must count characters: %+v", r.Findings)
	}
}

func hasRule(r Result, rule string) bool {
	for _, f := range r.Findings {
		if f.Rule == rule {
			return true
		}
	}
	return false
}

func TestHookPicksTheHostFromTheTool(t *testing.T) {
	long := strings.Repeat("x", 5000)
	run := func(in HookInput) string {
		return Evaluate(in, func(string) Options { return Options{} })
	}
	az := run(HookInput{ToolName: "mcp__azure-devops__repo_create_pull_request", ToolInput: map[string]any{"description": long}})
	if !strings.Contains(az, "Azure DevOps rejects more than 4000") {
		t.Errorf("an Azure DevOps tool call gets the 4,000 limit:\n%s", az)
	}
	gh := run(HookInput{ToolName: "mcp__github__create_pull_request", ToolInput: map[string]any{"body": long}})
	if strings.Contains(gh, "rejects more than") {
		t.Errorf("a GitHub tool call gets the 65,536 limit:\n%s", gh)
	}
}

func TestLegendAndLabelRules(t *testing.T) {
	cases := []struct{ name, from, to, rule, want string }{
		{"legend renamed", "subgraph LEG[\"Legend\"]", "subgraph LEG[\"Key\"]", "diagram", "subgraphs must be"},
		{"legend explains the wrong colour", "L1[\"added\"]:::added", "L1[\"modified\"]:::modified", "style", "Legend does not explain"},
		{"parenthesis in a function label", "F1[\"InviteCommand.Handle\"]", "F1[\"InviteCommand.Handle()\"]", "diagram", "no parentheses"},
		{"short port label", "I1([\"Input 1\"])", "I1([\"I1\"])", "diagram", "word and the number"},
		{"port label with the wrong number", "O2>\"Output 2\"]", "O2>\"Output 9\"]", "diagram", "word and the number"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			has(t, Check(mutate(c.from, c.to), Options{}), c.rule, c.want)
		})
	}
}
