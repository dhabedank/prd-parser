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
// The review focuses on structure (epic order, dependencies) and merges changes
// back onto the original to preserve subtasks and detailed data.
func ReviewAndFix(ctx context.Context, response *ParseResponse, prdContent string, reviewer Reviewer) (*ReviewResult, error) {
	// Build prompts
	userPrompt := BuildReviewPrompt(response, prdContent)

	// Call LLM
	output, err := reviewer.GenerateRaw(ctx, ReviewSystemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("review LLM call failed: %w", err)
	}

	// Parse the review response
	rawReviewed, err := parseReviewResponse(output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse review response: %w", err)
	}

	// Determine if it was modified
	reviewNotes := rawReviewed.ReviewNotes
	wasModified := reviewNotes != "No changes needed" && reviewNotes != ""

	// Merge the reviewed structure with original to preserve subtasks and detailed data
	mergedResponse := mergeReviewedStructure(response, rawReviewed)

	// Validate the merged result
	if err := mergedResponse.Validate(); err != nil {
		// If validation still fails, return original with warning
		return &ReviewResult{
			Response:    response,
			WasModified: false,
			ReviewNotes: fmt.Sprintf("Review merge failed validation: %v. Using original structure.", err),
		}, nil
	}

	return &ReviewResult{
		Response:    mergedResponse,
		WasModified: wasModified,
		ReviewNotes: reviewNotes,
	}, nil
}

// parseReviewResponse extracts the reviewed structure and notes from LLM output.
// Returns the raw reviewed response (which may be incomplete) and review notes.
func parseReviewResponse(output string) (*RawReviewResponse, error) {
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
		return nil, fmt.Errorf("no valid JSON found in review response")
	}

	jsonStr := output[start : end+1]

	// Parse as RawReviewResponse (includes review_notes)
	var rawResponse RawReviewResponse
	if err := json.Unmarshal([]byte(jsonStr), &rawResponse); err != nil {
		return nil, fmt.Errorf("failed to parse review JSON: %w", err)
	}

	return &rawResponse, nil
}

// mergeReviewedStructure merges the reviewed structure (which has structural changes
// like reordering, new dependencies, new epics) with the original response
// (which has full data including subtasks, testing requirements, etc.)
func mergeReviewedStructure(original *ParseResponse, reviewed *RawReviewResponse) *ParseResponse {
	// Build lookup maps from original
	originalEpicsByID := make(map[string]Epic)
	originalTasksByID := make(map[string]Task)
	for _, epic := range original.Epics {
		originalEpicsByID[epic.TempID] = epic
		for _, task := range epic.Tasks {
			originalTasksByID[task.TempID] = task
		}
	}

	// Build merged epics following the reviewed structure
	mergedEpics := make([]Epic, 0, len(reviewed.Epics))
	for _, reviewedEpic := range reviewed.Epics {
		var mergedEpic Epic

		// Check if this epic exists in original
		if origEpic, exists := originalEpicsByID[reviewedEpic.TempID]; exists {
			// Start with original epic (has full data)
			mergedEpic = origEpic
			// Apply reviewed changes (dependencies, etc.)
			mergedEpic.DependsOn = reviewedEpic.DependsOn
			if reviewedEpic.Title != "" {
				mergedEpic.Title = reviewedEpic.Title
			}
		} else {
			// New epic from review (e.g., added Project Foundation)
			mergedEpic = reviewedEpic
			// Ensure required fields
			if mergedEpic.AcceptanceCriteria == nil {
				mergedEpic.AcceptanceCriteria = []string{}
			}
		}

		// Merge tasks
		if len(reviewedEpic.Tasks) > 0 {
			mergedTasks := make([]Task, 0, len(reviewedEpic.Tasks))
			for _, reviewedTask := range reviewedEpic.Tasks {
				var mergedTask Task

				if origTask, exists := originalTasksByID[reviewedTask.TempID]; exists {
					// Start with original task (has subtasks, etc.)
					mergedTask = origTask
					// Apply reviewed changes
					mergedTask.DependsOn = reviewedTask.DependsOn
					if reviewedTask.Title != "" {
						mergedTask.Title = reviewedTask.Title
					}
				} else {
					// New task from review
					mergedTask = reviewedTask
					// Initialize empty subtasks if needed
					if mergedTask.Subtasks == nil {
						mergedTask.Subtasks = []Subtask{}
					}
				}
				mergedTasks = append(mergedTasks, mergedTask)
			}
			mergedEpic.Tasks = mergedTasks
		}

		mergedEpics = append(mergedEpics, mergedEpic)
	}

	// Build merged response
	merged := &ParseResponse{
		Project:  original.Project,
		Epics:    mergedEpics,
		Metadata: original.Metadata,
	}

	// Update project if review provided changes
	if reviewed.Project.ProductName != "" {
		merged.Project = reviewed.Project
	}

	// Recalculate metadata
	merged.Metadata.TotalEpics = len(merged.Epics)
	totalTasks := 0
	totalSubtasks := 0
	for _, epic := range merged.Epics {
		totalTasks += len(epic.Tasks)
		for _, task := range epic.Tasks {
			totalSubtasks += len(task.Subtasks)
		}
	}
	merged.Metadata.TotalTasks = totalTasks
	merged.Metadata.TotalSubtasks = totalSubtasks

	return merged
}
