package core

import "fmt"

// Stage 1: PRD → Epics (high-level structure only)
const Stage1SystemPrompt = `You are a PRD parser performing Stage 1: Epic extraction.

Your job is to extract the HIGH-LEVEL EPIC STRUCTURE from a PRD. Do NOT generate tasks or subtasks - those come later.

## OUTPUT FORMAT

Return a JSON object with:
1. "project" - Extracted context (product_name, elevator_pitch, target_audience, brand_guidelines, business_goals, user_goals, tech_stack, constraints)
2. "epics" - Array of epic summaries (WITHOUT tasks)

## MANDATORY: EPIC 1 MUST BE PROJECT FOUNDATION (CRITICAL)

**Epic 1 is ALWAYS "Project Foundation/Setup"** - this is not optional. Before ANY feature can be built, the project must exist!

**Epic 1 must include tasks for:**
- Initialize the project framework (Next.js, Vite, etc.)
- Install and configure database/backend (Convex, Supabase, Prisma, etc.)
- Set up authentication (Clerk, Auth0, NextAuth, etc.)
- Install core dependencies from the tech stack
- Environment variable configuration
- Basic project structure and configuration files

**Example Epic 1 for a Next.js + Convex + Clerk project:**
  temp_id: "1"
  title: "Project Foundation & Core Setup"
  description: "Initialize project, install dependencies, configure Convex database and Clerk authentication"
  acceptance_criteria: ["Can run npm dev and see basic page", "Convex backend is connected", "Clerk authentication works"]

**Feature epics (2, 3, 4...) MUST depend on Epic 1:**
  Epic 2: depends_on: ["1"]
  Epic 3: depends_on: ["1"]

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

## EPIC DEPENDENCIES

**All feature epics depend on Epic 1 (Project Foundation).**

**Feature epics may depend on each other when:**
- One epic provides infrastructure another needs (Auth epic → Dashboard epic)
- Data models from one epic are used by another

## DETERMINING EPIC COUNT (DON'T DEFAULT TO TARGET)

**Analyze the PRD to determine the RIGHT number of epics. The "target" is just a rough guide.**

- The PRD's actual scope should drive epic count, NOT the target number
- A complex PRD with 5 major features needs 5-6 feature epics (plus Epic 1 foundation)
- A simple PRD with 1-2 features might only need 2-3 total epics
- **Don't force features together just to hit the target**
- **Don't pad with unnecessary epics just to hit the target**

## OTHER GUIDELINES

- Epics should be independently deployable milestones
- Foundation epic (Epic 1) comes first
- Feature epics follow in logical dependency order
- Do NOT include "tasks" field - tasks are generated in Stage 2
- **Acceptance criteria must be verifiable** - include at least one "can run/demonstrate X"
- **Every feature needs an interface** - don't build backend without a way to interact with it

## OUTPUT REQUIREMENTS

- Return ONLY valid JSON, no markdown fencing
- No explanations or commentary
- Start with { and end with }`

const Stage1UserPromptTemplate = `Extract epics from this PRD.

CRITICAL REQUIREMENTS:
1. Epic 1 MUST be "Project Foundation/Setup" (initialize project, install dependencies, set up database, configure auth)
2. Feature epics (2, 3, 4...) follow based on the PRD's actual features
3. ALL feature epics must have depends_on: ["1"]

Suggested epic count: ~%d (but use the PRD's actual features to determine the real count - don't force this number)
Default priority: %s
Testing level: %s

---
PRD CONTENT:
---
%s
---

Return JSON with "project" and "epics" fields. Do NOT include tasks - only high-level epics.
Remember: Epic 1 is ALWAYS project foundation. Feature epics follow.`

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

## DEPENDENCIES (CRITICAL - MOST TASKS SHOULD HAVE depends_on)

**Dependencies are NOT optional.** Most tasks depend on something. Empty depends_on for ALL tasks is WRONG.

**Common dependency patterns:**
- Database/schema setup → API endpoints that query it
- Authentication setup → Pages/routes that require auth
- SDK/library installation → Code that uses the SDK
- Data models → CRUD operations on those models
- Backend endpoints → Frontend pages that call them
- Config/environment → Features that need that config

**Example within an epic:**
  Task 1.1: "Set up database schema" - depends_on: []
  Task 1.2: "Create API endpoints for users" - depends_on: ["1.1"]
  Task 1.3: "Build user management UI" - depends_on: ["1.2"]
  Task 1.4: "Add user authentication" - depends_on: ["1.1"]
  Task 1.5: "Create protected dashboard" - depends_on: ["1.3", "1.4"]

**How to decide depends_on:**
- "Can I start this task if X isn't done?" → If no, X goes in depends_on
- Foundation/setup tasks usually have no dependencies
- Most other tasks depend on at least one thing

## OTHER GUIDELINES

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

## DEPENDENCIES (SUBTASKS WITHIN A TASK)

**Subtasks often have sequential dependencies within their task.** Most subtasks depend on the previous step.

**Common patterns:**
- Create schema/types → Use those types in code
- Write failing test → Implement code to pass it
- Install package → Use package in implementation
- Create component → Wire component to data

**Example:**
  Subtask 1.1.1: "Create TypeScript types" - depends_on: []
  Subtask 1.1.2: "Write unit test for validation" - depends_on: ["1.1.1"]
  Subtask 1.1.3: "Implement validation function" - depends_on: ["1.1.1", "1.1.2"]
  Subtask 1.1.4: "Run tests and verify" - depends_on: ["1.1.3"]

## OTHER GUIDELINES

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
