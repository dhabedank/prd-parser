# PRD Parser - Portable Task Generation Library (Go)

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a portable Go library/CLI that parses PRDs into structured tasks with guardrails, supporting multiple LLM providers and task management backends.

**Architecture:** A Go module with:
- **Core Engine**: Prompt templates + struct validation (guardrails)
- **LLM Adapters**: Claude Code CLI, Codex CLI (preferred), Anthropic API, OpenAI API (fallback)
- **Output Adapters**: Beads (direct Go API + CLI), JSON, GitHub Issues
- **Distribution**: Single binary CLI, Go library import, MCP server

**Tech Stack:** Go, Cobra (CLI), beads Go API, LLM CLIs/APIs

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                     prd-parser (Go)                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────┐ │
│  │   Prompts   │    │   Structs   │    │   Core Parser       │ │
│  │  (embedded) │───▶│ (validation)│───▶│   (orchestration)   │ │
│  └─────────────┘    └─────────────┘    └──────────┬──────────┘ │
│                                                    │            │
│  ┌────────────────────────────────────────────────┼──────────┐ │
│  │                  LLM Adapters                   │          │ │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐      │          │ │
│  │  │ Claude   │  │  Codex   │  │  Ollama  │ ... ◀┘          │ │
│  │  │ Code CLI │  │   CLI    │  │   CLI    │                 │ │
│  │  └────┬─────┘  └────┬─────┘  └──────────┘                 │ │
│  │       │             │                                      │ │
│  │  ┌────▼─────┐  ┌────▼─────┐                               │ │
│  │  │Anthropic │  │ OpenAI   │  (API fallbacks)              │ │
│  │  │   API    │  │   API    │                               │ │
│  │  └──────────┘  └──────────┘                               │ │
│  └───────────────────────────────────────────────────────────┘ │
│                              │                                  │
│  ┌───────────────────────────┼───────────────────────────────┐ │
│  │                Output Adapters                             │ │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │ │
│  │  │  Beads   │  │  Beads   │  │  GitHub  │  │   JSON   │   │ │
│  │  │  Go API  │  │   CLI    │  │   CLI    │  │  stdout  │   │ │
│  │  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│  Distribution: Single Binary | Go Library | MCP Server         │
└─────────────────────────────────────────────────────────────────┘
```

## Why Go?

1. **Beads ecosystem**: Same language as beads - can use Go API directly
2. **Single binary**: No runtime dependencies (unlike Node.js)
3. **CLI-first LLM access**: Use `claude` and `codex` CLIs when available (pre-authenticated)
4. **Performance**: Faster startup, lower memory than interpreted languages
5. **Guardrails**: Go struct validation + JSON schema enforcement

## LLM Strategy: CLI First, API Fallback

```
User has Claude Code installed?
  ├── YES → Use `claude` CLI (no API key needed, already authenticated)
  └── NO  → Use Anthropic API (requires ANTHROPIC_API_KEY)

User has Codex installed?
  ├── YES → Use `codex` CLI (no API key needed)
  └── NO  → Use OpenAI API (requires OPENAI_API_KEY)
```

This is crucial for portability - users of Claude Code, Codex, Cursor, etc. already have authenticated CLI access.

---

## Project Structure

```
prd-parser/
├── go.mod
├── go.sum
├── main.go                      # CLI entry point
├── cmd/
│   └── parse.go                 # Parse command
├── internal/
│   ├── core/
│   │   ├── parser.go            # Main orchestration
│   │   ├── prompts.go           # Embedded prompt templates
│   │   └── types.go             # Struct definitions (guardrails)
│   ├── llm/
│   │   ├── adapter.go           # LLM adapter interface
│   │   ├── claude_cli.go        # Claude Code CLI adapter
│   │   ├── codex_cli.go         # Codex CLI adapter
│   │   ├── anthropic_api.go     # Anthropic API fallback
│   │   ├── openai_api.go        # OpenAI API fallback
│   │   └── detector.go          # Auto-detect available LLMs
│   ├── output/
│   │   ├── adapter.go           # Output adapter interface
│   │   ├── beads_api.go         # Direct beads Go API
│   │   ├── beads_cli.go         # Beads CLI fallback
│   │   ├── json.go              # JSON output
│   │   └── github.go            # GitHub Issues CLI
│   └── validation/
│       └── schema.go            # JSON schema validation
├── prompts/
│   └── parse_prd.go             # Embedded prompt template
└── tests/
    ├── core_test.go
    ├── llm_test.go
    └── output_test.go
```

---

## Task 1: Initialize Go Project

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `.gitignore`
- Create: `Makefile`

**Step 1: Initialize Go module**

```bash
go mod init github.com/yourusername/prd-parser
```

**Step 2: Create main.go**

```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.1.0"

func main() {
	rootCmd := &cobra.Command{
		Use:     "prd-parser",
		Short:   "Parse PRDs into structured tasks with LLM guardrails",
		Version: version,
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

**Step 3: Create .gitignore**

```
# Binaries
prd-parser
*.exe

# Build
dist/

# IDE
.idea/
.vscode/

# OS
.DS_Store

# Test
coverage.out
```

**Step 4: Create Makefile**

```makefile
.PHONY: build test lint clean install

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o prd-parser .

test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

lint:
	golangci-lint run

clean:
	rm -f prd-parser coverage.out

install: build
	cp prd-parser $(GOPATH)/bin/
```

**Step 5: Add dependencies**

```bash
go get github.com/spf13/cobra
go get github.com/anthropics/anthropic-sdk-go
go mod tidy
```

**Step 6: Verify setup**

Run: `make build`
Expected: Binary `prd-parser` created

**Step 7: Commit**

```bash
git init
git add go.mod go.sum main.go .gitignore Makefile
git commit -m "feat: initialize prd-parser Go project

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 2: Define Core Types (The Guardrails)

**Files:**
- Create: `internal/core/types.go`

**Step 1: Write the hierarchical types**

The types support **Epics → Tasks → Subtasks** with context propagation and testing at every level.

```go
package core

import "encoding/json"

// TestingRequirements captures testing needs at any level.
// Forces consideration of testing at epic, task, and subtask levels.
type TestingRequirements struct {
	UnitTests        *string `json:"unit_tests,omitempty"`        // Functions/methods to test in isolation
	IntegrationTests *string `json:"integration_tests,omitempty"` // How components interact
	TypeTests        *string `json:"type_tests,omitempty"`        // Type safety, runtime validation
	E2ETests         *string `json:"e2e_tests,omitempty"`         // User flows to verify
}

// ContextBlock propagates business context down the hierarchy.
// Keeps LLMs grounded in the business purpose.
type ContextBlock struct {
	BusinessContext *string `json:"business_context,omitempty"` // Why this exists
	TargetUsers     *string `json:"target_users,omitempty"`     // Who this is for
	BrandVoice      *string `json:"brand_voice,omitempty"`      // Brand/UX guidelines
	SuccessMetrics  *string `json:"success_metrics,omitempty"`  // How we know this succeeded
}

// Subtask is the atomic unit of work (30min - 2hrs)
type Subtask struct {
	TempID           string              `json:"temp_id"`                     // Hierarchical ID like "1.1.1"
	Title            string              `json:"title"`                       // Clear, atomic action
	Description      string              `json:"description"`                 // Specific implementation details
	Context          *string             `json:"context,omitempty"`           // Inherited context reminder
	Testing          TestingRequirements `json:"testing"`                     // Testing requirements
	EstimatedMinutes *int                `json:"estimated_minutes,omitempty"` // 15-120 minutes
	DependsOn        []string            `json:"depends_on"`                  // Temp IDs this depends on
}

// Task is a logical unit of work containing subtasks (2-8hrs total)
type Task struct {
	TempID         string              `json:"temp_id"`                   // Hierarchical ID like "1.1"
	Title          string              `json:"title"`                     // Clear, actionable title
	Description    string              `json:"description"`               // What needs to be accomplished
	Context        ContextBlock        `json:"context"`                   // Propagated + task-specific context
	DesignNotes    *string             `json:"design_notes,omitempty"`    // Technical approach
	Testing        TestingRequirements `json:"testing"`                   // Testing strategy
	Priority       Priority            `json:"priority"`                  // critical/high/medium/low
	Subtasks       []Subtask           `json:"subtasks"`                  // Atomic subtasks
	DependsOn      []string            `json:"depends_on"`                // Temp IDs this depends on
	EstimatedHours *float64            `json:"estimated_hours,omitempty"` // Total including subtasks
}

// Epic is a major feature or milestone containing tasks (1-4 weeks)
type Epic struct {
	TempID             string              `json:"temp_id"`                   // Simple ID like "1", "2"
	Title              string              `json:"title"`                     // Major feature or milestone
	Description        string              `json:"description"`               // What this delivers
	Context            ContextBlock        `json:"context"`                   // Business/user/brand context
	AcceptanceCriteria []string            `json:"acceptance_criteria"`       // When this epic is complete
	Testing            TestingRequirements `json:"testing"`                   // Epic-level testing strategy
	Tasks              []Task              `json:"tasks"`                     // Tasks that complete this epic
	DependsOn          []string            `json:"depends_on"`                // Epic temp IDs this depends on
	EstimatedDays      *float64            `json:"estimated_days,omitempty"`  // Working days for entire epic
}

// ProjectContext extracted from the PRD.
// Propagated into every epic, task, and subtask.
type ProjectContext struct {
	ProductName     string   `json:"product_name"`               // Name of the product
	ElevatorPitch   string   `json:"elevator_pitch"`             // One sentence: what and why
	TargetAudience  string   `json:"target_audience"`            // Primary and secondary users
	BusinessGoals   []string `json:"business_goals"`             // What the business wants
	UserGoals       []string `json:"user_goals"`                 // What users want
	BrandGuidelines *string  `json:"brand_guidelines,omitempty"` // Voice, tone, visual identity
	TechStack       []string `json:"tech_stack"`                 // Technologies and tools
	Constraints     []string `json:"constraints"`                // Technical/business constraints
}

// ParseResponse is the full PRD parsing output.
// Hierarchical: Project → Epics → Tasks → Subtasks
type ParseResponse struct {
	Project  ProjectContext   `json:"project"`  // Extracted project context
	Epics    []Epic           `json:"epics"`    // Major features/milestones
	Metadata ResponseMetadata `json:"metadata"` // Summary statistics
}

// ResponseMetadata provides summary stats about the parsed PRD.
type ResponseMetadata struct {
	TotalEpics         int              `json:"total_epics"`
	TotalTasks         int              `json:"total_tasks"`
	TotalSubtasks      int              `json:"total_subtasks"`
	EstimatedTotalDays *float64         `json:"estimated_total_days,omitempty"`
	TestingCoverage    TestingCoverage  `json:"testing_coverage"`
}

// TestingCoverage indicates what test types are included.
type TestingCoverage struct {
	HasUnitTests        bool `json:"has_unit_tests"`
	HasIntegrationTests bool `json:"has_integration_tests"`
	HasTypeTests        bool `json:"has_type_tests"`
	HasE2ETests         bool `json:"has_e2e_tests"`
}

// Priority levels for tasks.
type Priority string

const (
	PriorityCritical Priority = "critical"
	PriorityHigh     Priority = "high"
	PriorityMedium   Priority = "medium"
	PriorityLow      Priority = "low"
)

// ParseConfig configures PRD parsing behavior.
type ParseConfig struct {
	TargetEpics      int      `json:"target_epics"`       // Default: 3
	TasksPerEpic     int      `json:"tasks_per_epic"`     // Default: 5
	SubtasksPerTask  int      `json:"subtasks_per_task"`  // Default: 4
	DefaultPriority  Priority `json:"default_priority"`   // Default: medium
	TestingLevel     string   `json:"testing_level"`      // minimal/standard/comprehensive
	PropagateContext bool     `json:"propagate_context"`  // Default: true
}

// DefaultParseConfig returns sensible defaults.
func DefaultParseConfig() ParseConfig {
	return ParseConfig{
		TargetEpics:      3,
		TasksPerEpic:     5,
		SubtasksPerTask:  4,
		DefaultPriority:  PriorityMedium,
		TestingLevel:     "comprehensive",
		PropagateContext: true,
	}
}

// Validate checks the ParseResponse for required fields and consistency.
func (r *ParseResponse) Validate() error {
	if r.Project.ProductName == "" {
		return &ValidationError{Field: "project.product_name", Message: "required"}
	}
	if len(r.Epics) == 0 {
		return &ValidationError{Field: "epics", Message: "at least one epic required"}
	}
	for i, epic := range r.Epics {
		if epic.Title == "" {
			return &ValidationError{Field: fmt.Sprintf("epics[%d].title", i), Message: "required"}
		}
		if len(epic.Tasks) == 0 {
			return &ValidationError{Field: fmt.Sprintf("epics[%d].tasks", i), Message: "at least one task required"}
		}
	}
	return nil
}

// ValidationError represents a validation failure.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s - %s", e.Field, e.Message)
}
```

**Step 2: Create missing import**

Add `import "fmt"` at the top.

**Step 3: Verify types compile**

Run: `go build ./internal/core/`
Expected: No errors

**Step 4: Commit**

```bash
git add internal/core/types.go
git commit -m "feat: add hierarchical types with context propagation and testing

Supports Epics → Tasks → Subtasks with business context,
user personas, and brand guidelines flowing to all levels.
Testing requirements (unit, integration, type, e2e) at every level.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 3: Create Embedded Prompt Templates

**Files:**
- Create: `internal/core/prompts.go`

**Step 1: Write the embedded prompt template**

Go embeds templates at compile time - no external JSON files needed.

```go
package core

import (
	"fmt"
	"strings"
)

// SystemPrompt is the system instruction for PRD parsing.
// This enforces hierarchical structure, context propagation, and comprehensive testing.
const SystemPrompt = `You are an expert software architect and project manager. Your task is to analyze Product Requirements Documents (PRDs) and generate a HIERARCHICAL, dependency-aware breakdown.

## OUTPUT STRUCTURE

You MUST output:
1. **Project Context** - Extract business purpose, target users, brand guidelines
2. **Epics** - Major features/milestones (1-4 weeks each)
3. **Tasks** - Logical work units within epics (2-8 hours each)
4. **Subtasks** - Atomic actions within tasks (30min-2hrs each)

## CONTEXT PROPAGATION (CRITICAL)

The PRD contains valuable context that gets LOST if not propagated:
- **Business context**: WHY is this being built? What problem does it solve?
- **Target users**: WHO is this for? What are their goals and pain points?
- **Brand guidelines**: What voice/tone/style should the implementation reflect?

You MUST propagate relevant context DOWN the hierarchy:
- Epic context → inherited by all its tasks
- Task context → inherited by all its subtasks
- Each subtask's context field should remind the implementer WHY this matters

Example: If the PRD says "This is for busy parents who need quick meal planning", a subtask for "Implement recipe card component" should include context like: "For busy parents - must be scannable in <5 seconds, mobile-first, show prep time prominently"

## TESTING AT EVERY LEVEL (MANDATORY)

Every epic, task, AND subtask MUST have testing requirements:

- **unit_tests**: What functions/methods need isolated testing?
- **integration_tests**: How do components interact? What APIs to test?
- **type_tests**: Type safety, runtime validation, schema enforcement
- **e2e_tests**: What user flows need end-to-end verification?

Set to null ONLY if genuinely not applicable (rare).

Testing distribution guidelines:
- Setup/config work: type_tests, integration_tests
- UI components: unit_tests, e2e_tests
- API endpoints: unit_tests, integration_tests, type_tests
- Business logic: unit_tests, integration_tests
- User flows: e2e_tests

## HIERARCHY GUIDELINES

**Epics** (temp_id: "1", "2", "3"):
- Major features or milestones
- Should be independently deployable/releasable
- Include acceptance criteria (bullet points)
- 1-4 weeks of work

**Tasks** (temp_id: "1.1", "1.2", "2.1"):
- Logical groupings within an epic
- Design notes for technical approach
- 2-8 hours of work

**Subtasks** (temp_id: "1.1.1", "1.1.2"):
- Atomic, independently completable actions
- Specific enough that an LLM could implement without clarification
- 30 minutes to 2 hours of work

## DEPENDENCIES

- Use temp_ids for dependencies (e.g., "1.1" depends_on ["1.0"])
- Infrastructure/setup epics should come first
- Testing tasks should depend on implementation tasks
- Cross-epic dependencies are allowed

## ANTI-PATTERNS TO AVOID

1. Vague tasks like "Implement feature" - be SPECIFIC
2. Missing context - every subtask should know WHY it matters
3. Skipping tests - testing is NOT optional
4. Flat structure - USE the hierarchy
5. Disconnected work - every item should trace to business value`

// UserPromptTemplate is the template for user messages.
const UserPromptTemplate = `Analyze this PRD and generate a hierarchical breakdown.

Target structure:
- %d epics
- ~%d tasks per epic
- ~%d subtasks per task

Default priority: %s
Testing level: %s
Propagate context: %t

---
PRD CONTENT:
---
%s
---

Generate a JSON object with:

1. "project" - Extracted context (product_name, elevator_pitch, target_audience, business_goals, user_goals, brand_guidelines, tech_stack, constraints)

2. "epics" - Array with temp_id, title, description, context, acceptance_criteria, testing, tasks, depends_on, estimated_days

3. Each task with temp_id, title, description, context, design_notes, testing, subtasks, priority, depends_on, estimated_hours

4. Each subtask with temp_id, title, description, context, testing, estimated_minutes, depends_on

5. "metadata" - Counts and testing coverage summary

IMPORTANT: Propagate context! Every subtask should remind the implementer of the business purpose and user needs.

Return ONLY valid JSON, no markdown fencing.`

// BuildUserPrompt renders the user prompt with config values.
func BuildUserPrompt(prdContent string, config ParseConfig) string {
	return fmt.Sprintf(
		UserPromptTemplate,
		config.TargetEpics,
		config.TasksPerEpic,
		config.SubtasksPerTask,
		config.DefaultPriority,
		config.TestingLevel,
		config.PropagateContext,
		prdContent,
	)
}
```

**Step 2: Verify compiles**

Run: `go build ./internal/core/`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/core/prompts.go
git commit -m "feat: add embedded prompt templates for PRD parsing

Includes system prompt with context propagation and testing requirements.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 4: Define LLM Adapter Interface

**Files:**
- Create: `internal/llm/adapter.go`

**Step 1: Write the LLM adapter interface**

```go
package llm

import (
	"context"

	"github.com/yourusername/prd-parser/internal/core"
)

// Adapter is the interface all LLM adapters must implement.
type Adapter interface {
	// Name returns the adapter identifier for logging.
	Name() string

	// IsAvailable checks if this adapter can be used (CLI installed, API key set, etc.)
	IsAvailable() bool

	// Generate sends prompts to the LLM and returns parsed response.
	Generate(ctx context.Context, systemPrompt, userPrompt string) (*core.ParseResponse, error)
}

// Config holds configuration for LLM adapters.
type Config struct {
	// PreferCLI prefers CLI tools (claude, codex) over API when available.
	PreferCLI bool

	// Model specifies which model to use (optional, adapter chooses default).
	Model string

	// APIKey for direct API access (optional if CLI is used).
	APIKey string

	// MaxTokens limits response length.
	MaxTokens int
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		PreferCLI: true, // Use CLI tools when available (already authenticated)
		MaxTokens: 16384,
	}
}
```

**Step 2: Commit**

```bash
git add internal/llm/adapter.go
git commit -m "feat: define LLM adapter interface with CLI preference

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 5: Implement Claude CLI Adapter (Primary)

**Files:**
- Create: `internal/llm/claude_cli.go`

**Step 1: Write the Claude Code CLI adapter**

This adapter uses the `claude` CLI when available (users already have it authenticated).

```go
package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/yourusername/prd-parser/internal/core"
)

// ClaudeCLIAdapter uses the Claude Code CLI for generation.
// This is preferred because users already have it authenticated.
type ClaudeCLIAdapter struct {
	model string
}

// NewClaudeCLIAdapter creates a Claude CLI adapter.
func NewClaudeCLIAdapter(config Config) *ClaudeCLIAdapter {
	model := config.Model
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	return &ClaudeCLIAdapter{model: model}
}

func (a *ClaudeCLIAdapter) Name() string {
	return "claude-cli"
}

// IsAvailable checks if the claude CLI is installed.
func (a *ClaudeCLIAdapter) IsAvailable() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

func (a *ClaudeCLIAdapter) Generate(ctx context.Context, systemPrompt, userPrompt string) (*core.ParseResponse, error) {
	// Write prompts to temp files (claude CLI reads from files better than stdin for long content)
	systemFile, err := os.CreateTemp("", "prd-system-*.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to create system prompt file: %w", err)
	}
	defer os.Remove(systemFile.Name())

	userFile, err := os.CreateTemp("", "prd-user-*.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to create user prompt file: %w", err)
	}
	defer os.Remove(userFile.Name())

	if _, err := systemFile.WriteString(systemPrompt); err != nil {
		return nil, fmt.Errorf("failed to write system prompt: %w", err)
	}
	systemFile.Close()

	if _, err := userFile.WriteString(userPrompt); err != nil {
		return nil, fmt.Errorf("failed to write user prompt: %w", err)
	}
	userFile.Close()

	// Build claude command
	// claude --model <model> --system-prompt-file <file> --print "<user prompt file>"
	cmd := exec.CommandContext(ctx, "claude",
		"--model", a.model,
		"--system-prompt-file", systemFile.Name(),
		"--print",
		"--output-format", "text",
	)

	// Pass user prompt via stdin
	userContent, _ := os.ReadFile(userFile.Name())
	cmd.Stdin = strings.NewReader(string(userContent))

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("claude CLI failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("claude CLI failed: %w", err)
	}

	// Parse JSON from output
	return parseJSONResponse(string(output))
}

// parseJSONResponse extracts and validates JSON from LLM output.
func parseJSONResponse(output string) (*core.ParseResponse, error) {
	// Find JSON in output (may be wrapped in markdown fences)
	output = strings.TrimSpace(output)

	// Remove markdown fences if present
	if strings.HasPrefix(output, "```json") {
		output = strings.TrimPrefix(output, "```json")
		output = strings.TrimSuffix(output, "```")
		output = strings.TrimSpace(output)
	} else if strings.HasPrefix(output, "```") {
		output = strings.TrimPrefix(output, "```")
		output = strings.TrimSuffix(output, "```")
		output = strings.TrimSpace(output)
	}

	// Find JSON object
	start := strings.Index(output, "{")
	end := strings.LastIndex(output, "}")
	if start == -1 || end == -1 || end < start {
		return nil, fmt.Errorf("no valid JSON found in response")
	}

	jsonStr := output[start : end+1]

	var response core.ParseResponse
	if err := json.Unmarshal([]byte(jsonStr), &response); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Validate
	if err := response.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return &response, nil
}
```

**Step 2: Commit**

```bash
git add internal/llm/claude_cli.go
git commit -m "feat: implement Claude Code CLI adapter

Uses 'claude' CLI when available - no API key needed.
Users of Claude Code already have it authenticated.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 6: Implement Codex CLI Adapter

**Files:**
- Create: `internal/llm/codex_cli.go`

**Step 1: Write the Codex CLI adapter**

```go
package llm

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/yourusername/prd-parser/internal/core"
)

// CodexCLIAdapter uses the Codex CLI for generation.
type CodexCLIAdapter struct {
	model string
}

// NewCodexCLIAdapter creates a Codex CLI adapter.
func NewCodexCLIAdapter(config Config) *CodexCLIAdapter {
	model := config.Model
	if model == "" {
		model = "o3" // Default to o3 for best reasoning
	}
	return &CodexCLIAdapter{model: model}
}

func (a *CodexCLIAdapter) Name() string {
	return "codex-cli"
}

// IsAvailable checks if the codex CLI is installed.
func (a *CodexCLIAdapter) IsAvailable() bool {
	_, err := exec.LookPath("codex")
	return err == nil
}

func (a *CodexCLIAdapter) Generate(ctx context.Context, systemPrompt, userPrompt string) (*core.ParseResponse, error) {
	// Codex uses a slightly different invocation pattern
	// Combine system + user prompts for codex
	combinedPrompt := fmt.Sprintf("SYSTEM INSTRUCTIONS:\n%s\n\nUSER REQUEST:\n%s", systemPrompt, userPrompt)

	// Write to temp file
	promptFile, err := os.CreateTemp("", "prd-prompt-*.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to create prompt file: %w", err)
	}
	defer os.Remove(promptFile.Name())

	if _, err := promptFile.WriteString(combinedPrompt); err != nil {
		return nil, fmt.Errorf("failed to write prompt: %w", err)
	}
	promptFile.Close()

	// Run codex
	cmd := exec.CommandContext(ctx, "codex",
		"--model", a.model,
		"--quiet", // Less verbose output
	)
	cmd.Stdin = strings.NewReader(combinedPrompt)

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("codex CLI failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("codex CLI failed: %w", err)
	}

	return parseJSONResponse(string(output))
}
```

**Step 2: Commit**

```bash
git add internal/llm/codex_cli.go
git commit -m "feat: implement Codex CLI adapter

Uses 'codex' CLI when available - no API key needed.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 6b: Implement Anthropic API Adapter (Fallback)

**Files:**
- Create: `internal/llm/anthropic_api.go`

**Step 1: Write the Anthropic API fallback adapter**

Used when Claude CLI is not available.

```go
package llm

import (
	"context"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/yourusername/prd-parser/internal/core"
)

// AnthropicAPIAdapter uses the Anthropic API directly.
// Fallback when Claude CLI is not available.
type AnthropicAPIAdapter struct {
	client    *anthropic.Client
	model     string
	maxTokens int
}

// NewAnthropicAPIAdapter creates an Anthropic API adapter.
func NewAnthropicAPIAdapter(config Config) (*AnthropicAPIAdapter, error) {
	apiKey := config.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	model := config.Model
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	maxTokens := config.MaxTokens
	if maxTokens == 0 {
		maxTokens = 16384
	}

	return &AnthropicAPIAdapter{
		client:    client,
		model:     model,
		maxTokens: maxTokens,
	}, nil
}

func (a *AnthropicAPIAdapter) Name() string {
	return "anthropic-api"
}

func (a *AnthropicAPIAdapter) IsAvailable() bool {
	return os.Getenv("ANTHROPIC_API_KEY") != ""
}

func (a *AnthropicAPIAdapter) Generate(ctx context.Context, systemPrompt, userPrompt string) (*core.ParseResponse, error) {
	resp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.F(a.model),
		MaxTokens: anthropic.Int(int64(a.maxTokens)),
		System: anthropic.F([]anthropic.TextBlockParam{
			anthropic.NewTextBlock(systemPrompt),
		}),
		Messages: anthropic.F([]anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt)),
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("anthropic API error: %w", err)
	}

	// Extract text from response
	var output string
	for _, block := range resp.Content {
		if block.Type == anthropic.ContentBlockTypeText {
			output += block.Text
		}
	}

	return parseJSONResponse(output)
}
```

**Step 2: Commit**

```bash
git add internal/llm/anthropic_api.go
git commit -m "feat: implement Anthropic API adapter as fallback

Used when Claude CLI is not available.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 6c: Implement LLM Auto-Detection

**Files:**
- Create: `internal/llm/detector.go`

**Step 1: Write the auto-detector**

Automatically selects the best available LLM adapter.

```go
package llm

import (
	"fmt"
)

// DetectBestAdapter finds the best available LLM adapter.
// Priority: Claude CLI > Codex CLI > Anthropic API > OpenAI API
func DetectBestAdapter(config Config) (Adapter, error) {
	// Try Claude CLI first (preferred - already authenticated)
	if config.PreferCLI {
		claude := NewClaudeCLIAdapter(config)
		if claude.IsAvailable() {
			return claude, nil
		}

		// Try Codex CLI
		codex := NewCodexCLIAdapter(config)
		if codex.IsAvailable() {
			return codex, nil
		}
	}

	// Fall back to Anthropic API
	anthropic, err := NewAnthropicAPIAdapter(config)
	if err == nil && anthropic.IsAvailable() {
		return anthropic, nil
	}

	// Could add OpenAI API fallback here

	return nil, fmt.Errorf("no LLM adapter available - install Claude Code, Codex, or set ANTHROPIC_API_KEY")
}

// ListAvailableAdapters returns all adapters that could be used.
func ListAvailableAdapters(config Config) []string {
	var available []string

	claude := NewClaudeCLIAdapter(config)
	if claude.IsAvailable() {
		available = append(available, "claude-cli")
	}

	codex := NewCodexCLIAdapter(config)
	if codex.IsAvailable() {
		available = append(available, "codex-cli")
	}

	anthropic, _ := NewAnthropicAPIAdapter(config)
	if anthropic != nil && anthropic.IsAvailable() {
		available = append(available, "anthropic-api")
	}

	return available
}
```

**Step 2: Commit**

```bash
git add internal/llm/detector.go
git commit -m "feat: implement LLM auto-detection

Priority: Claude CLI > Codex CLI > Anthropic API
Users with Claude Code or Codex get zero-config experience.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 7: Define Output Adapter Interface

**Files:**
- Create: `src/output/types.ts`

**Step 1: Write the output adapter interface**

Updated to support hierarchical items (epics, tasks, subtasks) and parent-child relationships.

```typescript
import type { Epic, Task, Subtask, ParseResponse } from '../core/schemas.js';

/**
 * Represents any work item in the hierarchy
 */
export type WorkItem = {
  type: 'epic' | 'task' | 'subtask';
  tempId: string;
  title: string;
  parentTempId: string | null;
};

/**
 * Result of creating a single item in the target system
 */
export interface CreatedItem {
  /** ID assigned by the target system (e.g., bd-a3f8) */
  externalId: string;
  /** Original temp_id for dependency/parent mapping */
  tempId: string;
  /** Item type */
  type: 'epic' | 'task' | 'subtask';
  /** Title for display */
  title: string;
  /** Parent's external ID (if has parent) */
  parentExternalId: string | null;
}

/**
 * Result of creating all items
 */
export interface CreateResult {
  /** Successfully created items */
  created: CreatedItem[];
  /** Items that failed to create */
  failed: Array<{ item: WorkItem; error: string }>;
  /** Dependencies established */
  dependencies: Array<{ from: string; to: string; type: 'blocks' | 'parent-child' }>;
  /** Summary stats */
  stats: {
    epics: number;
    tasks: number;
    subtasks: number;
    dependencies: number;
  };
}

/**
 * Interface all output adapters must implement
 */
export interface OutputAdapter {
  /** Adapter name for logging */
  readonly name: string;

  /**
   * Create hierarchical items in the target system
   * @param response - Parsed PRD response with epics/tasks/subtasks
   * @returns Creation results with external IDs
   */
  createItems(response: ParseResponse): Promise<CreateResult>;

  /**
   * Check if the adapter is available (e.g., CLI installed)
   */
  isAvailable(): Promise<boolean>;
}

/**
 * Configuration for output adapters
 */
export interface OutputConfig {
  /** Working directory for CLI-based adapters */
  workingDir?: string;
  /** Dry run - don't actually create items */
  dryRun?: boolean;
  /** Include context in descriptions */
  includeContext?: boolean;
  /** Include testing requirements in items */
  includeTesting?: boolean;
  /** Additional adapter-specific options */
  [key: string]: unknown;
}

export type OutputAdapterFactory = (config: OutputConfig) => OutputAdapter;
```

**Step 2: Commit**

```bash
git add src/output/types.ts
git commit -m "feat: define hierarchical output adapter interface

Supports epics, tasks, subtasks with parent-child relationships.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 8: Implement Beads Output Adapter

**Files:**
- Create: `src/output/beads.ts`

**Step 1: Write the Beads adapter**

Creates hierarchical issues using beads' parent-child dependency type.

```typescript
import { exec } from 'child_process';
import { promisify } from 'util';
import type { Epic, Task, Subtask, ParseResponse } from '../core/schemas.js';
import type { OutputAdapter, OutputConfig, CreateResult, CreatedItem, WorkItem } from './types.js';

const execAsync = promisify(exec);

/**
 * Map prd-parser priority to beads priority (0-4)
 */
function mapPriority(priority: string): number {
  const map: Record<string, number> = { critical: 0, high: 1, medium: 2, low: 3 };
  return map[priority] ?? 2;
}

/**
 * Build description with context and testing info
 */
function buildDescription(
  base: string,
  context: Record<string, string | null> | string | null,
  testing: Record<string, string | null> | null,
  includeContext: boolean,
  includeTesting: boolean
): string {
  let desc = base;

  if (includeContext && context) {
    if (typeof context === 'string') {
      desc += `\n\n**Context:** ${context}`;
    } else {
      const contextParts = Object.entries(context)
        .filter(([_, v]) => v)
        .map(([k, v]) => `- **${k.replace(/_/g, ' ')}:** ${v}`);
      if (contextParts.length > 0) {
        desc += `\n\n**Context:**\n${contextParts.join('\n')}`;
      }
    }
  }

  if (includeTesting && testing) {
    const testParts = Object.entries(testing)
      .filter(([_, v]) => v)
      .map(([k, v]) => `- **${k.replace(/_/g, ' ')}:** ${v}`);
    if (testParts.length > 0) {
      desc += `\n\n**Testing Requirements:**\n${testParts.join('\n')}`;
    }
  }

  return desc;
}

export class BeadsAdapter implements OutputAdapter {
  readonly name = 'beads';
  private workingDir: string;
  private dryRun: boolean;
  private includeContext: boolean;
  private includeTesting: boolean;

  constructor(config: OutputConfig) {
    this.workingDir = config.workingDir || process.cwd();
    this.dryRun = config.dryRun || false;
    this.includeContext = config.includeContext ?? true;
    this.includeTesting = config.includeTesting ?? true;
  }

  async isAvailable(): Promise<boolean> {
    try {
      await execAsync('bd --version', { cwd: this.workingDir });
      return true;
    } catch {
      return false;
    }
  }

  async createItems(response: ParseResponse): Promise<CreateResult> {
    const created: CreatedItem[] = [];
    const failed: Array<{ item: WorkItem; error: string }> = [];
    const dependencies: Array<{ from: string; to: string; type: 'blocks' | 'parent-child' }> = [];
    const tempToExternal = new Map<string, string>();

    // Phase 1: Create all epics
    for (const epic of response.epics) {
      try {
        const id = await this.createEpic(epic);
        created.push({
          externalId: id,
          tempId: epic.temp_id,
          type: 'epic',
          title: epic.title,
          parentExternalId: null
        });
        tempToExternal.set(epic.temp_id, id);
      } catch (error) {
        failed.push({
          item: { type: 'epic', tempId: epic.temp_id, title: epic.title, parentTempId: null },
          error: error instanceof Error ? error.message : String(error)
        });
      }
    }

    // Phase 2: Create all tasks (as children of epics)
    for (const epic of response.epics) {
      const epicId = tempToExternal.get(epic.temp_id);
      if (!epicId) continue;

      for (const task of epic.tasks) {
        try {
          const id = await this.createTask(task, epicId);
          created.push({
            externalId: id,
            tempId: task.temp_id,
            type: 'task',
            title: task.title,
            parentExternalId: epicId
          });
          tempToExternal.set(task.temp_id, id);

          // Establish parent-child relationship
          await this.addDependency(epicId, id, 'parent-child');
          dependencies.push({ from: epicId, to: id, type: 'parent-child' });
        } catch (error) {
          failed.push({
            item: { type: 'task', tempId: task.temp_id, title: task.title, parentTempId: epic.temp_id },
            error: error instanceof Error ? error.message : String(error)
          });
        }
      }
    }

    // Phase 3: Create all subtasks (as children of tasks)
    for (const epic of response.epics) {
      for (const task of epic.tasks) {
        const taskId = tempToExternal.get(task.temp_id);
        if (!taskId) continue;

        for (const subtask of task.subtasks) {
          try {
            const id = await this.createSubtask(subtask, taskId);
            created.push({
              externalId: id,
              tempId: subtask.temp_id,
              type: 'subtask',
              title: subtask.title,
              parentExternalId: taskId
            });
            tempToExternal.set(subtask.temp_id, id);

            // Establish parent-child relationship
            await this.addDependency(taskId, id, 'parent-child');
            dependencies.push({ from: taskId, to: id, type: 'parent-child' });
          } catch (error) {
            failed.push({
              item: { type: 'subtask', tempId: subtask.temp_id, title: subtask.title, parentTempId: task.temp_id },
              error: error instanceof Error ? error.message : String(error)
            });
          }
        }
      }
    }

    // Phase 4: Establish explicit dependencies (blocks relationships)
    for (const epic of response.epics) {
      const epicId = tempToExternal.get(epic.temp_id);
      if (epicId) {
        for (const depTempId of epic.depends_on) {
          const blockerId = tempToExternal.get(depTempId);
          if (blockerId) {
            await this.addDependency(blockerId, epicId, 'blocks');
            dependencies.push({ from: blockerId, to: epicId, type: 'blocks' });
          }
        }
      }

      for (const task of epic.tasks) {
        const taskId = tempToExternal.get(task.temp_id);
        if (taskId) {
          for (const depTempId of task.depends_on) {
            const blockerId = tempToExternal.get(depTempId);
            if (blockerId) {
              await this.addDependency(blockerId, taskId, 'blocks');
              dependencies.push({ from: blockerId, to: taskId, type: 'blocks' });
            }
          }
        }

        for (const subtask of task.subtasks) {
          const subtaskId = tempToExternal.get(subtask.temp_id);
          if (subtaskId) {
            for (const depTempId of subtask.depends_on) {
              const blockerId = tempToExternal.get(depTempId);
              if (blockerId) {
                await this.addDependency(blockerId, subtaskId, 'blocks');
                dependencies.push({ from: blockerId, to: subtaskId, type: 'blocks' });
              }
            }
          }
        }
      }
    }

    return {
      created,
      failed,
      dependencies,
      stats: {
        epics: created.filter(c => c.type === 'epic').length,
        tasks: created.filter(c => c.type === 'task').length,
        subtasks: created.filter(c => c.type === 'subtask').length,
        dependencies: dependencies.length
      }
    };
  }

  private async createEpic(epic: Epic): Promise<string> {
    const desc = buildDescription(
      epic.description,
      epic.context,
      epic.testing,
      this.includeContext,
      this.includeTesting
    );

    const acceptance = epic.acceptance_criteria.join('\\n- ');

    return this.runBdCreate({
      title: epic.title,
      description: desc,
      type: 'epic',
      priority: 1, // Epics default to high priority
      design: null,
      acceptance: acceptance ? `- ${acceptance}` : null
    });
  }

  private async createTask(task: Task, _parentId: string): Promise<string> {
    const desc = buildDescription(
      task.description,
      task.context,
      task.testing,
      this.includeContext,
      this.includeTesting
    );

    return this.runBdCreate({
      title: task.title,
      description: desc,
      type: 'task',
      priority: mapPriority(task.priority),
      design: task.design_notes,
      acceptance: null
    });
  }

  private async createSubtask(subtask: Subtask, _parentId: string): Promise<string> {
    const desc = buildDescription(
      subtask.description,
      subtask.context,
      subtask.testing,
      this.includeContext,
      this.includeTesting
    );

    return this.runBdCreate({
      title: subtask.title,
      description: desc,
      type: 'task', // Beads uses 'task' for subtasks too
      priority: 2,
      design: null,
      acceptance: null
    });
  }

  private async runBdCreate(opts: {
    title: string;
    description: string;
    type: string;
    priority: number;
    design: string | null;
    acceptance: string | null;
  }): Promise<string> {
    const args = [
      'create',
      `"${this.escape(opts.title)}"`,
      `--description "${this.escape(opts.description)}"`,
      `--priority ${opts.priority}`,
      `--type ${opts.type}`
    ];

    if (opts.design) {
      args.push(`--design "${this.escape(opts.design)}"`);
    }
    if (opts.acceptance) {
      args.push(`--acceptance "${this.escape(opts.acceptance)}"`);
    }

    const command = `bd ${args.join(' ')}`;

    if (this.dryRun) {
      console.log(`[dry-run] ${command}`);
      return `dry-${Date.now()}-${Math.random().toString(36).slice(2, 6)}`;
    }

    const { stdout } = await execAsync(command, { cwd: this.workingDir });
    const match = stdout.match(/bd-[a-z0-9]+/i);
    if (!match) {
      throw new Error(`Could not extract issue ID from: ${stdout}`);
    }
    return match[0];
  }

  private async addDependency(
    fromId: string,
    toId: string,
    type: 'blocks' | 'parent-child'
  ): Promise<void> {
    const command = `bd dep add ${fromId} ${type} ${toId}`;

    if (this.dryRun) {
      console.log(`[dry-run] ${command}`);
      return;
    }

    await execAsync(command, { cwd: this.workingDir });
  }

  private escape(str: string): string {
    return str.replace(/"/g, '\\"').replace(/\n/g, '\\n');
  }
}

export function createBeadsAdapter(config: OutputConfig): OutputAdapter {
  return new BeadsAdapter(config);
}
```

**Step 2: Commit**

```bash
git add src/output/beads.ts
git commit -m "feat: implement hierarchical Beads output adapter

Creates epics, tasks, subtasks with parent-child relationships.
Propagates context and testing requirements into descriptions.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 9: Implement JSON Output Adapter

**Files:**
- Create: `src/output/json.ts`

**Step 1: Write the JSON adapter**

```typescript
import { writeFileSync } from 'fs';
import type { ParseResponse } from '../core/schemas.js';
import type { OutputAdapter, OutputConfig, CreateResult } from './types.js';

export class JsonAdapter implements OutputAdapter {
  readonly name = 'json';
  private outputPath: string | null;
  private dryRun: boolean;

  constructor(config: OutputConfig) {
    this.outputPath = (config.outputPath as string) || null;
    this.dryRun = config.dryRun || false;
  }

  async isAvailable(): Promise<boolean> {
    return true; // Always available
  }

  async createTasks(response: ParseResponse): Promise<CreateResult> {
    const output = JSON.stringify(response, null, 2);

    if (this.dryRun) {
      console.log('[dry-run] Would write:');
      console.log(output);
    } else if (this.outputPath) {
      writeFileSync(this.outputPath, output);
      console.log(`Tasks written to ${this.outputPath}`);
    } else {
      console.log(output);
    }

    return {
      created: response.tasks.map(task => ({
        externalId: `task-${task.temp_id}`,
        tempId: task.temp_id,
        title: task.title
      })),
      failed: [],
      dependencies: response.tasks.flatMap(task =>
        task.depends_on.map(dep => ({
          from: `task-${dep}`,
          to: `task-${task.temp_id}`
        }))
      )
    };
  }
}

export function createJsonAdapter(config: OutputConfig): OutputAdapter {
  return new JsonAdapter(config);
}
```

**Step 2: Commit**

```bash
git add src/output/json.ts
git commit -m "feat: implement JSON output adapter

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 10: Implement Core Parser

**Files:**
- Create: `src/core/parser.ts`

**Step 1: Write the core parser**

```typescript
import { readFileSync } from 'fs';
import { ParseConfigSchema, type ParseConfig, type ParseResponse } from './schemas.js';
import { buildParsePrdPrompts } from './prompts.js';
import type { LLMAdapter } from '../llm/types.js';
import type { OutputAdapter, CreateResult } from '../output/types.js';

export interface ParseOptions {
  /** Path to PRD file */
  prdPath: string;
  /** LLM adapter to use */
  llm: LLMAdapter;
  /** Output adapter to use */
  output: OutputAdapter;
  /** Parse configuration */
  config?: Partial<ParseConfig>;
}

export interface ParseResult {
  /** Parsed response from LLM */
  response: ParseResponse;
  /** Results from output adapter */
  createResult: CreateResult;
}

/**
 * Parse a PRD file and create tasks
 */
export async function parsePrd(options: ParseOptions): Promise<ParseResult> {
  // Validate and apply defaults to config
  const config = ParseConfigSchema.parse(options.config || {});

  // Read PRD content
  const prdContent = readFileSync(options.prdPath, 'utf-8');

  // Build prompts
  const { system, user } = buildParsePrdPrompts(prdContent, config);

  // Check output adapter availability
  const available = await options.output.isAvailable();
  if (!available) {
    throw new Error(`Output adapter '${options.output.name}' is not available`);
  }

  // Generate tasks via LLM
  console.log(`Generating tasks with ${options.llm.name}...`);
  const response = await options.llm.generateTasks(system, user);
  console.log(`Generated ${response.tasks.length} tasks`);

  // Create tasks in target system
  console.log(`Creating tasks in ${options.output.name}...`);
  const createResult = await options.output.createTasks(response);

  console.log(`Created ${createResult.created.length} tasks`);
  if (createResult.failed.length > 0) {
    console.warn(`Failed to create ${createResult.failed.length} tasks`);
  }
  console.log(`Established ${createResult.dependencies.length} dependencies`);

  return { response, createResult };
}
```

**Step 2: Commit**

```bash
git add src/core/parser.ts
git commit -m "feat: implement core PRD parser orchestration

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 11: Create Public API Exports

**Files:**
- Create: `src/index.ts`

**Step 1: Write the public API**

```typescript
// Core
export { parsePrd } from './core/parser.js';
export type { ParseOptions, ParseResult } from './core/parser.js';
export { ParseConfigSchema, TaskSchema, ParseResponseSchema } from './core/schemas.js';
export type { Task, ParseConfig, ParseResponse } from './core/schemas.js';

// LLM Adapters
export type { LLMAdapter, LLMConfig } from './llm/types.js';
export { createAnthropicAdapter, AnthropicAdapter } from './llm/anthropic.js';
export { createOpenAIAdapter, OpenAIAdapter } from './llm/openai.js';

// Output Adapters
export type { OutputAdapter, OutputConfig, CreateResult, CreatedTask } from './output/types.js';
export { createBeadsAdapter, BeadsAdapter } from './output/beads.js';
export { createJsonAdapter, JsonAdapter } from './output/json.js';
```

**Step 2: Commit**

```bash
git add src/index.ts
git commit -m "feat: create public API exports

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 12: Implement CLI

**Files:**
- Create: `src/cli/index.ts`

**Step 1: Write the CLI**

```typescript
#!/usr/bin/env node
import { Command } from 'commander';
import { parsePrd } from '../core/parser.js';
import { createAnthropicAdapter } from '../llm/anthropic.js';
import { createOpenAIAdapter } from '../llm/openai.js';
import { createBeadsAdapter } from '../output/beads.js';
import { createJsonAdapter } from '../output/json.js';

const program = new Command();

program
  .name('prd-parser')
  .description('Parse PRDs into structured tasks')
  .version('0.1.0');

program
  .command('parse')
  .description('Parse a PRD file and create tasks')
  .argument('<prd-file>', 'Path to PRD file')
  .option('-n, --num-tasks <number>', 'Number of tasks to generate', '10')
  .option('-p, --priority <level>', 'Default priority (critical/high/medium/low)', 'medium')
  .option('-l, --llm <provider>', 'LLM provider (anthropic/openai)', 'anthropic')
  .option('-m, --model <model>', 'Model to use')
  .option('-o, --output <adapter>', 'Output adapter (beads/json)', 'beads')
  .option('--output-path <path>', 'Output path for JSON adapter')
  .option('--dry-run', 'Preview without creating tasks')
  .action(async (prdFile, options) => {
    try {
      // Create LLM adapter
      const llmConfig = {
        model: options.model,
        apiKey: process.env[
          options.llm === 'openai' ? 'OPENAI_API_KEY' : 'ANTHROPIC_API_KEY'
        ]
      };
      const llm = options.llm === 'openai'
        ? createOpenAIAdapter(llmConfig)
        : createAnthropicAdapter(llmConfig);

      // Create output adapter
      const outputConfig = {
        dryRun: options.dryRun,
        outputPath: options.outputPath
      };
      const output = options.output === 'json'
        ? createJsonAdapter(outputConfig)
        : createBeadsAdapter(outputConfig);

      // Parse PRD
      const result = await parsePrd({
        prdPath: prdFile,
        llm,
        output,
        config: {
          num_tasks: parseInt(options.numTasks, 10),
          default_priority: options.priority
        }
      });

      // Print summary
      console.log('\n--- Summary ---');
      console.log(`Tasks created: ${result.createResult.created.length}`);
      for (const task of result.createResult.created) {
        console.log(`  ${task.externalId}: ${task.title}`);
      }

      if (result.createResult.failed.length > 0) {
        console.log(`\nFailed: ${result.createResult.failed.length}`);
        for (const { task, error } of result.createResult.failed) {
          console.log(`  ${task.title}: ${error}`);
        }
      }

    } catch (error) {
      console.error('Error:', error instanceof Error ? error.message : error);
      process.exit(1);
    }
  });

program.parse();
```

**Step 2: Make CLI executable**

Add shebang already included. Update package.json bin field (already done in Task 1).

**Step 3: Test CLI locally**

```bash
npm run cli -- parse --help
```

Expected: Help output showing all options

**Step 4: Commit**

```bash
git add src/cli/index.ts
git commit -m "feat: implement CLI with parse command

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 13: Write Tests

**Files:**
- Create: `tests/schemas.test.ts`
- Create: `tests/parser.test.ts`

**Step 1: Write schema tests**

```typescript
import { describe, it, expect } from 'vitest';
import { TaskSchema, ParseResponseSchema, ParseConfigSchema } from '../src/core/schemas.js';

describe('TaskSchema', () => {
  it('validates a valid task', () => {
    const task = {
      temp_id: 1,
      title: 'Set up project',
      description: 'Initialize the project structure',
      details: 'Create directories and config files',
      test_strategy: 'Verify directories exist',
      priority: 'high',
      task_type: 'task',
      depends_on: [],
      estimated_hours: 2
    };

    expect(() => TaskSchema.parse(task)).not.toThrow();
  });

  it('rejects task with invalid priority', () => {
    const task = {
      temp_id: 1,
      title: 'Test',
      description: 'Test',
      details: null,
      test_strategy: null,
      priority: 'invalid',
      task_type: 'task',
      depends_on: [],
      estimated_hours: null
    };

    expect(() => TaskSchema.parse(task)).toThrow();
  });

  it('rejects task with missing required fields', () => {
    const task = {
      temp_id: 1,
      title: 'Test'
    };

    expect(() => TaskSchema.parse(task)).toThrow();
  });
});

describe('ParseConfigSchema', () => {
  it('applies defaults', () => {
    const config = ParseConfigSchema.parse({});

    expect(config.num_tasks).toBe(10);
    expect(config.default_priority).toBe('medium');
    expect(config.include_test_tasks).toBe(true);
  });

  it('validates num_tasks range', () => {
    expect(() => ParseConfigSchema.parse({ num_tasks: 0 })).toThrow();
    expect(() => ParseConfigSchema.parse({ num_tasks: 101 })).toThrow();
    expect(() => ParseConfigSchema.parse({ num_tasks: 50 })).not.toThrow();
  });
});
```

**Step 2: Verify tests pass**

```bash
npm test
```

Expected: All tests pass

**Step 3: Commit**

```bash
git add tests/
git commit -m "test: add schema validation tests

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 14: Add README and Documentation

**Files:**
- Create: `README.md`

**Step 1: Write README**

```markdown
# prd-parser

Parse Product Requirements Documents into structured tasks with LLM guardrails.

## Features

- **Guardrails**: Zod schemas ensure LLMs output valid, structured tasks
- **Multi-LLM**: Works with Anthropic Claude, OpenAI, and more
- **Multi-Output**: Create tasks in Beads, export to JSON, or add your own adapter
- **Portable**: Use as CLI, library, or MCP server

## Installation

```bash
npm install prd-parser
```

## Quick Start

### CLI

```bash
# Parse PRD and create beads issues
prd-parser parse ./docs/prd.md --num-tasks 15

# Use OpenAI instead of Anthropic
prd-parser parse ./docs/prd.md --llm openai

# Output to JSON instead of beads
prd-parser parse ./docs/prd.md --output json --output-path tasks.json

# Dry run (preview without creating)
prd-parser parse ./docs/prd.md --dry-run
```

### Library

```typescript
import { parsePrd, createAnthropicAdapter, createBeadsAdapter } from 'prd-parser';

const result = await parsePrd({
  prdPath: './docs/prd.md',
  llm: createAnthropicAdapter({ model: 'claude-sonnet-4-20250514' }),
  output: createBeadsAdapter({ workingDir: process.cwd() }),
  config: {
    num_tasks: 20,
    default_priority: 'medium'
  }
});

console.log(`Created ${result.createResult.created.length} tasks`);
```

## Configuration

| Option | CLI Flag | Default | Description |
|--------|----------|---------|-------------|
| num_tasks | `-n, --num-tasks` | 10 | Target number of tasks |
| default_priority | `-p, --priority` | medium | Default task priority |
| include_test_tasks | - | true | Generate testing tasks |
| max_task_hours | - | 4 | Max hours per task |

## LLM Providers

### Anthropic (default)

```bash
export ANTHROPIC_API_KEY=your-key
prd-parser parse ./prd.md --llm anthropic --model claude-sonnet-4-20250514
```

### OpenAI

```bash
export OPENAI_API_KEY=your-key
prd-parser parse ./prd.md --llm openai --model gpt-4o
```

## Output Adapters

### Beads (default)

Creates issues in the current beads-initialized project:

```bash
bd init  # Initialize beads first
prd-parser parse ./prd.md --output beads
```

### JSON

Outputs tasks to JSON file or stdout:

```bash
prd-parser parse ./prd.md --output json --output-path tasks.json
```

## Adding Custom Adapters

### Custom LLM Adapter

```typescript
import type { LLMAdapter, ParseResponse } from 'prd-parser';

class MyLLMAdapter implements LLMAdapter {
  readonly name = 'my-llm';

  async generateTasks(system: string, user: string): Promise<ParseResponse> {
    // Call your LLM and parse response
    // Must return data matching ParseResponseSchema
  }
}
```

### Custom Output Adapter

```typescript
import type { OutputAdapter, CreateResult, ParseResponse } from 'prd-parser';

class MyOutputAdapter implements OutputAdapter {
  readonly name = 'my-output';

  async isAvailable(): Promise<boolean> {
    return true;
  }

  async createTasks(response: ParseResponse): Promise<CreateResult> {
    // Create tasks in your system
    // Return created task IDs for dependency mapping
  }
}
```

## License

MIT
```

**Step 2: Commit**

```bash
git add README.md
git commit -m "docs: add README with usage examples

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 15: Integration Test with Beads

**Files:**
- Create: `tests/integration/beads.test.ts`

**Step 1: Create integration test**

```typescript
import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { exec } from 'child_process';
import { promisify } from 'util';
import { mkdtempSync, writeFileSync, rmSync } from 'fs';
import { join } from 'path';
import { tmpdir } from 'os';

const execAsync = promisify(exec);

describe('Beads Integration', () => {
  let testDir: string;

  beforeAll(async () => {
    // Create temp directory
    testDir = mkdtempSync(join(tmpdir(), 'prd-parser-test-'));

    // Initialize beads
    await execAsync('bd init', { cwd: testDir });

    // Create test PRD
    writeFileSync(join(testDir, 'prd.md'), `
# Test Project

## Overview
Build a simple todo app.

## Requirements
1. Create todos
2. Mark todos complete
3. Delete todos
    `);
  });

  afterAll(() => {
    rmSync(testDir, { recursive: true, force: true });
  });

  it('creates beads issues from PRD', async () => {
    // This test requires ANTHROPIC_API_KEY to be set
    if (!process.env.ANTHROPIC_API_KEY) {
      console.log('Skipping: ANTHROPIC_API_KEY not set');
      return;
    }

    const { stdout } = await execAsync(
      `npx tsx src/cli/index.ts parse ${join(testDir, 'prd.md')} --num-tasks 5`,
      { cwd: process.cwd() }
    );

    expect(stdout).toContain('Tasks created:');

    // Verify issues exist in beads
    const { stdout: listOutput } = await execAsync('bd list', { cwd: testDir });
    expect(listOutput).toContain('bd-');
  });
});
```

**Step 2: Run integration test (manual)**

```bash
ANTHROPIC_API_KEY=your-key npm test -- tests/integration/
```

**Step 3: Commit**

```bash
git add tests/integration/
git commit -m "test: add beads integration test

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Summary

After completing all tasks, you will have:

### Core Features

1. **Hierarchical Task Generation**: Epics → Tasks → Subtasks
   - Smart grouping based on PRD structure
   - Proper parent-child relationships in beads

2. **Context Propagation**: Business purpose, target users, brand guidelines flow to all levels
   - Every subtask knows WHY it matters
   - LLMs stay grounded in user value

3. **Testing at Every Level**: Unit, integration, type, e2e requirements at all levels
   - Not optional - guardrails enforce it
   - Testing distribution based on work type

### Architecture

1. **Core library** (`internal/core/`) with:
   - Go structs enforcing structured output (guardrails)
   - Embedded prompt templates (no external files)
   - Validation functions

2. **LLM adapters** (`internal/llm/`) with CLI-first approach:
   - **Claude Code CLI** (primary) - already authenticated
   - **Codex CLI** (secondary) - already authenticated
   - **Anthropic API** (fallback) - requires API key
   - Auto-detection picks best available

3. **Output adapters** (`internal/output/`) for:
   - **Beads Go API** - direct integration
   - **Beads CLI** - fallback
   - **JSON** - for other integrations
   - Extensible interface for Linear, GitHub, etc.

4. **Distribution**:
   - Single binary CLI (`prd-parser parse`)
   - Go library import
   - Future: MCP server

### Usage Examples

```bash
# Zero-config if you have Claude Code installed
prd-parser parse ./docs/prd.md

# Specify targets
prd-parser parse ./docs/prd.md --epics 5 --tasks-per-epic 8

# Dry run to preview
prd-parser parse ./docs/prd.md --dry-run

# Output to JSON instead of beads
prd-parser parse ./docs/prd.md --output json > tasks.json

# Use specific LLM
prd-parser parse ./docs/prd.md --llm codex
prd-parser parse ./docs/prd.md --llm anthropic-api
```

## Future Tasks (Not in This Plan)

- **Task 16**: Implement MCP server for Claude Desktop/Cursor
- **Task 17**: Add Ollama adapter for local LLMs
- **Task 18**: Add GitHub Issues output adapter
- **Task 19**: Add Linear output adapter
- **Task 20**: Publish to Homebrew and as Go module

## Note on Remaining Tasks

Tasks 7-15 in this plan still show TypeScript code but should be converted to Go following the same patterns established in Tasks 1-6. The key conversions:

- Output adapter interface → Go interface
- Beads adapter → Use beads Go API directly + CLI fallback
- CLI → Cobra commands
- Tests → Go testing package

---

**Plan complete and saved to `docs/plans/2026-01-29-parse-prd-skill.md`.**

**Two execution options:**

**1. Subagent-Driven (this session)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

Which approach?
