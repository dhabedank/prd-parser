//go:build integration

package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestBeadsIntegration tests the full parse-to-beads workflow.
// Run with: go test -tags=integration -v ./tests/integration/...
func TestBeadsIntegration(t *testing.T) {
	// Skip if no LLM available
	if !hasLLM() {
		t.Skip("Skipping: No LLM adapter available (need Claude Code, Codex, or ANTHROPIC_API_KEY)")
	}

	// Skip if bd not installed
	if _, err := exec.LookPath("bd"); err != nil {
		t.Skip("Skipping: beads CLI (bd) not installed")
	}

	// Create temp directory
	testDir, err := os.MkdirTemp("", "prd-parser-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Initialize git (beads requires git)
	runCmd(t, testDir, "git", "init")
	runCmd(t, testDir, "git", "config", "user.email", "test@test.com")
	runCmd(t, testDir, "git", "config", "user.name", "Test")

	// Initialize beads
	runCmd(t, testDir, "bd", "init")

	// Create test PRD
	prdPath := filepath.Join(testDir, "prd.md")
	prdContent := `# Test Project

## Overview
Build a simple todo application for testing purposes.

## Requirements

### Core Features
1. Create todos with title and description
2. Mark todos as complete
3. Delete todos
4. List all todos

### Technical Requirements
- Use a simple JSON file for storage
- Implement as a CLI tool
- Include unit tests

## User Story
As a developer, I want to track my tasks so that I stay organized.

## Success Criteria
- All CRUD operations work correctly
- Tests pass
- Documentation is complete
`
	if err := os.WriteFile(prdPath, []byte(prdContent), 0644); err != nil {
		t.Fatalf("Failed to create PRD: %v", err)
	}

	// Get the prd-parser binary path (assumes it's built)
	binaryPath, err := filepath.Abs("../../prd-parser")
	if err != nil {
		t.Fatalf("Failed to get binary path: %v", err)
	}

	// Check binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Fatal("prd-parser binary not found - run 'make build' first")
	}

	// Run prd-parser with minimal epics for faster testing
	cmd := exec.Command(binaryPath, "parse", prdPath, "--epics", "1", "--tasks", "2", "--subtasks", "2")
	cmd.Dir = testDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("prd-parser failed: %v\nOutput: %s", err, string(output))
	}
	t.Logf("prd-parser output:\n%s", string(output))

	// Verify issues were created
	listCmd := exec.Command("bd", "list")
	listCmd.Dir = testDir
	listOutput, err := listCmd.Output()
	if err != nil {
		t.Fatalf("bd list failed: %v", err)
	}

	// Check that at least one issue exists
	if len(listOutput) < 10 {
		t.Errorf("Expected beads issues to be created, got: %s", string(listOutput))
	}
	t.Logf("Created issues:\n%s", string(listOutput))
}

func hasLLM() bool {
	// Check Claude CLI
	if _, err := exec.LookPath("claude"); err == nil {
		return true
	}
	// Check Codex CLI
	if _, err := exec.LookPath("codex"); err == nil {
		return true
	}
	// Check API key
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		return true
	}
	return false
}

func runCmd(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Command '%s %v' failed: %v\nOutput: %s", name, args, err, string(output))
	}
}
