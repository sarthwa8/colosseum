package agent

import (
	"strings"
	"testing"
)

func TestExtractCode(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain block", "```python\nprint(1)\n```", "print(1)"},
		{"no language tag", "```\nprint(2)\n```", "print(2)"},
		{"prefers last block", "first:\n```python\nA\n```\nfinal:\n```python\nB\n```", "B"},
		{"no fence falls back to text", "print(3)", "print(3)"},
		{"strips surrounding prose", "Here is the answer:\n```py\nx=1\n```\nDone.", "x=1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractCode(tt.in); got != tt.want {
				t.Errorf("ExtractCode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFeedbackPromptHidesExpected(t *testing.T) {
	p := FeedbackPrompt("WA", 3, 5, "")
	if !strings.Contains(p, "3/5") {
		t.Errorf("feedback should report pass count: %q", p)
	}
	// Must not fabricate/leak an expected-output section on a wrong answer.
	if strings.Contains(strings.ToLower(p), "expected output") {
		t.Errorf("feedback should not leak expected outputs: %q", p)
	}
}

func TestFeedbackPromptIncludesStderr(t *testing.T) {
	p := FeedbackPrompt("RE", 0, 5, "Traceback: NameError")
	if !strings.Contains(p, "NameError") {
		t.Errorf("runtime-error feedback should include stderr: %q", p)
	}
}

func TestCostUSD(t *testing.T) {
	// Haiku: $1/MTok in, $5/MTok out.
	got := CostUSD("claude-haiku-4-5", Usage{InputTokens: 1_000_000, OutputTokens: 1_000_000})
	if got != 6.0 {
		t.Errorf("cost = %v, want 6.0", got)
	}
	// Unknown/local models are free.
	if c := CostUSD("ollama-qwen", Usage{InputTokens: 1_000_000}); c != 0 {
		t.Errorf("local model cost = %v, want 0", c)
	}
	// Date-suffixed ids resolve by prefix.
	if c := CostUSD("claude-haiku-4-5-20251001", Usage{OutputTokens: 1_000_000}); c != 5.0 {
		t.Errorf("suffixed cost = %v, want 5.0", c)
	}
	// Gemini rates: 3.1 Pro is $2/MTok in, $12/MTok out.
	if c := CostUSD("gemini-3.1-pro", Usage{InputTokens: 1_000_000, OutputTokens: 1_000_000}); c != 14.0 {
		t.Errorf("gemini pro cost = %v, want 14.0", c)
	}
	// Longest prefix wins: a flash-lite preview id must not price as flash.
	if c := CostUSD("gemini-3.1-flash-lite-preview-06-17", Usage{OutputTokens: 1_000_000}); c != 1.5 {
		t.Errorf("flash-lite preview cost = %v, want 1.5 (lite rate, not flash)", c)
	}
}

func TestMockProvider(t *testing.T) {
	p := NewMockProvider("test", "one", "two")
	r1, _ := p.Complete(nil, Request{})
	r2, _ := p.Complete(nil, Request{})
	r3, _ := p.Complete(nil, Request{}) // exhausted -> repeats last
	if r1.Text != "one" || r2.Text != "two" || r3.Text != "two" {
		t.Errorf("scripted replies wrong: %q %q %q", r1.Text, r2.Text, r3.Text)
	}
}
