// Package ladder runs a tournament across a pool of fighters, problems, and
// formats, persisting every match and aggregating the results into the eval
// report. It sits above match + rank (which don't import each other) to avoid a
// cycle.
package ladder

import (
	"context"
	"math/rand"

	"github.com/sarthaksukhral/colosseum/internal/agent"
	"github.com/sarthaksukhral/colosseum/internal/judge"
	"github.com/sarthaksukhral/colosseum/internal/match"
	"github.com/sarthaksukhral/colosseum/internal/problem"
	"github.com/sarthaksukhral/colosseum/internal/rank"
)

// Fighter is a competitor in the ladder. New builds its provider for a given
// problem — real providers ignore the problem; mock/offline fighters use it to
// produce problem-specific solutions and attacks.
type Fighter struct {
	Label string
	New   func(prob *problem.Problem) agent.Provider
}

// Config parameterizes a ladder run.
type Config struct {
	Fighters    []Fighter
	Problems    []*problem.Problem
	Formats     []match.Format
	Rounds      int // times to replay each (pair, problem, format) combination
	Judge       *judge.Judge
	MaxIters    int
	TokenBudget int
	DataDir     string // where match records are saved ("" = don't persist)
	Seed        int64
	Progress    func(done, total int, rec match.Record)
}

// pairing is one scheduled match.
type pairing struct {
	i, j   int
	prob   *problem.Problem
	format match.Format
}

// Run executes the whole schedule sequentially (the judge is internally
// concurrent; serializing matches keeps Docker and provider rate limits sane)
// and returns the per-match summaries for the report.
func Run(ctx context.Context, cfg Config) ([]rank.MatchSummary, error) {
	if cfg.Rounds < 1 {
		cfg.Rounds = 1
	}
	var schedule []pairing
	for i := 0; i < len(cfg.Fighters); i++ {
		for j := i + 1; j < len(cfg.Fighters); j++ {
			for _, prob := range cfg.Problems {
				for _, f := range cfg.Formats {
					for r := 0; r < cfg.Rounds; r++ {
						schedule = append(schedule, pairing{i, j, prob, f})
					}
				}
			}
		}
	}
	rand.New(rand.NewSource(cfg.Seed)).Shuffle(len(schedule), func(a, b int) {
		schedule[a], schedule[b] = schedule[b], schedule[a]
	})

	summaries := make([]rank.MatchSummary, 0, len(schedule))
	for idx, p := range schedule {
		if ctx.Err() != nil {
			return summaries, ctx.Err()
		}
		fa, fb := cfg.Fighters[p.i], cfg.Fighters[p.j]
		fighters := map[string]agent.Config{
			"A": {ID: "A", Model: fa.Label, Provider: fa.New(p.prob), MaxTokens: 4096, Temperature: 0.7},
			"B": {ID: "B", Model: fb.Label, Provider: fb.New(p.prob), MaxTokens: 4096, Temperature: 0.7},
		}
		m := match.New(p.prob, cfg.Judge, fighters, []string{"A", "B"}, p.format, cfg.MaxIters, cfg.TokenBudget)
		outcome, err := m.Run(ctx, p.format)
		if err != nil {
			return summaries, err
		}
		rec := m.ToRecord(outcome)
		if cfg.DataDir != "" {
			_, _ = match.SaveRecord(cfg.DataDir, rec)
		}
		summaries = append(summaries, rank.MatchSummary{
			Format:  p.format.Name(),
			Problem: p.prob.Slug,
			Outcome: outcome,
		})
		if cfg.Progress != nil {
			cfg.Progress(idx+1, len(schedule), rec)
		}
	}
	return summaries, nil
}
