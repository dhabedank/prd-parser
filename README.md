# prd-parser

Parse Product Requirements Documents into structured tasks with LLM guardrails.

## Features

- **Guardrails**: Go structs ensure LLMs output valid, structured tasks
- **Hierarchical**: Epics → Tasks → Subtasks with context propagation
- **CLI-First LLM Access**: Uses `claude` and `codex` CLIs when available (already authenticated)
- **Multi-LLM**: Works with Claude Code, Codex, Anthropic API
- **Multi-Output**: Create tasks in Beads, export to JSON, or add your own adapter
- **Testing at Every Level**: Unit, integration, type, and E2E testing requirements enforced

## Installation

### From Source

```bash
git clone https://github.com/yourusername/prd-parser.git
cd prd-parser
make build
make install  # Copies to $GOPATH/bin
```

### Binary Download

Coming soon to Homebrew and releases.

## Quick Start

### Basic Usage

```bash
# Parse PRD and create beads issues (auto-detects LLM)
prd-parser parse ./docs/prd.md

# Specify target structure
prd-parser parse ./docs/prd.md --epics 5 --tasks 8 --subtasks 4

# Use specific LLM
prd-parser parse ./docs/prd.md --llm claude-cli
prd-parser parse ./docs/prd.md --llm codex-cli
prd-parser parse ./docs/prd.md --llm anthropic-api

# Output to JSON instead of beads
prd-parser parse ./docs/prd.md --output json --output-path tasks.json

# Dry run (preview without creating)
prd-parser parse ./docs/prd.md --dry-run
```

### Zero-Config Experience

If you have Claude Code or Codex installed, prd-parser will use it automatically - no API key needed.

```bash
# Just works if you have Claude Code
prd-parser parse ./prd.md

# Just works if you have Codex
prd-parser parse ./prd.md
```

## Configuration

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
| `--dry-run` | | false | Preview without creating items |

## LLM Providers

### Auto-Detection (default)

prd-parser automatically finds the best available LLM:

1. **Claude Code CLI** (preferred) - If `claude` is installed
2. **Codex CLI** - If `codex` is installed
3. **Anthropic API** - If `ANTHROPIC_API_KEY` is set

### Claude Code CLI

```bash
# Requires Claude Code to be installed and authenticated
prd-parser parse ./prd.md --llm claude-cli --model claude-sonnet-4-20250514
```

### Codex CLI

```bash
# Requires Codex to be installed and authenticated
prd-parser parse ./prd.md --llm codex-cli --model o3
```

### Anthropic API

```bash
export ANTHROPIC_API_KEY=your-key
prd-parser parse ./prd.md --llm anthropic-api --model claude-sonnet-4-20250514
```

## Output Adapters

### Beads (default)

Creates issues in the current beads-initialized project:

```bash
bd init  # Initialize beads first
prd-parser parse ./prd.md --output beads
```

The adapter creates:
- Epics as high-priority issues
- Tasks as children of epics
- Subtasks as children of tasks
- Dependencies between related items

### JSON

Outputs tasks to JSON file or stdout:

```bash
# Write to file
prd-parser parse ./prd.md --output json --output-path tasks.json

# Write to stdout
prd-parser parse ./prd.md --output json
```

## Architecture

```
prd-parser/
├── cmd/                    # CLI commands
├── internal/
│   ├── core/              # Types, prompts, parser orchestration
│   │   ├── types.go       # Hierarchical structs (guardrails)
│   │   ├── prompts.go     # Embedded prompt templates
│   │   └── parser.go      # Main orchestration
│   ├── llm/               # LLM adapters
│   │   ├── adapter.go     # Interface definition
│   │   ├── claude_cli.go  # Claude Code CLI
│   │   ├── codex_cli.go   # Codex CLI
│   │   ├── anthropic_api.go # API fallback
│   │   └── detector.go    # Auto-detection
│   └── output/            # Output adapters
│       ├── adapter.go     # Interface definition
│       ├── beads.go       # Beads issue tracker
│       └── json.go        # JSON output
└── tests/                 # Unit tests
```

## Adding Custom Adapters

### Custom LLM Adapter

Implement the `llm.Adapter` interface:

```go
type Adapter interface {
    Name() string
    IsAvailable() bool
    Generate(ctx context.Context, systemPrompt, userPrompt string) (*core.ParseResponse, error)
}
```

### Custom Output Adapter

Implement the `output.Adapter` interface:

```go
type Adapter interface {
    Name() string
    IsAvailable() (bool, error)
    CreateItems(response *core.ParseResponse, config Config) (*CreateResult, error)
}
```

## License

MIT
