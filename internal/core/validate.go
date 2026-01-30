package core

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ValidationResult contains the results of plan validation.
type ValidationResult struct {
	IsValid  bool     `json:"is_valid"`
	Gaps     []string `json:"gaps,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// Validator validates a complete ParseResponse for gaps.
type Validator interface {
	Validate(ctx context.Context, response *ParseResponse, prdContent string) (*ValidationResult, error)
}

// ValidationPrompt is the system prompt for validation.
const ValidationPrompt = `You are a plan validator. You review a generated task breakdown and check for GAPS that would prevent successful implementation.

Your job is to identify MISSING steps, not to critique the plan's quality.

## WHAT TO CHECK

1. **Setup gaps**: Is there a task to initialize the project, install dependencies, set up environment?
2. **Build gaps**: If code is generated/compiled, is there a task to build it?
3. **Interface gaps**: If backend is built, is there a UI to interact with it?
4. **Verification gaps**: Can each epic's acceptance criteria actually be tested with the tasks provided?
5. **Dependency gaps**: Are dependencies installed before code that uses them?
6. **Order gaps**: Are tasks in a logical order (setup → implement → verify)?

## OUTPUT FORMAT

Return JSON:
{
  "is_valid": true/false,
  "gaps": ["Missing task to install dependencies", "No UI to test the API"],
  "warnings": ["Task 2.3 might need to come before 2.2"]
}

If no gaps found, return {"is_valid": true, "gaps": [], "warnings": []}

## IMPORTANT

- Focus on PRACTICAL gaps that would block implementation
- Don't nitpick - only flag things that would actually cause problems
- A warning is something suboptimal; a gap is something that would break the build/flow`

// BuildValidationPrompt creates the user prompt for validation.
func BuildValidationPrompt(response *ParseResponse, prdContent string) string {
	// Summarize the plan
	var summary strings.Builder
	summary.WriteString("## GENERATED PLAN SUMMARY\n\n")
	summary.WriteString(fmt.Sprintf("Project: %s\n", response.Project.ProductName))
	summary.WriteString(fmt.Sprintf("Tech Stack: %v\n\n", response.Project.TechStack))

	for _, epic := range response.Epics {
		summary.WriteString(fmt.Sprintf("### Epic %s: %s\n", epic.TempID, epic.Title))
		summary.WriteString(fmt.Sprintf("Acceptance: %v\n", epic.AcceptanceCriteria))
		for _, task := range epic.Tasks {
			summary.WriteString(fmt.Sprintf("  - Task %s: %s\n", task.TempID, task.Title))
			for _, subtask := range task.Subtasks {
				summary.WriteString(fmt.Sprintf("    - %s: %s\n", subtask.TempID, subtask.Title))
			}
		}
		summary.WriteString("\n")
	}

	return fmt.Sprintf(`Review this plan for GAPS that would prevent successful implementation.

%s

## ORIGINAL PRD (for context)
%s

Check for:
1. Missing setup/initialization tasks
2. Backend without UI to test it
3. Dependencies not installed
4. Acceptance criteria that can't be verified with current tasks
5. Tasks in wrong order

Return JSON with is_valid, gaps, and warnings.`, summary.String(), prdContent)
}

// ParseValidationResult parses the LLM response into a ValidationResult.
func ParseValidationResult(output string) (*ValidationResult, error) {
	// Find JSON in output
	output = strings.TrimSpace(output)

	// Handle CLI wrapper
	if strings.HasPrefix(output, "{\"type\":") {
		var wrapper struct {
			Result string `json:"result"`
		}
		if err := json.Unmarshal([]byte(output), &wrapper); err == nil {
			output = wrapper.Result
		}
	}

	// Remove markdown fences
	if strings.HasPrefix(output, "```") {
		lines := strings.Split(output, "\n")
		var jsonLines []string
		inJSON := false
		for _, line := range lines {
			if strings.HasPrefix(line, "```") {
				inJSON = !inJSON
				continue
			}
			if inJSON || (!strings.HasPrefix(line, "```") && strings.Contains(line, "{")) {
				jsonLines = append(jsonLines, line)
			}
		}
		output = strings.Join(jsonLines, "\n")
	}

	// Find JSON object
	start := strings.Index(output, "{")
	end := strings.LastIndex(output, "}")
	if start == -1 || end == -1 {
		return nil, fmt.Errorf("no JSON found in validation response")
	}

	jsonStr := output[start : end+1]

	var result ValidationResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse validation JSON: %w", err)
	}

	return &result, nil
}
