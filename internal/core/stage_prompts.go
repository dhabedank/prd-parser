package core

import "fmt"

// Stage 1: PRD → Epics (high-level structure only)
const Stage1SystemPrompt = `You are a PRD parser performing Stage 1: Epic extraction.

Your job is to extract the HIGH-LEVEL EPIC STRUCTURE from a PRD. Do NOT generate tasks or subtasks - those come later.

## OUTPUT FORMAT

Return a JSON object with:
1. "project" - Extracted context (product_name, elevator_pitch, target_audience, brand_guidelines, business_goals, user_goals, tech_stack, constraints)
2. "epics" - Array of epic summaries (WITHOUT tasks)

## EPIC STRUCTURE

Each epic should have:
- temp_id: "1", "2", "3", etc.
- title: Major feature or milestone name
- description: What this epic delivers
- context: Business/user context for why this matters
- acceptance_criteria: Array of completion conditions
- testing: Testing requirements object
- depends_on: Array of epic temp_ids this depends on
- estimated_days: Estimated working days
- labels: Categorization tags

## GUIDELINES

- Create 3-8 epics based on PRD complexity
- Epics should be independently deployable milestones
- Foundation/infrastructure epics should come first
- Do NOT include "tasks" field - tasks are generated in Stage 2
- **First epic should include project setup** - environment, dependencies, basic config
- **Acceptance criteria must be verifiable** - include at least one "can run/demonstrate X"
- **Every feature needs an interface** - don't build backend without a way to interact with it; users need to SEE features work

## OUTPUT REQUIREMENTS

- Return ONLY valid JSON, no markdown fencing
- No explanations or commentary
- Start with { and end with }`

const Stage1UserPromptTemplate = `Extract epics from this PRD.

Target epics: ~%d (adjust based on actual PRD complexity)
Default priority: %s
Testing level: %s

---
PRD CONTENT:
---
%s
---

Return JSON with "project" and "epics" fields. Do NOT include tasks - only high-level epics.`

// Stage 2: Epic → Tasks
const Stage2SystemPrompt = `You are a PRD parser performing Stage 2: Task generation.

Your job is to break down ONE EPIC into its component TASKS. Do NOT generate subtasks - those come later.

## OUTPUT FORMAT

Return a JSON object with:
{
  "tasks": [
    {
      "temp_id": "1.1", "1.2", etc. (epic.temp_id + "." + task_number)
      "title": "Task name",
      "description": "What needs to be done",
      "context": "Propagated context + task-specific context",
      "design_notes": "Technical approach",
      "testing": { testing requirements },
      "priority": "critical/high/medium/low/very-low",
      "depends_on": ["1.1"] (other task temp_ids),
      "estimated_hours": 4,
      "labels": ["backend", "api"]
    }
  ]
}

## GUIDELINES

- Generate 3-8 tasks per epic based on complexity
- Tasks should be 2-8 hours of work each
- PROPAGATE CONTEXT: Include why this task matters to the business
- Set appropriate priority based on dependencies and risk
- Do NOT include "subtasks" field - subtasks are generated in Stage 3
- **Include operational tasks**: If code adds dependencies, include installation; if it changes schemas, include migration/type generation
- **End sequences with verification**: After a set of related tasks, include a task to verify everything works together
- **UI before or with backend**: Don't create API endpoints without a page/component to call them - users need to SEE features work, not just trust the backend exists

## OUTPUT REQUIREMENTS

- Return ONLY valid JSON with "tasks" array
- No markdown fencing, no explanations
- Start with { and end with }`

const Stage2UserPromptTemplate = `Break down this epic into tasks.

EPIC:
- ID: %s
- Title: %s
- Description: %s
- Context: %v
- Acceptance Criteria: %v

PROJECT CONTEXT:
- Product: %s
- Target Users: %s
- Tech Stack: %v

Target tasks: ~%d
Default priority: %s

Return JSON with "tasks" array. Do NOT include subtasks.`

// Stage 3: Task → Subtasks
const Stage3SystemPrompt = `You are a PRD parser performing Stage 3: Subtask generation.

Your job is to break down ONE TASK into ATOMIC SUBTASKS that can be completed independently.

## OUTPUT FORMAT

Return a JSON object with:
{
  "subtasks": [
    {
      "temp_id": "1.1.1", "1.1.2", etc. (task.temp_id + "." + subtask_number)
      "title": "Atomic action",
      "description": "Specific implementation details",
      "context": "Why this matters",
      "testing": { testing requirements },
      "estimated_minutes": 45,
      "depends_on": ["1.1.1"],
      "labels": ["backend"]
    }
  ]
}

## GUIDELINES

- Generate 2-6 subtasks per task based on complexity
- Subtasks should be 30 minutes to 2 hours of work
- Subtasks should be specific enough for an LLM to implement without clarification
- Keep context propagation - include why this matters
- **Don't skip practical steps**: Installing packages, running builds, verifying output
- **Last subtask should verify**: Run tests, start dev server, check the feature works

## OUTPUT REQUIREMENTS

- Return ONLY valid JSON with "subtasks" array
- No markdown fencing, no explanations
- Start with { and end with }`

const Stage3UserPromptTemplate = `Break down this task into subtasks.

TASK:
- ID: %s
- Title: %s
- Description: %s
- Context: %v
- Design Notes: %v

EPIC CONTEXT: %s

PROJECT:
- Product: %s
- Target Users: %s

Target subtasks: ~%d

Return JSON with "subtasks" array.`

// BuildStage1Prompt builds the Stage 1 user prompt.
func BuildStage1Prompt(prdContent string, config ParseConfig) string {
	return fmt.Sprintf(
		Stage1UserPromptTemplate,
		config.TargetEpics,
		config.DefaultPriority,
		config.TestingLevel,
		prdContent,
	)
}

// BuildStage2Prompt builds the Stage 2 user prompt.
func BuildStage2Prompt(epic Epic, project ProjectContext, config ParseConfig) string {
	return fmt.Sprintf(
		Stage2UserPromptTemplate,
		epic.TempID,
		epic.Title,
		epic.Description,
		epic.Context,
		epic.AcceptanceCriteria,
		project.ProductName,
		project.TargetAudience,
		project.TechStack,
		config.TasksPerEpic,
		config.DefaultPriority,
	)
}

// BuildStage3Prompt builds the Stage 3 user prompt.
func BuildStage3Prompt(task Task, epicContext string, project ProjectContext, config ParseConfig) string {
	designNotes := ""
	if task.DesignNotes != nil {
		designNotes = *task.DesignNotes
	}
	return fmt.Sprintf(
		Stage3UserPromptTemplate,
		task.TempID,
		task.Title,
		task.Description,
		task.Context,
		designNotes,
		epicContext,
		project.ProductName,
		project.TargetAudience,
		config.SubtasksPerTask,
	)
}
