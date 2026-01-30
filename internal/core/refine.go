package core

import (
	"encoding/json"
	"fmt"
	"strings"
)

// BeadsIssue represents an issue loaded from beads
type BeadsIssue struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Type        string `json:"type"` // epic, task, subtask
	Parent      string `json:"parent,omitempty"`
	Status      string `json:"status,omitempty"`
}

// AnalysisResult contains the analysis of what's wrong and how to fix it
type AnalysisResult struct {
	WrongConcepts        []string `json:"wrong_concepts"`
	CorrectConcepts      []string `json:"correct_concepts"`
	CorrectedTitle       string   `json:"corrected_title"`
	CorrectedDescription string   `json:"corrected_description"`
}

// ParseBeadsJSON parses JSON output from bd show --format json
func ParseBeadsJSON(data []byte) (*BeadsIssue, error) {
	var issue BeadsIssue
	if err := json.Unmarshal(data, &issue); err != nil {
		return nil, fmt.Errorf("failed to parse beads JSON: %w", err)
	}
	return &issue, nil
}

// ParseAnalysisResult parses the LLM analysis response
func ParseAnalysisResult(output string) (*AnalysisResult, error) {
	output = strings.TrimSpace(output)

	// Handle CLI wrapper
	if strings.HasPrefix(output, "{\"type\":") {
		var wrapper struct {
			Type    string `json:"type"`
			Result  string `json:"result"`
			IsError bool   `json:"is_error"`
		}
		if err := json.Unmarshal([]byte(output), &wrapper); err == nil {
			if wrapper.IsError {
				return nil, fmt.Errorf("CLI returned error: %s", wrapper.Result)
			}
			output = wrapper.Result
		}
	}

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
		return nil, fmt.Errorf("no valid JSON found in analysis response")
	}

	jsonStr := output[start : end+1]

	var result AnalysisResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse analysis JSON: %w", err)
	}

	return &result, nil
}
