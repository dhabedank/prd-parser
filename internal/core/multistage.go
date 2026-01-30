package core

import (
	"context"
	"fmt"
	"sync"
)

// MultiStageParser implements progressive decomposition:
// Stage 1: PRD → Epics (high-level only)
// Stage 2: For each Epic → Tasks (parallel)
// Stage 3: For each Task → Subtasks (parallel)
// Stage 4: Cross-stage dependency resolution
type MultiStageParser struct {
	generator  Generator
	config     ParseConfig
	prdContent string // Stored for full-context mode
}

// Generator is the interface for LLM generation at each stage.
type Generator interface {
	GenerateEpics(ctx context.Context, prdContent string, config ParseConfig) (*EpicsResponse, error)
	GenerateTasks(ctx context.Context, epic Epic, projectContext ProjectContext, config ParseConfig, prdContent string) ([]Task, error)
	GenerateSubtasks(ctx context.Context, task Task, epicContext string, projectContext ProjectContext, config ParseConfig, prdContent string) ([]Subtask, error)
}

// EpicsResponse is the Stage 1 response - epics without tasks.
type EpicsResponse struct {
	Project ProjectContext `json:"project"`
	Epics   []EpicSummary  `json:"epics"`
}

// EpicSummary is a lightweight epic without tasks (Stage 1).
type EpicSummary struct {
	TempID             string              `json:"temp_id"`
	Title              string              `json:"title"`
	Description        string              `json:"description"`
	Context            interface{}         `json:"context"`
	AcceptanceCriteria []string            `json:"acceptance_criteria"`
	Testing            TestingRequirements `json:"testing"`
	DependsOn          []string            `json:"depends_on"`
	EstimatedDays      *float64            `json:"estimated_days,omitempty"`
	Labels             []string            `json:"labels,omitempty"`
}

// TasksResponse is the Stage 2 response - tasks without subtasks.
type TasksResponse struct {
	Tasks []TaskSummary `json:"tasks"`
}

// TaskSummary is a lightweight task without subtasks (Stage 2).
type TaskSummary struct {
	TempID         string              `json:"temp_id"`
	Title          string              `json:"title"`
	Description    string              `json:"description"`
	Context        interface{}         `json:"context"`
	DesignNotes    *string             `json:"design_notes,omitempty"`
	Testing        TestingRequirements `json:"testing"`
	Priority       Priority            `json:"priority"`
	DependsOn      []string            `json:"depends_on"`
	EstimatedHours *float64            `json:"estimated_hours,omitempty"`
	Labels         []string            `json:"labels,omitempty"`
}

// NewMultiStageParser creates a multi-stage parser.
func NewMultiStageParser(generator Generator, config ParseConfig) *MultiStageParser {
	return &MultiStageParser{
		generator: generator,
		config:    config,
	}
}

// Parse executes the multi-stage parsing pipeline.
func (p *MultiStageParser) Parse(ctx context.Context, prdContent string) (*ParseResponse, error) {
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
	fmt.Printf("  Generated %d epics\n", len(epicsResp.Epics))

	// Stage 2: Generate tasks for each epic (parallel)
	fmt.Println("Stage 2: Generating tasks for each epic...")
	epics, err := p.generateTasksParallel(ctx, epicsResp)
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
	fmt.Println("Stage 3: Generating subtasks for each task...")
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

// generateTasksParallel generates tasks for all epics in parallel.
func (p *MultiStageParser) generateTasksParallel(ctx context.Context, epicsResp *EpicsResponse) ([]Epic, error) {
	epics := make([]Epic, len(epicsResp.Epics))
	errs := make([]error, len(epicsResp.Epics))

	var wg sync.WaitGroup
	sem := make(chan struct{}, 3) // Limit parallelism to 3

	for i, epicSummary := range epicsResp.Epics {
		wg.Add(1)
		go func(idx int, es EpicSummary) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release

			// Convert summary to full epic for task generation
			epic := Epic{
				TempID:             es.TempID,
				Title:              es.Title,
				Description:        es.Description,
				Context:            es.Context,
				AcceptanceCriteria: es.AcceptanceCriteria,
				Testing:            es.Testing,
				DependsOn:          es.DependsOn,
				EstimatedDays:      es.EstimatedDays,
				Labels:             es.Labels,
			}

			// Pass PRD content if full-context mode is enabled
			prd := ""
			if p.config.FullContext {
				prd = p.prdContent
			}

			tasks, err := p.generator.GenerateTasks(ctx, epic, epicsResp.Project, p.config, prd)
			if err != nil {
				errs[idx] = fmt.Errorf("epic %s: %w", es.TempID, err)
				return
			}

			epic.Tasks = tasks
			epics[idx] = epic
			fmt.Printf("    Epic %s: %d tasks\n", es.TempID, len(tasks))
		}(i, epicSummary)
	}

	wg.Wait()

	// Check for errors
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}

	return epics, nil
}

// generateSubtasksParallel generates subtasks for all tasks in parallel.
func (p *MultiStageParser) generateSubtasksParallel(ctx context.Context, epics []Epic, projectCtx ProjectContext) ([]Epic, error) {
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
	sem := make(chan struct{}, 5) // Higher parallelism for subtasks (smaller requests)

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
