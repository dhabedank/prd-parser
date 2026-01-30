package llm

import (
	"context"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/yourusername/prd-parser/internal/core"
)

// AnthropicAPIAdapter uses the Anthropic API directly.
// Fallback when Claude CLI is not available.
type AnthropicAPIAdapter struct {
	client    anthropic.Client
	model     string
	maxTokens int
}

// NewAnthropicAPIAdapter creates an Anthropic API adapter.
func NewAnthropicAPIAdapter(config Config) (*AnthropicAPIAdapter, error) {
	apiKey := config.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	model := config.Model
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	maxTokens := config.MaxTokens
	if maxTokens == 0 {
		maxTokens = 16384
	}

	return &AnthropicAPIAdapter{
		client:    client,
		model:     model,
		maxTokens: maxTokens,
	}, nil
}

func (a *AnthropicAPIAdapter) Name() string {
	return "anthropic-api"
}

func (a *AnthropicAPIAdapter) IsAvailable() bool {
	return os.Getenv("ANTHROPIC_API_KEY") != ""
}

func (a *AnthropicAPIAdapter) Generate(ctx context.Context, systemPrompt, userPrompt string) (*core.ParseResponse, error) {
	resp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(a.model),
		MaxTokens: int64(a.maxTokens),
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("anthropic API error: %w", err)
	}

	// Extract text from response
	var output string
	for _, block := range resp.Content {
		if block.Type == "text" {
			output += block.Text
		}
	}

	return parseJSONResponse(output)
}
