package llm

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/yourusername/prd-parser/internal/core"
)

// CodexCLIAdapter uses the Codex CLI for generation.
type CodexCLIAdapter struct {
	model string
}

// NewCodexCLIAdapter creates a Codex CLI adapter.
func NewCodexCLIAdapter(config Config) *CodexCLIAdapter {
	model := config.Model
	if model == "" {
		model = "o3" // Default to o3 for best reasoning
	}
	return &CodexCLIAdapter{model: model}
}

func (a *CodexCLIAdapter) Name() string {
	return "codex-cli"
}

// IsAvailable checks if the codex CLI is installed.
func (a *CodexCLIAdapter) IsAvailable() bool {
	_, err := exec.LookPath("codex")
	return err == nil
}

func (a *CodexCLIAdapter) Generate(ctx context.Context, systemPrompt, userPrompt string) (*core.ParseResponse, error) {
	// Codex uses a slightly different invocation pattern
	// Combine system + user prompts for codex
	combinedPrompt := fmt.Sprintf("SYSTEM INSTRUCTIONS:\n%s\n\nUSER REQUEST:\n%s", systemPrompt, userPrompt)

	// Write to temp file
	promptFile, err := os.CreateTemp("", "prd-prompt-*.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to create prompt file: %w", err)
	}
	defer os.Remove(promptFile.Name())

	if _, err := promptFile.WriteString(combinedPrompt); err != nil {
		return nil, fmt.Errorf("failed to write prompt: %w", err)
	}
	promptFile.Close()

	// Run codex
	cmd := exec.CommandContext(ctx, "codex",
		"--model", a.model,
		"--quiet", // Less verbose output
	)
	cmd.Stdin = strings.NewReader(combinedPrompt)

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("codex CLI failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("codex CLI failed: %w", err)
	}

	return parseJSONResponse(string(output))
}
