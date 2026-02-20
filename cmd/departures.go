package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/glundgren93/sl-cli/internal/api"
	"github.com/glundgren93/sl-cli/internal/format"
	"github.com/glundgren93/sl-cli/internal/model"
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

Also fetches relevant service deviations and shows them inline.

Examples:
  sl departures --site 9530                                  # By site ID
  sl departures --stop "Medborgarplatsen"                    # By stop name
  sl departures --address "Magnus Ladulåsgatan 7" --line 55  # By address + line
  sl departures --address "Drottninggatan 45" --mode TRAIN   # Nearest train
  sl departures --address "Stureplan" --mode METRO           # Nearest metro
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

	return fetchAndPrintDepartures(ctx, client, siteID, "", 0)
}

func runDeparturesByAddress(ctx context.Context, client *api.Client) error {
	lat, lon, resolvedName, err := geocodeAddress(ctx, client, depAddress)
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

	nearby := api.FindNearestSites(sites, lat, lon, depRadius)
	if len(nearby) == 0 {
		return fmt.Errorf("no stops found within %.0fm of %q", depRadius*1000, depAddress)
	}

	if depLine != "" || depMode != "" {
		return departuresFromNearestMatching(ctx, client, nearby)
	}

	closest := nearby[0]
	if !jsonOutput {
		fmt.Fprintf(os.Stderr, "🚏 Nearest stop: %s (%dm)\n\n", closest.Site.Name, int(closest.DistanceKm*1000))
	}
	return fetchAndPrintDepartures(ctx, client, closest.Site.ID, closest.Site.Name, int(closest.DistanceKm*1000))
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

		if !jsonOutput {
			fmt.Fprintf(os.Stderr, "🚏 %s — %dm away (%s found)\n\n",
				stop.Site.Name, int(stop.DistanceKm*1000), filterDesc)
		}

		// Fetch deviations for the lines we found
		deviations := fetchRelevantDeviations(ctx, client, parsed)

		if depLimit > 0 && len(parsed) > depLimit {
			parsed = parsed[:depLimit]
		}

		if jsonOutput {
			return format.JSON(departureResult{
				Stop:       stop.Site.Name,
				SiteID:     stop.Site.ID,
				DistanceM:  int(stop.DistanceKm * 1000),
				Departures: parsed,
				Deviations: deviations,
			})
		}

		format.Departures(parsed, stop.Site.Name)
		format.DeviationWarnings(deviations)
		return nil
	}

	return fmt.Errorf("%s not found at any stop within %.0fm of %q", filterDesc, depRadius*1000, depAddress)
}

// departureResult is the consistent JSON output for all departures queries.
type departureResult struct {
	Stop       string                  `json:"stop"`
	SiteID     int                     `json:"site_id"`
	DistanceM  int                     `json:"distance_m,omitempty"`
	Departures []model.ParsedDeparture `json:"departures"`
	Deviations []deviationSummary      `json:"deviations,omitempty"`
}

type deviationSummary struct {
	Line    string `json:"line,omitempty"`
	Header  string `json:"header"`
	Details string `json:"details,omitempty"`
	Scope   string `json:"scope,omitempty"`
}

func fetchAndPrintDepartures(ctx context.Context, client *api.Client, siteID int, stopName string, distanceM int) error {
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

	// Fetch deviations for the lines we found
	deviations := fetchRelevantDeviations(ctx, client, parsed)

	if depLimit > 0 && len(parsed) > depLimit {
		parsed = parsed[:depLimit]
	}

	if stopName == "" {
		stopName = fmt.Sprintf("Site %d", siteID)
	}
	if len(parsed) > 0 {
		stopName = parsed[0].StopArea
	}

	if jsonOutput {
		return format.JSON(departureResult{
			Stop:       stopName,
			SiteID:     siteID,
			DistanceM:  distanceM,
			Departures: parsed,
			Deviations: deviations,
		})
	}

	format.Departures(parsed, stopName)
	format.DeviationWarnings(deviations)
	return nil
}

// fetchRelevantDeviations fetches deviations for lines present in the departures.
func fetchRelevantDeviations(ctx context.Context, client *api.Client, deps []model.ParsedDeparture) []deviationSummary {
	// Collect unique line IDs
	lineSet := make(map[string]bool)
	for _, d := range deps {
		if d.Line != "" {
			lineSet[d.Line] = true
		}
	}

	if len(lineSet) == 0 {
		return nil
	}

	// Fetch deviations for these transport modes
	modeSet := make(map[string]bool)
	for _, d := range deps {
		if d.TransportMode != "" {
			modeSet[d.TransportMode] = true
		}
	}
	var modes []string
	for m := range modeSet {
		modes = append(modes, m)
	}

	devs, err := client.GetDeviations(ctx, api.DeviationOptions{
		TransportModes: modes,
	})
	if err != nil {
		return nil // Don't fail departures if deviations fail
	}

	// Filter to only deviations affecting our lines
	var results []deviationSummary
	for _, dev := range devs {
		if dev.Scope == nil {
			continue
		}
		for _, line := range dev.Scope.Lines {
			if lineSet[line.Designation] {
				for _, msg := range dev.MessageVariants {
					if msg.Language == "en" || (msg.Language == "sv" && len(dev.MessageVariants) == 1) {
						results = append(results, deviationSummary{
							Line:    line.Designation,
							Header:  msg.Header,
							Details: truncate(msg.Details, 150),
							Scope:   msg.ScopeAlias,
						})
						break
					}
				}
				break // One summary per deviation
			}
		}
	}

	return results
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
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
	sites, err := client.GetSitesCached(ctx)
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
