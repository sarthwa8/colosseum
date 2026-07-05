// Package match runs a contest between two LLM fighters over a problem, on the
// sandboxed judge, recording everything to an event log. Formats (Race,
// AttackDefense) are pluggable behind the Format interface; the match itself
// owns the fighters, judge, problem, log, and the reproducibility manifest.
package match

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/sarthaksukhral/colosseum/internal/agent"
	"github.com/sarthaksukhral/colosseum/internal/events"
	"github.com/sarthaksukhral/colosseum/internal/judge"
	"github.com/sarthaksukhral/colosseum/internal/problem"
)

// Format is a pluggable contest type. Run drives the match to completion,
// emitting events to m.Log, and returns the outcome.
type Format interface {
	Name() string
	Run(ctx context.Context, m *Match) (Outcome, error)
}

// FighterInfo is the manifest record for one fighter (no secrets, just what's
// needed to reproduce and audit).
type FighterInfo struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

// Manifest pins everything needed to reproduce a match: fighters, problem
// version, judge image, seed, and budget. Stored as the first event's payload.
type Manifest struct {
	MatchID        string                 `json:"match_id"`
	Format         string                 `json:"format"`
	Problem        string                 `json:"problem"`
	ProblemVersion string                 `json:"problem_version"`
	Fighters       map[string]FighterInfo `json:"fighters"`
	MaxIterations  int                    `json:"max_iterations"`
	TokenBudget    int                    `json:"token_budget"`
	CreatedAt      time.Time              `json:"created_at"`
	Seed           int64                  `json:"seed"`
}

// FighterScore is one fighter's result across all its axes — the raw material
// for both the winner decision and the eval report.
type FighterScore struct {
	ID          string  `json:"id"`
	Model       string  `json:"model"`
	Solved      bool    `json:"solved"`
	CasesPassed int     `json:"cases_passed"`
	CasesTotal  int     `json:"cases_total"`
	Iterations  int     `json:"iterations"`
	TokensIn    int     `json:"tokens_in"`
	TokensOut   int     `json:"tokens_out"`
	CostUSD     float64 `json:"cost_usd"`
	WallMs      int64   `json:"wall_ms"`
	Forfeit     string  `json:"forfeit,omitempty"` // reason if the fighter forfeited
	Code        string  `json:"code,omitempty"`    // final submitted solution (defender code in A/D)

	// Attack/Defense axes (unset in Race).
	Survived bool `json:"survived,omitempty"` // as defender, resisted the attack
	Broke    bool `json:"broke,omitempty"`    // as attacker, broke the opponent
}

// Outcome is the final result of a match.
type Outcome struct {
	WinnerID string                  `json:"winner_id"` // "" => draw
	Reason   string                  `json:"reason"`
	Scores   map[string]FighterScore `json:"scores"`
}

// Match bundles the runtime state a Format operates on.
type Match struct {
	ID       string
	Problem  *problem.Problem
	Judge    *judge.Judge
	Log      *events.Log
	Manifest Manifest
	Fighters map[string]agent.Config // id -> fighter config
	Order    []string                // stable fighter order (e.g. ["A","B"])

	MaxIters    int
	TokenBudget int // 0 => unlimited
	Lang        string
}

// New assembles a Match and its event log/manifest. fighters is keyed by id;
// order fixes a stable presentation order.
func New(prob *problem.Problem, j *judge.Judge, fighters map[string]agent.Config, order []string, format Format, maxIters, tokenBudget int) *Match {
	id := newID()
	log := events.NewLog(id)

	info := make(map[string]FighterInfo, len(fighters))
	for fid, cfg := range fighters {
		info[fid] = FighterInfo{ID: fid, Provider: cfg.Provider.Name(), Model: cfg.Model}
	}
	lang := "python"
	man := Manifest{
		MatchID:        id,
		Format:         format.Name(),
		Problem:        prob.Slug,
		ProblemVersion: prob.Version,
		Fighters:       info,
		MaxIterations:  maxIters,
		TokenBudget:    tokenBudget,
		CreatedAt:      time.Now().UTC(),
		Seed:           time.Now().UnixNano(),
	}
	return &Match{
		ID:          id,
		Problem:     prob,
		Judge:       j,
		Log:         log,
		Manifest:    man,
		Fighters:    fighters,
		Order:       order,
		MaxIters:    maxIters,
		TokenBudget: tokenBudget,
		Lang:        lang,
	}
}

func newID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return "m_" + hex.EncodeToString(b)
}

// Run drives this match with the given format.
func (m *Match) Run(ctx context.Context, f Format) (Outcome, error) {
	return f.Run(ctx, m)
}
