package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/glundgren93/sl-cli/internal/api"
	"github.com/glundgren93/sl-cli/internal/model"
	"github.com/glundgren93/sl-cli/internal/format"
	"github.com/spf13/cobra"
)

var (
	nearbyLat       float64
	nearbyLon       float64
	nearbyRadius    float64
	nearbyLimit     int
	nearbyAddr      string
	nearbyShowLines bool
)

var nearbyCmd = &cobra.Command{
	Use:   "nearby",
	Short: "Find stops near a location",
	Long: `Find nearby stops by coordinates or address.

Use --lines to also show which transit lines serve each stop (slower, makes API calls per stop).

Examples:
  sl nearby --lat 59.3121 --lon 18.0643         # By coordinates
  sl nearby --address "Magnus Ladulåsgatan"      # By address
  sl nearby --lat 59.3121 --lon 18.0643 -r 0.3  # 300m radius
  sl nearby --address "Stureplan" --lines        # Show lines per stop
  sl nearby --lat 59.3121 --lon 18.0643 --json   # JSON output`,
	Aliases: []string{"near", "n"},
	RunE:    runNearby,
}

func init() {
	nearbyCmd.Flags().Float64Var(&nearbyLat, "lat", 0, "Latitude (WGS84)")
	nearbyCmd.Flags().Float64Var(&nearbyLon, "lon", 0, "Longitude (WGS84)")
	nearbyCmd.Flags().Float64VarP(&nearbyRadius, "radius", "r", 0.5, "Search radius in km (default 0.5)")
	nearbyCmd.Flags().IntVar(&nearbyLimit, "limit", 10, "Max results")
	nearbyCmd.Flags().StringVar(&nearbyAddr, "address", "", "Address to geocode (uses SL stop-finder)")
	nearbyCmd.Flags().BoolVar(&nearbyShowLines, "lines", false, "Show which lines serve each stop (slower)")

	rootCmd.AddCommand(nearbyCmd)
}

func runNearby(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	client := api.NewClient()

	lat, lon := nearbyLat, nearbyLon

	// Try to resolve address
	if lat == 0 && lon == 0 {
		addr := nearbyAddr
		if addr == "" && len(args) > 0 {
			addr = strings.Join(args, " ")
		}
		if addr == "" {
			return fmt.Errorf("provide --lat/--lon coordinates or --address")
		}

		// Try parsing as "lat,lon"
		if parts := strings.Split(addr, ","); len(parts) == 2 {
			if la, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64); err == nil {
				if lo, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil {
					lat, lon = la, lo
				}
			}
		}

		if lat == 0 && lon == 0 {
			locations, err := client.FindAddress(ctx, addr)
			if err != nil {
				return fmt.Errorf("geocoding address: %w", err)
			}
			if len(locations) == 0 {
				return fmt.Errorf("no location found for %q", addr)
			}
			loc := locations[0]
			lat, lon = loc.Coord[0], loc.Coord[1]
			fmt.Fprintf(cmd.ErrOrStderr(), "📍 Resolved: %s (%.4f, %.4f)\n\n", loc.Name, lat, lon)
		}
	}

	sites, err := client.GetSitesCached(ctx)
	if err != nil {
		return fmt.Errorf("fetching sites: %w", err)
	}

	nearby := api.FindNearestSites(sites, lat, lon, nearbyRadius)

	for i := range nearby {
		nearby[i].DistanceM = int(nearby[i].DistanceKm * 1000)
	}

	if nearbyLimit > 0 && len(nearby) > nearbyLimit {
		nearby = nearby[:nearbyLimit]
	}

	if !nearbyShowLines {
		if jsonOutput {
			return format.JSON(nearby)
		}
		format.NearbyStops(nearby)
		return nil
	}

	// Enrich with line info
	results := []format.NearbyStopWithLines{}
	for _, s := range nearby {
		entry := format.NearbyStopWithLines{
			Stop:      s.Site.Name,
			SiteID:    s.Site.ID,
			DistanceM: s.DistanceM,
		}

		resp, err := client.GetDepartures(ctx, api.DepartureOptions{SiteID: s.Site.ID})
		if err != nil {
			results = append(results, entry)
			continue
		}

		parsed := api.ParseDepartures(resp.Departures)
		entry.Lines = extractLines(parsed)
		results = append(results, entry)
	}

	if jsonOutput {
		return format.JSON(results)
	}

	format.NearbyStopsWithLines(results)
	return nil
}

// extractLines groups parsed departures into unique lines with destinations.
func extractLines(parsed []model.ParsedDeparture) []format.StopInfoLine {
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
	return lines
}
