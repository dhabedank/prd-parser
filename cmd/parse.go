package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yourusername/prd-parser/internal/core"
	"github.com/yourusername/prd-parser/internal/llm"
	"github.com/yourusername/prd-parser/internal/output"
)

var (
	targetEpics     int
	tasksPerEpic    int
	subtasksPerTask int
	defaultPriority string
	testingLevel    string
	llmProvider     string
	llmModel        string
	outputAdapter   string
	outputPath      string
	dryRun          bool
)

// ParseCmd represents the parse command
var ParseCmd = &cobra.Command{
	Use:   "parse <prd-file>",
	Short: "Parse a PRD file and create tasks",
	Long: `Parse a Product Requirements Document and generate hierarchical tasks.

The parser uses AI to analyze the PRD and create:
- Epics (major features/milestones)
- Tasks (work units within epics)
- Subtasks (atomic actions within tasks)

Each item includes context propagation and testing requirements.`,
	Args: cobra.ExactArgs(1),
	RunE: runParse,
}

func init() {
	// Parsing options
	ParseCmd.Flags().IntVarP(&targetEpics, "epics", "e", 3, "Target number of epics")
	ParseCmd.Flags().IntVarP(&tasksPerEpic, "tasks", "t", 5, "Target tasks per epic")
	ParseCmd.Flags().IntVarP(&subtasksPerTask, "subtasks", "s", 4, "Target subtasks per task")
	ParseCmd.Flags().StringVarP(&defaultPriority, "priority", "p", "medium", "Default priority (critical/high/medium/low)")
	ParseCmd.Flags().StringVar(&testingLevel, "testing", "comprehensive", "Testing level (minimal/standard/comprehensive)")

	// LLM options
	ParseCmd.Flags().StringVarP(&llmProvider, "llm", "l", "auto", "LLM provider (auto/claude-cli/codex-cli/anthropic-api)")
	ParseCmd.Flags().StringVarP(&llmModel, "model", "m", "", "Model to use (provider-specific)")

	// Output options
	ParseCmd.Flags().StringVarP(&outputAdapter, "output", "o", "beads", "Output adapter (beads/json)")
	ParseCmd.Flags().StringVar(&outputPath, "output-path", "", "Output path for JSON adapter")
	ParseCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview without creating items")
}

func runParse(cmd *cobra.Command, args []string) error {
	prdPath := args[0]

	// Check PRD file exists
	if _, err := os.Stat(prdPath); os.IsNotExist(err) {
		return fmt.Errorf("PRD file not found: %s", prdPath)
	}

	// Create LLM adapter
	llmAdapter, err := createLLMAdapter()
	if err != nil {
		return fmt.Errorf("failed to create LLM adapter: %w", err)
	}
	fmt.Printf("Using LLM: %s\n", llmAdapter.Name())

	// Create output adapter
	outAdapter, outConfig, err := createOutputAdapter()
	if err != nil {
		return fmt.Errorf("failed to create output adapter: %w", err)
	}
	fmt.Printf("Using output: %s\n", outAdapter.Name())

	// Build config
	priority := core.Priority(defaultPriority)
	config := core.ParseConfig{
		TargetEpics:      targetEpics,
		TasksPerEpic:     tasksPerEpic,
		SubtasksPerTask:  subtasksPerTask,
		DefaultPriority:  priority,
		TestingLevel:     testingLevel,
		PropagateContext: true,
	}

	// Wrap the output adapter to satisfy core.OutputAdapter interface
	wrappedOutput := &outputAdapterWrapper{
		adapter: outAdapter,
		config:  outConfig,
	}

	// Parse PRD
	ctx := context.Background()
	result, err := core.ParsePRD(ctx, core.ParseOptions{
		PRDPath:       prdPath,
		LLMAdapter:    llmAdapter,
		OutputAdapter: wrappedOutput,
		Config:        &config,
	})
	if err != nil {
		return fmt.Errorf("parsing failed: %w", err)
	}

	// Print summary
	fmt.Println("\n--- Summary ---")
	fmt.Printf("Epics: %d\n", result.CreateResult.Stats.Epics)
	fmt.Printf("Tasks: %d\n", result.CreateResult.Stats.Tasks)
	fmt.Printf("Subtasks: %d\n", result.CreateResult.Stats.Subtasks)
	fmt.Printf("Dependencies: %d\n", result.CreateResult.Stats.Dependencies)

	if len(result.CreateResult.Failed) > 0 {
		fmt.Printf("\nFailed to create %d items:\n", len(result.CreateResult.Failed))
		for _, f := range result.CreateResult.Failed {
			fmt.Printf("  - %v: %s\n", f.Item, f.Error)
		}
	}

	return nil
}

func createLLMAdapter() (llm.Adapter, error) {
	config := llm.Config{
		Model:     llmModel,
		PreferCLI: true,
	}

	switch llmProvider {
	case "auto":
		return llm.DetectBestAdapter(config)
	case "claude-cli":
		adapter := llm.NewClaudeCLIAdapter(config)
		if !adapter.IsAvailable() {
			return nil, fmt.Errorf("Claude CLI not available - install Claude Code")
		}
		return adapter, nil
	case "codex-cli":
		adapter := llm.NewCodexCLIAdapter(config)
		if !adapter.IsAvailable() {
			return nil, fmt.Errorf("Codex CLI not available - install Codex")
		}
		return adapter, nil
	case "anthropic-api":
		return llm.NewAnthropicAPIAdapter(config)
	default:
		return nil, fmt.Errorf("unknown LLM provider: %s", llmProvider)
	}
}

func createOutputAdapter() (output.Adapter, output.Config, error) {
	config := output.Config{
		WorkingDir:     ".",
		DryRun:         dryRun,
		IncludeContext: true,
		IncludeTesting: true,
	}

	switch outputAdapter {
	case "beads":
		adapter := output.NewBeadsAdapter(config)
		available, _ := adapter.IsAvailable()
		if !available {
			return nil, config, fmt.Errorf("Beads not available - run 'bd init' first")
		}
		return adapter, config, nil
	case "json":
		return output.NewJSONAdapter(config, outputPath), config, nil
	default:
		return nil, config, fmt.Errorf("unknown output adapter: %s", outputAdapter)
	}
}

// outputAdapterWrapper wraps output.Adapter to satisfy core.OutputAdapter.
// This bridges the interface difference where output.Adapter.CreateItems takes
// a Config parameter but core.OutputAdapter.CreateItems does not.
type outputAdapterWrapper struct {
	adapter output.Adapter
	config  output.Config
}

func (w *outputAdapterWrapper) Name() string {
	return w.adapter.Name()
}

func (w *outputAdapterWrapper) IsAvailable() (bool, error) {
	return w.adapter.IsAvailable()
}

func (w *outputAdapterWrapper) CreateItems(response *core.ParseResponse) (*core.OutputCreateResult, error) {
	result, err := w.adapter.CreateItems(response, w.config)
	if err != nil {
		return nil, err
	}

	// Convert output.CreateResult to core.OutputCreateResult
	coreResult := &core.OutputCreateResult{}
	coreResult.Stats.Epics = result.Stats.Epics
	coreResult.Stats.Tasks = result.Stats.Tasks
	coreResult.Stats.Subtasks = result.Stats.Subtasks
	coreResult.Stats.Dependencies = result.Stats.Dependencies

	for _, f := range result.Failed {
		coreResult.Failed = append(coreResult.Failed, struct {
			Item  interface{}
			Error string
		}{
			Item:  f.Item,
			Error: f.Error,
		})
	}

	return coreResult, nil
}
