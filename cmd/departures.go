package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/glundgren/sl-cli/internal/api"
	"github.com/glundgren/sl-cli/internal/format"
	"github.com/spf13/cobra"
)

var (
	depSiteID    int
	depStopName  string
	depLine      string
	depMode      string
	depDirection int
	depLimit     int
)

var departuresCmd = &cobra.Command{
	Use:   "departures",
	Short: "Get real-time departures from a stop",
	Long: `Get real-time departures from a stop. Specify the stop by ID or name.

Examples:
  sl departures --site 9530                     # Stockholms södra by ID
  sl departures --stop "Medborgarplatsen"        # Search by name
  sl departures --site 9530 --line 55            # Only bus 55
  sl departures --site 9530 --mode TRAIN         # Only trains
  sl departures --site 9530 --json               # JSON output for agents`,
	Aliases: []string{"dep", "d"},
	RunE:    runDepartures,
}

func init() {
	departuresCmd.Flags().IntVar(&depSiteID, "site", 0, "Site ID (use 'sl search' to find IDs)")
	departuresCmd.Flags().StringVar(&depStopName, "stop", "", "Stop name (fuzzy search)")
	departuresCmd.Flags().StringVar(&depLine, "line", "", "Filter by line designation (e.g. 55, 18)")
	departuresCmd.Flags().StringVar(&depMode, "mode", "", "Filter by transport mode (BUS, METRO, TRAIN, TRAM, SHIP)")
	departuresCmd.Flags().IntVar(&depDirection, "direction", 0, "Filter by direction (1 or 2)")
	departuresCmd.Flags().IntVar(&depLimit, "limit", 20, "Max departures to show")

	rootCmd.AddCommand(departuresCmd)
}

func runDepartures(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	client := api.NewClient()

	siteID := depSiteID

	// If no site ID, try resolving from stop name or args
	if siteID == 0 && depStopName == "" && len(args) > 0 {
		depStopName = strings.Join(args, " ")
	}

	if siteID == 0 {
		if depStopName == "" {
			return fmt.Errorf("provide --site ID or --stop name (use 'sl search <name>' to find stops)")
		}

		// Try parsing as number first
		if id, err := strconv.Atoi(depStopName); err == nil {
			siteID = id
		} else {
			// Resolve name to site ID
			resolved, err := resolveSiteID(ctx, client, depStopName)
			if err != nil {
				return err
			}
			siteID = resolved
		}
	}

	resp, err := client.GetDepartures(ctx, api.DepartureOptions{
		SiteID:        siteID,
		TransportMode: depMode,
		Line:          depLine,
		Direction:     depDirection,
	})
	if err != nil {
		return fmt.Errorf("fetching departures: %w", err)
	}

	parsed := api.ParseDepartures(resp.Departures)

	// Apply additional filters
	if depMode != "" {
		parsed = api.FilterByTransportMode(parsed, depMode)
	}

	// Limit results
	if depLimit > 0 && len(parsed) > depLimit {
		parsed = parsed[:depLimit]
	}

	if jsonOutput {
		return format.JSON(parsed)
	}

	// Get stop name for display
	stopName := depStopName
	if stopName == "" {
		stopName = fmt.Sprintf("Site %d", siteID)
	}
	if len(parsed) > 0 {
		stopName = parsed[0].StopArea
	}
	format.Departures(parsed, stopName)
	return nil
}

func resolveSiteID(ctx context.Context, client *api.Client, name string) (int, error) {
	// First try the Transport API sites list
	sites, err := client.GetSites(ctx)
	if err != nil {
		return 0, fmt.Errorf("fetching sites: %w", err)
	}

	nameLower := strings.ToLower(name)
	var matches []struct {
		id   int
		name string
	}

	for _, s := range sites {
		sNameLower := strings.ToLower(s.Name)
		// Exact match
		if sNameLower == nameLower {
			return s.ID, nil
		}
		// Contains match
		if strings.Contains(sNameLower, nameLower) {
			matches = append(matches, struct {
				id   int
				name string
			}{s.ID, s.Name})
		}
	}

	if len(matches) == 1 {
		return matches[0].id, nil
	}
	if len(matches) > 1 {
		fmt.Fprintf(os.Stderr, "Multiple matches found:\n")
		for _, m := range matches {
			fmt.Fprintf(os.Stderr, "  %s (id:%d)\n", m.name, m.id)
		}
		fmt.Fprintf(os.Stderr, "\nUse --site <id> to specify.\n")
		return 0, fmt.Errorf("ambiguous stop name %q — %d matches", name, len(matches))
	}

	return 0, fmt.Errorf("no stop found matching %q", name)
}
