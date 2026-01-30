package tui

import "fmt"

// ModelPricing contains pricing per 1M tokens for various models.
// Prices are in USD. Updated: 2026-01-30 from https://docs.anthropic.com/en/docs/about-claude/models
var ModelPricing = map[string]struct {
	InputPer1M  float64
	OutputPer1M float64
}{
	// Claude 4.5 models (latest)
	"claude-opus-4-5-20251101":   {InputPer1M: 5.0, OutputPer1M: 25.0},
	"claude-sonnet-4-5-20250929": {InputPer1M: 3.0, OutputPer1M: 15.0},
	"claude-haiku-4-5-20251001":  {InputPer1M: 1.0, OutputPer1M: 5.0},

	// Claude 4.x legacy models
	"claude-opus-4-1-20250805": {InputPer1M: 15.0, OutputPer1M: 75.0},
	"claude-sonnet-4-20250514": {InputPer1M: 3.0, OutputPer1M: 15.0},
	"claude-opus-4-20250514":   {InputPer1M: 15.0, OutputPer1M: 75.0},

	// Claude 3.x legacy models
	"claude-3-7-sonnet-20250219": {InputPer1M: 3.0, OutputPer1M: 15.0},
	"claude-3-haiku-20240307":    {InputPer1M: 0.25, OutputPer1M: 1.25},

	// OpenAI models
	"gpt-4o":      {InputPer1M: 2.5, OutputPer1M: 10.0},
	"gpt-4o-mini": {InputPer1M: 0.15, OutputPer1M: 0.60},
	"gpt-4-turbo": {InputPer1M: 10.0, OutputPer1M: 30.0},
	"o1":          {InputPer1M: 15.0, OutputPer1M: 60.0},
	"o1-mini":     {InputPer1M: 1.10, OutputPer1M: 4.40},
	"o3":          {InputPer1M: 10.0, OutputPer1M: 40.0},
	"o3-mini":     {InputPer1M: 1.10, OutputPer1M: 4.40},
	"codex":       {InputPer1M: 15.0, OutputPer1M: 60.0},

	// Fallback for unknown models (use conservative estimate)
	"default": {InputPer1M: 5.0, OutputPer1M: 15.0},
}

// EstimateTokens estimates token count from character count.
// Uses the approximation that 1 token ≈ 4 characters.
func EstimateTokens(chars int) int {
	if chars <= 0 {
		return 0
	}
	return chars / 4
}

// EstimateCost calculates the estimated cost for a model given token counts.
// Returns cost in USD.
func EstimateCost(model string, inputTokens, outputTokens int) float64 {
	pricing, ok := ModelPricing[model]
	if !ok {
		pricing = ModelPricing["default"]
	}

	inputCost := float64(inputTokens) * pricing.InputPer1M / 1_000_000
	outputCost := float64(outputTokens) * pricing.OutputPer1M / 1_000_000

	return inputCost + outputCost
}

// FormatCost formats a cost in USD for display.
// Uses appropriate precision based on the magnitude.
func FormatCost(cost float64) string {
	if cost < 0.001 {
		return fmt.Sprintf("$%.4f", cost)
	}
	if cost < 0.01 {
		return fmt.Sprintf("$%.3f", cost)
	}
	if cost < 1.0 {
		return fmt.Sprintf("$%.2f", cost)
	}
	return fmt.Sprintf("$%.2f", cost)
}

// FormatTokens formats a token count for display.
// Uses k suffix for thousands.
func FormatTokens(tokens int) string {
	if tokens < 1000 {
		return fmt.Sprintf("%d", tokens)
	}
	if tokens < 10000 {
		return fmt.Sprintf("%.1fk", float64(tokens)/1000)
	}
	return fmt.Sprintf("%dk", tokens/1000)
}
