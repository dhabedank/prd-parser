package output

import (
	"github.com/dhabedank/prd-parser/internal/core"
)

// WorkItem represents any work item in the hierarchy.
type WorkItem struct {
	Type         string // "epic", "task", or "subtask"
	TempID       string
	Title        string
	ParentTempID string // empty if no parent
}

// CreatedItem represents a successfully created item in the target system.
type CreatedItem struct {
	ExternalID       string // ID assigned by target system (e.g., bd-a3f8)
	TempID           string // Original temp_id for dependency mapping
	Type             string // "epic", "task", or "subtask"
	Title            string
	ParentExternalID string // empty if no parent
}

// CreateResult is the result of creating all items.
type CreateResult struct {
	Created      []CreatedItem
	Failed       []FailedItem
	Dependencies []Dependency
	Stats        Stats
}

// FailedItem represents an item that failed to create.
type FailedItem struct {
	Item  WorkItem
	Error string
}

// Dependency represents a relationship between items.
type Dependency struct {
	From string // external ID
	To   string // external ID
	Type string // "blocks" or "parent-child"
}

// Stats provides summary statistics.
type Stats struct {
	Epics        int
	Tasks        int
	Subtasks     int
	Dependencies int
}

// Adapter is the interface all output adapters must implement.
type Adapter interface {
	// Name returns the adapter identifier for logging.
	Name() string

	// IsAvailable checks if the adapter can be used (e.g., CLI installed).
	IsAvailable() (bool, error)

	// CreateItems creates hierarchical items in the target system.
	CreateItems(response *core.ParseResponse, config Config) (*CreateResult, error)
}

// Config configures output adapter behavior.
type Config struct {
	// WorkingDir for CLI-based adapters.
	WorkingDir string

	// DryRun previews without creating items.
	DryRun bool

	// IncludeContext adds context blocks to descriptions.
	IncludeContext bool

	// IncludeTesting adds testing requirements to descriptions.
	IncludeTesting bool
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		WorkingDir:     ".",
		DryRun:         false,
		IncludeContext: true,
		IncludeTesting: true,
	}
}
