package tui

import (
	"testing"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name     string
		chars    int
		expected int
	}{
		{"empty", 0, 0},
		{"negative", -10, 0},
		{"small", 40, 10},
		{"medium", 1000, 250},
		{"large", 4000, 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EstimateTokens(tt.chars)
			if result != tt.expected {
				t.Errorf("EstimateTokens(%d) = %d, want %d", tt.chars, result, tt.expected)
			}
		})
	}
}

func TestEstimateCost(t *testing.T) {
	tests := []struct {
		name         string
		model        string
		inputTokens  int
		outputTokens int
		wantMin      float64
		wantMax      float64
	}{
		{
			name:         "claude opus 4.5",
			model:        "claude-opus-4-5-20251101",
			inputTokens:  1000,
			outputTokens: 500,
			wantMin:      0.017,
			wantMax:      0.018,
		},
		{
			name:         "claude haiku 4.5",
			model:        "claude-haiku-4-5-20251001",
			inputTokens:  1000,
			outputTokens: 500,
			wantMin:      0.003,
			wantMax:      0.004,
		},
		{
			name:         "unknown model uses default",
			model:        "unknown-model",
			inputTokens:  1000,
			outputTokens: 500,
			wantMin:      0.01,
			wantMax:      0.02,
		},
		{
			name:         "zero tokens",
			model:        "claude-opus-4-5-20251101",
			inputTokens:  0,
			outputTokens: 0,
			wantMin:      0,
			wantMax:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EstimateCost(tt.model, tt.inputTokens, tt.outputTokens)
			if result < tt.wantMin || result > tt.wantMax {
				t.Errorf("EstimateCost(%s, %d, %d) = %f, want between %f and %f",
					tt.model, tt.inputTokens, tt.outputTokens, result, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestFormatCost(t *testing.T) {
	tests := []struct {
		name     string
		cost     float64
		expected string
	}{
		{"tiny", 0.0001, "$0.0001"},
		{"small", 0.005, "$0.005"},
		{"medium", 0.05, "$0.05"},
		{"large", 1.50, "$1.50"},
		{"very large", 100.00, "$100.00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatCost(tt.cost)
			if result != tt.expected {
				t.Errorf("FormatCost(%f) = %s, want %s", tt.cost, result, tt.expected)
			}
		})
	}
}

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		name     string
		tokens   int
		expected string
	}{
		{"small", 500, "500"},
		{"thousand", 1500, "1.5k"},
		{"large", 15000, "15k"},
		{"very large", 150000, "150k"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatTokens(tt.tokens)
			if result != tt.expected {
				t.Errorf("FormatTokens(%d) = %s, want %s", tt.tokens, result, tt.expected)
			}
		})
	}
}
