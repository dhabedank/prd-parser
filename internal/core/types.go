package core

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FlexibleString is a type that can unmarshal from either a JSON string or an array of strings.
// Arrays are joined with "; " to form a single string.
type FlexibleString string

// UnmarshalJSON implements custom JSON unmarshaling for FlexibleString.
func (f *FlexibleString) UnmarshalJSON(data []byte) error {
	// Handle null
	if string(data) == "null" {
		*f = FlexibleString("")
		return nil
	}

	// Handle boolean (LLMs sometimes return false/true instead of strings)
	if string(data) == "false" || string(data) == "true" {
		*f = FlexibleString("") // Treat as empty/not applicable
		return nil
	}

	// Try unmarshaling as a string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*f = FlexibleString(s)
		return nil
	}

	// Try unmarshaling as an array of strings
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*f = FlexibleString(strings.Join(arr, "; "))
		return nil
	}

	// If neither works, return an error
	return fmt.Errorf("FlexibleString: cannot unmarshal %s", string(data))
}

// String returns the string value.
func (f FlexibleString) String() string {
	return string(f)
}

// StringPtr returns a pointer to the string value, or nil if empty.
func (f FlexibleString) StringPtr() *string {
	if f == "" {
		return nil
	}
	s := string(f)
	return &s
}

// FlexibleStringSlice is a type that can unmarshal from either a JSON array
// of strings or an object (extracting values).
type FlexibleStringSlice []string

// UnmarshalJSON implements custom JSON unmarshaling for FlexibleStringSlice.
func (f *FlexibleStringSlice) UnmarshalJSON(data []byte) error {
	// Try unmarshaling as an array of strings first
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*f = arr
		return nil
	}

	// Try unmarshaling as an object and extract values
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err == nil {
		var values []string
		for _, v := range obj {
			if s, ok := v.(string); ok {
				values = append(values, s)
			}
		}
		*f = values
		return nil
	}

	// Try unmarshaling as a single string
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*f = []string{s}
		return nil
	}

	return fmt.Errorf("FlexibleStringSlice: cannot unmarshal %s", string(data))
}

// ToSlice returns the underlying slice.
func (f FlexibleStringSlice) ToSlice() []string {
	return []string(f)
}

// TestingRequirements captures testing needs at any level.
// Forces consideration of testing at epic, task, and subtask levels.
type TestingRequirements struct {
	UnitTests        *FlexibleString `json:"unit_tests,omitempty"`        // Functions/methods to test in isolation
	IntegrationTests *FlexibleString `json:"integration_tests,omitempty"` // How components interact
	TypeTests        *FlexibleString `json:"type_tests,omitempty"`        // Type safety, runtime validation
	E2ETests         *FlexibleString `json:"e2e_tests,omitempty"`         // User flows to verify
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
	Labels           []string            `json:"labels,omitempty"`            // Tags for categorization
}

// Task is a logical unit of work containing subtasks (2-8hrs total)
type Task struct {
	TempID         string              `json:"temp_id"`                   // Hierarchical ID like "1.1"
	Title          string              `json:"title"`                     // Clear, actionable title
	Description    string              `json:"description"`               // What needs to be accomplished
	Context        interface{}         `json:"context"`                   // Propagated + task-specific context (object or string)
	DesignNotes    *string             `json:"design_notes,omitempty"`    // Technical approach
	Testing        TestingRequirements `json:"testing"`                   // Testing strategy
	Priority       Priority            `json:"priority"`                  // critical/high/medium/low/very-low
	Subtasks       []Subtask           `json:"subtasks"`                  // Atomic subtasks
	DependsOn      []string            `json:"depends_on"`                // Temp IDs this depends on
	EstimatedHours *float64            `json:"estimated_hours,omitempty"` // Total including subtasks
	Labels         []string            `json:"labels,omitempty"`          // Tags for categorization
}

// Epic is a major feature or milestone containing tasks (1-4 weeks)
type Epic struct {
	TempID             string              `json:"temp_id"`                  // Simple ID like "1", "2"
	Title              string              `json:"title"`                    // Major feature or milestone
	Description        string              `json:"description"`              // What this delivers
	Context            interface{}         `json:"context"`                  // Business/user/brand context (object or string)
	AcceptanceCriteria []string            `json:"acceptance_criteria"`      // When this epic is complete
	Testing            TestingRequirements `json:"testing"`                  // Epic-level testing strategy
	Tasks              []Task              `json:"tasks"`                    // Tasks that complete this epic
	DependsOn          []string            `json:"depends_on"`               // Epic temp IDs this depends on
	EstimatedDays      *float64            `json:"estimated_days,omitempty"` // Working days for entire epic
	Labels             []string            `json:"labels,omitempty"`         // Tags for categorization
}

// ProjectContext extracted from the PRD.
// Propagated into every epic, task, and subtask.
type ProjectContext struct {
	ProductName     string              `json:"product_name"`                   // Name of the product
	ElevatorPitch   string              `json:"elevator_pitch"`                 // One sentence: what and why
	TargetAudience  string              `json:"target_audience"`                // Primary and secondary users
	BusinessGoals   FlexibleStringSlice `json:"business_goals"`                 // What the business wants
	UserGoals       FlexibleStringSlice `json:"user_goals"`                     // What users want
	BrandGuidelines interface{}         `json:"brand_guidelines,omitempty"`     // Voice, tone, visual identity (string or object)
	TechStack       FlexibleStringSlice `json:"tech_stack"`                     // Technologies and tools
	Constraints     FlexibleStringSlice `json:"constraints"`                    // Technical/business constraints
}

// ParseResponse is the full PRD parsing output.
// Hierarchical: Project -> Epics -> Tasks -> Subtasks
type ParseResponse struct {
	Project  ProjectContext   `json:"project"`  // Extracted project context
	Epics    []Epic           `json:"epics"`    // Major features/milestones
	Metadata ResponseMetadata `json:"metadata"` // Summary statistics
}

// ResponseMetadata provides summary stats about the parsed PRD.
type ResponseMetadata struct {
	TotalEpics         int             `json:"total_epics"`
	TotalTasks         int             `json:"total_tasks"`
	TotalSubtasks      int             `json:"total_subtasks"`
	EstimatedTotalDays *float64        `json:"estimated_total_days,omitempty"`
	TestingCoverage    TestingCoverage `json:"testing_coverage"`
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
	PriorityVeryLow  Priority = "very-low"
)

// ParseConfig configures PRD parsing behavior.
type ParseConfig struct {
	TargetEpics      int      `json:"target_epics"`      // Default: 3
	TasksPerEpic     int      `json:"tasks_per_epic"`    // Default: 5
	SubtasksPerTask  int      `json:"subtasks_per_task"` // Default: 4
	DefaultPriority  Priority `json:"default_priority"`  // Default: medium
	TestingLevel     string   `json:"testing_level"`     // minimal/standard/comprehensive
	PropagateContext bool     `json:"propagate_context"` // Default: true
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
			return &ValidationError{
				Field:   fmt.Sprintf("epics[%d].tasks", i),
				Message: fmt.Sprintf("epic '%s' has empty tasks array - must decompose into tasks", epic.Title),
			}
		}
		for j, task := range epic.Tasks {
			if task.Title == "" {
				return &ValidationError{Field: fmt.Sprintf("epics[%d].tasks[%d].title", i, j), Message: "required"}
			}
			if len(task.Subtasks) == 0 {
				return &ValidationError{
					Field:   fmt.Sprintf("epics[%d].tasks[%d].subtasks", i, j),
					Message: fmt.Sprintf("task '%s' has empty subtasks array - must decompose into subtasks", task.Title),
				}
			}
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
