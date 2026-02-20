package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	jsonOutput bool
)

var rootCmd = &cobra.Command{
	Use:   "sl",
	Short: "Stockholm public transport CLI",
	Long: `sl-cli — A command-line interface for Stockholm's public transport (SL).

Query real-time departures, plan journeys, find nearby stops, and check
service deviations. Designed for both humans and AI agents.

No API key required. Data sourced from SL via Trafiklab.`,
	SilenceErrors: true,
}

// Execute runs the root command and handles errors.
func Execute() error {
	err := rootCmd.Execute()
	if err != nil {
		if jsonOutput {
			enc := json.NewEncoder(os.Stderr)
			enc.SetEscapeHTML(false)
			enc.Encode(map[string]string{"error": err.Error()})
		} else {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		}
	}
	return err
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format (for agent/machine consumption)")

	// Silence usage on RunE errors (not flag errors).
	// Cobra shows usage by default on all errors; we only want it for bad flags/args.
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		// If we got past flag parsing, silence usage for runtime errors
		cmd.SilenceUsage = true
	}
}
