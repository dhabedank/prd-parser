package output

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/yourusername/prd-parser/internal/core"
)

// JSONAdapter outputs the parsed response as JSON.
type JSONAdapter struct {
	outputPath string
	dryRun     bool
}

// NewJSONAdapter creates a JSON adapter.
func NewJSONAdapter(config Config, outputPath string) *JSONAdapter {
	return &JSONAdapter{
		outputPath: outputPath,
		dryRun:     config.DryRun,
	}
}

func (a *JSONAdapter) Name() string {
	return "json"
}

func (a *JSONAdapter) IsAvailable() (bool, error) {
	return true, nil // Always available
}

func (a *JSONAdapter) CreateItems(response *core.ParseResponse, config Config) (*CreateResult, error) {
	output, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if a.dryRun {
		fmt.Println("[dry-run] Would write:")
		fmt.Println(string(output))
	} else if a.outputPath != "" {
		if err := os.WriteFile(a.outputPath, output, 0644); err != nil {
			return nil, fmt.Errorf("failed to write file: %w", err)
		}
		fmt.Printf("Tasks written to %s\n", a.outputPath)
	} else {
		fmt.Println(string(output))
	}

	// Build result from response
	result := &CreateResult{
		Created:      []CreatedItem{},
		Failed:       []FailedItem{},
		Dependencies: []Dependency{},
		Stats:        Stats{},
	}

	// Add all items as "created"
	for _, epic := range response.Epics {
		result.Created = append(result.Created, CreatedItem{
			ExternalID:       fmt.Sprintf("epic-%s", epic.TempID),
			TempID:           epic.TempID,
			Type:             "epic",
			Title:            epic.Title,
			ParentExternalID: "",
		})
		result.Stats.Epics++

		for _, task := range epic.Tasks {
			result.Created = append(result.Created, CreatedItem{
				ExternalID:       fmt.Sprintf("task-%s", task.TempID),
				TempID:           task.TempID,
				Type:             "task",
				Title:            task.Title,
				ParentExternalID: fmt.Sprintf("epic-%s", epic.TempID),
			})
			result.Stats.Tasks++

			for _, subtask := range task.Subtasks {
				result.Created = append(result.Created, CreatedItem{
					ExternalID:       fmt.Sprintf("subtask-%s", subtask.TempID),
					TempID:           subtask.TempID,
					Type:             "subtask",
					Title:            subtask.Title,
					ParentExternalID: fmt.Sprintf("task-%s", task.TempID),
				})
				result.Stats.Subtasks++
			}
		}
	}

	// Add dependencies
	for _, epic := range response.Epics {
		for _, dep := range epic.DependsOn {
			result.Dependencies = append(result.Dependencies, Dependency{
				From: fmt.Sprintf("epic-%s", dep),
				To:   fmt.Sprintf("epic-%s", epic.TempID),
				Type: "blocks",
			})
			result.Stats.Dependencies++
		}

		for _, task := range epic.Tasks {
			// Parent-child
			result.Dependencies = append(result.Dependencies, Dependency{
				From: fmt.Sprintf("epic-%s", epic.TempID),
				To:   fmt.Sprintf("task-%s", task.TempID),
				Type: "parent-child",
			})
			result.Stats.Dependencies++

			for _, dep := range task.DependsOn {
				result.Dependencies = append(result.Dependencies, Dependency{
					From: fmt.Sprintf("task-%s", dep),
					To:   fmt.Sprintf("task-%s", task.TempID),
					Type: "blocks",
				})
				result.Stats.Dependencies++
			}

			for _, subtask := range task.Subtasks {
				// Parent-child
				result.Dependencies = append(result.Dependencies, Dependency{
					From: fmt.Sprintf("task-%s", task.TempID),
					To:   fmt.Sprintf("subtask-%s", subtask.TempID),
					Type: "parent-child",
				})
				result.Stats.Dependencies++

				for _, dep := range subtask.DependsOn {
					result.Dependencies = append(result.Dependencies, Dependency{
						From: fmt.Sprintf("subtask-%s", dep),
						To:   fmt.Sprintf("subtask-%s", subtask.TempID),
						Type: "blocks",
					})
					result.Stats.Dependencies++
				}
			}
		}
	}

	return result, nil
}
