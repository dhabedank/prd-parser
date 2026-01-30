package tests

import (
	"strings"
	"testing"

	"github.com/dhabedank/prd-parser/internal/core"
)

func TestParseResponseValidate(t *testing.T) {
	tests := []struct {
		name    string
		resp    core.ParseResponse
		wantErr bool
	}{
		{
			name: "valid response",
			resp: core.ParseResponse{
				Project: core.ProjectContext{
					ProductName: "Test Product",
				},
				Epics: []core.Epic{
					{
						Title: "Epic 1",
						Tasks: []core.Task{
							{
								Title: "Task 1",
								Subtasks: []core.Subtask{
									{Title: "Subtask 1"},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing product name",
			resp: core.ParseResponse{
				Project: core.ProjectContext{},
				Epics: []core.Epic{
					{Title: "Epic 1", Tasks: []core.Task{{Title: "Task 1"}}},
				},
			},
			wantErr: true,
		},
		{
			name: "no epics",
			resp: core.ParseResponse{
				Project: core.ProjectContext{ProductName: "Test"},
				Epics:   []core.Epic{},
			},
			wantErr: true,
		},
		{
			name: "epic without tasks",
			resp: core.ParseResponse{
				Project: core.ProjectContext{ProductName: "Test"},
				Epics: []core.Epic{
					{Title: "Epic 1", Tasks: []core.Task{}},
				},
			},
			wantErr: true,
		},
		{
			name: "epic without title",
			resp: core.ParseResponse{
				Project: core.ProjectContext{ProductName: "Test"},
				Epics: []core.Epic{
					{Title: "", Tasks: []core.Task{{Title: "Task 1"}}},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.resp.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultParseConfig(t *testing.T) {
	config := core.DefaultParseConfig()

	if config.TargetEpics != 3 {
		t.Errorf("TargetEpics = %d, want 3", config.TargetEpics)
	}
	if config.TasksPerEpic != 5 {
		t.Errorf("TasksPerEpic = %d, want 5", config.TasksPerEpic)
	}
	if config.SubtasksPerTask != 4 {
		t.Errorf("SubtasksPerTask = %d, want 4", config.SubtasksPerTask)
	}
	if config.DefaultPriority != core.PriorityMedium {
		t.Errorf("DefaultPriority = %s, want medium", config.DefaultPriority)
	}
	if config.TestingLevel != "comprehensive" {
		t.Errorf("TestingLevel = %s, want comprehensive", config.TestingLevel)
	}
	if !config.PropagateContext {
		t.Error("PropagateContext should be true by default")
	}
}

func TestBuildUserPrompt(t *testing.T) {
	config := core.DefaultParseConfig()
	prd := "# Test PRD\nThis is a test."

	prompt := core.BuildUserPrompt(prd, config)

	// Check that config values are interpolated
	if !strings.Contains(prompt, "~3") {
		t.Error("Prompt should contain target epics")
	}
	if !strings.Contains(prompt, "Test PRD") {
		t.Error("Prompt should contain PRD content")
	}
	if !strings.Contains(prompt, "comprehensive") {
		t.Error("Prompt should contain testing level")
	}
	if !strings.Contains(prompt, "medium") {
		t.Error("Prompt should contain default priority")
	}
}

func TestValidationError(t *testing.T) {
	resp := core.ParseResponse{
		Project: core.ProjectContext{},
		Epics:   []core.Epic{},
	}

	err := resp.Validate()
	if err == nil {
		t.Fatal("Expected validation error")
	}

	// Check error message contains useful information
	errMsg := err.Error()
	if !strings.Contains(errMsg, "validation error") {
		t.Errorf("Error message should contain 'validation error', got: %s", errMsg)
	}
}

func TestPriorityConstants(t *testing.T) {
	// Verify priority constants have expected values
	if core.PriorityCritical != "critical" {
		t.Errorf("PriorityCritical = %s, want critical", core.PriorityCritical)
	}
	if core.PriorityHigh != "high" {
		t.Errorf("PriorityHigh = %s, want high", core.PriorityHigh)
	}
	if core.PriorityMedium != "medium" {
		t.Errorf("PriorityMedium = %s, want medium", core.PriorityMedium)
	}
	if core.PriorityLow != "low" {
		t.Errorf("PriorityLow = %s, want low", core.PriorityLow)
	}
}
