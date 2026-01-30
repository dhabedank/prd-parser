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
	prefix         string // Beads issue prefix (e.g., "my-project")
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

// getPrefix retrieves the beads prefix from the database.
func (a *BeadsAdapter) getPrefix() string {
	if a.prefix != "" {
		return a.prefix
	}

	// Get prefix from beads database config
	// The prefix is stored in the config table as issue_prefix
	dbPath := ".beads/beads.db"
	if a.workingDir != "." && a.workingDir != "" {
		dbPath = a.workingDir + "/" + dbPath
	}

	cmd := exec.Command("sqlite3", dbPath, "SELECT value FROM config WHERE key='issue_prefix';")
	output, err := cmd.Output()
	if err == nil {
		prefix := strings.TrimSpace(string(output))
		if prefix != "" {
			a.prefix = prefix
			return a.prefix
		}
	}

	// Fallback: try to extract from bd list output
	cmd = exec.Command("bd", "list", "--limit", "1")
	cmd.Dir = a.workingDir
	output, err = cmd.Output()
	if err == nil {
		// Extract prefix from issue ID like "my-project-abc"
		re := regexp.MustCompile(`\b([\w-]+)-[a-z0-9]{2,}\b`)
		match := re.FindStringSubmatch(string(output))
		if len(match) > 1 {
			a.prefix = match[1]
			return a.prefix
		}
	}

	return "prd" // last resort fallback
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
			// Parent-child relationship established via --parent flag in bd create
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
				// Parent-child relationship established via --parent flag in bd create
			}
		}
	}

	// Phase 4: Establish explicit dependencies (depends_on relationships)
	// Format: taskID depends on blockerID
	for _, epic := range response.Epics {
		epicID := tempToExternal[epic.TempID]
		for _, depTempID := range epic.DependsOn {
			if blockerID, ok := tempToExternal[depTempID]; ok && epicID != "" {
				if err := a.addDependency(epicID, blockerID); err == nil {
					result.Dependencies = append(result.Dependencies, Dependency{From: epicID, To: blockerID, Type: "depends_on"})
					result.Stats.Dependencies++
				}
			}
		}

		for _, task := range epic.Tasks {
			taskID := tempToExternal[task.TempID]
			for _, depTempID := range task.DependsOn {
				if blockerID, ok := tempToExternal[depTempID]; ok && taskID != "" {
					if err := a.addDependency(taskID, blockerID); err == nil {
						result.Dependencies = append(result.Dependencies, Dependency{From: taskID, To: blockerID, Type: "depends_on"})
						result.Stats.Dependencies++
					}
				}
			}

			for _, subtask := range task.Subtasks {
				subtaskID := tempToExternal[subtask.TempID]
				for _, depTempID := range subtask.DependsOn {
					if blockerID, ok := tempToExternal[depTempID]; ok && subtaskID != "" {
						if err := a.addDependency(subtaskID, blockerID); err == nil {
							result.Dependencies = append(result.Dependencies, Dependency{From: subtaskID, To: blockerID, Type: "depends_on"})
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
	acceptance := strings.Join(epic.AcceptanceCriteria, "\n- ")
	if acceptance != "" {
		acceptance = "- " + acceptance
	}

	var estimateMinutes int
	if epic.EstimatedDays != nil {
		estimateMinutes = int(*epic.EstimatedDays * 8 * 60) // 8 hours per day
	}

	// Generate readable ID like "prefix-e1"
	readableID := tempIDToReadableID(a.getPrefix(), epic.TempID)

	return a.runBdCreate(createOptions{
		title:       epic.Title,
		description: desc,
		itemType:    "epic",
		priority:    1, // Epics default to high priority
		acceptance:  acceptance,
		estimate:    estimateMinutes,
		labels:      epic.Labels,
		explicitID:  readableID,
	})
}

func (a *BeadsAdapter) createTask(task *core.Task, parentID string) (string, error) {
	desc := a.buildDescription(task.Description, task.Context, &task.Testing)
	priority := mapPriority(task.Priority)

	var designNotes string
	if task.DesignNotes != nil {
		designNotes = *task.DesignNotes
	}

	var estimateMinutes int
	if task.EstimatedHours != nil {
		estimateMinutes = int(*task.EstimatedHours * 60)
	}

	// Generate readable ID like "prefix-e1t1"
	readableID := tempIDToReadableID(a.getPrefix(), task.TempID)

	// Create without parent (can't use both --id and --parent)
	id, err := a.runBdCreate(createOptions{
		title:       task.Title,
		description: desc,
		itemType:    "task",
		priority:    priority,
		design:      designNotes,
		estimate:    estimateMinutes,
		labels:      task.Labels,
		explicitID:  readableID,
	})
	if err != nil {
		return "", err
	}

	// Set parent relationship after creation
	if parentID != "" {
		if err := a.setParent(id, parentID); err != nil {
			// Don't fail - issue is created, just without parent
			fmt.Printf("Warning: failed to set parent for %s: %v\n", id, err)
		}
	}

	return id, nil
}

func (a *BeadsAdapter) createSubtask(subtask *core.Subtask, parentID string) (string, error) {
	desc := a.buildDescriptionWithContext(subtask.Description, subtask.Context, &subtask.Testing)

	var estimateMinutes int
	if subtask.EstimatedMinutes != nil {
		estimateMinutes = *subtask.EstimatedMinutes
	}

	// Generate readable ID like "prefix-e1t1s1"
	readableID := tempIDToReadableID(a.getPrefix(), subtask.TempID)

	// Create without parent (can't use both --id and --parent)
	id, err := a.runBdCreate(createOptions{
		title:       subtask.Title,
		description: desc,
		itemType:    "task", // Beads uses "task" for subtasks too
		priority:    2,      // Subtasks default to medium
		estimate:    estimateMinutes,
		labels:      subtask.Labels,
		explicitID:  readableID,
	})
	if err != nil {
		return "", err
	}

	// Set parent relationship after creation
	if parentID != "" {
		if err := a.setParent(id, parentID); err != nil {
			// Don't fail - issue is created, just without parent
			fmt.Printf("Warning: failed to set parent for %s: %v\n", id, err)
		}
	}

	return id, nil
}

// setParent sets the parent of an issue using bd update --parent
func (a *BeadsAdapter) setParent(childID, parentID string) error {
	if a.dryRun {
		fmt.Printf("[dry-run] bd update %s --parent %s\n", childID, parentID)
		return nil
	}

	cmd := exec.Command("bd", "update", childID, "--parent", parentID)
	cmd.Dir = a.workingDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("bd update failed: %s", string(output))
	}
	return nil
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

// createOptions holds all parameters for bd create
type createOptions struct {
	title       string
	description string
	itemType    string
	priority    int
	acceptance  string   // Acceptance criteria (epics)
	design      string   // Design notes (tasks)
	estimate    int      // Estimate in minutes
	labels      []string // Labels/tags
	explicitID  string   // Readable ID (e.g., "prefix-e1", "prefix-e1t1")
}

// tempIDToReadableID converts a temp_id like "1", "1.1", or "1.1.1" to a
// readable beads ID like "prefix-e1", "prefix-e1t1", "prefix-e1t1s1".
func tempIDToReadableID(prefix string, tempID string) string {
	parts := strings.Split(tempID, ".")
	if len(parts) == 0 {
		return ""
	}

	var suffix string
	for i, part := range parts {
		if i == 0 {
			suffix += "e" + part // Epic
		} else if i == 1 {
			suffix += "t" + part // Task
		} else {
			suffix += "s" + part // Subtask
		}
	}

	return prefix + "-" + suffix
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

func (a *BeadsAdapter) runBdCreate(opts createOptions) (string, error) {
	args := []string{
		"create",
		opts.title,
		"--description", opts.description,
		"--priority", fmt.Sprintf("%d", opts.priority),
		"--type", opts.itemType,
	}

	// Add explicit readable ID (e.g., "prefix-e1", "prefix-e1t1")
	if opts.explicitID != "" {
		args = append(args, "--id", opts.explicitID)
	}

	// Add acceptance criteria for epics
	if opts.acceptance != "" {
		args = append(args, "--acceptance", opts.acceptance)
	}

	// Add design notes for tasks
	if opts.design != "" {
		args = append(args, "--design", opts.design)
	}

	// Add time estimate
	if opts.estimate > 0 {
		args = append(args, "--estimate", fmt.Sprintf("%d", opts.estimate))
	}

	// Add labels
	if len(opts.labels) > 0 {
		args = append(args, "--labels", strings.Join(opts.labels, ","))
	}

	if a.dryRun {
		fmt.Printf("[dry-run] bd %s\n", strings.Join(args, " "))
		if opts.explicitID != "" {
			return opts.explicitID, nil
		}
		return fmt.Sprintf("dry-%d", len(opts.title)), nil
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

	// If we specified an explicit ID, return that
	if opts.explicitID != "" {
		return opts.explicitID, nil
	}

	// Otherwise extract issue ID from output (e.g., "prefix-a3f8")
	re := regexp.MustCompile(`\b[\w-]+-[a-z0-9]{2,}\b`)
	match := re.FindString(string(output))
	if match == "" {
		return "", fmt.Errorf("could not extract issue ID from: %s", string(output))
	}

	return match, nil
}

// addDependency adds a dependency where dependentID depends on blockerID.
// Syntax: bd dep add <dependent> <blocker>
func (a *BeadsAdapter) addDependency(dependentID, blockerID string) error {
	if a.dryRun {
		fmt.Printf("[dry-run] bd dep add %s %s\n", dependentID, blockerID)
		return nil
	}

	cmd := exec.Command("bd", "dep", "add", dependentID, blockerID)
	cmd.Dir = a.workingDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("bd dep add failed: %s", string(output))
	}
	return nil
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
	case core.PriorityVeryLow:
		return 4
	default:
		return 2
	}
}
