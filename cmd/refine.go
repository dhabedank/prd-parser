package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/dhabedank/prd-parser/internal/core"
	"github.com/dhabedank/prd-parser/internal/llm"
)

var (
	refineFeedback    string
	refineCascade     bool
	refineScanAll     bool
	refineDryRun      bool
	refinePRDPath     string
)

// RefineCmd represents the refine command
var RefineCmd = &cobra.Command{
	Use:   "refine <issue-id>",
	Short: "Refine an issue and propagate corrections",
	Long: `Refine an issue based on feedback and propagate corrections to related issues.

This command:
1. Analyzes the target issue against your feedback
2. Identifies what concepts are misaligned
3. Regenerates the issue with corrections
4. Scans children and (optionally) all issues for the same misalignment
5. Updates affected issues via beads

Example:
  prd-parser refine test-e6 --feedback "RealHerd is voice-first, not a CRM"
  prd-parser refine test-e3t2 --feedback "This should use OpenRouter, not direct OpenAI" --scan-all`,
	Args: cobra.ExactArgs(1),
	RunE: runRefine,
}

func init() {
	RefineCmd.Flags().StringVarP(&refineFeedback, "feedback", "f", "", "Correction feedback (required)")
	RefineCmd.Flags().BoolVar(&refineCascade, "cascade", true, "Also update children of the target issue")
	RefineCmd.Flags().BoolVar(&refineScanAll, "scan-all", true, "Scan ALL issues for the same misalignment (not just children)")
	RefineCmd.Flags().BoolVar(&refineDryRun, "dry-run", false, "Preview changes without applying them")
	RefineCmd.Flags().StringVar(&refinePRDPath, "prd", "", "Path to PRD file for context (recommended)")
	RefineCmd.MarkFlagRequired("feedback")
}

func runRefine(cmd *cobra.Command, args []string) error {
	issueID := args[0]
	ctx := context.Background()

	// Load PRD if provided
	var prdContent string
	if refinePRDPath != "" {
		data, err := os.ReadFile(refinePRDPath)
		if err != nil {
			return fmt.Errorf("failed to read PRD: %w", err)
		}
		prdContent = string(data)
	}

	// Step 1: Load target issue from beads
	fmt.Printf("Loading issue %s...\n", issueID)
	targetIssue, err := loadBeadsIssue(issueID)
	if err != nil {
		return fmt.Errorf("failed to load issue: %w", err)
	}
	fmt.Printf("  Found: %s\n", targetIssue.Title)

	// Step 2: Load all issues for scanning
	fmt.Println("Loading all issues for analysis...")
	allIssues, err := loadAllBeadsIssues()
	if err != nil {
		return fmt.Errorf("failed to load issues: %w", err)
	}
	fmt.Printf("  Loaded %d issues\n", len(allIssues))

	// Step 3: Create LLM adapter
	llmConfig := llm.Config{
		PreferCLI: true,
	}
	adapter := llm.NewClaudeCLIAdapter(llmConfig)
	if !adapter.IsAvailable() {
		return fmt.Errorf("claude CLI not available")
	}

	// Step 4: Analyze misalignment and get corrections
	fmt.Println("\nAnalyzing misalignment...")
	analysis, err := analyzeAndCorrect(ctx, adapter, targetIssue, refineFeedback, prdContent)
	if err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	fmt.Printf("\nIdentified misalignment:\n")
	for _, concept := range analysis.WrongConcepts {
		fmt.Printf("  - %s\n", concept)
	}
	fmt.Printf("\nCorrected version:\n")
	fmt.Printf("  Title: %s\n", analysis.CorrectedTitle)
	fmt.Printf("  Description: %s\n", truncate(analysis.CorrectedDescription, 100))

	// Step 5: Find affected issues
	fmt.Println("\nScanning for affected issues...")
	var affectedIssues []core.BeadsIssue

	// Always include children if cascade is enabled
	if refineCascade {
		children := findChildren(allIssues, issueID)
		affectedIssues = append(affectedIssues, children...)
		if len(children) > 0 {
			fmt.Printf("  Found %d children\n", len(children))
		}
	}

	// Scan all issues for wrong concepts if enabled
	if refineScanAll && len(analysis.WrongConcepts) > 0 {
		matches := findIssuesWithConcepts(allIssues, analysis.WrongConcepts, issueID)
		// Deduplicate
		seen := make(map[string]bool)
		for _, issue := range affectedIssues {
			seen[issue.ID] = true
		}
		for _, issue := range matches {
			if !seen[issue.ID] {
				affectedIssues = append(affectedIssues, issue)
				seen[issue.ID] = true
			}
		}
		fmt.Printf("  Found %d issues with similar misalignment\n", len(matches))
	}

	// Step 6: Apply corrections
	fmt.Printf("\n--- Changes to apply ---\n")
	fmt.Printf("Target: %s\n", issueID)
	for _, issue := range affectedIssues {
		fmt.Printf("  + %s: %s\n", issue.ID, truncate(issue.Title, 50))
	}

	if refineDryRun {
		fmt.Println("\n[dry-run] No changes applied")
		return nil
	}

	// Apply to target
	fmt.Printf("\nApplying corrections...\n")
	if err := applyCorrection(issueID, analysis.CorrectedTitle, analysis.CorrectedDescription); err != nil {
		fmt.Printf("  Warning: failed to update %s: %v\n", issueID, err)
	} else {
		fmt.Printf("  ✓ Updated %s\n", issueID)
	}

	// Apply to affected issues (regenerate each with context)
	for _, issue := range affectedIssues {
		corrected, err := regenerateWithContext(ctx, adapter, issue, analysis.WrongConcepts, analysis.CorrectConcepts, prdContent)
		if err != nil {
			fmt.Printf("  Warning: failed to regenerate %s: %v\n", issue.ID, err)
			continue
		}
		if err := applyCorrection(issue.ID, corrected.CorrectedTitle, corrected.CorrectedDescription); err != nil {
			fmt.Printf("  Warning: failed to update %s: %v\n", issue.ID, err)
		} else {
			fmt.Printf("  ✓ Updated %s\n", issue.ID)
		}
	}

	fmt.Printf("\n--- Summary ---\n")
	fmt.Printf("Updated: 1 target + %d related issues\n", len(affectedIssues))

	return nil
}

// Type aliases for convenience
type BeadsIssue = core.BeadsIssue
type AnalysisResult = core.AnalysisResult

// loadBeadsIssue loads a single issue from beads
func loadBeadsIssue(id string) (*core.BeadsIssue, error) {
	cmd := exec.Command("bd", "show", id, "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		// Fallback to parsing text output
		cmd = exec.Command("bd", "show", id)
		output, err = cmd.Output()
		if err != nil {
			return nil, err
		}
		return parseBeadsShowOutput(id, string(output))
	}
	return core.ParseBeadsJSON(output)
}

// parseBeadsShowOutput parses the text output of bd show
func parseBeadsShowOutput(id, output string) (*core.BeadsIssue, error) {
	issue := &core.BeadsIssue{ID: id}

	lines := strings.Split(output, "\n")
	for i, line := range lines {
		// First line has format: ○ test-e1 [EPIC] · Title   [● P1 · OPEN]
		if i == 0 && strings.Contains(line, "·") {
			parts := strings.Split(line, "·")
			if len(parts) >= 2 {
				issue.Title = strings.TrimSpace(parts[1])
			}
			if strings.Contains(line, "[EPIC]") {
				issue.Type = "epic"
			} else {
				issue.Type = "task"
			}
		}

		// Look for DESCRIPTION section
		if strings.HasPrefix(line, "DESCRIPTION") {
			// Collect lines until next section
			var desc []string
			for j := i + 1; j < len(lines); j++ {
				if strings.HasPrefix(lines[j], "ACCEPTANCE") ||
				   strings.HasPrefix(lines[j], "LABELS") ||
				   strings.HasPrefix(lines[j], "DEPENDS") ||
				   strings.HasPrefix(lines[j], "CHILDREN") ||
				   strings.HasPrefix(lines[j], "BLOCKS") {
					break
				}
				desc = append(desc, lines[j])
			}
			issue.Description = strings.TrimSpace(strings.Join(desc, "\n"))
		}

		// Look for parent
		if strings.Contains(line, "PARENT") {
			for j := i + 1; j < len(lines); j++ {
				if strings.Contains(lines[j], "→") {
					// Extract parent ID
					parts := strings.Fields(lines[j])
					for _, p := range parts {
						if strings.HasPrefix(p, "test-") || strings.HasPrefix(p, "prd-") {
							issue.Parent = strings.TrimSuffix(p, ":")
							break
						}
					}
					break
				}
			}
		}
	}

	return issue, nil
}

// loadAllBeadsIssues loads all issues from beads
func loadAllBeadsIssues() ([]core.BeadsIssue, error) {
	cmd := exec.Command("bd", "list", "--status=all", "--limit", "0")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var issues []core.BeadsIssue
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if !strings.HasPrefix(line, "○") {
			continue
		}

		// Parse: ○ test-e1t1 [● P0] [task] [labels] - Title
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		id := fields[1]

		// Find title after " - "
		titleIdx := strings.Index(line, " - ")
		title := ""
		if titleIdx != -1 {
			title = line[titleIdx+3:]
		}

		// Determine type from ID pattern
		issueType := "task"
		if strings.Contains(id, "t") && !strings.Contains(id, "s") {
			// Has 't' but no 's' - could be task under epic
			if strings.Count(id, "t") == 1 && strings.Count(id, "e") == 1 {
				issueType = "task"
			}
		} else if !strings.Contains(id, "t") {
			issueType = "epic"
		}

		issues = append(issues, core.BeadsIssue{
			ID:    id,
			Title: title,
			Type:  issueType,
		})
	}

	return issues, nil
}

// analyzeAndCorrect uses LLM to analyze the misalignment and generate corrections
func analyzeAndCorrect(ctx context.Context, adapter *llm.ClaudeCLIAdapter, issue *core.BeadsIssue, feedback, prdContent string) (*AnalysisResult, error) {
	systemPrompt := `You analyze misaligned project issues and generate corrections.

Given an issue and user feedback about what's wrong, you:
1. Identify the WRONG CONCEPTS (phrases, terms, framing that are incorrect)
2. Identify the CORRECT CONCEPTS (what should replace them)
3. Generate a corrected title and description

Return JSON:
{
  "wrong_concepts": ["pipeline tracking", "CRM management", "deal stages"],
  "correct_concepts": ["conversation insights", "voice intelligence", "activity visibility"],
  "corrected_title": "Corrected title here",
  "corrected_description": "Corrected description that maintains the original intent but fixes the framing"
}

Be specific about wrong concepts - they'll be used to search other issues.
Keep the corrected content the same length/detail as original, just fix the framing.`

	userPrompt := fmt.Sprintf(`Analyze this issue and fix the misalignment.

ISSUE ID: %s
ISSUE TYPE: %s
CURRENT TITLE: %s
CURRENT DESCRIPTION:
%s

USER FEEDBACK (what's wrong):
%s
`, issue.ID, issue.Type, issue.Title, issue.Description, feedback)

	if prdContent != "" {
		userPrompt += fmt.Sprintf(`
ORIGINAL PRD (for correct context):
%s
`, truncate(prdContent, 4000))
	}

	userPrompt += `
Return JSON with wrong_concepts, correct_concepts, corrected_title, and corrected_description.`

	output, err := adapter.GenerateRaw(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	return core.ParseAnalysisResult(output)
}

// findChildren finds all issues that are children of the given parent
func findChildren(allIssues []core.BeadsIssue, parentID string) []core.BeadsIssue {
	var children []core.BeadsIssue

	// Extract the pattern (e.g., "test-e6" -> look for "test-e6t")
	for _, issue := range allIssues {
		if issue.ID == parentID {
			continue
		}
		// Check if this issue's ID starts with the parent ID pattern
		// e.g., test-e6t1 is child of test-e6
		// e.g., test-e6t1s1 is child of test-e6t1
		if strings.HasPrefix(issue.ID, parentID) && issue.ID != parentID {
			// Verify it's a direct or indirect child
			suffix := strings.TrimPrefix(issue.ID, parentID)
			if len(suffix) > 0 && (suffix[0] == 't' || suffix[0] == 's') {
				children = append(children, issue)
			}
		}
	}

	return children
}

// findIssuesWithConcepts finds issues containing any of the wrong concepts
func findIssuesWithConcepts(allIssues []core.BeadsIssue, concepts []string, excludeID string) []core.BeadsIssue {
	var matches []core.BeadsIssue

	for _, issue := range allIssues {
		if issue.ID == excludeID {
			continue
		}

		// Check title and description for wrong concepts
		text := strings.ToLower(issue.Title + " " + issue.Description)
		for _, concept := range concepts {
			if strings.Contains(text, strings.ToLower(concept)) {
				matches = append(matches, issue)
				break
			}
		}
	}

	return matches
}

// regenerateWithContext regenerates an issue with correction context
func regenerateWithContext(ctx context.Context, adapter *llm.ClaudeCLIAdapter, issue core.BeadsIssue, wrongConcepts, correctConcepts []string, prdContent string) (*AnalysisResult, error) {
	// First load full issue details
	fullIssue, err := loadBeadsIssue(issue.ID)
	if err != nil {
		fullIssue = &issue // Use what we have
	}

	systemPrompt := `You fix misaligned concepts in project issues.

Given an issue and a list of wrong concepts to replace with correct concepts,
generate a corrected version that maintains the original structure but fixes the framing.

Return JSON:
{
  "corrected_title": "Fixed title",
  "corrected_description": "Fixed description"
}`

	userPrompt := fmt.Sprintf(`Fix this issue by replacing wrong concepts with correct ones.

WRONG CONCEPTS (replace these):
%v

CORRECT CONCEPTS (use these instead):
%v

ISSUE TO FIX:
ID: %s
Title: %s
Description: %s

Return JSON with corrected_title and corrected_description.
Keep the same structure and detail level, just fix the conceptual framing.`,
		wrongConcepts, correctConcepts, fullIssue.ID, fullIssue.Title, fullIssue.Description)

	output, err := adapter.GenerateRaw(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	return core.ParseAnalysisResult(output)
}

// applyCorrection applies a correction to an issue via bd update
func applyCorrection(id, title, description string) error {
	// Update title
	if title != "" {
		cmd := exec.Command("bd", "update", id, "--title", title)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to update title: %w", err)
		}
	}

	// Update description
	if description != "" {
		cmd := exec.Command("bd", "update", id, "--description", description)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to update description: %w", err)
		}
	}

	return nil
}

// truncate shortens a string to maxLen
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// Progress indicator helper
func startProgress() chan bool {
	done := make(chan bool)
	go func() {
		start := time.Now()
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				fmt.Printf("  Still processing... (%s)\n", time.Since(start).Truncate(time.Second))
			}
		}
	}()
	return done
}
