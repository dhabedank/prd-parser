package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/dhabedank/prd-parser/cmd"
)

// Set by goreleaser ldflags
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	versionStr := fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date)

	rootCmd := &cobra.Command{
		Use:     "prd-parser",
		Short:   "Parse PRDs into structured tasks with LLM guardrails",
		Version: versionStr,
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
