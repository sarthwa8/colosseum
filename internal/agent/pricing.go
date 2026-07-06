package agent

import "strings"

// rate is USD per 1M tokens.
type rate struct{ in, out float64 }

// pricing maps a model id (or prefix) to its per-MTok cost. Local/Ollama models
// and anything unknown are treated as free. Update as prices change; this only
// affects the reported cost meter, never behavior.
var pricing = map[string]rate{
	"claude-fable-5":    {10.0, 50.0},
	"claude-opus-4-8":   {5.0, 25.0},
	"claude-opus-4-7":   {5.0, 25.0},
	"claude-sonnet-5":   {3.0, 15.0},
	"claude-sonnet-4-6": {3.0, 15.0},
	"claude-haiku-4-5":  {1.0, 5.0},
	// Gemini rates as of 2026-07 (standard context tier); verify against
	// ai.google.dev/gemini-api/docs/pricing before a paid run.
	"gemini-3.1-pro":        {2.0, 12.0},
	"gemini-3.5-flash":      {1.5, 9.0},
	"gemini-3-flash":        {0.5, 3.0},
	"gemini-3.1-flash-lite": {0.25, 1.5},
}

// CostUSD estimates the dollar cost of usage for a model. Unknown/local models
// return 0.
func CostUSD(model string, u Usage) float64 {
	r, ok := pricing[model]
	if !ok {
		// Tolerate suffixed ids like claude-haiku-4-5-20251001 or
		// gemini-3.5-flash-preview. Longest prefix wins, so an id can't
		// accidentally price as a shorter sibling (e.g. -flash vs -flash-lite).
		bestLen := -1
		for k, v := range pricing {
			if strings.HasPrefix(model, k) && len(k) > bestLen {
				r, ok, bestLen = v, true, len(k)
			}
		}
	}
	if !ok {
		return 0
	}
	return float64(u.InputTokens)/1e6*r.in + float64(u.OutputTokens)/1e6*r.out
}
