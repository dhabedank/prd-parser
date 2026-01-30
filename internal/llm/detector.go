package llm

import (
	"fmt"
	"os/exec"
)

// ModelInfo describes an available model.
type ModelInfo struct {
	ID          string // Model identifier (e.g., "claude-opus-4-5-20251101")
	Name        string // Human-readable name (e.g., "Claude Opus 4.5")
	Description string // Brief description
	Provider    string // Provider name (e.g., "anthropic", "openai")
}

// claudeModels lists Claude models available via CLI.
var claudeModels = []ModelInfo{
	{ID: "claude-opus-4-5-20251101", Name: "Claude Opus 4.5", Description: "Most capable, best for complex tasks", Provider: "anthropic"},
	{ID: "claude-opus-4-20250514", Name: "Claude Opus 4", Description: "High capability, great for analysis", Provider: "anthropic"},
	{ID: "claude-sonnet-4-20250514", Name: "Claude Sonnet 4", Description: "Balanced speed and capability", Provider: "anthropic"},
	{ID: "claude-3-5-sonnet-20241022", Name: "Claude 3.5 Sonnet", Description: "Fast and capable", Provider: "anthropic"},
	{ID: "claude-3-5-haiku-20241022", Name: "Claude 3.5 Haiku", Description: "Fastest, most cost-effective", Provider: "anthropic"},
}

// codexModels lists Codex/OpenAI models available via CLI.
var codexModels = []ModelInfo{
	{ID: "o3", Name: "O3", Description: "Most capable reasoning model", Provider: "openai"},
	{ID: "o3-mini", Name: "O3 Mini", Description: "Fast reasoning model", Provider: "openai"},
	{ID: "o1", Name: "O1", Description: "Advanced reasoning", Provider: "openai"},
	{ID: "o1-mini", Name: "O1 Mini", Description: "Efficient reasoning", Provider: "openai"},
	{ID: "gpt-4o", Name: "GPT-4o", Description: "Fast multimodal model", Provider: "openai"},
	{ID: "gpt-4o-mini", Name: "GPT-4o Mini", Description: "Most cost-effective", Provider: "openai"},
}

// AvailableModels returns models grouped by provider based on available CLIs.
func AvailableModels() map[string][]ModelInfo {
	result := make(map[string][]ModelInfo)

	// Check for Claude CLI
	if _, err := exec.LookPath("claude"); err == nil {
		result["anthropic"] = claudeModels
	}

	// Check for Codex CLI
	if _, err := exec.LookPath("codex"); err == nil {
		result["openai"] = codexModels
	}

	return result
}

// AllModels returns a flat list of all available models.
func AllModels() []ModelInfo {
	available := AvailableModels()
	var result []ModelInfo

	// Add Claude models first (preferred)
	if models, ok := available["anthropic"]; ok {
		result = append(result, models...)
	}

	// Add OpenAI models
	if models, ok := available["openai"]; ok {
		result = append(result, models...)
	}

	return result
}

// DetectBestAdapter finds the best available LLM adapter.
// Priority: Claude CLI > Codex CLI > Anthropic API
func DetectBestAdapter(config Config) (Adapter, error) {
	// Try Claude CLI first (preferred - already authenticated)
	if config.PreferCLI {
		claude := NewClaudeCLIAdapter(config)
		if claude.IsAvailable() {
			return claude, nil
		}

		// Try Codex CLI
		codex := NewCodexCLIAdapter(config)
		if codex.IsAvailable() {
			return codex, nil
		}
	}

	// Fall back to Anthropic API
	anthropic, err := NewAnthropicAPIAdapter(config)
	if err == nil && anthropic.IsAvailable() {
		return anthropic, nil
	}

	// Could add OpenAI API fallback here

	return nil, fmt.Errorf("no LLM adapter available - install Claude Code, Codex, or set ANTHROPIC_API_KEY")
}

// ListAvailableAdapters returns all adapters that could be used.
func ListAvailableAdapters(config Config) []string {
	available := []string{}

	claude := NewClaudeCLIAdapter(config)
	if claude.IsAvailable() {
		available = append(available, "claude-cli")
	}

	codex := NewCodexCLIAdapter(config)
	if codex.IsAvailable() {
		available = append(available, "codex-cli")
	}

	anthropic, _ := NewAnthropicAPIAdapter(config)
	if anthropic != nil && anthropic.IsAvailable() {
		available = append(available, "anthropic-api")
	}

	return available
}
