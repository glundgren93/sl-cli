package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/glundgren/sl-cli/internal/api"
	"github.com/glundgren/sl-cli/internal/format"
	"github.com/spf13/cobra"
)

var (
	devLines  string
	devSites  string
	devModes  string
	devFuture bool
)

var deviationsCmd = &cobra.Command{
	Use:   "deviations",
	Short: "Check service deviations and disruptions",
	Long: `Check current service disruptions across the SL network.

Examples:
  sl deviations                                # All current deviations
  sl deviations --mode METRO                   # Metro only
  sl deviations --line 55                      # Line 55 only
  sl deviations --future                       # Include planned deviations
  sl deviations --json                         # JSON output`,
	Aliases: []string{"dev", "status"},
	RunE:    runDeviations,
}

func init() {
	deviationsCmd.Flags().StringVar(&devLines, "line", "", "Filter by line ID(s), comma-separated")
	deviationsCmd.Flags().StringVar(&devSites, "site", "", "Filter by site ID(s), comma-separated")
	deviationsCmd.Flags().StringVar(&devModes, "mode", "", "Filter by transport mode(s): BUS,METRO,TRAIN,TRAM,SHIP")
	deviationsCmd.Flags().BoolVar(&devFuture, "future", false, "Include future/planned deviations")

	rootCmd.AddCommand(deviationsCmd)
}

func runDeviations(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	client := api.NewClient()

	opts := api.DeviationOptions{
		Future: devFuture,
	}

	if devLines != "" {
		for _, s := range strings.Split(devLines, ",") {
			if id, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
				opts.LineIDs = append(opts.LineIDs, id)
			}
		}
	}
	if devSites != "" {
		for _, s := range strings.Split(devSites, ",") {
			if id, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
				opts.SiteIDs = append(opts.SiteIDs, id)
			}
		}
	}
	if devModes != "" {
		for _, m := range strings.Split(devModes, ",") {
			opts.TransportModes = append(opts.TransportModes, strings.TrimSpace(m))
		}
	}

	devs, err := client.GetDeviations(ctx, opts)
	if err != nil {
		return fmt.Errorf("fetching deviations: %w", err)
	}

	if jsonOutput {
		return format.JSON(devs)
	}

	format.Deviations(devs)
	return nil
}
