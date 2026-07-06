package rank

import (
	"fmt"
	"io"
	"math"
	"sort"

	"github.com/sarthaksukhral/colosseum/internal/match"
)

// MatchSummary is the ladder's per-match input to the report: which format was
// played and the outcome (whose Scores carry each fighter's model id and axes).
type MatchSummary struct {
	Format  string        `json:"format"`
	Problem string        `json:"problem"`
	Outcome match.Outcome `json:"outcome"`
}

// ModelRow is one model's aggregated metrics across all its games.
type ModelRow struct {
	Model       string  `json:"model"`
	Games       int     `json:"games"`
	Wins        int     `json:"wins"`
	Losses      int     `json:"losses"`
	Draws       int     `json:"draws"`
	Solved      int     `json:"solved"`
	SolveGames  int     `json:"solve_games"`
	SolveRate   float64 `json:"solve_rate"`
	AvgTokens   float64 `json:"avg_tokens"`
	tokensTotal int
	ADGames     int     `json:"ad_games"`
	Survivals   int     `json:"survivals"`
	Breaks      int     `json:"breaks"`
	SurviveRate float64 `json:"survive_rate"`
	BreakRate   float64 `json:"break_rate"`
	CostUSD     float64 `json:"cost_usd"`
}

// Report is the full eval output.
type Report struct {
	TotalMatches int                       `json:"total_matches"`
	TotalCostUSD float64                   `json:"total_cost_usd"`
	Models       []ModelRow                `json:"models"`
	RaceElo      map[string]CI             `json:"race_elo"`
	RobustElo    map[string]CI             `json:"robustness_elo"`
	WinMatrix    map[string]map[string]int `json:"win_matrix"`
}

// Build aggregates match summaries into a Report. Race games feed the Race Elo
// (speed/efficiency proxy); attack_defense games feed the robustness Elo. Both
// contribute to solve rate and the win matrix.
func Build(summaries []MatchSummary, bootstrapIters int, seed int64) Report {
	rows := map[string]*ModelRow{}
	winMatrix := map[string]map[string]int{}
	var raceResults, robustResults []Result

	row := func(model string) *ModelRow {
		r, ok := rows[model]
		if !ok {
			r = &ModelRow{Model: model}
			rows[model] = r
		}
		return r
	}
	recordWin := func(winner, loser string) {
		if winMatrix[winner] == nil {
			winMatrix[winner] = map[string]int{}
		}
		winMatrix[winner][loser]++
	}

	for _, s := range summaries {
		ids := sortedKeys(s.Outcome.Scores)
		if len(ids) != 2 {
			continue
		}
		sa, sb := s.Outcome.Scores[ids[0]], s.Outcome.Scores[ids[1]]
		ma, mb := sa.Model, sb.Model
		ra, rb := row(ma), row(mb)

		ra.Games++
		rb.Games++
		accrue(ra, sa, s.Format)
		accrue(rb, sb, s.Format)

		// Win/loss/draw + win matrix + Elo score.
		var scoreA float64
		switch s.Outcome.WinnerID {
		case ids[0]:
			ra.Wins++
			rb.Losses++
			recordWin(ma, mb)
			scoreA = 1
		case ids[1]:
			rb.Wins++
			ra.Losses++
			recordWin(mb, ma)
			scoreA = 0
		default:
			ra.Draws++
			rb.Draws++
			scoreA = 0.5
		}

		res := Result{A: ma, B: mb, ScoreA: scoreA}
		if s.Format == "attack_defense" {
			robustResults = append(robustResults, res)
		} else {
			raceResults = append(raceResults, res)
		}
	}

	// Finalize rates and sort rows by solve rate then break rate.
	out := make([]ModelRow, 0, len(rows))
	var totalCost float64
	for _, r := range rows {
		if r.SolveGames > 0 {
			r.SolveRate = float64(r.Solved) / float64(r.SolveGames)
		}
		if r.Games > 0 {
			r.AvgTokens = float64(r.tokensTotal) / float64(r.Games)
		}
		if r.ADGames > 0 {
			r.SurviveRate = float64(r.Survivals) / float64(r.ADGames)
			r.BreakRate = float64(r.Breaks) / float64(r.ADGames)
		}
		totalCost += r.CostUSD
		out = append(out, *r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SolveRate != out[j].SolveRate {
			return out[i].SolveRate > out[j].SolveRate
		}
		return out[i].BreakRate > out[j].BreakRate
	})

	return Report{
		TotalMatches: len(summaries),
		TotalCostUSD: totalCost,
		Models:       out,
		RaceElo:      BootstrapCI(raceResults, defaultK, bootstrapIters, seed),
		RobustElo:    BootstrapCI(robustResults, defaultK, bootstrapIters, seed+1),
		WinMatrix:    winMatrix,
	}
}

// accrue folds one fighter's per-match score into its aggregate row.
func accrue(r *ModelRow, s match.FighterScore, format string) {
	r.SolveGames++
	if s.Solved {
		r.Solved++
	}
	r.tokensTotal += s.TokensIn + s.TokensOut
	r.CostUSD += s.CostUSD
	if format == "attack_defense" {
		r.ADGames++
		if s.Survived {
			r.Survivals++
		}
		if s.Broke {
			r.Breaks++
		}
	}
}

func sortedKeys(m map[string]match.FighterScore) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

// RankingBySolveRate returns models ordered by solve rate (the saturating axis).
func (r Report) RankingBySolveRate() []string { return r.rankBy(func(m ModelRow) float64 { return m.SolveRate }) }

// RankingByRobustness returns models ordered by robustness Elo (the axis that
// separates models the pass-rate can't).
func (r Report) RankingByRobustness() []string {
	return r.rankByElo(r.RobustElo)
}

func (r Report) rankBy(key func(ModelRow) float64) []string {
	ms := make([]ModelRow, len(r.Models))
	copy(ms, r.Models)
	sort.SliceStable(ms, func(i, j int) bool { return key(ms[i]) > key(ms[j]) })
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.Model
	}
	return out
}

func (r Report) rankByElo(elo map[string]CI) []string {
	type kv struct {
		m string
		v float64
	}
	var xs []kv
	for _, row := range r.Models {
		xs = append(xs, kv{row.Model, elo[row.Model].Rating})
	}
	sort.SliceStable(xs, func(i, j int) bool { return xs[i].v > xs[j].v })
	out := make([]string, len(xs))
	for i, x := range xs {
		out[i] = x.m
	}
	return out
}

// Diverges is the report's headline test: are there two models that tie on the
// pass rate (the saturating axis) yet differ on robustness Elo (the adversarial
// axis)? When true, the standard benchmark can't tell those models apart but
// Colosseum's adversarial ranking can — the whole thesis.
func (r Report) Diverges() bool {
	for i := 0; i < len(r.Models); i++ {
		for j := i + 1; j < len(r.Models); j++ {
			a, b := r.Models[i], r.Models[j]
			if math.Abs(a.SolveRate-b.SolveRate) > 1e-9 {
				continue // not tied on pass rate
			}
			ea, eb := r.RobustElo[a.Model].Rating, r.RobustElo[b.Model].Rating
			if math.Abs(ea-eb) > 1e-6 {
				return true // tied on pass rate, separated on robustness
			}
		}
	}
	return false
}

// Render writes a human-readable report.
func (r Report) Render(w io.Writer) {
	fmt.Fprintf(w, "Colosseum eval report — %d matches\n", r.TotalMatches)
	fmt.Fprintln(w, "================================================================")
	fmt.Fprintf(w, "%-22s %6s %6s %8s %9s %9s %10s %8s\n", "model", "games", "solve%", "avgTok", "survive%", "break%", "robustElo", "cost$")
	fmt.Fprintln(w, "----------------------------------------------------------------")
	for _, m := range r.Models {
		fmt.Fprintf(w, "%-22s %6d %5.0f%% %8.0f %8.0f%% %8.0f%% %10s %8.2f\n",
			m.Model, m.Games, m.SolveRate*100, m.AvgTokens,
			m.SurviveRate*100, m.BreakRate*100, eloCell(r.RobustElo[m.Model]), m.CostUSD)
	}
	fmt.Fprintln(w, "================================================================")
	fmt.Fprintf(w, "total cost: $%.2f\n", r.TotalCostUSD)
	fmt.Fprintf(w, "solve-rate ranking:  %v\n", r.RankingBySolveRate())
	fmt.Fprintf(w, "robustness ranking:  %v\n", r.RankingByRobustness())
	if r.Diverges() {
		fmt.Fprintln(w, "→ DIVERGENCE: models that tie on pass rate are separated by adversarial robustness.")
	} else {
		fmt.Fprintln(w, "→ no divergence yet (add models/problems, or the field is already stratified by pass rate).")
	}
}

func eloCell(c CI) string {
	if c.Rating == 0 && c.Low == 0 && c.High == 0 {
		return "—"
	}
	return fmt.Sprintf("%.0f±%.0f", c.Rating, (c.High-c.Low)/2)
}
