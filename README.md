# prd-parser

**Turn your PRD into a ready-to-work beads project in one command.**

prd-parser uses LLM guardrails to transform Product Requirements Documents into a hierarchical issue structure (Epics → Tasks → Subtasks) and creates them directly in [beads](https://github.com/beads-project/beads) - the git-backed issue tracker for AI-driven development.

```bash
# One command: PRD → structured beads issues
prd-parser parse ./docs/prd.md
```

## Why prd-parser + beads?

**The Problem**: You have a PRD. You need to break it into trackable tasks. Doing this manually loses context - by the time you're implementing subtask 47, you've forgotten the business purpose. LLMs can help, but they output unstructured text that still needs manual entry.

**The Solution**: prd-parser uses Go struct guardrails to force the LLM to output valid, hierarchical JSON with:
- **Context propagation** - Business purpose flows from PRD → Epic → Task → Subtask
- **Testing at every level** - Unit, integration, type, and E2E requirements enforced
- **Dependencies tracked** - Issues know what blocks them
- **Direct beads integration** - Issues created with one command, ready to work

## Getting Started: Your First PRD → beads Project

### 1. Install prd-parser

```bash
git clone https://github.com/dhabedank/prd-parser.git
cd prd-parser
make build
make install  # Copies to $GOPATH/bin
```

### 2. Create a new project with beads

```bash
mkdir my-project && cd my-project
git init
bd init --prefix my-project
```

### 3. Write your PRD

Create `docs/prd.md` with your product requirements. Include:
- What you're building and why
- Who the target users are
- Technical constraints
- Key features

Example:
```markdown
# Task Management CLI

## Overview
A fast, developer-friendly command-line task manager for teams
who prefer terminal workflows.

## Target Users
Software developers who live in the terminal and want sub-100ms
task operations without context-switching to a GUI.

## Core Features
1. Create tasks with title, description, priority
2. List and filter tasks by status/priority
3. Update task status (todo → in-progress → done)
4. Local JSON storage for offline-first operation

## Technical Constraints
- Sub-100ms response for all operations
- Single binary, no runtime dependencies
- Config stored in ~/.taskman/
```

### 4. Parse your PRD into beads issues

```bash
prd-parser parse docs/prd.md
```

That's it. Your PRD is now a structured beads project:

```bash
$ bd list
○ my-project-9x3 [P1] [epic] - Core Task Management System
○ my-project-4og [P1] [task] - Implement Task Data Model
○ my-project-67g [P1] [task] - Build CLI Interface
○ my-project-3m7 [P2] [task] - Add JSON Storage Layer
○ my-project-bzy [P2] [task] - Implement List Filtering
...
```

### 5. Start working with beads + Claude

```bash
# See what's ready to work on
bd ready

# Pick an issue and let Claude implement it
bd show my-project-4og  # Shows full context, testing requirements

# Or let Claude pick and work autonomously
# (beads integrates with Claude Code via the beads skill)
```

## What prd-parser Creates

### Hierarchical Structure

```
Epic: Core Task Management System
├── Task: Implement Task Data Model
│   ├── Subtask: Define Task struct with JSON tags
│   └── Subtask: Implement JSON file storage
├── Task: Build CLI Interface
│   ├── Subtask: Implement create command
│   └── Subtask: Implement list command
└── ...
```

### Context Propagation

Every issue includes propagated context so implementers understand WHY:

```markdown
**Context:**
- **Business Context:** Developers need fast, frictionless task management
- **Target Users:** Terminal-first developers who want <100ms operations
- **Success Metrics:** All CRUD operations complete in under 100ms
```

### Testing Requirements

Every issue specifies what testing is needed:

```markdown
**Testing Requirements:**
- **Unit Tests:** Task struct validation, JSON marshaling/unmarshaling
- **Integration Tests:** Full storage layer integration, concurrent access
- **Type Tests:** Go struct tags validation, JSON schema compliance
```

### Dependencies

Issues are linked with proper blocking relationships:
- Tasks depend on setup tasks
- Subtasks depend on parent task completion
- Cross-epic dependencies are tracked

## Configuration

### Parse Options

```bash
# Control structure size
prd-parser parse ./prd.md --epics 5 --tasks 8 --subtasks 4

# Set default priority
prd-parser parse ./prd.md --priority high

# Choose testing level
prd-parser parse ./prd.md --testing comprehensive  # or minimal, standard

# Preview without creating (dry run)
prd-parser parse ./prd.md --dry-run

# Include/exclude sections in issue descriptions
prd-parser parse ./prd.md --include-context --include-testing
```

### Full Options

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--epics` | `-e` | 3 | Target number of epics |
| `--tasks` | `-t` | 5 | Target tasks per epic |
| `--subtasks` | `-s` | 4 | Target subtasks per task |
| `--priority` | `-p` | medium | Default priority (critical/high/medium/low) |
| `--testing` | | comprehensive | Testing level (minimal/standard/comprehensive) |
| `--llm` | `-l` | auto | LLM provider (auto/claude-cli/codex-cli/anthropic-api) |
| `--model` | `-m` | | Model to use (provider-specific) |
| `--output` | `-o` | beads | Output adapter (beads/json) |
| `--output-path` | | | Output path for JSON adapter |
| `--working-dir` | `-w` | . | Working directory for beads |
| `--dry-run` | | false | Preview without creating items |
| `--include-context` | | true | Include context in descriptions |
| `--include-testing` | | true | Include testing requirements |

## LLM Providers

### Zero-Config (Recommended)

prd-parser auto-detects installed LLM CLIs - no API keys needed:

```bash
# If you have Claude Code installed, it just works
prd-parser parse ./prd.md

# If you have Codex installed, it just works
prd-parser parse ./prd.md
```

### Detection Priority

1. **Claude Code CLI** (`claude`) - Preferred, already authenticated
2. **Codex CLI** (`codex`) - Already authenticated
3. **Anthropic API** - Fallback if `ANTHROPIC_API_KEY` is set

### Explicit Selection

```bash
# Force specific provider
prd-parser parse ./prd.md --llm claude-cli
prd-parser parse ./prd.md --llm codex-cli
prd-parser parse ./prd.md --llm anthropic-api

# Specify model
prd-parser parse ./prd.md --llm claude-cli --model claude-sonnet-4-20250514
prd-parser parse ./prd.md --llm codex-cli --model o3
```

## Output Options

### beads (Default)

Creates issues directly in the current beads-initialized project:

```bash
bd init --prefix myproject
prd-parser parse ./prd.md --output beads
bd list  # See created issues
```

### JSON

Export to JSON for inspection or custom processing:

```bash
# Write to file
prd-parser parse ./prd.md --output json --output-path tasks.json

# Write to stdout (pipe to other tools)
prd-parser parse ./prd.md --output json | jq '.epics[0].tasks'
```

## The Guardrails System

prd-parser isn't just a prompt wrapper. It uses Go structs as **guardrails** to enforce valid output:

```go
type Epic struct {
    TempID             string              `json:"temp_id"`
    Title              string              `json:"title"`
    Description        string              `json:"description"`
    Context            interface{}         `json:"context"`
    AcceptanceCriteria []string            `json:"acceptance_criteria"`
    Testing            TestingRequirements `json:"testing"`
    Tasks              []Task              `json:"tasks"`
    DependsOn          []string            `json:"depends_on"`
}

type TestingRequirements struct {
    UnitTests        *string `json:"unit_tests,omitempty"`
    IntegrationTests *string `json:"integration_tests,omitempty"`
    TypeTests        *string `json:"type_tests,omitempty"`
    E2ETests         *string `json:"e2e_tests,omitempty"`
}
```

The LLM MUST produce output that matches these structs. Missing required fields? Validation fails. Wrong types? Parse fails. This ensures every PRD produces consistent, complete issue structures.

## Architecture

```
prd-parser/
├── cmd/                    # CLI commands (Cobra)
│   └── parse.go           # Main parse command
├── internal/
│   ├── core/              # Core types and orchestration
│   │   ├── types.go       # Hierarchical structs (guardrails)
│   │   ├── prompts.go     # Embedded system/user prompts
│   │   └── parser.go      # LLM → Output orchestration
│   ├── llm/               # LLM adapters
│   │   ├── adapter.go     # Interface definition
│   │   ├── claude_cli.go  # Claude Code CLI adapter
│   │   ├── codex_cli.go   # Codex CLI adapter
│   │   ├── anthropic_api.go # API fallback
│   │   └── detector.go    # Auto-detection logic
│   └── output/            # Output adapters
│       ├── adapter.go     # Interface definition
│       ├── beads.go       # beads issue tracker
│       └── json.go        # JSON file output
└── tests/                 # Unit tests
```

## Adding Custom Adapters

### Custom LLM Adapter

```go
type Adapter interface {
    Name() string
    IsAvailable() bool
    Generate(ctx context.Context, systemPrompt, userPrompt string) (*core.ParseResponse, error)
}
```

### Custom Output Adapter

```go
type Adapter interface {
    Name() string
    IsAvailable() (bool, error)
    CreateItems(response *core.ParseResponse, config Config) (*CreateResult, error)
}
```

## Related Projects

- **[beads](https://github.com/beads-project/beads)** - Git-backed issue tracker for AI-driven development
- **[Claude Code](https://claude.ai/claude-code)** - Claude's official CLI with beads integration

## License

MIT
