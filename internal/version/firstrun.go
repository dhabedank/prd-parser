package version

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dhabedank/prd-parser/internal/tui"
)

// IsFirstRun returns true if this appears to be the first run.
// Checks for existence of config file or first-run marker.
func IsFirstRun() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	// Check for config file
	configPath := filepath.Join(home, ".prd-parser.yaml")
	if _, err := os.Stat(configPath); err == nil {
		return false // Config exists, not first run
	}

	// Check for first-run marker
	markerPath := filepath.Join(home, ".prd-parser", ".initialized")
	if _, err := os.Stat(markerPath); err == nil {
		return false // Already initialized
	}

	return true
}

// MarkInitialized creates the first-run marker.
func MarkInitialized() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	dir := filepath.Join(home, ".prd-parser")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}

	markerPath := filepath.Join(dir, ".initialized")
	_ = os.WriteFile(markerPath, []byte{}, 0644)
}

// PrintFirstRunNotice prints a welcome message for first-time users.
func PrintFirstRunNotice() {
	fmt.Println()
	fmt.Printf("%s Welcome to prd-parser!\n", tui.TitleStyle.Render("*"))
	fmt.Println()
	fmt.Println("  Quick start:")
	fmt.Printf("    1. Run %s to configure your preferred models\n", tui.ModelStyle.Render("prd-parser setup"))
	fmt.Printf("    2. Initialize beads in your project: %s\n", tui.ModelStyle.Render("bd init --prefix myproject"))
	fmt.Printf("    3. Parse your PRD: %s\n", tui.ModelStyle.Render("prd-parser parse docs/prd.md"))
	fmt.Println()
	fmt.Printf("  %s\n", tui.HelpStyle.Render("Run 'prd-parser --help' for all options"))
	fmt.Println()

	// Mark as initialized so we don't show this again
	MarkInitialized()
}
