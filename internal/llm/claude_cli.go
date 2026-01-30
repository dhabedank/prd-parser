package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/yourusername/prd-parser/internal/core"
)

// ClaudeCLIAdapter uses the Claude Code CLI for generation.
// This is preferred because users already have it authenticated.
type ClaudeCLIAdapter struct {
	model string
}

// NewClaudeCLIAdapter creates a Claude CLI adapter.
func NewClaudeCLIAdapter(config Config) *ClaudeCLIAdapter {
	model := config.Model
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	return &ClaudeCLIAdapter{model: model}
}

func (a *ClaudeCLIAdapter) Name() string {
	return "claude-cli"
}

// IsAvailable checks if the claude CLI is installed.
func (a *ClaudeCLIAdapter) IsAvailable() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

func (a *ClaudeCLIAdapter) Generate(ctx context.Context, systemPrompt, userPrompt string) (*core.ParseResponse, error) {
	// Write prompts to temp files (claude CLI reads from files better than stdin for long content)
	systemFile, err := os.CreateTemp("", "prd-system-*.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to create system prompt file: %w", err)
	}
	defer os.Remove(systemFile.Name())

	userFile, err := os.CreateTemp("", "prd-user-*.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to create user prompt file: %w", err)
	}
	defer os.Remove(userFile.Name())

	if _, err := systemFile.WriteString(systemPrompt); err != nil {
		return nil, fmt.Errorf("failed to write system prompt: %w", err)
	}
	systemFile.Close()

	if _, err := userFile.WriteString(userPrompt); err != nil {
		return nil, fmt.Errorf("failed to write user prompt: %w", err)
	}
	userFile.Close()

	// Build claude command
	// claude --model <model> --system-prompt-file <file> --print "<user prompt file>"
	cmd := exec.CommandContext(ctx, "claude",
		"--model", a.model,
		"--system-prompt-file", systemFile.Name(),
		"--print",
		"--output-format", "text",
	)

	// Pass user prompt via stdin
	userContent, _ := os.ReadFile(userFile.Name())
	cmd.Stdin = strings.NewReader(string(userContent))

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("claude CLI failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("claude CLI failed: %w", err)
	}

	// Parse JSON from output
	return parseJSONResponse(string(output))
}

// parseJSONResponse extracts and validates JSON from LLM output.
func parseJSONResponse(output string) (*core.ParseResponse, error) {
	// Find JSON in output (may be wrapped in markdown fences)
	output = strings.TrimSpace(output)

	// Remove markdown fences if present
	if strings.HasPrefix(output, "```json") {
		output = strings.TrimPrefix(output, "```json")
		output = strings.TrimSuffix(output, "```")
		output = strings.TrimSpace(output)
	} else if strings.HasPrefix(output, "```") {
		output = strings.TrimPrefix(output, "```")
		output = strings.TrimSuffix(output, "```")
		output = strings.TrimSpace(output)
	}

	// Find JSON object
	start := strings.Index(output, "{")
	end := strings.LastIndex(output, "}")
	if start == -1 || end == -1 || end < start {
		return nil, fmt.Errorf("no valid JSON found in response")
	}

	jsonStr := output[start : end+1]

	var response core.ParseResponse
	if err := json.Unmarshal([]byte(jsonStr), &response); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Validate
	if err := response.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return &response, nil
}
