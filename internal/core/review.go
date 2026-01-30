package core

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ReviewResult contains the result of a review pass.
type ReviewResult struct {
	Response    *ParseResponse
	WasModified bool
	ReviewNotes string
}

// RawReviewResponse is the structure returned by the LLM during review.
// It includes the review notes alongside the potentially modified structure.
type RawReviewResponse struct {
	ReviewNotes string          `json:"review_notes"`
	Project     ProjectContext  `json:"project"`
	Epics       []Epic          `json:"epics"`
	Metadata    ResponseMetadata `json:"metadata"`
}

// Reviewer interface for LLM adapters that can review.
type Reviewer interface {
	GenerateRaw(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

// ReviewAndFix reviews a generated ParseResponse and fixes structural issues.
// It calls the LLM with a review prompt and returns a potentially modified response.
func ReviewAndFix(ctx context.Context, response *ParseResponse, prdContent string, reviewer Reviewer) (*ReviewResult, error) {
	// Build prompts
	userPrompt := BuildReviewPrompt(response, prdContent)

	// Call LLM
	output, err := reviewer.GenerateRaw(ctx, ReviewSystemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("review LLM call failed: %w", err)
	}

	// Parse the review response
	reviewedResponse, reviewNotes, err := parseReviewResponse(output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse review response: %w", err)
	}

	// Determine if it was modified
	wasModified := reviewNotes != "No changes needed" && reviewNotes != ""

	// If the LLM returned an incomplete structure, merge with original
	if reviewedResponse != nil {
		// Preserve original metadata if not provided
		if reviewedResponse.Metadata.TotalEpics == 0 {
			reviewedResponse.Metadata = response.Metadata
		}
		// Update metadata counts
		reviewedResponse.Metadata.TotalEpics = len(reviewedResponse.Epics)
		totalTasks := 0
		totalSubtasks := 0
		for _, epic := range reviewedResponse.Epics {
			totalTasks += len(epic.Tasks)
			for _, task := range epic.Tasks {
				totalSubtasks += len(task.Subtasks)
			}
		}
		reviewedResponse.Metadata.TotalTasks = totalTasks
		reviewedResponse.Metadata.TotalSubtasks = totalSubtasks
	} else {
		// If parsing failed to get a response, use original
		reviewedResponse = response
	}

	return &ReviewResult{
		Response:    reviewedResponse,
		WasModified: wasModified,
		ReviewNotes: reviewNotes,
	}, nil
}

// parseReviewResponse extracts the reviewed structure and notes from LLM output.
func parseReviewResponse(output string) (*ParseResponse, string, error) {
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
				return nil, "", fmt.Errorf("CLI returned error: %s", wrapper.Result)
			}
			output = wrapper.Result
		}
	}

	// Remove markdown fences
	output = strings.TrimSpace(output)
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
		return nil, "", fmt.Errorf("no valid JSON found in review response")
	}

	jsonStr := output[start : end+1]

	// First try to parse as RawReviewResponse (includes review_notes)
	var rawResponse RawReviewResponse
	if err := json.Unmarshal([]byte(jsonStr), &rawResponse); err != nil {
		return nil, "", fmt.Errorf("failed to parse review JSON: %w", err)
	}

	// Convert to ParseResponse
	response := &ParseResponse{
		Project:  rawResponse.Project,
		Epics:    rawResponse.Epics,
		Metadata: rawResponse.Metadata,
	}

	// Validate the response
	if err := response.Validate(); err != nil {
		return nil, rawResponse.ReviewNotes, fmt.Errorf("review response validation failed: %w", err)
	}

	return response, rawResponse.ReviewNotes, nil
}
