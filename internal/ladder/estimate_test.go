package ladder

import (
	"testing"

	"github.com/sarthaksukhral/colosseum/internal/agent"
	"github.com/sarthaksukhral/colosseum/internal/match"
	"github.com/sarthaksukhral/colosseum/internal/problem"
)

func estFixture() Config {
	prob := &problem.Problem{Slug: "p1", Statement: "solve it", Constraints: "1 <= n"}
	prob2 := &problem.Problem{Slug: "p2", Statement: "solve that", Constraints: "1 <= n"}
	mk := func(label string) Fighter {
		return Fighter{Label: label, New: func(*problem.Problem) agent.Provider {
			return agent.NewMockProvider(label, "x")
		}}
	}
	return Config{
		Fighters: []Fighter{mk("claude-haiku-4-5"), mk("mock:wrong")},
		Problems: []*problem.Problem{prob, prob2},
		Formats:  []match.Format{match.Race{}, match.AttackDefense{}},
		Rounds:   2,
		MaxIters: 3,
	}
}

func TestEstimateScheduleSize(t *testing.T) {
	est := EstimateWorstCase(estFixture())
	// C(2,2)=1 pair x 2 problems x 2 formats x 2 rounds = 8 matches.
	if est.Matches != 8 {
		t.Fatalf("matches = %d, want 8", est.Matches)
	}
}

func TestEstimatePricesKnownModelsOnly(t *testing.T) {
	est := EstimateWorstCase(estFixture())
	if est.PerModel["claude-haiku-4-5"] <= 0 {
		t.Fatalf("priced model should have non-zero worst case, got %v", est.PerModel["claude-haiku-4-5"])
	}
	if est.PerModel["mock:wrong"] != 0 {
		t.Fatalf("mock model should be free, got %v", est.PerModel["mock:wrong"])
	}
	if est.TotalUSD != est.PerModel["claude-haiku-4-5"] {
		t.Fatalf("total %v should equal the only paid model's cost %v", est.TotalUSD, est.PerModel["claude-haiku-4-5"])
	}
}

func TestEstimateBudgetCapLowersCeiling(t *testing.T) {
	cfg := estFixture()
	uncapped := EstimateWorstCase(cfg)
	cfg.TokenBudget = 1000 // far below the worst-case per-match usage
	capped := EstimateWorstCase(cfg)
	if capped.TotalUSD >= uncapped.TotalUSD {
		t.Fatalf("budget cap should lower the ceiling: capped %v >= uncapped %v", capped.TotalUSD, uncapped.TotalUSD)
	}
}
