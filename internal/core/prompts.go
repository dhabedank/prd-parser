package core

import (
	"fmt"
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
