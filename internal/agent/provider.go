// Package agent is the fighter runtime: a provider-agnostic LLM client plus the
// solve→judge→debug loop that turns a model into a competitor. Providers are
// swappable (Anthropic, any OpenAI-compatible endpoint incl. Ollama, and a mock
// for cost-free tests) behind one interface.
package agent

import "context"

// Role is a chat role.
type Role string

const (
	User      Role = "user"
	Assistant Role = "assistant"
)

// Message is one chat turn.
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

// Request is a single completion request.
type Request struct {
	System      string
	Messages    []Message
	Model       string
	Temperature float64
	MaxTokens   int
}

// Usage reports token consumption for cost metering.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Response is a completion result.
type Response struct {
	Text  string
	Usage Usage
}

// Provider is any backend that can complete a chat request.
type Provider interface {
	Name() string
	Complete(ctx context.Context, req Request) (Response, error)
}

// MockProvider returns scripted replies in order (repeating the last once
// exhausted). It makes match/format logic testable with no network or cost.
type MockProvider struct {
	name    string
	replies []string
	i       int
}

// NewMockProvider builds a provider that yields the given replies in sequence.
func NewMockProvider(name string, replies ...string) *MockProvider {
	return &MockProvider{name: name, replies: replies}
}

func (m *MockProvider) Name() string { return "mock:" + m.name }

func (m *MockProvider) Complete(_ context.Context, req Request) (Response, error) {
	var text string
	switch {
	case len(m.replies) == 0:
		text = ""
	case m.i < len(m.replies):
		text = m.replies[m.i]
		m.i++
	default:
		text = m.replies[len(m.replies)-1]
	}
	return Response{
		Text:  text,
		Usage: estimateUsage(req, text),
	}, nil
}

// FuncProvider computes each reply from the request via a caller-supplied
// function. It's the mock used to test formats (like Attack/Defense) where a
// fighter must answer differently depending on the role/prompt it's given.
type FuncProvider struct {
	name string
	fn   func(Request) string
}

// NewFuncProvider builds a provider whose reply text is fn(req).
func NewFuncProvider(name string, fn func(Request) string) *FuncProvider {
	return &FuncProvider{name: name, fn: fn}
}

func (p *FuncProvider) Name() string { return "mock:" + p.name }

func (p *FuncProvider) Complete(_ context.Context, req Request) (Response, error) {
	text := p.fn(req)
	return Response{Text: text, Usage: estimateUsage(req, text)}, nil
}

// estimateUsage gives a rough token count (~4 chars/token) used by the mock and
// as a fallback when a provider omits usage.
func estimateUsage(req Request, out string) Usage {
	in := len(req.System)
	for _, msg := range req.Messages {
		in += len(msg.Content)
	}
	return Usage{InputTokens: in / 4, OutputTokens: len(out) / 4}
}
