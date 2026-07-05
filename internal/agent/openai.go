package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OpenAICompatProvider targets any OpenAI-compatible /chat/completions endpoint.
// Its purpose here is Ollama (http://localhost:11434/v1), so local models can
// fight for free and keep ladder cost at ~$0 — but it works against any
// compatible server. Deliberately raw net/http, not the OpenAI SDK: one small
// request shape, no dependency.
type OpenAICompatProvider struct {
	baseURL string
	apiKey  string
	label   string
	http    *http.Client
}

// NewOllamaProvider points at a local Ollama server (or OLLAMA_HOST override
// handled by the caller). base "" defaults to the standard local endpoint.
func NewOllamaProvider(base string) *OpenAICompatProvider {
	if base == "" {
		base = "http://localhost:11434/v1"
	}
	return &OpenAICompatProvider{
		baseURL: base,
		label:   "ollama",
		http:    &http.Client{Timeout: 5 * time.Minute},
	}
}

// NewOpenAICompatProvider is the general constructor (custom endpoint + key).
func NewOpenAICompatProvider(base, apiKey, label string) *OpenAICompatProvider {
	if label == "" {
		label = "openai-compat"
	}
	return &OpenAICompatProvider{
		baseURL: base,
		apiKey:  apiKey,
		label:   label,
		http:    &http.Client{Timeout: 5 * time.Minute},
	}
}

func (p *OpenAICompatProvider) Name() string { return p.label }

type ocMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ocRequest struct {
	Model    string      `json:"model"`
	Messages []ocMessage `json:"messages"`
	Stream   bool        `json:"stream"`
}

type ocResponse struct {
	Choices []struct {
		Message ocMessage `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (p *OpenAICompatProvider) Complete(ctx context.Context, req Request) (Response, error) {
	msgs := make([]ocMessage, 0, len(req.Messages)+1)
	if req.System != "" {
		msgs = append(msgs, ocMessage{Role: "system", Content: req.System})
	}
	for _, m := range req.Messages {
		msgs = append(msgs, ocMessage{Role: string(m.Role), Content: m.Content})
	}

	body, err := json.Marshal(ocRequest{Model: req.Model, Messages: msgs, Stream: false})
	if err != nil {
		return Response{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Response{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.http.Do(httpReq)
	if err != nil {
		return Response{}, fmt.Errorf("%s request: %w", p.label, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return Response{}, fmt.Errorf("%s returned %d: %s", p.label, resp.StatusCode, truncate(string(raw), 300))
	}

	var parsed ocResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return Response{}, fmt.Errorf("%s decode: %w", p.label, err)
	}
	if parsed.Error != nil {
		return Response{}, fmt.Errorf("%s error: %s", p.label, parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return Response{}, fmt.Errorf("%s returned no choices", p.label)
	}

	usage := Usage{InputTokens: parsed.Usage.PromptTokens, OutputTokens: parsed.Usage.CompletionTokens}
	text := parsed.Choices[0].Message.Content
	if usage.InputTokens == 0 && usage.OutputTokens == 0 {
		// Some servers omit usage; fall back to a rough estimate so cost/metrics
		// aren't silently zero.
		usage = estimateUsage(req, text)
	}
	return Response{Text: text, Usage: usage}, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
