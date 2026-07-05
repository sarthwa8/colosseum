package match

import (
	"context"

	"github.com/sarthaksukhral/colosseum/internal/agent"
	"github.com/sarthaksukhral/colosseum/internal/events"
	"github.com/sarthaksukhral/colosseum/internal/judge"
)

// AttackDefense is the adversarial format and the project's differentiator: each
// fighter first defends (writes and debugs a solution), then attacks the
// opponent's solution by crafting an input that breaks it. An attack only counts
// if it's validated against the pinned reference solution — the defender's
// output must differ from the reference's (or the defender must crash) on an
// input the reference itself handles cleanly. No judging by vibes.
type AttackDefense struct{}

func (AttackDefense) Name() string { return "attack_defense" }

func (AttackDefense) Run(ctx context.Context, m *Match) (Outcome, error) {
	m.Log.Emit(events.MatchScheduled, "system", map[string]any{"manifest": m.Manifest})
	m.Log.Emit(events.MatchStarted, "system", map[string]any{"format": "attack_defense", "problem": m.Problem.Slug, "title": m.Problem.Title})

	if m.Problem.Reference == "" {
		// Without a reference oracle, attacks can't be validated.
		return Outcome{Reason: "no_reference_solution"}, nil
	}

	a, b := m.Order[0], m.Order[1]

	// Round 1: A defends, B attacks.
	scoreA := m.runSolve(ctx, a) // A's defensive solution (with debug loop)
	bBrokeA := m.runAttack(ctx, b, a, scoreA.Code)

	// Round 2: B defends, A attacks.
	scoreB := m.runSolve(ctx, b)
	aBrokeB := m.runAttack(ctx, a, b, scoreB.Code)

	// Fold attack outcomes and attacker token spend into each fighter's score.
	scoreA.Survived = !bBrokeA.broke
	scoreA.Broke = aBrokeB.broke
	scoreA.TokensIn += aBrokeB.usage.InputTokens
	scoreA.TokensOut += aBrokeB.usage.OutputTokens
	scoreA.CostUSD += agent.CostUSD(m.Fighters[a].Model, aBrokeB.usage)

	scoreB.Survived = !aBrokeB.broke
	scoreB.Broke = bBrokeA.broke
	scoreB.TokensIn += bBrokeA.usage.InputTokens
	scoreB.TokensOut += bBrokeA.usage.OutputTokens
	scoreB.CostUSD += agent.CostUSD(m.Fighters[b].Model, bBrokeA.usage)

	scores := map[string]FighterScore{a: scoreA, b: scoreB}
	outcome := decideADOutcome(a, b, scores)
	m.Log.Emit(events.MatchFinished, "system", map[string]any{
		"winner": outcome.WinnerID, "reason": outcome.Reason, "scores": scores,
	})
	return outcome, nil
}

// attackResult holds one attack attempt's outcome.
type attackResult struct {
	broke bool
	usage agent.Usage
}

// runAttack has attacker try to break defender's code. Emits the proposed input
// and the validated result.
func (m *Match) runAttack(ctx context.Context, attacker, defender, defenderCode string) attackResult {
	m.Log.Emit(events.PhaseStarted, "system", map[string]any{"phase": "attack", "attacker": attacker, "defender": defender})

	if defenderCode == "" {
		// Defender never produced a solution; nothing to attack, no break.
		m.Log.Emit(events.AttackResult, attacker, map[string]any{"broke": false, "detail": "defender produced no solution"})
		return attackResult{}
	}

	cfg := m.Fighters[attacker]
	sess := agent.NewSession(cfg, agent.AttackSystemPrompt())
	prompt := agent.AttackPrompt(m.Problem.Statement, m.Problem.Constraints, defenderCode)

	reply, err := sess.Send(ctx, prompt)
	if err != nil {
		m.Log.Emit(events.AttackResult, attacker, map[string]any{"broke": false, "detail": "attacker error: " + err.Error()})
		return attackResult{usage: sess.Usage}
	}
	input := agent.ExtractCode(reply)
	m.Log.Emit(events.AttackSubmitted, attacker, map[string]any{"input": input})

	broke, detail := m.validateAttack(ctx, defenderCode, input)
	m.Log.Emit(events.AttackResult, attacker, map[string]any{"broke": broke, "detail": detail})
	return attackResult{broke: broke, usage: sess.Usage}
}

// validateAttack runs the input through both the reference oracle and the
// defender. The attack succeeds only if the reference handled the input cleanly
// AND the defender disagreed with it (or failed to run). This rejects
// out-of-spec inputs (which the reference itself can't handle) as invalid.
func (m *Match) validateAttack(ctx context.Context, defenderCode, input string) (bool, string) {
	refOut, refClean, refVerdict, err := m.Judge.RunInput(ctx, m.Problem.RefLang, m.Problem.Reference, input, m.Problem.Limits)
	if err != nil {
		return false, "reference run error"
	}
	if !refClean {
		// The reference couldn't process this input → it violates the constraints.
		return false, "invalid input (reference verdict " + string(refVerdict) + ")"
	}

	defOut, defClean, defVerdict, err := m.Judge.RunInput(ctx, m.Lang, defenderCode, input, m.Problem.Limits)
	if err != nil {
		return false, "defender run error"
	}
	if !defClean {
		return true, "defender failed (" + string(defVerdict) + ") on a valid input"
	}
	if !judge.SameOutput(refOut, defOut) {
		return true, "defender output disagrees with the reference"
	}
	return false, "defender matched the reference"
}

// decideADOutcome: the fighter who broke its opponent while surviving wins. If
// both broke each other (or both survived), it's a draw — with a tiebreak
// toward the defender who actually solved the problem, then fewer tokens.
func decideADOutcome(a, b string, scores map[string]FighterScore) Outcome {
	sa, sb := scores[a], scores[b]
	aWins := sa.Broke && sa.Survived
	bWins := sb.Broke && sb.Survived

	switch {
	case aWins && !bWins:
		return Outcome{WinnerID: a, Reason: "broke_and_survived", Scores: scores}
	case bWins && !aWins:
		return Outcome{WinnerID: b, Reason: "broke_and_survived", Scores: scores}
	}

	// Symmetric result: prefer the one who solved; then fewer tokens.
	if sa.Solved != sb.Solved {
		if sa.Solved {
			return Outcome{WinnerID: a, Reason: "defender_solved", Scores: scores}
		}
		return Outcome{WinnerID: b, Reason: "defender_solved", Scores: scores}
	}
	ta, tb := sa.TokensIn+sa.TokensOut, sb.TokensIn+sb.TokensOut
	if ta != tb {
		if ta < tb {
			return Outcome{WinnerID: a, Reason: "fewer_tokens", Scores: scores}
		}
		return Outcome{WinnerID: b, Reason: "fewer_tokens", Scores: scores}
	}
	return Outcome{WinnerID: "", Reason: "draw", Scores: scores}
}
