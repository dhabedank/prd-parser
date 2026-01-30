package llm

import (
	"context"

	"github.com/yourusername/prd-parser/internal/core"
)

// Adapter is the interface all LLM adapters must implement.
type Adapter interface {
	// Name returns the adapter identifier for logging.
	Name() string

	// IsAvailable checks if this adapter can be used (CLI installed, API key set, etc.)
	IsAvailable() bool

	// Generate sends prompts to the LLM and returns parsed response.
	Generate(ctx context.Context, systemPrompt, userPrompt string) (*core.ParseResponse, error)
}

// Config holds configuration for LLM adapters.
type Config struct {
	// PreferCLI prefers CLI tools (claude, codex) over API when available.
	PreferCLI bool

	// Model specifies which model to use (optional, adapter chooses default).
	Model string

	// APIKey for direct API access (optional if CLI is used).
	APIKey string

	// MaxTokens limits response length.
	MaxTokens int
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		PreferCLI: true, // Use CLI tools when available (already authenticated)
		MaxTokens: 16384,
	}
}
