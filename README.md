# prd-parser

> **🚧 Active Development** - This project is new and actively evolving. Expect breaking changes. Contributions and feedback welcome!

**Turn your PRD into a ready-to-work beads project in one command.**

prd-parser uses LLM guardrails to transform Product Requirements Documents into a hierarchical issue structure (Epics → Tasks → Subtasks) and creates them directly in [beads](https://github.com/beads-project/beads) - the git-backed issue tracker for AI-driven development.

```bash
# One command: PRD → structured beads issues
prd-parser parse ./docs/prd.md
```

## The 0→1 Problem

Starting a new project is exciting. You have a vision, maybe a PRD, and you're ready to build. But then:

1. **The breakdown problem** - You need to turn that PRD into actionable tasks. This is tedious and error-prone. You lose context as you go.

2. **The context problem** - By the time you're implementing subtask #47, you've forgotten why it matters. What was the business goal? Who are the users? What constraints apply?

3. **The handoff problem** - If you're using AI to help implement, it needs that context too. Copy-pasting from your PRD for every task doesn't scale.

**prd-parser + beads solves all three.** Write your PRD once, run one command, and get a complete project structure with context propagated to every level - ready for you or Claude to start implementing.

## Why prd-parser + beads?

**For greenfield projects**, this is the fastest path from idea to structured, trackable work:

| Without prd-parser | With prd-parser |
|-------------------|-----------------|
| Read PRD, manually create issues | One command |
| Forget context by subtask #10 | Context propagated everywhere |
| Testing requirements? Maybe later | Testing enforced at every level |
| Dependencies tracked in your head | Dependencies explicit and tracked |
| Copy-paste context for AI helpers | AI has full context in every issue |

**How it works**: prd-parser uses Go struct guardrails to force the LLM to output valid, hierarchical JSON with:
- **Context propagation** - Business purpose flows from PRD → Epic → Task → Subtask
- **Testing at every level** - Unit, integration, type, and E2E requirements enforced
- **Dependencies tracked** - Issues know what blocks them
- **Direct beads integration** - Issues created with one command, ready to work

## Getting Started: Your First PRD → beads Project

### 1. Install prd-parser

**Via npm/bun (easiest):**
```bash
npm install -g prd-parser
# or
bun install -g prd-parser
# or
npx prd-parser parse ./docs/prd.md  # run without installing
```

**Via Go:**
```bash
go install github.com/dhabedank/prd-parser@latest
```

**From source:**
```bash
cd /tmp && git clone https://github.com/dhabedank/prd-parser.git && cd prd-parser && make install
```

If you see "Make sure ~/go/bin is in your PATH", run:
```bash
echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.zshrc && source ~/.zshrc
```

Now go back to your project.

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

That's it. Your PRD is now a structured beads project with **readable hierarchical IDs**:

```bash
$ bd list
○ my-project-e1 [P1] [epic] - Core Task Management System
○ my-project-e1t1 [P0] [task] - Implement Task Data Model
○ my-project-e1t1s1 [P2] [task] - Define Task struct with JSON tags
○ my-project-e1t1s2 [P2] [task] - Implement JSON file storage
○ my-project-e1t2 [P1] [task] - Build CLI Interface
○ my-project-e2 [P1] [epic] - User Authentication
...
```

IDs follow a logical hierarchy: `e1` (epic 1) → `e1t1` (task 1) → `e1t1s1` (subtask 1). Use `bd show <id>` to see parent/children relationships.

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

### Priority Evaluation

The LLM evaluates each task and assigns appropriate priority (not just a default):

| Priority | When to Use |
|----------|-------------|
| **P0 (critical)** | Blocks all work, security issues, launch blockers |
| **P1 (high)** | Core functionality, enables other tasks |
| **P2 (medium)** | Important features, standard work |
| **P3 (low)** | Nice-to-haves, polish |
| **P4 (very-low)** | Future considerations, can defer indefinitely |

Foundation/setup work gets higher priority. Polish/UI tweaks get lower priority.

### Labels

Issues are automatically labeled based on:
- **Layer**: frontend, backend, api, database, infra
- **Domain**: auth, payments, search, notifications
- **Skill**: react, go, sql, typescript
- **Type**: setup, feature, refactor, testing

Labels are extracted from the PRD's tech stack and feature descriptions.

### Design Notes & Acceptance Criteria

- **Epics** include acceptance criteria for when the epic is complete
- **Tasks** include design notes for technical approach

### Time Estimates

All items include time estimates that flow to beads:
- Epics: estimated days
- Tasks: estimated hours
- Subtasks: estimated minutes

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

# Save/resume from checkpoint (useful for large PRDs)
prd-parser parse ./prd.md --save-json checkpoint.json
prd-parser parse ./prd.md --from-json checkpoint.json
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
| `--multi-stage` | | false | Force multi-stage parsing |
| `--single-shot` | | false | Force single-shot parsing |
| `--smart-threshold` | | 300 | Line count for auto multi-stage (0 to disable) |
| `--validate` | | false | Run validation pass to check for gaps |
| `--no-review` | | false | Disable automatic LLM review pass (review ON by default) |
| `--interactive` | | false | Human-in-the-loop mode (review epics before task generation) |
| `--subtask-model` | | | Model for subtasks in multi-stage (can be faster/cheaper) |
| `--output` | `-o` | beads | Output adapter (beads/json) |
| `--output-path` | | | Output path for JSON adapter |
| `--dry-run` | | false | Preview without creating items |
| `--from-json` | | | Resume from saved JSON checkpoint (skip LLM) |
| `--save-json` | | | Save generated JSON to file (for resume) |
| `--config` | | | Config file path (default: .prd-parser.yaml) |

### Smart Parsing (Default Behavior)

prd-parser automatically chooses the best parsing strategy based on PRD size:

- **Small PRDs** (< 300 lines): Single-shot parsing (faster)
- **Large PRDs** (≥ 300 lines): Multi-stage parallel parsing (more reliable)

Override with `--single-shot` or `--multi-stage` flags, or adjust threshold with `--smart-threshold`.

### Validation Pass

Use `--validate` to run a final review that checks for gaps in the generated plan:

```bash
prd-parser parse ./prd.md --validate
```

This asks the LLM to review the complete plan and identify:
- Missing setup/initialization tasks
- Backend without UI to test it
- Dependencies not installed
- Acceptance criteria that can't be verified
- Tasks in wrong order

Example output:
```
✓ Plan validation passed - no gaps found
```
or
```
⚠ Plan validation found gaps:
  • No task to install dependencies after adding @clerk/nextjs
  • Auth API built but no login page to test it
```

### Review Pass (Default)

By default, prd-parser runs an automatic review pass after generation that checks for and fixes structural issues:

- **Missing "Project Foundation" epic** as Epic 1 (setup should come first)
- **Feature epics not depending on Epic 1** (all work depends on setup)
- **Missing setup tasks** in foundation epic
- **Incorrect dependency chains** (setup → backend → frontend)

```bash
# Review is on by default
prd-parser parse ./prd.md

# See: "Reviewing structure..."
# See: "✓ Review fixed issues: Added Project Foundation epic..."
# Or:  "✓ Review passed - no changes needed"

# Disable if you want raw output
prd-parser parse ./prd.md --no-review
```

### Interactive Mode

For human-in-the-loop review during generation:

```bash
prd-parser parse docs/prd.md --interactive
```

In interactive mode, you'll review epics after Stage 1 before task generation continues:

```
=== Stage 1 Complete: 4 Epics Generated ===

Proposed Epics:
  1. Project Foundation (depends on: none)
      Initialize Next.js, Convex, Clerk setup
  2. Voice Infrastructure (depends on: 1)
      Telnyx phone system integration
  3. AI Conversations (depends on: 1)
      LFM 2.5 integration for call handling
  4. CRM Integration (depends on: 1)
      Follow Up Boss sync

[Enter] continue, [e] edit in $EDITOR, [r] regenerate, [a] add epic:
```

**Options:**
- **Enter** - Accept epics and continue to task generation
- **e** - Open epics in your `$EDITOR` for manual editing
- **r** - Regenerate epics from scratch
- **a** - Add a new epic

Interactive mode skips the automatic review pass since you are the reviewer.

### Checkpoint Workflow (Manual Review)

For full manual control over the generated structure:

**Step 1: Generate Draft**
```bash
prd-parser parse docs/prd.md --save-json draft.json --dry-run
```

**Step 2: Review and Edit**

Open `draft.json` in your editor. You can:
- Reorder epics (change array order)
- Add/remove epics, tasks, or subtasks
- Fix dependencies
- Adjust priorities and estimates

**Step 3: Create from Edited Draft**
```bash
prd-parser parse --from-json draft.json
```

The PRD file argument is optional when using `--from-json`.

**Auto-Recovery**: If creation fails mid-way, prd-parser saves a checkpoint to `/tmp/prd-parser-checkpoint.json`. Retry with:
```bash
prd-parser parse --from-json /tmp/prd-parser-checkpoint.json
```

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
│   │   ├── prompts.go     # Single-shot system/user prompts
│   │   ├── stage_prompts.go # Multi-stage prompts (Stages 1-3)
│   │   ├── parser.go      # Single-shot LLM → Output orchestration
│   │   ├── multistage.go  # Multi-stage parallel parser
│   │   └── validate.go    # Validation pass logic
│   ├── llm/               # LLM adapters
│   │   ├── adapter.go     # Interface definition
│   │   ├── claude_cli.go  # Claude Code CLI adapter
│   │   ├── codex_cli.go   # Codex CLI adapter
│   │   ├── anthropic_api.go # API fallback
│   │   ├── detector.go    # Auto-detection logic
│   │   └── multistage_generator.go # Multi-stage LLM calls
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
