package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/dhabedank/prd-parser/cmd"
)

var version = "0.1.0"

func main() {
	rootCmd := &cobra.Command{
		Use:     "prd-parser",
		Short:   "Parse PRDs into structured tasks with LLM guardrails",
		Version: version,
	}

	// Add commands
	rootCmd.AddCommand(cmd.ParseCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
