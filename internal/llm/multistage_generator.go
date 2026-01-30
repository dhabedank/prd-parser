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

// MultiStageGenerator implements core.Generator for multi-stage parsing.
type MultiStageGenerator struct {
	mainModel    string // For Stage 1 (epics) and Stage 2 (tasks)
	subtaskModel string // For Stage 3 (subtasks) - can be cheaper/faster
}

// NewMultiStageGenerator creates a generator for multi-stage parsing.
func NewMultiStageGenerator(config Config) *MultiStageGenerator {
	mainModel := config.Model
	if mainModel == "" {
		mainModel = "claude-opus-4-5-20250514" // Use Opus 4.5 for best quality
	}

	// Use same model for subtasks unless specified (consistency over cost savings)
	subtaskModel := config.SubtaskModel
	if subtaskModel == "" {
		subtaskModel = mainModel
	}

	return &MultiStageGenerator{
		mainModel:    mainModel,
		subtaskModel: subtaskModel,
	}
}

// GenerateEpics implements Stage 1: PRD → Epics.
func (g *MultiStageGenerator) GenerateEpics(ctx context.Context, prdContent string, config core.ParseConfig) (*core.EpicsResponse, error) {
	userPrompt := core.BuildStage1Prompt(prdContent, config)

	output, err := g.callClaude(ctx, g.mainModel, core.Stage1SystemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	jsonStr := extractJSON(output)
	if jsonStr == "" {
		return nil, fmt.Errorf("no valid JSON in Stage 1 response")
	}

	var response core.EpicsResponse
	if err := json.Unmarshal([]byte(jsonStr), &response); err != nil {
		return nil, fmt.Errorf("Stage 1 JSON parse error: %w", err)
	}

	if len(response.Epics) == 0 {
		return nil, fmt.Errorf("Stage 1 returned no epics")
	}

	return &response, nil
}

// GenerateTasks implements Stage 2: Epic → Tasks.
func (g *MultiStageGenerator) GenerateTasks(ctx context.Context, epic core.Epic, project core.ProjectContext, config core.ParseConfig) ([]core.Task, error) {
	userPrompt := core.BuildStage2Prompt(epic, project, config)

	output, err := g.callClaude(ctx, g.mainModel, core.Stage2SystemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	jsonStr := extractJSON(output)
	if jsonStr == "" {
		return nil, fmt.Errorf("no valid JSON in Stage 2 response for epic %s", epic.TempID)
	}

	var response struct {
		Tasks []core.Task `json:"tasks"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &response); err != nil {
		return nil, fmt.Errorf("Stage 2 JSON parse error for epic %s: %w", epic.TempID, err)
	}

	if len(response.Tasks) == 0 {
		return nil, fmt.Errorf("Stage 2 returned no tasks for epic %s", epic.TempID)
	}

	return response.Tasks, nil
}

// GenerateSubtasks implements Stage 3: Task → Subtasks.
func (g *MultiStageGenerator) GenerateSubtasks(ctx context.Context, task core.Task, epicContext string, project core.ProjectContext, config core.ParseConfig) ([]core.Subtask, error) {
	userPrompt := core.BuildStage3Prompt(task, epicContext, project, config)

	// Retry up to 2 times for transient LLM output issues
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		output, err := g.callClaude(ctx, g.subtaskModel, core.Stage3SystemPrompt, userPrompt)
		if err != nil {
			lastErr = err
			continue
		}

		jsonStr := extractJSON(output)
		if jsonStr == "" {
			lastErr = fmt.Errorf("no valid JSON in Stage 3 response for task %s", task.TempID)
			continue
		}

		var response struct {
			Subtasks []core.Subtask `json:"subtasks"`
		}
		if err := json.Unmarshal([]byte(jsonStr), &response); err != nil {
			// Save debug info on parse error
			debugFile := filepath.Join(os.TempDir(), fmt.Sprintf("prd-parser-stage3-%s.json", task.TempID))
			_ = os.WriteFile(debugFile, []byte(jsonStr), 0644)
			lastErr = fmt.Errorf("Stage 3 JSON parse error for task %s: %w", task.TempID, err)
			continue
		}

		if len(response.Subtasks) == 0 {
			lastErr = fmt.Errorf("Stage 3 returned no subtasks for task %s", task.TempID)
			continue
		}

		return response.Subtasks, nil
	}

	return nil, lastErr
}

// callClaude invokes the Claude CLI with the given prompts.
func (g *MultiStageGenerator) callClaude(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
	// Write prompts to temp files
	systemFile, err := os.CreateTemp("", "stage-system-*.txt")
	if err != nil {
		return "", fmt.Errorf("failed to create system prompt file: %w", err)
	}
	defer os.Remove(systemFile.Name())

	if _, err := systemFile.WriteString(systemPrompt); err != nil {
		return "", fmt.Errorf("failed to write system prompt: %w", err)
	}
	systemFile.Close()

	// Progress indicator for longer stages
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
				fmt.Printf("    Still generating... (%s elapsed)\n", elapsed)
			}
		}
	}()

	cmd := exec.CommandContext(ctx, "claude",
		"--model", model,
		"--system-prompt-file", systemFile.Name(),
		"--print",
		"--output-format", "json",
		"--tools", "",
		"--no-session-persistence",
	)

	cmd.Stdin = strings.NewReader(userPrompt)

	output, err := cmd.Output()
	close(done)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("claude CLI failed: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("claude CLI failed: %w", err)
	}

	return string(output), nil
}

// extractJSON extracts JSON from CLI output, handling wrappers and markdown.
func extractJSON(output string) string {
	output = strings.TrimSpace(output)

	// Handle CLI JSON wrapper
	if strings.HasPrefix(output, "{\"type\":") {
		var wrapper struct {
			Type    string `json:"type"`
			Result  string `json:"result"`
			IsError bool   `json:"is_error"`
		}
		if err := json.Unmarshal([]byte(output), &wrapper); err == nil {
			if wrapper.IsError {
				return ""
			}
			output = wrapper.Result
		}
	}

	output = strings.TrimSpace(output)

	// Remove markdown fences
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
		return ""
	}

	return output[start : end+1]
}
