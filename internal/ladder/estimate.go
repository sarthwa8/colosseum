package ladder

import (
	"github.com/sarthaksukhral/colosseum/internal/agent"
	"github.com/sarthaksukhral/colosseum/internal/problem"
)

// CostEstimate is the worst-case spend for a ladder schedule, computed before
// any provider or Docker call. Real spend is typically 3–10x lower: fighters
// usually solve before exhausting their iterations and replies run far shorter
// than the max_tokens cap.
type CostEstimate struct {
	Matches  int
	PerModel map[string]float64 // fighter label -> worst-case USD
	TotalUSD float64
}

// EstimateWorstCase prices the schedule at its ceiling: every completion runs
// to the full MaxTokens, every solve loop burns all MaxIters, and every
// attack/defense match includes the adversarial turn. When TokenBudget is set,
// each fighter's per-match spend is capped at the budget priced entirely at
// the output rate — a strict upper bound, since output never costs less than
// input.
func EstimateWorstCase(cfg Config) CostEstimate {
	if cfg.Rounds < 1 {
		cfg.Rounds = 1
	}
	if cfg.MaxTokens <= 0 {
		cfg.MaxTokens = 4096
	}
	if cfg.MaxIters <= 0 {
		cfg.MaxIters = 3
	}

	est := CostEstimate{PerModel: map[string]float64{}}
	pairs := len(cfg.Fighters) * (len(cfg.Fighters) - 1) / 2
	est.Matches = pairs * len(cfg.Problems) * len(cfg.Formats) * cfg.Rounds

	for i := 0; i < len(cfg.Fighters); i++ {
		for j := i + 1; j < len(cfg.Fighters); j++ {
			for _, prob := range cfg.Problems {
				for _, f := range cfg.Formats {
					u := fighterMatchUsage(cfg, prob, f.Name())
					for _, fighter := range []Fighter{cfg.Fighters[i], cfg.Fighters[j]} {
						cost := agent.CostUSD(fighter.Label, u)
						if cfg.TokenBudget > 0 {
							if capped := agent.CostUSD(fighter.Label, agent.Usage{OutputTokens: cfg.TokenBudget}); capped < cost {
								cost = capped
							}
						}
						cost *= float64(cfg.Rounds)
						est.PerModel[fighter.Label] += cost
						est.TotalUSD += cost
					}
				}
			}
		}
	}
	return est
}

// fighterMatchUsage is one fighter's worst-case tokens for a single match. The
// session resends the full history each iteration, so iteration i's input is
// the problem prompt plus every prior (reply + feedback) pair.
func fighterMatchUsage(cfg Config, prob *problem.Problem, formatName string) agent.Usage {
	prompt := len(prob.Statement+prob.Constraints)/4 + 100 // ~4 chars/token + persona overhead
	var u agent.Usage
	for i := 0; i < cfg.MaxIters; i++ {
		u.InputTokens += prompt + i*(cfg.MaxTokens+100)
		u.OutputTokens += cfg.MaxTokens
	}
	if formatName == "attack_defense" {
		// One adversarial turn: the attacker reads the defender's code too.
		u.InputTokens += prompt + cfg.MaxTokens
		u.OutputTokens += cfg.MaxTokens
	}
	return u
}
