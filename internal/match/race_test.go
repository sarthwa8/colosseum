package match

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sarthaksukhral/colosseum/internal/agent"
	"github.com/sarthaksukhral/colosseum/internal/judge"
	"github.com/sarthaksukhral/colosseum/internal/problem"
)

func TestDecideOutcome(t *testing.T) {
	order := []string{"A", "B"}

	t.Run("fastest solver wins", func(t *testing.T) {
		scores := map[string]FighterScore{
			"A": {ID: "A", Solved: true, WallMs: 500},
			"B": {ID: "B", Solved: true, WallMs: 300},
		}
		if o := decideOutcome(order, scores); o.WinnerID != "B" || o.Reason != "solved" {
			t.Fatalf("got %+v", o)
		}
	})

	t.Run("more cases passed wins when neither solves", func(t *testing.T) {
		scores := map[string]FighterScore{
			"A": {ID: "A", CasesPassed: 4},
			"B": {ID: "B", CasesPassed: 2},
		}
		if o := decideOutcome(order, scores); o.WinnerID != "A" || o.Reason != "more_tests_passed" {
			t.Fatalf("got %+v", o)
		}
	})

	t.Run("fewer tokens breaks a cases tie", func(t *testing.T) {
		scores := map[string]FighterScore{
			"A": {ID: "A", CasesPassed: 2, TokensIn: 100, TokensOut: 100},
			"B": {ID: "B", CasesPassed: 2, TokensIn: 50, TokensOut: 10},
		}
		if o := decideOutcome(order, scores); o.WinnerID != "B" || o.Reason != "more_tests_passed" {
			t.Fatalf("got %+v", o)
		}
	})

	t.Run("true tie is a draw", func(t *testing.T) {
		scores := map[string]FighterScore{
			"A": {ID: "A", CasesPassed: 0, TokensIn: 10, TokensOut: 10},
			"B": {ID: "B", CasesPassed: 0, TokensIn: 10, TokensOut: 10},
		}
		if o := decideOutcome(order, scores); o.WinnerID != "" || o.Reason != "draw" {
			t.Fatalf("got %+v", o)
		}
	})
}

// End-to-end Race with scripted "models" (no API cost) but the REAL Docker
// judge: a correct fighter must beat a wrong one, and events must be logged.
func TestRaceEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Docker integration test in -short mode")
	}
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skip("Docker not available; skipping race integration test")
	}

	prob, err := problem.Load(filepath.Join("..", "..", "problems", "sum-two"))
	if err != nil {
		t.Fatal(err)
	}

	correct := "```python\na,b=map(int,input().split())\nprint(a+b)\n```"
	wrong := "```python\na,b=map(int,input().split())\nprint(a*b)\n```"

	fighters := map[string]agent.Config{
		"A": {ID: "A", Provider: agent.NewMockProvider("good", correct), Model: "mock-good", MaxTokens: 512},
		"B": {ID: "B", Provider: agent.NewMockProvider("bad", wrong), Model: "mock-bad", MaxTokens: 512},
	}
	j := judge.New(judge.NewDockerRunner(""))
	m := New(prob, j, fighters, []string{"A", "B"}, Race{}, 2, 0)

	outcome, err := m.Run(context.Background(), Race{})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.WinnerID != "A" {
		t.Fatalf("correct fighter should win, got winner=%q reason=%q", outcome.WinnerID, outcome.Reason)
	}
	if !outcome.Scores["A"].Solved {
		t.Errorf("fighter A should have solved")
	}
	if outcome.Scores["B"].Solved {
		t.Errorf("fighter B should not have solved")
	}
	// The event log must contain a start and a finish.
	evs := m.Log.Snapshot()
	if len(evs) < 3 {
		t.Fatalf("expected a populated event log, got %d events", len(evs))
	}
	if evs[len(evs)-1].Type != "match_finished" {
		t.Errorf("last event should be match_finished, got %s", evs[len(evs)-1].Type)
	}
}
