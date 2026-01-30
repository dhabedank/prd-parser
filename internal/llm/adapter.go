package llm

import (
	"context"

	"github.com/dhabedank/prd-parser/internal/core"
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

	// Per-stage model configuration for multi-stage parsing.
	// These override Model when set.
	EpicModel    string `yaml:"epic_model"`
	TaskModel    string `yaml:"task_model"`
	SubtaskModel string `yaml:"subtask_model"`

	// APIKey for direct API access (optional if CLI is used).
	APIKey string

	// MaxTokens limits response length.
	MaxTokens int
}

// ModelForStage returns the model to use for a given stage.
// Falls back to the default Model if no stage-specific model is set.
func (c Config) ModelForStage(stage string) string {
	switch stage {
	case "epic":
		if c.EpicModel != "" {
			return c.EpicModel
		}
	case "task":
		if c.TaskModel != "" {
			return c.TaskModel
		}
	case "subtask":
		if c.SubtaskModel != "" {
			return c.SubtaskModel
		}
	}
	return c.Model
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		PreferCLI: true, // Use CLI tools when available (already authenticated)
		MaxTokens: 16384,
	}
}
