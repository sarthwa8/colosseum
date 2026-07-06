package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestOpenAICompatRequestShape asserts the wire format the Gemini/OpenAI/Ollama
// providers all share: path, bearer auth, system-role injection, and the
// max_tokens/temperature caps that bound per-completion spend.
func TestOpenAICompatRequestShape(t *testing.T) {
	var got ocRequest
	var auth, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		auth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"hi"}}],"usage":{"prompt_tokens":10,"completion_tokens":5}}`))
	}))
	defer srv.Close()

	p := NewGeminiProvider(srv.URL, "test-key")
	resp, err := p.Complete(context.Background(), Request{
		System:      "you are a solver",
		Messages:    []Message{{Role: User, Content: "solve"}},
		Model:       "gemini-3.5-flash",
		MaxTokens:   4096,
		Temperature: 0.7,
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}

	if path != "/chat/completions" {
		t.Errorf("path = %q, want /chat/completions", path)
	}
	if auth != "Bearer test-key" {
		t.Errorf("auth = %q, want Bearer test-key", auth)
	}
	if got.Model != "gemini-3.5-flash" {
		t.Errorf("model = %q", got.Model)
	}
	if got.MaxTokens != 4096 {
		t.Errorf("max_tokens = %d, want 4096 (uncapped output is a cost hole)", got.MaxTokens)
	}
	if got.Temperature != 0.7 {
		t.Errorf("temperature = %v, want 0.7", got.Temperature)
	}
	if len(got.Messages) != 2 || got.Messages[0].Role != "system" || got.Messages[1].Role != "user" {
		t.Errorf("messages = %+v, want [system, user]", got.Messages)
	}
	if resp.Text != "hi" {
		t.Errorf("text = %q", resp.Text)
	}
	if resp.Usage.InputTokens != 10 || resp.Usage.OutputTokens != 5 {
		t.Errorf("usage = %+v, want 10/5", resp.Usage)
	}
	if p.Name() != "gemini" {
		t.Errorf("Name() = %q, want gemini", p.Name())
	}
}

// TestOpenAICompatUsageFallback: servers that omit usage (some Ollama builds)
// must still produce non-zero estimated usage so cost metering isn't silently 0.
func TestOpenAICompatUsageFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"a reply of some length"}}]}`))
	}))
	defer srv.Close()

	p := NewOpenAICompatProvider(srv.URL, "", "test")
	resp, err := p.Complete(context.Background(), Request{
		Messages: []Message{{Role: User, Content: "a prompt with enough characters"}},
		Model:    "m",
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Usage.InputTokens == 0 || resp.Usage.OutputTokens == 0 {
		t.Errorf("usage = %+v, want estimateUsage fallback (non-zero)", resp.Usage)
	}
}

// TestOpenAICompatErrorLabel: HTTP errors must carry the provider label so a
// ladder forfeit reason points at the right provider.
func TestOpenAICompatErrorLabel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"quota exceeded"}}`, http.StatusForbidden)
	}))
	defer srv.Close()

	p := NewGeminiProvider(srv.URL, "k")
	_, err := p.Complete(context.Background(), Request{Model: "m"})
	if err == nil || !strings.Contains(err.Error(), "gemini") {
		t.Errorf("err = %v, want label 'gemini' in message", err)
	}
}

// TestOpenAICompatRetries429: one transient 429 must be retried, not forfeited.
func TestOpenAICompatRetries429(t *testing.T) {
	old := compatRetryDelay
	compatRetryDelay = 0
	defer func() { compatRetryDelay = old }()

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			http.Error(w, "rate limited", http.StatusTooManyRequests)
			return
		}
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`))
	}))
	defer srv.Close()

	p := NewOpenAICompatProvider(srv.URL, "", "test")
	resp, err := p.Complete(context.Background(), Request{Model: "m"})
	if err != nil {
		t.Fatalf("Complete after retry: %v", err)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2 (one retry)", calls)
	}
	if resp.Text != "ok" {
		t.Errorf("text = %q", resp.Text)
	}
}
