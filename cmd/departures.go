package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/glundgren/sl-cli/internal/api"
	"github.com/glundgren/sl-cli/internal/format"
	"github.com/glundgren/sl-cli/internal/model"
	"github.com/spf13/cobra"
)

var (
	depSiteID    int
	depStopName  string
	depAddress   string
	depLine      string
	depMode      string
	depDirection int
	depLimit     int
	depRadius    float64
)

var departuresCmd = &cobra.Command{
	Use:   "departures",
	Short: "Get real-time departures from a stop",
	Long: `Get real-time departures from a stop. Specify by site ID, stop name, or address.

When using --address, the CLI geocodes the address, finds nearby stops,
and returns departures from the closest stop(s) that serve the requested line/mode.

Examples:
  sl departures --site 9530                                  # By site ID
  sl departures --stop "Medborgarplatsen"                    # By stop name
  sl departures --address "Magnus Ladulåsgatan 7" --line 55  # By address + line
  sl departures --address "Magnus Ladulåsgatan 7" --mode TRAIN  # Nearest train
  sl departures --address "Magnus Ladulåsgatan 7"            # All nearby departures
  sl departures --site 9530 --json                           # JSON for agents`,
	Aliases: []string{"dep", "d"},
	RunE:    runDepartures,
}

func init() {
	departuresCmd.Flags().IntVar(&depSiteID, "site", 0, "Site ID (use 'sl search' to find IDs)")
	departuresCmd.Flags().StringVar(&depStopName, "stop", "", "Stop name (fuzzy search)")
	departuresCmd.Flags().StringVar(&depAddress, "address", "", "Street address (geocodes and finds nearest stops)")
	departuresCmd.Flags().StringVar(&depLine, "line", "", "Filter by line designation (e.g. 55, 18)")
	departuresCmd.Flags().StringVar(&depMode, "mode", "", "Filter by transport mode (BUS, METRO, TRAIN, TRAM, SHIP)")
	departuresCmd.Flags().IntVar(&depDirection, "direction", 0, "Filter by direction (1 or 2)")
	departuresCmd.Flags().IntVar(&depLimit, "limit", 20, "Max departures to show")
	departuresCmd.Flags().Float64Var(&depRadius, "radius", 1.0, "Search radius in km when using --address")

	rootCmd.AddCommand(departuresCmd)
}

func runDepartures(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	client := api.NewClient()

	// Address mode: geocode → find nearby → get departures from matching stops
	if depAddress != "" {
		return runDeparturesByAddress(ctx, client)
	}

	siteID := depSiteID

	if siteID == 0 && depStopName == "" && len(args) > 0 {
		depStopName = strings.Join(args, " ")
	}

	if siteID == 0 {
		if depStopName == "" {
			return fmt.Errorf("provide --site, --stop, or --address (use 'sl search <name>' to find stops)")
		}

		if id, err := strconv.Atoi(depStopName); err == nil {
			siteID = id
		} else {
			resolved, err := resolveSiteID(ctx, client, depStopName)
			if err != nil {
				return err
			}
			siteID = resolved
		}
	}

	return fetchAndPrintDepartures(ctx, client, siteID, "")
}

func runDeparturesByAddress(ctx context.Context, client *api.Client) error {
	// Step 1: Geocode
	lat, lon, resolvedName, err := geocodeAddress(ctx, client, depAddress)
	if err != nil {
		return fmt.Errorf("geocoding address: %w", err)
	}

	if !jsonOutput {
		fmt.Fprintf(os.Stderr, "📍 Resolved: %s (%.4f, %.4f)\n", resolvedName, lat, lon)
	}

	// Step 2: Find nearby stops
	sites, err := client.GetSites(ctx)
	if err != nil {
		return fmt.Errorf("fetching sites: %w", err)
	}

	nearby := api.FindNearestSites(sites, lat, lon, depRadius)
	if len(nearby) == 0 {
		return fmt.Errorf("no stops found within %.0fm of %q", depRadius*1000, depAddress)
	}

	// Step 3: If a line or mode filter is set, scan stops to find the nearest one with results
	if depLine != "" || depMode != "" {
		return departuresFromNearestMatching(ctx, client, nearby)
	}

	// No filters: just use the closest stop
	closest := nearby[0]
	if !jsonOutput {
		fmt.Fprintf(os.Stderr, "🚏 Nearest stop: %s (%dm)\n\n", closest.Site.Name, int(closest.DistanceKm*1000))
	}
	return fetchAndPrintDepartures(ctx, client, closest.Site.ID, closest.Site.Name)
}

func departuresFromNearestMatching(ctx context.Context, client *api.Client, nearby []api.SiteWithDistance) error {
	maxScan := 15
	if len(nearby) < maxScan {
		maxScan = len(nearby)
	}

	filterDesc := depLine
	if filterDesc == "" {
		filterDesc = depMode
	}

	for _, stop := range nearby[:maxScan] {
		resp, err := client.GetDepartures(ctx, api.DepartureOptions{
			SiteID:        stop.Site.ID,
			TransportMode: depMode,
			Line:          depLine,
			Direction:     depDirection,
		})
		if err != nil {
			continue
		}

		if len(resp.Departures) == 0 {
			continue
		}

		parsed := api.ParseDepartures(resp.Departures)
		if depMode != "" {
			parsed = api.FilterByTransportMode(parsed, depMode)
		}

		if len(parsed) == 0 {
			continue
		}

		// Found a stop with matching departures
		if !jsonOutput {
			fmt.Fprintf(os.Stderr, "🚏 %s — %dm away (%s found)\n\n",
				stop.Site.Name, int(stop.DistanceKm*1000), filterDesc)
		}

		if depLimit > 0 && len(parsed) > depLimit {
			parsed = parsed[:depLimit]
		}

		if jsonOutput {
			type addressResult struct {
				Address    string                  `json:"address"`
				Stop       string                  `json:"stop"`
				SiteID     int                     `json:"site_id"`
				DistanceM  int                     `json:"distance_m"`
				Departures []model.ParsedDeparture `json:"departures"`
			}
			return format.JSON(addressResult{
				Address:    depAddress,
				Stop:       stop.Site.Name,
				SiteID:     stop.Site.ID,
				DistanceM:  int(stop.DistanceKm * 1000),
				Departures: parsed,
			})
		}

		format.Departures(parsed, stop.Site.Name)
		return nil
	}

	return fmt.Errorf("%s not found at any stop within %.0fm of %q", filterDesc, depRadius*1000, depAddress)
}

func fetchAndPrintDepartures(ctx context.Context, client *api.Client, siteID int, stopName string) error {
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
	if depMode != "" {
		parsed = api.FilterByTransportMode(parsed, depMode)
	}
	if depLimit > 0 && len(parsed) > depLimit {
		parsed = parsed[:depLimit]
	}

	if jsonOutput {
		return format.JSON(parsed)
	}

	if stopName == "" {
		stopName = fmt.Sprintf("Site %d", siteID)
	}
	if len(parsed) > 0 {
		stopName = parsed[0].StopArea
	}
	format.Departures(parsed, stopName)
	return nil
}

func geocodeAddress(ctx context.Context, client *api.Client, address string) (lat, lon float64, name string, err error) {
	locations, err := client.FindAddress(ctx, address)
	if err != nil {
		return 0, 0, "", err
	}
	if len(locations) == 0 {
		return 0, 0, "", fmt.Errorf("no location found for %q", address)
	}
	loc := locations[0]
	return loc.Coord[0], loc.Coord[1], loc.Name, nil
}

func resolveSiteID(ctx context.Context, client *api.Client, name string) (int, error) {
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
		if sNameLower == nameLower {
			return s.ID, nil
		}
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
