package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dhabedank/prd-parser/internal/core"
)

const maxRetries = 3

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
	var lastErr error
	var lastOutput string

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			fmt.Printf("Retry attempt %d/%d...\n", attempt, maxRetries)
			time.Sleep(time.Duration(attempt) * 2 * time.Second) // Exponential backoff
		}

		output, err := a.callClaude(ctx, systemPrompt, userPrompt)
		if err != nil {
			lastErr = err
			lastOutput = ""
			fmt.Printf("LLM call failed: %v\n", err)
			continue
		}

		lastOutput = output
		response, err := parseJSONResponse(output)
		if err != nil {
			lastErr = err
			fmt.Printf("JSON parsing failed: %v\n", err)
			continue
		}

		return response, nil
	}

	// All retries failed - save raw response for debugging
	if lastOutput != "" {
		debugFile := filepath.Join(os.TempDir(), "prd-parser-last-response.txt")
		os.WriteFile(debugFile, []byte(lastOutput), 0644)
		return nil, fmt.Errorf("%w (raw response saved to %s)", lastErr, debugFile)
	}

	return nil, lastErr
}

// GenerateRaw sends prompts to Claude and returns raw string output.
// Used for validation and other non-structured responses.
func (a *ClaudeCLIAdapter) GenerateRaw(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return a.callClaude(ctx, systemPrompt, userPrompt)
}

func (a *ClaudeCLIAdapter) callClaude(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	// Write prompts to temp files (claude CLI reads from files better than stdin for long content)
	systemFile, err := os.CreateTemp("", "prd-system-*.txt")
	if err != nil {
		return "", fmt.Errorf("failed to create system prompt file: %w", err)
	}
	defer os.Remove(systemFile.Name())

	userFile, err := os.CreateTemp("", "prd-user-*.txt")
	if err != nil {
		return "", fmt.Errorf("failed to create user prompt file: %w", err)
	}
	defer os.Remove(userFile.Name())

	if _, err := systemFile.WriteString(systemPrompt); err != nil {
		return "", fmt.Errorf("failed to write system prompt: %w", err)
	}
	systemFile.Close()

	if _, err := userFile.WriteString(userPrompt); err != nil {
		return "", fmt.Errorf("failed to write user prompt: %w", err)
	}
	userFile.Close()

	// Start progress indicator in background
	done := make(chan bool)
	go func() {
		startTime := time.Now()
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				elapsed := time.Since(startTime).Truncate(time.Second)
				fmt.Printf("  Still generating... (%s elapsed)\n", elapsed)
			}
		}
	}()

	// Build claude command
	// Key flags for clean, isolated execution:
	// --tools "" disables all tools so LLM just responds to the prompt
	// --output-format json returns structured result
	// --no-session-persistence avoids picking up session context
	cmd := exec.CommandContext(ctx, "claude",
		"--model", a.model,
		"--system-prompt-file", systemFile.Name(),
		"--print",
		"--output-format", "json",
		"--tools", "",
		"--no-session-persistence",
	)

	// Pass user prompt via stdin
	userContent, _ := os.ReadFile(userFile.Name())
	cmd.Stdin = strings.NewReader(string(userContent))

	output, err := cmd.Output()
	close(done) // Stop progress indicator

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("claude CLI failed: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("claude CLI failed: %w", err)
	}

	return string(output), nil
}

// cliJSONResponse is the wrapper structure from --output-format json
type cliJSONResponse struct {
	Type    string `json:"type"`
	Result  string `json:"result"`
	IsError bool   `json:"is_error"`
}

// parseJSONResponse extracts and validates JSON from LLM output.
func parseJSONResponse(output string) (*core.ParseResponse, error) {
	if len(output) == 0 {
		return nil, fmt.Errorf("empty response from LLM")
	}

	output = strings.TrimSpace(output)

	// Check if this is wrapped JSON from --output-format json
	if strings.HasPrefix(output, "{\"type\":") {
		var wrapper cliJSONResponse
		if err := json.Unmarshal([]byte(output), &wrapper); err == nil {
			if wrapper.IsError {
				return nil, fmt.Errorf("CLI returned error: %s", wrapper.Result)
			}
			// Extract the actual response from the wrapper
			output = wrapper.Result
		}
	}

	// Find JSON in output (may be wrapped in markdown fences)
	output = strings.TrimSpace(output)

	// Remove markdown fences if present
	if strings.HasPrefix(output, "```json") {
		output = strings.TrimPrefix(output, "```json")
		if idx := strings.LastIndex(output, "```"); idx != -1 {
			output = output[:idx]
		}
		output = strings.TrimSpace(output)
	} else if strings.HasPrefix(output, "```") {
		output = strings.TrimPrefix(output, "```")
		if idx := strings.LastIndex(output, "```"); idx != -1 {
			output = output[:idx]
		}
		output = strings.TrimSpace(output)
	}

	// Find JSON object
	start := strings.Index(output, "{")
	end := strings.LastIndex(output, "}")
	if start == -1 || end == -1 || end < start {
		// Show first 200 chars to help debug
		preview := output
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		return nil, fmt.Errorf("no valid JSON found in response (starts with: %q)", preview)
	}

	jsonStr := output[start : end+1]

	var response core.ParseResponse
	if err := json.Unmarshal([]byte(jsonStr), &response); err != nil {
		// Try to find the error location
		if syntaxErr, ok := err.(*json.SyntaxError); ok {
			context := jsonStr
			offset := int(syntaxErr.Offset)
			if offset > 50 {
				context = "..." + jsonStr[offset-50:]
			}
			if len(context) > 100 {
				context = context[:100] + "..."
			}
			return nil, fmt.Errorf("JSON syntax error at position %d: %v (near: %q)", syntaxErr.Offset, err, context)
		}
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Validate
	if err := response.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return &response, nil
}
