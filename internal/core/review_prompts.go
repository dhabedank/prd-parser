package core

import (
	"fmt"
	"strings"
)

// ReviewSystemPrompt asks LLM to check and fix structural issues in a generated PRD breakdown.
const ReviewSystemPrompt = `You review a generated PRD breakdown and fix structural issues.

## CHECK FOR AND FIX

1. **Missing "Project Foundation" epic as Epic 1**
   - Epic 1 MUST be "Project Foundation" or "Project Setup" with setup tasks
   - If missing, add it with temp_id: "1" and renumber other epics

2. **Feature epics not depending on Epic 1**
   - All epics with temp_id > "1" MUST have depends_on: ["1"]
   - Add missing dependencies

3. **Setup tasks missing from foundation epic**
   - Epic 1 should have tasks for: project initialization, dependency installation,
     database setup, authentication setup, environment configuration
   - Add any missing setup tasks based on the tech stack

4. **Incorrect ordering (features before setup)**
   - Epic 1 = Foundation, Epic 2+ = Features
   - Reorder if setup work is scattered across feature epics

5. **Missing cross-task dependencies**
   - If Task B uses something Task A creates, B depends on A
   - Check for database → API → UI dependency chains

6. **Tasks that assume infrastructure exists without depending on setup**
   - Any task using the tech stack must depend on the setup task that installs it

## OUTPUT FORMAT

Return the FIXED JSON structure with corrections applied. The structure should be:

{
  "review_notes": "What was fixed (or 'No changes needed')",
  "project": { ... },
  "epics": [ ... ],
  "metadata": { ... }
}

If no fixes are needed, return the original structure unchanged with review_notes: "No changes needed".

IMPORTANT:
- Return ONLY valid JSON - no explanations before or after
- Preserve all existing fields, testing requirements, context, labels
- Only modify structure/dependencies to fix issues
- Renumber temp_ids if epics are reordered (maintain 1.1, 1.2 format for tasks)
- Start your response with { and end with }`

// ReviewUserPromptTemplate is the template for the review user prompt.
const ReviewUserPromptTemplate = `Review this generated PRD breakdown and fix any structural issues.

## TECH STACK FROM PRD
%s

## GENERATED STRUCTURE
%s

## REQUIREMENTS

1. Epic 1 MUST be "Project Foundation" with setup for: %s
2. All feature epics (2, 3, 4...) MUST have depends_on: ["1"]
3. Dependencies should follow: setup → backend → frontend chains
4. Tasks should not assume infrastructure exists without dependencies

Fix any issues and return the corrected JSON with a "review_notes" field explaining changes.
If no changes needed, return the original with review_notes: "No changes needed".`

// BuildReviewPrompt creates the user prompt for review.
func BuildReviewPrompt(response *ParseResponse, prdContent string) string {
	// Extract tech stack as a string
	techStack := strings.Join(response.Project.TechStack.ToSlice(), ", ")
	if techStack == "" {
		techStack = "Not specified"
	}

	// Serialize the response to JSON for review
	responseJSON := serializeForReview(response)

	return fmt.Sprintf(
		ReviewUserPromptTemplate,
		techStack,
		responseJSON,
		techStack,
	)
}

// serializeForReview converts ParseResponse to a readable JSON string for the review prompt.
func serializeForReview(response *ParseResponse) string {
	var sb strings.Builder

	sb.WriteString("{\n")
	sb.WriteString(fmt.Sprintf("  \"project\": {\n    \"product_name\": %q,\n    \"tech_stack\": %v\n  },\n",
		response.Project.ProductName,
		response.Project.TechStack.ToSlice()))

	sb.WriteString("  \"epics\": [\n")
	for i, epic := range response.Epics {
		sb.WriteString(fmt.Sprintf("    {\n      \"temp_id\": %q,\n      \"title\": %q,\n      \"depends_on\": %v,\n",
			epic.TempID, epic.Title, epic.DependsOn))
		sb.WriteString("      \"tasks\": [\n")
		for j, task := range epic.Tasks {
			sb.WriteString(fmt.Sprintf("        {\"temp_id\": %q, \"title\": %q, \"depends_on\": %v}",
				task.TempID, task.Title, task.DependsOn))
			if j < len(epic.Tasks)-1 {
				sb.WriteString(",")
			}
			sb.WriteString("\n")
		}
		sb.WriteString("      ]\n    }")
		if i < len(response.Epics)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}
	sb.WriteString("  ]\n}")

	return sb.String()
}
