package output

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/dhabedank/prd-parser/internal/core"
)

// BeadsAdapter creates issues in beads using the bd CLI.
type BeadsAdapter struct {
	workingDir     string
	dryRun         bool
	includeContext bool
	includeTesting bool
}

// NewBeadsAdapter creates a Beads adapter.
func NewBeadsAdapter(config Config) *BeadsAdapter {
	return &BeadsAdapter{
		workingDir:     config.WorkingDir,
		dryRun:         config.DryRun,
		includeContext: config.IncludeContext,
		includeTesting: config.IncludeTesting,
	}
}

func (a *BeadsAdapter) Name() string {
	return "beads"
}

func (a *BeadsAdapter) IsAvailable() (bool, error) {
	cmd := exec.Command("bd", "--version")
	cmd.Dir = a.workingDir
	if err := cmd.Run(); err != nil {
		return false, nil
	}
	return true, nil
}

func (a *BeadsAdapter) CreateItems(response *core.ParseResponse, config Config) (*CreateResult, error) {
	result := &CreateResult{
		Created:      []CreatedItem{},
		Failed:       []FailedItem{},
		Dependencies: []Dependency{},
		Stats:        Stats{},
	}
	tempToExternal := make(map[string]string)

	// Phase 1: Create all epics
	for _, epic := range response.Epics {
		id, err := a.createEpic(&epic)
		if err != nil {
			result.Failed = append(result.Failed, FailedItem{
				Item:  WorkItem{Type: "epic", TempID: epic.TempID, Title: epic.Title, ParentTempID: ""},
				Error: err.Error(),
			})
			continue
		}
		result.Created = append(result.Created, CreatedItem{
			ExternalID:       id,
			TempID:           epic.TempID,
			Type:             "epic",
			Title:            epic.Title,
			ParentExternalID: "",
		})
		tempToExternal[epic.TempID] = id
		result.Stats.Epics++
	}

	// Phase 2: Create all tasks (as children of epics)
	for _, epic := range response.Epics {
		epicID, ok := tempToExternal[epic.TempID]
		if !ok {
			continue
		}

		for _, task := range epic.Tasks {
			id, err := a.createTask(&task, epicID)
			if err != nil {
				result.Failed = append(result.Failed, FailedItem{
					Item:  WorkItem{Type: "task", TempID: task.TempID, Title: task.Title, ParentTempID: epic.TempID},
					Error: err.Error(),
				})
				continue
			}
			result.Created = append(result.Created, CreatedItem{
				ExternalID:       id,
				TempID:           task.TempID,
				Type:             "task",
				Title:            task.Title,
				ParentExternalID: epicID,
			})
			tempToExternal[task.TempID] = id
			result.Stats.Tasks++

			// Establish parent-child relationship
			if err := a.addDependency(epicID, id, "parent-child"); err == nil {
				result.Dependencies = append(result.Dependencies, Dependency{From: epicID, To: id, Type: "parent-child"})
				result.Stats.Dependencies++
			}
		}
	}

	// Phase 3: Create all subtasks (as children of tasks)
	for _, epic := range response.Epics {
		for _, task := range epic.Tasks {
			taskID, ok := tempToExternal[task.TempID]
			if !ok {
				continue
			}

			for _, subtask := range task.Subtasks {
				id, err := a.createSubtask(&subtask, taskID)
				if err != nil {
					result.Failed = append(result.Failed, FailedItem{
						Item:  WorkItem{Type: "subtask", TempID: subtask.TempID, Title: subtask.Title, ParentTempID: task.TempID},
						Error: err.Error(),
					})
					continue
				}
				result.Created = append(result.Created, CreatedItem{
					ExternalID:       id,
					TempID:           subtask.TempID,
					Type:             "subtask",
					Title:            subtask.Title,
					ParentExternalID: taskID,
				})
				tempToExternal[subtask.TempID] = id
				result.Stats.Subtasks++

				// Establish parent-child relationship
				if err := a.addDependency(taskID, id, "parent-child"); err == nil {
					result.Dependencies = append(result.Dependencies, Dependency{From: taskID, To: id, Type: "parent-child"})
					result.Stats.Dependencies++
				}
			}
		}
	}

	// Phase 4: Establish explicit dependencies (blocks relationships)
	for _, epic := range response.Epics {
		epicID := tempToExternal[epic.TempID]
		for _, depTempID := range epic.DependsOn {
			if blockerID, ok := tempToExternal[depTempID]; ok && epicID != "" {
				if err := a.addDependency(blockerID, epicID, "blocks"); err == nil {
					result.Dependencies = append(result.Dependencies, Dependency{From: blockerID, To: epicID, Type: "blocks"})
					result.Stats.Dependencies++
				}
			}
		}

		for _, task := range epic.Tasks {
			taskID := tempToExternal[task.TempID]
			for _, depTempID := range task.DependsOn {
				if blockerID, ok := tempToExternal[depTempID]; ok && taskID != "" {
					if err := a.addDependency(blockerID, taskID, "blocks"); err == nil {
						result.Dependencies = append(result.Dependencies, Dependency{From: blockerID, To: taskID, Type: "blocks"})
						result.Stats.Dependencies++
					}
				}
			}

			for _, subtask := range task.Subtasks {
				subtaskID := tempToExternal[subtask.TempID]
				for _, depTempID := range subtask.DependsOn {
					if blockerID, ok := tempToExternal[depTempID]; ok && subtaskID != "" {
						if err := a.addDependency(blockerID, subtaskID, "blocks"); err == nil {
							result.Dependencies = append(result.Dependencies, Dependency{From: blockerID, To: subtaskID, Type: "blocks"})
							result.Stats.Dependencies++
						}
					}
				}
			}
		}
	}

	return result, nil
}

func (a *BeadsAdapter) createEpic(epic *core.Epic) (string, error) {
	desc := a.buildDescription(epic.Description, epic.Context, &epic.Testing)
	return a.runBdCreate(epic.Title, desc, "epic", 1)
}

func (a *BeadsAdapter) createTask(task *core.Task, parentID string) (string, error) {
	desc := a.buildDescription(task.Description, task.Context, &task.Testing)
	priority := mapPriority(task.Priority)
	return a.runBdCreate(task.Title, desc, "task", priority)
}

func (a *BeadsAdapter) createSubtask(subtask *core.Subtask, parentID string) (string, error) {
	desc := a.buildDescriptionWithContext(subtask.Description, subtask.Context, &subtask.Testing)
	return a.runBdCreate(subtask.Title, desc, "task", 2) // Beads uses "task" for subtasks too
}

func (a *BeadsAdapter) buildDescription(base string, context interface{}, testing *core.TestingRequirements) string {
	desc := base

	if a.includeContext && context != nil {
		// Handle context as either string or object
		switch ctx := context.(type) {
		case string:
			if ctx != "" {
				desc += fmt.Sprintf("\n\n**Context:** %s", ctx)
			}
		case map[string]interface{}:
			parts := []string{}
			if bc, ok := ctx["business_context"].(string); ok && bc != "" {
				parts = append(parts, fmt.Sprintf("- **Business Context:** %s", bc))
			}
			if tu, ok := ctx["target_users"].(string); ok && tu != "" {
				parts = append(parts, fmt.Sprintf("- **Target Users:** %s", tu))
			}
			if bv, ok := ctx["brand_voice"].(string); ok && bv != "" {
				parts = append(parts, fmt.Sprintf("- **Brand Voice:** %s", bv))
			}
			if sm, ok := ctx["success_metrics"].(string); ok && sm != "" {
				parts = append(parts, fmt.Sprintf("- **Success Metrics:** %s", sm))
			}
			if len(parts) > 0 {
				desc += "\n\n**Context:**\n" + strings.Join(parts, "\n")
			}
		}
	}

	if a.includeTesting && testing != nil {
		parts := []string{}
		if testing.UnitTests != nil {
			parts = append(parts, fmt.Sprintf("- **Unit Tests:** %s", *testing.UnitTests))
		}
		if testing.IntegrationTests != nil {
			parts = append(parts, fmt.Sprintf("- **Integration Tests:** %s", *testing.IntegrationTests))
		}
		if testing.TypeTests != nil {
			parts = append(parts, fmt.Sprintf("- **Type Tests:** %s", *testing.TypeTests))
		}
		if testing.E2ETests != nil {
			parts = append(parts, fmt.Sprintf("- **E2E Tests:** %s", *testing.E2ETests))
		}
		if len(parts) > 0 {
			desc += "\n\n**Testing Requirements:**\n" + strings.Join(parts, "\n")
		}
	}

	return desc
}

func (a *BeadsAdapter) buildDescriptionWithContext(base string, context *string, testing *core.TestingRequirements) string {
	desc := base

	if a.includeContext && context != nil {
		desc += fmt.Sprintf("\n\n**Context:** %s", *context)
	}

	if a.includeTesting && testing != nil {
		parts := []string{}
		if testing.UnitTests != nil {
			parts = append(parts, fmt.Sprintf("- **Unit Tests:** %s", *testing.UnitTests))
		}
		if testing.IntegrationTests != nil {
			parts = append(parts, fmt.Sprintf("- **Integration Tests:** %s", *testing.IntegrationTests))
		}
		if testing.TypeTests != nil {
			parts = append(parts, fmt.Sprintf("- **Type Tests:** %s", *testing.TypeTests))
		}
		if testing.E2ETests != nil {
			parts = append(parts, fmt.Sprintf("- **E2E Tests:** %s", *testing.E2ETests))
		}
		if len(parts) > 0 {
			desc += "\n\n**Testing Requirements:**\n" + strings.Join(parts, "\n")
		}
	}

	return desc
}

func (a *BeadsAdapter) runBdCreate(title, description, itemType string, priority int) (string, error) {
	args := []string{
		"create",
		title,
		"--description", description,
		"--priority", fmt.Sprintf("%d", priority),
		"--type", itemType,
	}

	if a.dryRun {
		fmt.Printf("[dry-run] bd %s\n", strings.Join(args, " "))
		return fmt.Sprintf("dry-%d", len(title)), nil
	}

	cmd := exec.Command("bd", args...)
	cmd.Dir = a.workingDir
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("bd create failed: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("bd create failed: %w", err)
	}

	// Extract issue ID from output (e.g., "beads-test-a3f8" or "myproject-x7f2")
	// Beads uses the format: <prefix>-<hash> where prefix is set during bd init
	re := regexp.MustCompile(`\b[\w-]+-[a-z0-9]{2,}\b`)
	match := re.FindString(string(output))
	if match == "" {
		return "", fmt.Errorf("could not extract issue ID from: %s", string(output))
	}

	return match, nil
}

func (a *BeadsAdapter) addDependency(fromID, toID, depType string) error {
	if a.dryRun {
		fmt.Printf("[dry-run] bd dep add %s %s %s\n", fromID, depType, toID)
		return nil
	}

	cmd := exec.Command("bd", "dep", "add", fromID, depType, toID)
	cmd.Dir = a.workingDir
	return cmd.Run()
}

func mapPriority(p core.Priority) int {
	switch p {
	case core.PriorityCritical:
		return 0
	case core.PriorityHigh:
		return 1
	case core.PriorityMedium:
		return 2
	case core.PriorityLow:
		return 3
	default:
		return 2
	}
}
