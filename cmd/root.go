package cmd

import (
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
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format (for agent/machine consumption)")
}
