package version

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dhabedank/prd-parser/internal/tui"
)

const (
	// GitHubRepo is the repository for version checks.
	GitHubRepo = "dhabedank/prd-parser"

	// CheckInterval is how often to check for updates (24 hours).
	CheckInterval = 24 * time.Hour
)

// GitHubRelease represents a GitHub release.
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// CheckResult holds the result of a version check.
type CheckResult struct {
	CurrentVersion  string
	LatestVersion   string
	UpdateAvailable bool
	ReleaseURL      string
}

// CheckForUpdate checks if a newer version is available.
// Returns nil if check should be skipped (checked recently) or on error.
func CheckForUpdate(currentVersion string) *CheckResult {
	// Skip if running dev version
	if currentVersion == "dev" || currentVersion == "" {
		return nil
	}

	// Check if we should skip (checked recently)
	if shouldSkipCheck() {
		return nil
	}

	// Mark that we checked
	markChecked()

	// Fetch latest release from GitHub
	latest, err := fetchLatestRelease()
	if err != nil {
		return nil // Silently fail - don't block user
	}

	// Compare versions
	latestClean := strings.TrimPrefix(latest.TagName, "v")
	currentClean := strings.TrimPrefix(currentVersion, "v")

	if isNewerVersion(latestClean, currentClean) {
		return &CheckResult{
			CurrentVersion:  currentVersion,
			LatestVersion:   latest.TagName,
			UpdateAvailable: true,
			ReleaseURL:      latest.HTMLURL,
		}
	}

	return nil
}

// PrintUpdateNotice prints a notice if an update is available.
func PrintUpdateNotice(result *CheckResult) {
	if result == nil || !result.UpdateAvailable {
		return
	}

	fmt.Println()
	fmt.Printf("%s A new version of prd-parser is available: %s (you have %s)\n",
		tui.WarningStyle.Render("!"),
		tui.SuccessStyle.Render(result.LatestVersion),
		result.CurrentVersion,
	)
	fmt.Printf("  Update: %s\n", tui.HelpStyle.Render("npm update -g prd-parser"))
	fmt.Printf("  Or: %s\n", tui.HelpStyle.Render("go install github.com/dhabedank/prd-parser@latest"))
	fmt.Println()
}

// fetchLatestRelease fetches the latest release from GitHub.
func fetchLatestRelease() (*GitHubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", GitHubRepo)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

// shouldSkipCheck returns true if we checked recently.
func shouldSkipCheck() bool {
	markerPath := getMarkerPath()
	info, err := os.Stat(markerPath)
	if err != nil {
		return false // No marker, should check
	}

	return time.Since(info.ModTime()) < CheckInterval
}

// markChecked updates the marker file timestamp.
func markChecked() {
	markerPath := getMarkerPath()
	dir := filepath.Dir(markerPath)

	// Create directory if needed
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}

	// Touch the file
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		_ = os.WriteFile(markerPath, []byte{}, 0644)
	} else {
		_ = os.Chtimes(markerPath, time.Now(), time.Now())
	}
}

// getMarkerPath returns the path to the version check marker file.
func getMarkerPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".prd-parser", ".last-update-check")
}

// isNewerVersion returns true if latest is newer than current.
// Simple comparison: splits by dots and compares numerically.
func isNewerVersion(latest, current string) bool {
	latestParts := strings.Split(latest, ".")
	currentParts := strings.Split(current, ".")

	// Compare each part
	for i := 0; i < len(latestParts) && i < len(currentParts); i++ {
		l := parseVersionPart(latestParts[i])
		c := parseVersionPart(currentParts[i])

		if l > c {
			return true
		}
		if l < c {
			return false
		}
	}

	// If all compared parts are equal, longer version is newer
	return len(latestParts) > len(currentParts)
}

// parseVersionPart extracts a number from a version part (e.g., "1" from "1-beta").
func parseVersionPart(s string) int {
	var n int
	_, _ = fmt.Sscanf(s, "%d", &n)
	return n
}
