package core

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

// InteractiveParser wraps multi-stage parsing with human-in-the-loop review.
// It pauses after Stage 1 (epics) for human review before continuing.
type InteractiveParser struct {
	generator  Generator
	config     ParseConfig
	prdContent string // Stored for full-context mode
}

// NewInteractiveParser creates an interactive parser.
func NewInteractiveParser(generator Generator, config ParseConfig) *InteractiveParser {
	return &InteractiveParser{
		generator: generator,
		config:    config,
	}
}

// Parse executes the interactive multi-stage parsing pipeline.
func (p *InteractiveParser) Parse(ctx context.Context, prdContent string) (*ParseResponse, error) {
	// Store PRD for full-context mode
	p.prdContent = prdContent

	if p.config.FullContext {
		fmt.Println("Full context mode: PRD will be passed to all stages")
	}

	// Stage 1: Generate epics (high-level only)
	fmt.Println("Stage 1: Generating epics from PRD...")
	epicsResp, err := p.generator.GenerateEpics(ctx, prdContent, p.config)
	if err != nil {
		return nil, fmt.Errorf("stage 1 (epics) failed: %w", err)
	}

	// Convert to full epics for display
	epics := summariesToEpics(epicsResp.Epics)

	// Interactive: Review epics
	fmt.Printf("\n=== Stage 1 Complete: %d Epics Generated ===\n", len(epics))
	printEpicsSummary(epics)

	epics, err = p.interactiveEpicReview(ctx, epics, prdContent, epicsResp.Project)
	if err != nil {
		return nil, fmt.Errorf("epic review failed: %w", err)
	}

	// Stage 2: Generate tasks for each epic (parallel)
	fmt.Println("\nStage 2: Generating tasks for each epic...")
	epics, err = p.generateTasksParallel(ctx, epics, epicsResp.Project)
	if err != nil {
		return nil, fmt.Errorf("stage 2 (tasks) failed: %w", err)
	}

	// Count total tasks
	totalTasks := 0
	for _, epic := range epics {
		totalTasks += len(epic.Tasks)
	}
	fmt.Printf("  Generated %d tasks across %d epics\n", totalTasks, len(epics))

	// Stage 3: Generate subtasks for each task (parallel)
	fmt.Println("\nStage 3: Generating subtasks for each task...")
	epics, err = p.generateSubtasksParallel(ctx, epics, epicsResp.Project)
	if err != nil {
		return nil, fmt.Errorf("stage 3 (subtasks) failed: %w", err)
	}

	// Count total subtasks
	totalSubtasks := 0
	for _, epic := range epics {
		for _, task := range epic.Tasks {
			totalSubtasks += len(task.Subtasks)
		}
	}
	fmt.Printf("  Generated %d subtasks\n", totalSubtasks)

	// Build final response
	response := &ParseResponse{
		Project: epicsResp.Project,
		Epics:   epics,
		Metadata: ResponseMetadata{
			TotalEpics:    len(epics),
			TotalTasks:    totalTasks,
			TotalSubtasks: totalSubtasks,
			TestingCoverage: TestingCoverage{
				HasUnitTests:        true,
				HasIntegrationTests: true,
				HasTypeTests:        true,
				HasE2ETests:         true,
			},
		},
	}

	return response, nil
}

// interactiveEpicReview handles human review of epics.
func (p *InteractiveParser) interactiveEpicReview(ctx context.Context, epics []Epic, prdContent string, project ProjectContext) ([]Epic, error) {
	for {
		fmt.Print("\n[Enter] continue, [e] edit in $EDITOR, [r] regenerate, [a] add epic: ")

		choice, err := promptChoice()
		if err != nil {
			return nil, err
		}

		switch choice {
		case "": // Enter - continue
			return epics, nil

		case "e": // Edit in editor
			edited, err := reviewEpicsInEditor(epics)
			if err != nil {
				fmt.Printf("Edit failed: %v\n", err)
				continue
			}
			if len(edited) == 0 {
				fmt.Println("No epics after edit. Keeping original.")
				continue
			}
			epics = edited
			fmt.Printf("\nUpdated epics (%d total):\n", len(epics))
			printEpicsSummary(epics)

		case "r": // Regenerate
			fmt.Println("\nRegenerating epics...")
			epicsResp, err := p.generator.GenerateEpics(ctx, prdContent, p.config)
			if err != nil {
				fmt.Printf("Regeneration failed: %v\n", err)
				continue
			}
			epics = summariesToEpics(epicsResp.Epics)
			fmt.Printf("\nRegenerated %d epics:\n", len(epics))
			printEpicsSummary(epics)

		case "a": // Add epic
			fmt.Print("Enter epic title: ")
			title, _ := promptChoice()
			title = strings.TrimSpace(title)
			if title == "" {
				fmt.Println("Empty title, not adding.")
				continue
			}
			fmt.Print("Enter epic description (or Enter to skip): ")
			desc, _ := promptChoice()
			desc = strings.TrimSpace(desc)

			newID := strconv.Itoa(len(epics) + 1)
			epics = append(epics, Epic{
				TempID:             newID,
				Title:              title,
				Description:        desc,
				AcceptanceCriteria: []string{},
				Tasks:              []Task{},
				DependsOn:          []string{"1"}, // Depends on foundation
			})
			fmt.Printf("\nAdded epic %s: %s\n", newID, title)
			printEpicsSummary(epics)

		default:
			fmt.Println("Unknown option. Press Enter to continue.")
		}
	}
}

// summariesToEpics converts EpicSummary slice to Epic slice.
func summariesToEpics(summaries []EpicSummary) []Epic {
	epics := make([]Epic, len(summaries))
	for i, es := range summaries {
		epics[i] = Epic{
			TempID:             es.TempID,
			Title:              es.Title,
			Description:        es.Description,
			Context:            es.Context,
			AcceptanceCriteria: es.AcceptanceCriteria,
			Testing:            es.Testing,
			DependsOn:          es.DependsOn,
			EstimatedDays:      es.EstimatedDays,
			Labels:             es.Labels,
			Tasks:              []Task{}, // Will be filled in Stage 2
		}
	}
	return epics
}

// generateTasksParallel generates tasks for all epics in parallel.
func (p *InteractiveParser) generateTasksParallel(ctx context.Context, epics []Epic, project ProjectContext) ([]Epic, error) {
	errs := make([]error, len(epics))
	results := make([][]Task, len(epics))

	var wg sync.WaitGroup
	sem := make(chan struct{}, 3) // Limit parallelism to 3

	for i, epic := range epics {
		wg.Add(1)
		go func(idx int, e Epic) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release

			// Pass PRD content if full-context mode is enabled
			prd := ""
			if p.config.FullContext {
				prd = p.prdContent
			}

			tasks, err := p.generator.GenerateTasks(ctx, e, project, p.config, prd)
			if err != nil {
				errs[idx] = fmt.Errorf("epic %s: %w", e.TempID, err)
				return
			}

			results[idx] = tasks
			fmt.Printf("    Epic %s: %d tasks\n", e.TempID, len(tasks))
		}(i, epic)
	}

	wg.Wait()

	// Check for errors
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}

	// Assign tasks to epics
	for i := range epics {
		epics[i].Tasks = results[i]
	}

	return epics, nil
}

// generateSubtasksParallel generates subtasks for all tasks in parallel.
func (p *InteractiveParser) generateSubtasksParallel(ctx context.Context, epics []Epic, projectCtx ProjectContext) ([]Epic, error) {
	// Collect all tasks to process
	type taskRef struct {
		epicIdx int
		taskIdx int
		task    Task
		epicCtx string
	}

	var taskRefs []taskRef
	for ei, epic := range epics {
		epicCtx := ""
		if epic.Context != nil {
			if s, ok := epic.Context.(string); ok {
				epicCtx = s
			}
		}
		for ti, task := range epic.Tasks {
			taskRefs = append(taskRefs, taskRef{
				epicIdx: ei,
				taskIdx: ti,
				task:    task,
				epicCtx: epicCtx,
			})
		}
	}

	results := make([][]Subtask, len(taskRefs))
	errs := make([]error, len(taskRefs))

	var wg sync.WaitGroup
	sem := make(chan struct{}, 5) // Higher parallelism for subtasks

	for i, ref := range taskRefs {
		wg.Add(1)
		go func(idx int, r taskRef) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release

			// Pass PRD content if full-context mode is enabled
			prd := ""
			if p.config.FullContext {
				prd = p.prdContent
			}

			subtasks, err := p.generator.GenerateSubtasks(ctx, r.task, r.epicCtx, projectCtx, p.config, prd)
			if err != nil {
				errs[idx] = fmt.Errorf("task %s: %w", r.task.TempID, err)
				return
			}

			results[idx] = subtasks
		}(i, ref)
	}

	wg.Wait()

	// Check for errors
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}

	// Assign subtasks back to tasks
	for i, ref := range taskRefs {
		epics[ref.epicIdx].Tasks[ref.taskIdx].Subtasks = results[i]
	}

	return epics, nil
}

// ---- Interactive Helpers ----

// promptChoice prompts the user for input and returns the trimmed string.
func promptChoice() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(strings.ToLower(input)), nil
}

// printEpicsSummary prints a summary of epics for review.
func printEpicsSummary(epics []Epic) {
	fmt.Println("\nProposed Epics:")
	for _, epic := range epics {
		deps := ""
		if len(epic.DependsOn) > 0 {
			deps = fmt.Sprintf(" (depends on: %s)", strings.Join(epic.DependsOn, ", "))
		}
		fmt.Printf("  %s. %s%s\n", epic.TempID, epic.Title, deps)
		if epic.Description != "" {
			// Truncate long descriptions
			desc := epic.Description
			if len(desc) > 60 {
				desc = desc[:57] + "..."
			}
			fmt.Printf("      %s\n", desc)
		}
	}
}

// reviewEpicsInEditor opens an editor with epics for human review.
func reviewEpicsInEditor(epics []Epic) ([]Epic, error) {
	// Format epics as editable text
	content := formatEpicsForEdit(epics)

	// Write to temp file
	tmpFile, err := os.CreateTemp("", "prd-epics-*.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	// Open editor
	if err := openEditor(tmpFile.Name()); err != nil {
		return nil, fmt.Errorf("failed to open editor: %w", err)
	}

	// Read back and parse
	edited, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read edited file: %w", err)
	}

	return parseEditedEpics(string(edited), epics)
}

// formatEpicsForEdit formats epics in an editable text format.
func formatEpicsForEdit(epics []Epic) string {
	var sb strings.Builder

	sb.WriteString("# Epic Review\n")
	sb.WriteString("# ------------\n")
	sb.WriteString("# Reorder lines to change epic order.\n")
	sb.WriteString("# Edit titles and descriptions inline.\n")
	sb.WriteString("# Delete a line or prefix with # to remove an epic.\n")
	sb.WriteString("# Add new: + Title | Description\n")
	sb.WriteString("# Save and close to continue. Empty file to cancel.\n")
	sb.WriteString("#\n")
	sb.WriteString("# Format: NUMBER. TITLE | DESCRIPTION\n")
	sb.WriteString("# ---\n\n")

	for _, epic := range epics {
		sb.WriteString(fmt.Sprintf("%s. %s | %s\n", epic.TempID, epic.Title, epic.Description))
	}

	return sb.String()
}

// parseEditedEpics parses the edited text back into epics.
func parseEditedEpics(content string, originals []Epic) ([]Epic, error) {
	// Build a map of original epics by temp_id for field preservation
	originalMap := make(map[string]Epic)
	for _, epic := range originals {
		originalMap[epic.TempID] = epic
	}

	var result []Epic
	newID := len(originals) + 1

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for new epic syntax: + Title | Description
		if strings.HasPrefix(line, "+") {
			line = strings.TrimPrefix(line, "+")
			line = strings.TrimSpace(line)

			parts := strings.SplitN(line, "|", 2)
			title := strings.TrimSpace(parts[0])
			desc := ""
			if len(parts) > 1 {
				desc = strings.TrimSpace(parts[1])
			}

			if title != "" {
				result = append(result, Epic{
					TempID:             strconv.Itoa(newID),
					Title:              title,
					Description:        desc,
					AcceptanceCriteria: []string{},
					Tasks:              []Task{},
					DependsOn:          []string{},
				})
				newID++
			}
			continue
		}

		// Parse existing epic: NUMBER. TITLE | DESCRIPTION
		dotIdx := strings.Index(line, ".")
		if dotIdx == -1 {
			continue
		}

		idPart := strings.TrimSpace(line[:dotIdx])
		rest := strings.TrimSpace(line[dotIdx+1:])

		parts := strings.SplitN(rest, "|", 2)
		title := strings.TrimSpace(parts[0])
		desc := ""
		if len(parts) > 1 {
			desc = strings.TrimSpace(parts[1])
		}

		if title == "" {
			continue
		}

		// Check if this is an existing epic (preserve fields)
		if orig, ok := originalMap[idPart]; ok {
			epic := orig
			epic.Title = title
			if desc != "" {
				epic.Description = desc
			}
			result = append(result, epic)
		} else {
			// New epic from reordering or addition
			result = append(result, Epic{
				TempID:             idPart,
				Title:              title,
				Description:        desc,
				AcceptanceCriteria: []string{},
				Tasks:              []Task{},
				DependsOn:          []string{},
			})
		}
	}

	// Renumber epics sequentially
	for i := range result {
		result[i].TempID = strconv.Itoa(i + 1)
	}

	return result, nil
}

// openEditor opens the system editor for the given file.
func openEditor(path string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		// Try common editors
		for _, e := range []string{"nano", "vim", "vi"} {
			if _, err := exec.LookPath(e); err == nil {
				editor = e
				break
			}
		}
	}
	if editor == "" {
		return fmt.Errorf("no editor found - set $EDITOR environment variable")
	}

	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
