package match

import (
	"context"
	"time"

	"github.com/sarthaksukhral/colosseum/internal/agent"
	"github.com/sarthaksukhral/colosseum/internal/events"
	"github.com/sarthaksukhral/colosseum/internal/judge"
)

// runSolve executes one fighter's solve→judge→debug loop against the problem's
// hidden tests, emitting events as it goes, and returns its score. It stops
// early on the first Accepted verdict, on context cancellation (a winner was
// declared elsewhere), on token-budget exhaustion (forfeit), or after MaxIters.
// The named return lets the deferred finalizer stamp usage/timing on every exit
// path.
func (m *Match) runSolve(ctx context.Context, fid string) (score FighterScore) {
	cfg := m.Fighters[fid]
	score = FighterScore{ID: fid, Model: cfg.Model, CasesTotal: len(m.Problem.Cases)}
	sess := agent.NewSession(cfg, agent.SolveSystemPrompt(m.Lang))
	start := time.Now()

	defer func() {
		score.WallMs = time.Since(start).Milliseconds()
		score.TokensIn = sess.Usage.InputTokens
		score.TokensOut = sess.Usage.OutputTokens
		score.CostUSD = agent.CostUSD(cfg.Model, sess.Usage)
		if score.Iterations == 0 {
			score.Iterations = sess.Turns
		}
	}()

	prompt := agent.SolvePrompt(m.Problem.Statement, m.Problem.Constraints)

	for iter := 0; iter < m.MaxIters; iter++ {
		if ctx.Err() != nil {
			return
		}
		if m.TokenBudget > 0 && sess.Usage.InputTokens+sess.Usage.OutputTokens >= m.TokenBudget {
			score.Forfeit = "token_budget_exhausted"
			m.Log.Emit(events.FighterForfeit, fid, map[string]any{"reason": score.Forfeit})
			return
		}

		m.Log.Emit(events.FighterThinking, fid, map[string]any{"iteration": iter})
		reply, err := sess.Send(ctx, prompt)
		if err != nil {
			if ctx.Err() != nil {
				return // cancelled by a winner; not a real forfeit
			}
			score.Forfeit = "provider_error"
			m.Log.Emit(events.FighterForfeit, fid, map[string]any{"reason": score.Forfeit, "error": err.Error()})
			return
		}

		code := agent.ExtractCode(reply)
		score.Iterations = iter + 1
		score.Code = code
		m.Log.Emit(events.FighterCode, fid, map[string]any{"iteration": iter, "code": code})

		report, err := m.Judge.Judge(ctx, judge.Submission{
			Language: m.Lang,
			Code:     code,
			Cases:    m.Problem.Cases,
			Limits:   m.Problem.Limits,
		})
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			m.Log.Emit(events.Submission, fid, map[string]any{"iteration": iter, "verdict": string(judge.InternalErr), "error": err.Error()})
			prompt = agent.FeedbackPrompt(string(judge.InternalErr), score.CasesPassed, score.CasesTotal, err.Error())
			continue
		}

		if report.Passed > score.CasesPassed {
			score.CasesPassed = report.Passed
		}
		m.Log.Emit(events.Submission, fid, map[string]any{
			"iteration": iter, "verdict": string(report.Verdict),
			"passed": report.Passed, "total": report.Total,
		})
		m.Log.Emit(events.FighterProgress, fid, map[string]any{"passed": report.Passed, "total": report.Total})

		if report.Verdict == judge.Accepted {
			score.Solved = true
			score.CasesPassed = report.Total
			return
		}
		prompt = agent.FeedbackPrompt(string(report.Verdict), report.Passed, report.Total, firstStderr(report))
	}
	return
}

// firstStderr returns the most useful error text to feed back to a fighter: the
// compile error, or the stderr of the first failing case.
func firstStderr(rep judge.Report) string {
	if rep.Compile != nil && rep.Compile.Stderr != "" {
		return rep.Compile.Stderr
	}
	for _, c := range rep.Cases {
		if c.Verdict != judge.Accepted && c.Stderr != "" {
			return c.Stderr
		}
	}
	return ""
}
