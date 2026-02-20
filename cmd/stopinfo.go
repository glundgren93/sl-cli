package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/glundgren93/sl-cli/internal/api"
	"github.com/glundgren93/sl-cli/internal/format"
	"github.com/spf13/cobra"
)

var (
	stopInfoSite    int
	stopInfoStop    string
	stopInfoAddress string
)

var stopInfoCmd = &cobra.Command{
	Use:   "stop-info",
	Short: "Show which lines serve a stop",
	Long: `Show all transit lines that serve a specific stop.

Uses real-time departure data to identify which lines currently operate at the stop.
Results are grouped by transport mode (Metro, Bus, Train, etc).

Examples:
  sl stop-info --site 9530                          # By site ID
  sl stop-info --stop "Medborgarplatsen"             # By stop name
  sl stop-info --address "Magnus Ladulåsgatan 7"     # By address (nearest stop)
  sl stop-info --json                                # JSON for agents`,
	Aliases: []string{"si", "info"},
	RunE:    runStopInfo,
}

func init() {
	stopInfoCmd.Flags().IntVar(&stopInfoSite, "site", 0, "Site ID")
	stopInfoCmd.Flags().StringVar(&stopInfoStop, "stop", "", "Stop name (fuzzy search)")
	stopInfoCmd.Flags().StringVar(&stopInfoAddress, "address", "", "Street address (finds nearest stop)")

	rootCmd.AddCommand(stopInfoCmd)
}

// stopInfoResult is the JSON output for stop-info.
type stopInfoResult struct {
	Stop      string               `json:"stop"`
	SiteID    int                  `json:"site_id"`
	DistanceM int                  `json:"distance_m,omitempty"`
	Lines     []format.StopInfoLine `json:"lines"`
}

func runStopInfo(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	client := api.NewClient()

	siteID := stopInfoSite
	stopName := ""
	distanceM := 0

	// Resolve by address
	if siteID == 0 && stopInfoAddress != "" {
		lat, lon, resolvedName, err := geocodeAddress(ctx, client, stopInfoAddress)
		if err != nil {
			return fmt.Errorf("geocoding address: %w", err)
		}

		if !jsonOutput {
			fmt.Fprintf(os.Stderr, "📍 Resolved: %s (%.4f, %.4f)\n", resolvedName, lat, lon)
		}

		sites, err := client.GetSitesCached(ctx)
		if err != nil {
			return fmt.Errorf("fetching sites: %w", err)
		}

		nearby := api.FindNearestSites(sites, lat, lon, 1.0)
		if len(nearby) == 0 {
			return fmt.Errorf("no stops found near %q", stopInfoAddress)
		}

		siteID = nearby[0].Site.ID
		stopName = nearby[0].Site.Name
		distanceM = int(nearby[0].DistanceKm * 1000)

		if !jsonOutput {
			fmt.Fprintf(os.Stderr, "🚏 Nearest stop: %s (%dm)\n\n", stopName, distanceM)
		}
	}

	// Resolve by stop name
	if siteID == 0 {
		name := stopInfoStop
		if name == "" && len(args) > 0 {
			name = strings.Join(args, " ")
		}
		if name == "" {
			return fmt.Errorf("provide --site, --stop, or --address")
		}

		resolved, err := resolveSiteID(ctx, client, name)
		if err != nil {
			return err
		}
		siteID = resolved
	}

	// Fetch departures (all modes, no line filter)
	resp, err := client.GetDepartures(ctx, api.DepartureOptions{
		SiteID: siteID,
	})
	if err != nil {
		return fmt.Errorf("fetching departures: %w", err)
	}

	parsed := api.ParseDepartures(resp.Departures)

	// Group by line
	type lineKey struct {
		designation   string
		transportMode string
		groupOfLines  string
	}
	lineMap := make(map[lineKey]map[string]bool)
	var lineOrder []lineKey

	for _, d := range parsed {
		key := lineKey{d.Line, d.TransportMode, d.GroupOfLines}
		if _, exists := lineMap[key]; !exists {
			lineMap[key] = make(map[string]bool)
			lineOrder = append(lineOrder, key)
		}
		if d.Destination != "" {
			lineMap[key][d.Destination] = true
		}
	}

	// Build result
	lines := []format.StopInfoLine{}
	for _, key := range lineOrder {
		dests := lineMap[key]
		var destList []string
		for d := range dests {
			destList = append(destList, d)
		}
		lines = append(lines, format.StopInfoLine{
			Designation:   key.designation,
			TransportMode: key.transportMode,
			GroupOfLines:  key.groupOfLines,
			Destinations:  destList,
		})
	}

	if stopName == "" {
		stopName = fmt.Sprintf("Site %d", siteID)
		if len(parsed) > 0 {
			stopName = parsed[0].StopArea
		}
	}

	if jsonOutput {
		return format.JSON(stopInfoResult{
			Stop:      stopName,
			SiteID:    siteID,
			DistanceM: distanceM,
			Lines:     lines,
		})
	}

	format.StopInfo(stopName, siteID, lines)
	return nil
}
