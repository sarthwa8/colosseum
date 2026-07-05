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
}

// CostUSD estimates the dollar cost of usage for a model. Unknown/local models
// return 0.
func CostUSD(model string, u Usage) float64 {
	r, ok := pricing[model]
	if !ok {
		// tolerate date-suffixed ids like claude-haiku-4-5-20251001
		for k, v := range pricing {
			if strings.HasPrefix(model, k) {
				r, ok = v, true
				break
			}
		}
	}
	if !ok {
		return 0
	}
	return float64(u.InputTokens)/1e6*r.in + float64(u.OutputTokens)/1e6*r.out
}
