package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/glundgren93/sl-cli/internal/api"
	"github.com/glundgren93/sl-cli/internal/format"
	"github.com/glundgren93/sl-cli/internal/model"
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

Filter by line designation (e.g. 55, 17), transport mode, or site ID.

Examples:
  sl deviations                                # All current deviations
  sl deviations --mode METRO                   # Metro only
  sl deviations --line 55                      # Line 55 only
  sl deviations --line 17,18,19                # Multiple lines
  sl deviations --future                       # Include planned deviations
  sl deviations --json                         # JSON output`,
	Aliases: []string{"dev", "status"},
	RunE:    runDeviations,
}

func init() {
	deviationsCmd.Flags().StringVar(&devLines, "line", "", "Filter by line designation(s), comma-separated (e.g. 55,17)")
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

	// Parse line designations to filter client-side (API uses internal IDs, not designations)
	var lineDesignations []string
	if devLines != "" {
		for _, s := range strings.Split(devLines, ",") {
			lineDesignations = append(lineDesignations, strings.TrimSpace(s))
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

	// Filter by line designation client-side
	if len(lineDesignations) > 0 {
		devs = filterDeviationsByLine(devs, lineDesignations)
	}

	if jsonOutput {
		return format.JSON(devs)
	}

	format.Deviations(devs)
	return nil
}

// filterDeviationsByLine filters deviations to only those affecting the given line designations.
func filterDeviationsByLine(devs []model.Deviation, designations []string) []model.Deviation {
	designSet := make(map[string]bool)
	for _, d := range designations {
		designSet[strings.ToLower(d)] = true
	}

	var filtered []model.Deviation
	for _, dev := range devs {
		if dev.Scope == nil {
			continue
		}
		for _, line := range dev.Scope.Lines {
			if designSet[strings.ToLower(line.Designation)] {
				filtered = append(filtered, dev)
				break
			}
		}
	}
	return filtered
}
