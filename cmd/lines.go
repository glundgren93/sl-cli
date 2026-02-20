package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/glundgren93/sl-cli/internal/api"
	"github.com/glundgren93/sl-cli/internal/format"
	"github.com/spf13/cobra"
)

var linesMode string

var linesCmd = &cobra.Command{
	Use:   "lines",
	Short: "List all transit lines",
	Long: `List all transit lines in the SL network, optionally filtered by transport mode.

Examples:
  sl lines                    # All lines
  sl lines --mode BUS         # Bus lines only
  sl lines --mode METRO       # Metro lines only
  sl lines --json             # JSON output`,
	Aliases: []string{"line", "l"},
	RunE:    runLines,
}

func init() {
	linesCmd.Flags().StringVar(&linesMode, "mode", "", "Filter by transport mode: BUS, METRO, TRAIN, TRAM, SHIP")
	rootCmd.AddCommand(linesCmd)
}

func runLines(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	client := api.NewClient()

	lines, err := client.GetLines(ctx)
	if err != nil {
		return fmt.Errorf("fetching lines: %w", err)
	}

	if linesMode != "" {
		mode := strings.ToUpper(linesMode)
		n := 0
		for _, l := range lines {
			if strings.EqualFold(l.TransportMode, mode) {
				lines[n] = l
				n++
			}
		}
		lines = lines[:n]
	}

	if jsonOutput {
		return format.JSON(lines)
	}

	format.Lines(lines)
	return nil
}
