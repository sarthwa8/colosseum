// Package rank turns match results into rankings: Elo ratings with bootstrap
// confidence intervals, a win matrix, and the multi-axis eval report that is the
// project's headline — showing where adversarial and efficiency rankings diverge
// from a saturated pass-rate ranking.
package rank

import (
	"math"
	"math/rand"
	"sort"
)

// Result is one settled game between two models from Elo's point of view.
// ScoreA is 1 (A won), 0.5 (draw), or 0 (A lost).
type Result struct {
	A      string
	B      string
	ScoreA float64
}

const (
	baseRating = 1500.0
	defaultK   = 32.0
)

// expected is A's expected score against B under the logistic Elo model.
func expected(ra, rb float64) float64 {
	return 1.0 / (1.0 + math.Pow(10, (rb-ra)/400.0))
}

// ComputeElo runs sequential Elo updates over results in order. Every model
// named in results (or in the seed list) starts at 1500. K defaults to 32.
func ComputeElo(results []Result, k float64) map[string]float64 {
	if k <= 0 {
		k = defaultK
	}
	ratings := map[string]float64{}
	ensure := func(m string) {
		if _, ok := ratings[m]; !ok {
			ratings[m] = baseRating
		}
	}
	for _, r := range results {
		ensure(r.A)
		ensure(r.B)
		ra, rb := ratings[r.A], ratings[r.B]
		ea := expected(ra, rb)
		ratings[r.A] = ra + k*(r.ScoreA-ea)
		ratings[r.B] = rb + k*((1-r.ScoreA)-(1-ea))
	}
	return ratings
}

// CI is a rating with a 95% bootstrap confidence interval.
type CI struct {
	Rating float64 `json:"rating"`
	Low    float64 `json:"low"`
	High   float64 `json:"high"`
}

// BootstrapCI estimates each model's rating and 95% CI by resampling the result
// set with replacement `iters` times and recomputing Elo. With few games the
// interval is wide — which is the honest thing to show, and the reason a ladder
// needs enough matches before its rankings mean anything.
func BootstrapCI(results []Result, k float64, iters int, seed int64) map[string]CI {
	if iters <= 0 {
		iters = 1000
	}
	rng := rand.New(rand.NewSource(seed))

	// Collect the model set and per-model sampled ratings.
	samples := map[string][]float64{}
	for i := 0; i < iters; i++ {
		resampled := make([]Result, len(results))
		for j := range results {
			resampled[j] = results[rng.Intn(len(results))]
		}
		for m, r := range ComputeElo(resampled, k) {
			samples[m] = append(samples[m], r)
		}
	}

	point := ComputeElo(results, k)
	out := map[string]CI{}
	for m, xs := range samples {
		sort.Float64s(xs)
		out[m] = CI{
			Rating: point[m],
			Low:    percentile(xs, 2.5),
			High:   percentile(xs, 97.5),
		}
	}
	return out
}

// percentile returns the p-th percentile of a sorted slice (linear interp).
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}
	rank := p / 100 * float64(len(sorted)-1)
	lo := int(math.Floor(rank))
	hi := int(math.Ceil(rank))
	if lo == hi {
		return sorted[lo]
	}
	frac := rank - float64(lo)
	return sorted[lo]*(1-frac) + sorted[hi]*frac
}
