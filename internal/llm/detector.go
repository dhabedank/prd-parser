package llm

import (
	"fmt"
)

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
	var available []string

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
