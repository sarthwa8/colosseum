package rank

import (
	"math"
	"testing"

	"github.com/sarthaksukhral/colosseum/internal/match"
)

func TestExpectedSymmetry(t *testing.T) {
	// Equal ratings => 50/50.
	if e := expected(1500, 1500); math.Abs(e-0.5) > 1e-9 {
		t.Fatalf("expected 0.5, got %v", e)
	}
	// Higher rating => >0.5, and the two expectations sum to 1.
	ea := expected(1600, 1400)
	eb := expected(1400, 1600)
	if ea <= 0.5 || math.Abs(ea+eb-1) > 1e-9 {
		t.Fatalf("asymmetric expectations: %v %v", ea, eb)
	}
}

func TestComputeEloWinnerGains(t *testing.T) {
	// A beats B repeatedly: A should end above 1500, B below, and they should
	// stay roughly balanced around the base (zero-sum updates).
	var results []Result
	for i := 0; i < 20; i++ {
		results = append(results, Result{A: "A", B: "B", ScoreA: 1})
	}
	r := ComputeElo(results, 32)
	if r["A"] <= 1500 || r["B"] >= 1500 {
		t.Fatalf("winner should rise, loser fall: %+v", r)
	}
	if math.Abs((r["A"]-1500)+(r["B"]-1500)) > 1e-6 {
		t.Fatalf("Elo updates should be zero-sum: %+v", r)
	}
}

func TestBootstrapCIBracketsRating(t *testing.T) {
	var results []Result
	for i := 0; i < 30; i++ {
		results = append(results, Result{A: "strong", B: "weak", ScoreA: 1})
	}
	ci := BootstrapCI(results, 32, 300, 42)
	s := ci["strong"]
	if !(s.Low <= s.Rating && s.Rating <= s.High) {
		t.Fatalf("point rating should sit within its CI: %+v", s)
	}
	if s.Rating <= ci["weak"].Rating {
		t.Fatalf("strong should outrank weak: %v vs %v", s.Rating, ci["weak"].Rating)
	}
}

// The headline claim: the report detects when the solve-rate ranking and the
// robustness ranking disagree. Two models tie on pass rate (both solve
// everything), but one wins every attack/defense game — robustness separates
// them where the pass rate can't.
func TestReportDetectsDivergence(t *testing.T) {
	score := func(model string, solved, survived, broke bool) match.FighterScore {
		return match.FighterScore{Model: model, Solved: solved, Survived: survived, Broke: broke, TokensIn: 100, TokensOut: 100}
	}
	var summaries []MatchSummary

	// Race: both solve every race → identical solve rate on this axis.
	for i := 0; i < 6; i++ {
		summaries = append(summaries, MatchSummary{
			Format: "race",
			Outcome: match.Outcome{
				WinnerID: "A", Reason: "solved",
				Scores: map[string]match.FighterScore{
					"A": score("robust", true, false, false),
					"B": score("fragile", true, false, false),
				},
			},
		})
	}
	// Attack/Defense: "robust" always breaks "fragile" and survives → robust wins.
	for i := 0; i < 6; i++ {
		summaries = append(summaries, MatchSummary{
			Format: "attack_defense",
			Outcome: match.Outcome{
				WinnerID: "A", Reason: "broke_and_survived",
				Scores: map[string]match.FighterScore{
					"A": score("robust", true, true, true),
					"B": score("fragile", true, false, false),
				},
			},
		})
	}

	rep := Build(summaries, 200, 7)

	// Both models solve everything → equal solve rate.
	var robustRow, fragileRow ModelRow
	for _, m := range rep.Models {
		if m.Model == "robust" {
			robustRow = m
		} else {
			fragileRow = m
		}
	}
	if robustRow.SolveRate != fragileRow.SolveRate {
		t.Fatalf("solve rates should tie: %v vs %v", robustRow.SolveRate, fragileRow.SolveRate)
	}
	// Robustness must separate them.
	if rep.RobustElo["robust"].Rating <= rep.RobustElo["fragile"].Rating {
		t.Fatalf("robust should outrank fragile on robustness Elo")
	}
	if !rep.Diverges() {
		t.Fatalf("report should report divergence when robustness reorders a solve-rate tie")
	}
	if robustRow.BreakRate != 1.0 || fragileRow.BreakRate != 0.0 {
		t.Fatalf("break rates wrong: robust=%v fragile=%v", robustRow.BreakRate, fragileRow.BreakRate)
	}
}
