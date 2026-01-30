package core

import (
	"context"
	"fmt"
	"os"
)

// LLMAdapter is the interface for LLM providers used by the parser.
// This matches llm.Adapter but is defined here to avoid import cycles.
type LLMAdapter interface {
	// Name returns the adapter identifier for logging.
	Name() string

	// Generate sends prompts to the LLM and returns parsed response.
	Generate(ctx context.Context, systemPrompt, userPrompt string) (*ParseResponse, error)
}

// OutputCreateResult mirrors output.CreateResult to avoid import cycles.
type OutputCreateResult struct {
	Stats struct {
		Epics        int
		Tasks        int
		Subtasks     int
		Dependencies int
	}
	Failed []struct {
		Item  interface{}
		Error string
	}
}

// OutputAdapter is the interface for output providers used by the parser.
// This matches output.Adapter but is defined here to avoid import cycles.
type OutputAdapter interface {
	// Name returns the adapter identifier for logging.
	Name() string

	// IsAvailable checks if the adapter can be used.
	IsAvailable() (bool, error)

	// CreateItems creates hierarchical items in the target system.
	CreateItems(response *ParseResponse) (*OutputCreateResult, error)
}

// ParseOptions configures the PRD parsing.
type ParseOptions struct {
	// PRDPath is the path to the PRD file.
	PRDPath string

	// LLMAdapter is the LLM to use for generation.
	LLMAdapter LLMAdapter

	// OutputAdapter is where to create tasks.
	OutputAdapter OutputAdapter

	// Config overrides default parsing configuration.
	Config *ParseConfig
}

// ParseResult is the result of parsing a PRD.
type ParseResult struct {
	// Response is the parsed PRD with epics/tasks/subtasks.
	Response *ParseResponse

	// CreateResult is the output from creating items.
	CreateResult *OutputCreateResult
}

// ParsePRD parses a PRD file and creates tasks.
func ParsePRD(ctx context.Context, opts ParseOptions) (*ParseResult, error) {
	// Apply defaults
	config := DefaultParseConfig()
	if opts.Config != nil {
		config = *opts.Config
	}

	// Read PRD content
	content, err := os.ReadFile(opts.PRDPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read PRD file: %w", err)
	}

	// Build prompts
	userPrompt := BuildUserPrompt(string(content), config)

	// Check output adapter availability
	available, err := opts.OutputAdapter.IsAvailable()
	if err != nil {
		return nil, fmt.Errorf("output adapter error: %w", err)
	}
	if !available {
		return nil, fmt.Errorf("output adapter '%s' is not available", opts.OutputAdapter.Name())
	}

	// Generate tasks via LLM
	fmt.Printf("Generating tasks with %s...\n", opts.LLMAdapter.Name())
	response, err := opts.LLMAdapter.Generate(ctx, SystemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	// Count items
	totalTasks := 0
	totalSubtasks := 0
	for _, epic := range response.Epics {
		totalTasks += len(epic.Tasks)
		for _, task := range epic.Tasks {
			totalSubtasks += len(task.Subtasks)
		}
	}
	fmt.Printf("Generated %d epics, %d tasks, %d subtasks\n", len(response.Epics), totalTasks, totalSubtasks)

	// Create tasks in target system
	fmt.Printf("Creating tasks in %s...\n", opts.OutputAdapter.Name())
	createResult, err := opts.OutputAdapter.CreateItems(response)
	if err != nil {
		return nil, fmt.Errorf("failed to create tasks: %w", err)
	}

	fmt.Printf("Created %d items (%d epics, %d tasks, %d subtasks)\n",
		createResult.Stats.Epics+createResult.Stats.Tasks+createResult.Stats.Subtasks,
		createResult.Stats.Epics,
		createResult.Stats.Tasks,
		createResult.Stats.Subtasks,
	)
	if len(createResult.Failed) > 0 {
		fmt.Printf("Warning: %d items failed to create\n", len(createResult.Failed))
	}
	fmt.Printf("Established %d dependencies\n", createResult.Stats.Dependencies)

	return &ParseResult{
		Response:     response,
		CreateResult: createResult,
	}, nil
}
