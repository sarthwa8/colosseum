package match

import (
	"context"

	"github.com/sarthaksukhral/colosseum/internal/events"
)

// Race is the head-to-head format: both fighters solve the same problem against
// the same hidden tests; first to full green wins. Ties (neither solves) are
// broken by tests passed, then tokens spent — the efficiency axis the eval
// thesis leans on.
type Race struct{}

func (Race) Name() string { return "race" }

func (Race) Run(ctx context.Context, m *Match) (Outcome, error) {
	m.Log.Emit(events.MatchScheduled, "system", map[string]any{"manifest": m.Manifest})
	m.Log.Emit(events.MatchStarted, "system", map[string]any{"format": "race", "problem": m.Problem.Slug, "title": m.Problem.Title})
	m.Log.Emit(events.PhaseStarted, "system", map[string]any{"phase": "solve"})

	// Cancel the loser as soon as anyone solves, to save tokens and wall time.
	cctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type res struct {
		id    string
		score FighterScore
	}
	ch := make(chan res, len(m.Order))
	for _, id := range m.Order {
		go func(id string) { ch <- res{id, m.runSolve(cctx, id)} }(id)
	}

	scores := make(map[string]FighterScore, len(m.Order))
	for range m.Order {
		r := <-ch
		scores[r.id] = r.score
		if r.score.Solved {
			cancel()
		}
	}

	outcome := decideOutcome(m.Order, scores)
	m.Log.Emit(events.MatchFinished, "system", map[string]any{
		"winner": outcome.WinnerID,
		"reason": outcome.Reason,
		"scores": scores,
	})
	return outcome, nil
}

// decideOutcome picks the winner. Among fighters that solved, the fastest
// (lowest wall time) wins. If none solved, rank by cases passed then fewer
// tokens; a genuine tie on both is a draw.
func decideOutcome(order []string, scores map[string]FighterScore) Outcome {
	winner := ""
	var bestWall int64 = 1<<63 - 1
	for _, id := range order {
		s := scores[id]
		if s.Solved && s.WallMs < bestWall {
			bestWall, winner = s.WallMs, id
		}
	}
	if winner != "" {
		return Outcome{WinnerID: winner, Reason: "solved", Scores: scores}
	}

	best := order[0]
	for _, id := range order[1:] {
		if cmpScore(scores[id], scores[best]) > 0 {
			best = id
		}
	}
	// Draw if anyone ties the best on the ranking keys.
	for _, id := range order {
		if id != best && cmpScore(scores[id], scores[best]) == 0 {
			return Outcome{WinnerID: "", Reason: "draw", Scores: scores}
		}
	}
	reason := "fewer_tokens"
	if scores[best].CasesPassed > 0 {
		reason = "more_tests_passed"
	}
	return Outcome{WinnerID: best, Reason: reason, Scores: scores}
}

// cmpScore ranks two non-solving fighters: more cases passed is better; on a
// tie, fewer total tokens is better. Returns >0 if a is better than b.
func cmpScore(a, b FighterScore) int {
	if a.CasesPassed != b.CasesPassed {
		if a.CasesPassed > b.CasesPassed {
			return 1
		}
		return -1
	}
	at, bt := a.TokensIn+a.TokensOut, b.TokensIn+b.TokensOut
	if at != bt {
		if at < bt {
			return 1
		}
		return -1
	}
	return 0
}
