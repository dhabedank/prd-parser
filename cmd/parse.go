package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/dhabedank/prd-parser/internal/core"
	"github.com/dhabedank/prd-parser/internal/llm"
	"github.com/dhabedank/prd-parser/internal/output"
	"gopkg.in/yaml.v3"
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
	fromJSON        string // Resume from checkpoint
	saveJSON        string // Save checkpoint
	configFile      string // Config file path
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

	// Checkpoint/resume options
	ParseCmd.Flags().StringVar(&fromJSON, "from-json", "", "Resume from saved JSON checkpoint (skip LLM)")
	ParseCmd.Flags().StringVar(&saveJSON, "save-json", "", "Save generated JSON to file (for resume)")

	// Config file
	ParseCmd.Flags().StringVar(&configFile, "config", "", "Config file (default: .prd-parser.yaml)")
}

func runParse(cmd *cobra.Command, args []string) error {
	prdPath := args[0]

	// Load config file (flags override config file values)
	if err := loadConfig(cmd); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check PRD file exists (unless resuming from JSON)
	if fromJSON == "" {
		if _, err := os.Stat(prdPath); os.IsNotExist(err) {
			return fmt.Errorf("PRD file not found: %s", prdPath)
		}
	}

	// Create output adapter
	outAdapter, outConfig, err := createOutputAdapter()
	if err != nil {
		return fmt.Errorf("failed to create output adapter: %w", err)
	}
	fmt.Printf("Using output: %s\n", outAdapter.Name())

	// Wrap the output adapter to satisfy core.OutputAdapter interface
	wrappedOutput := &outputAdapterWrapper{
		adapter: outAdapter,
		config:  outConfig,
	}

	var parseResponse *core.ParseResponse

	// Either resume from JSON checkpoint or generate new
	if fromJSON != "" {
		// Resume from checkpoint
		fmt.Printf("Resuming from checkpoint: %s\n", fromJSON)
		data, err := os.ReadFile(fromJSON)
		if err != nil {
			return fmt.Errorf("failed to read checkpoint: %w", err)
		}
		parseResponse = &core.ParseResponse{}
		if err := json.Unmarshal(data, parseResponse); err != nil {
			return fmt.Errorf("failed to parse checkpoint JSON: %w", err)
		}
		fmt.Printf("Loaded %d epics from checkpoint\n", len(parseResponse.Epics))
	} else {
		// Generate new via LLM
		llmAdapter, err := createLLMAdapter()
		if err != nil {
			return fmt.Errorf("failed to create LLM adapter: %w", err)
		}
		fmt.Printf("Using LLM: %s\n", llmAdapter.Name())

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

		// Parse PRD
		ctx := context.Background()
		result, err := core.ParsePRD(ctx, core.ParseOptions{
			PRDPath:       prdPath,
			LLMAdapter:    llmAdapter,
			OutputAdapter: nil, // Don't create items yet
			Config:        &config,
		})
		if err != nil {
			return fmt.Errorf("parsing failed: %w", err)
		}
		parseResponse = result.ParseResponse

		// Save checkpoint if requested
		if saveJSON != "" {
			data, err := json.MarshalIndent(parseResponse, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			if err := os.WriteFile(saveJSON, data, 0644); err != nil {
				return fmt.Errorf("failed to save checkpoint: %w", err)
			}
			fmt.Printf("Saved checkpoint to: %s\n", saveJSON)
		}
	}

	// Create items via output adapter
	fmt.Println("Creating items...")
	createResult, err := wrappedOutput.CreateItems(parseResponse)
	if err != nil {
		// Auto-save checkpoint on failure for retry
		checkpointPath := filepath.Join(os.TempDir(), "prd-parser-checkpoint.json")
		data, _ := json.MarshalIndent(parseResponse, "", "  ")
		os.WriteFile(checkpointPath, data, 0644)
		return fmt.Errorf("creating items failed: %w\n\nCheckpoint saved to: %s\nRetry with: prd-parser parse %s --from-json %s", err, checkpointPath, prdPath, checkpointPath)
	}

	// Print summary
	fmt.Println("\n--- Summary ---")
	fmt.Printf("Epics: %d\n", createResult.Stats.Epics)
	fmt.Printf("Tasks: %d\n", createResult.Stats.Tasks)
	fmt.Printf("Subtasks: %d\n", createResult.Stats.Subtasks)
	fmt.Printf("Dependencies: %d\n", createResult.Stats.Dependencies)

	if len(createResult.Failed) > 0 {
		fmt.Printf("\nFailed to create %d items:\n", len(createResult.Failed))
		for _, f := range createResult.Failed {
			fmt.Printf("  - %v: %s\n", f.Item, f.Error)
		}
	}

	return nil
}

// Config file structure
type configFileData struct {
	LLM             string `yaml:"llm"`
	Model           string `yaml:"model"`
	Epics           int    `yaml:"epics"`
	TasksPerEpic    int    `yaml:"tasks_per_epic"`
	SubtasksPerTask int    `yaml:"subtasks_per_task"`
	Priority        string `yaml:"priority"`
	Testing         string `yaml:"testing"`
	Output          string `yaml:"output"`
}

func loadConfig(cmd *cobra.Command) error {
	// Find config file
	configPath := configFile
	if configPath == "" {
		// Check .prd-parser.yaml in current dir
		if _, err := os.Stat(".prd-parser.yaml"); err == nil {
			configPath = ".prd-parser.yaml"
		} else if home, err := os.UserHomeDir(); err == nil {
			// Check ~/.prd-parser.yaml
			homePath := filepath.Join(home, ".prd-parser.yaml")
			if _, err := os.Stat(homePath); err == nil {
				configPath = homePath
			}
		}
	}

	if configPath == "" {
		return nil // No config file, use defaults
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg configFileData
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	fmt.Printf("Loaded config from: %s\n", configPath)

	// Apply config values only if flags weren't explicitly set
	if !cmd.Flags().Changed("llm") && cfg.LLM != "" {
		llmProvider = cfg.LLM
	}
	if !cmd.Flags().Changed("model") && cfg.Model != "" {
		llmModel = cfg.Model
	}
	if !cmd.Flags().Changed("epics") && cfg.Epics > 0 {
		targetEpics = cfg.Epics
	}
	if !cmd.Flags().Changed("tasks") && cfg.TasksPerEpic > 0 {
		tasksPerEpic = cfg.TasksPerEpic
	}
	if !cmd.Flags().Changed("subtasks") && cfg.SubtasksPerTask > 0 {
		subtasksPerTask = cfg.SubtasksPerTask
	}
	if !cmd.Flags().Changed("priority") && cfg.Priority != "" {
		defaultPriority = cfg.Priority
	}
	if !cmd.Flags().Changed("testing") && cfg.Testing != "" {
		testingLevel = cfg.Testing
	}
	if !cmd.Flags().Changed("output") && cfg.Output != "" {
		outputAdapter = cfg.Output
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
