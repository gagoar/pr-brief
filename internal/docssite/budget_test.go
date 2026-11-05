package docssite

import (
	"os"
	"path/filepath"
	"testing"
)

// The skill is loaded into the model's context on every run, so its size is a cost. These budgets
// stop it from growing back. A run reads SKILL.md and body-template.md always. It reads
// diagram-convention.md only when there is a diagram to draw. The rest is read on demand.
// Raise a budget only with a reason in the PR: every byte is paid for on every /pr-brief run.
const (
	budgetSkill       = 22_000 // SKILL.md alone
	budgetNoDiagram   = 27_000 // SKILL.md + body-template.md: a config-only or small PR
	budgetWithDiagram = 37_000 // the above + diagram-convention.md: a PR with flows
)

func size(t *testing.T, rel string) int {
	t.Helper()
	info, err := os.Stat(filepath.Join("..", "..", "plugin", "skills", "pr-brief", rel))
	if err != nil {
		t.Fatal(err)
	}
	return int(info.Size())
}

func TestSkillContextBudget(t *testing.T) {
	skill := size(t, "SKILL.md")
	noDiagram := skill + size(t, filepath.Join("references", "body-template.md"))
	withDiagram := noDiagram + size(t, filepath.Join("references", "diagram-convention.md"))
	for _, c := range []struct {
		name        string
		got, budget int
	}{
		{"SKILL.md", skill, budgetSkill},
		{"a run with no diagram (SKILL.md + body-template.md)", noDiagram, budgetNoDiagram},
		{"a run with a diagram (+ diagram-convention.md)", withDiagram, budgetWithDiagram},
	} {
		if c.got > c.budget {
			t.Errorf("%s is %d bytes, over its budget of %d. Move text into a reference that loads on demand, or give a reason to raise the budget.", c.name, c.got, c.budget)
		}
	}
}
