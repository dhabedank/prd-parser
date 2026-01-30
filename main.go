package main

import (
	"fmt"
	"os"

	"github.com/dhabedank/prd-parser/cmd"
	versionpkg "github.com/dhabedank/prd-parser/internal/version"
	"github.com/spf13/cobra"
)

// Set by goreleaser ldflags
var (
	versionNum = "dev"      // Just the version number (e.g., "0.4.0")
	version    = "dev"      // Full version string
	commit     = "none"
	date       = "unknown"
)

func main() {
	versionStr := fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date)

	// Check for first run
	if versionpkg.IsFirstRun() {
		versionpkg.PrintFirstRunNotice()
	}

	// Check for updates (runs in background, cached for 24h)
	updateResult := versionpkg.CheckForUpdate(versionNum)

	rootCmd := &cobra.Command{
		Use:     "prd-parser",
		Short:   "Parse PRDs into structured tasks with LLM guardrails",
		Version: versionStr,
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			// Show update notice after command completes
			versionpkg.PrintUpdateNotice(updateResult)
		},
	}

	// Add commands
	rootCmd.AddCommand(cmd.ParseCmd)
	rootCmd.AddCommand(cmd.RefineCmd)
	rootCmd.AddCommand(cmd.SetupCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
