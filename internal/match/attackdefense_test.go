package match

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sarthaksukhral/colosseum/internal/agent"
	"github.com/sarthaksukhral/colosseum/internal/judge"
	"github.com/sarthaksukhral/colosseum/internal/problem"
)

func TestDecideADOutcome(t *testing.T) {
	t.Run("broke and survived wins", func(t *testing.T) {
		scores := map[string]FighterScore{
			"A": {ID: "A", Broke: true, Survived: true},
			"B": {ID: "B", Broke: false, Survived: false},
		}
		if o := decideADOutcome("A", "B", scores); o.WinnerID != "A" || o.Reason != "broke_and_survived" {
			t.Fatalf("got %+v", o)
		}
	})
	t.Run("both broken falls back to who solved", func(t *testing.T) {
		scores := map[string]FighterScore{
			"A": {ID: "A", Broke: true, Survived: false, Solved: true},
			"B": {ID: "B", Broke: true, Survived: false, Solved: false},
		}
		if o := decideADOutcome("A", "B", scores); o.WinnerID != "A" || o.Reason != "defender_solved" {
			t.Fatalf("got %+v", o)
		}
	})
	t.Run("symmetric and equal is a draw", func(t *testing.T) {
		scores := map[string]FighterScore{
			"A": {ID: "A", Broke: true, Survived: true, Solved: true, TokensIn: 5, TokensOut: 5},
			"B": {ID: "B", Broke: true, Survived: true, Solved: true, TokensIn: 5, TokensOut: 5},
		}
		if o := decideADOutcome("A", "B", scores); o.WinnerID != "" || o.Reason != "draw" {
			t.Fatalf("got %+v", o)
		}
	})
}

// End-to-end Attack/Defense with scripted models and the real Docker judge +
// reference oracle. A defends with a classic buggy solution (best initialized to
// 0, wrong for all-negative arrays); both attackers submit an all-negative input.
// Expected: B (correct defender) breaks A and survives A's attack, so B wins.
func TestAttackDefenseEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Docker integration test in -short mode")
	}
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skip("Docker not available; skipping A/D integration test")
	}

	prob, err := problem.Load(filepath.Join("..", "..", "problems", "max-subarray"))
	if err != nil {
		t.Fatal(err)
	}

	buggy := "```python\n" +
		"import sys\n" +
		"data=sys.stdin.read().split()\n" +
		"n=int(data[0]); nums=list(map(int,data[1:1+n]))\n" +
		"best=0\ncur=0\n" + // BUG: best should start at nums[0]
		"for x in nums:\n    cur=max(x,cur+x)\n    best=max(best,cur)\n" +
		"print(best)\n```"
	correct := "```python\n" + strings.TrimSpace(prob.Reference) + "\n```"
	breakingInput := "```\n1\n-3\n```" // all-negative: reference=-3, buggy=0

	isAttack := func(r agent.Request) bool {
		return strings.Contains(strings.ToLower(r.System), "adversarial tester")
	}
	// A: buggy defender; attacks with the all-negative input.
	aProv := agent.NewFuncProvider("A", func(r agent.Request) string {
		if isAttack(r) {
			return breakingInput
		}
		return buggy
	})
	// B: correct defender; attacks with the same input.
	bProv := agent.NewFuncProvider("B", func(r agent.Request) string {
		if isAttack(r) {
			return breakingInput
		}
		return correct
	})

	fighters := map[string]agent.Config{
		"A": {ID: "A", Provider: aProv, Model: "mock-buggy", MaxTokens: 512},
		"B": {ID: "B", Provider: bProv, Model: "mock-correct", MaxTokens: 512},
	}
	j := judge.New(judge.NewDockerRunner(""))
	m := New(prob, j, fighters, []string{"A", "B"}, AttackDefense{}, 1, 0)

	outcome, err := m.Run(context.Background(), AttackDefense{})
	if err != nil {
		t.Fatal(err)
	}

	if outcome.WinnerID != "B" {
		t.Fatalf("correct defender B should win, got winner=%q reason=%q scores=%+v", outcome.WinnerID, outcome.Reason, outcome.Scores)
	}
	if !outcome.Scores["B"].Broke {
		t.Errorf("B should have broken A")
	}
	if !outcome.Scores["B"].Survived {
		t.Errorf("B's correct solution should have survived the attack")
	}
	if outcome.Scores["A"].Survived {
		t.Errorf("A's buggy solution should have been broken")
	}
}
