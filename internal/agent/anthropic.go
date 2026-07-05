package agent

import (
	"context"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// AnthropicProvider talks to the Claude API. Requests are kept deliberately
// minimal (model, system, messages, max_tokens) so the same code path works
// across Haiku/Sonnet/Opus without per-model thinking/effort parameters — a
// fighter is a plain chat loop, and portability matters more than squeezing out
// extended thinking here.
type AnthropicProvider struct {
	client anthropic.Client
}

// NewAnthropicProvider builds a provider. Empty apiKey falls back to the SDK's
// normal credential resolution (ANTHROPIC_API_KEY, then ant profiles).
func NewAnthropicProvider(apiKey string) *AnthropicProvider {
	var opts []option.RequestOption
	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}
	return &AnthropicProvider{client: anthropic.NewClient(opts...)}
}

func (p *AnthropicProvider) Name() string { return "anthropic" }

func (p *AnthropicProvider) Complete(ctx context.Context, req Request) (Response, error) {
	msgs := make([]anthropic.MessageParam, 0, len(req.Messages))
	for _, m := range req.Messages {
		block := anthropic.NewTextBlock(m.Content)
		if m.Role == Assistant {
			msgs = append(msgs, anthropic.NewAssistantMessage(block))
		} else {
			msgs = append(msgs, anthropic.NewUserMessage(block))
		}
	}

	maxTok := req.MaxTokens
	if maxTok <= 0 {
		maxTok = 4096
	}
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(req.Model),
		MaxTokens: int64(maxTok),
		Messages:  msgs,
	}
	if req.System != "" {
		params.System = []anthropic.TextBlockParam{{Text: req.System}}
	}

	resp, err := p.client.Messages.New(ctx, params)
	if err != nil {
		return Response{}, err
	}

	var sb strings.Builder
	for _, block := range resp.Content {
		if tb, ok := block.AsAny().(anthropic.TextBlock); ok {
			sb.WriteString(tb.Text)
		}
	}
	return Response{
		Text: sb.String(),
		Usage: Usage{
			InputTokens:  int(resp.Usage.InputTokens),
			OutputTokens: int(resp.Usage.OutputTokens),
		},
	}, nil
}
