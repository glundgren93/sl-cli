package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/glundgren/sl-cli/internal/api"
	"github.com/glundgren/sl-cli/internal/format"
	"github.com/spf13/cobra"
)

var (
	tripFrom       string
	tripTo         string
	tripNumTrips   int
	tripLang       string
	tripMaxChanges int
	tripRouteType  string
)

var tripCmd = &cobra.Command{
	Use:   "trip",
	Short: "Plan a journey between two locations",
	Long: `Plan a trip from A to B using SL's journey planner.

Examples:
  sl trip --from "Medborgarplatsen" --to "T-Centralen"
  sl trip --from "Magnus Ladulåsgatan" --to "Kungsträdgården" --results 3
  sl trip --from 9191 --to 1080 --json`,
	Aliases: []string{"plan", "route"},
	RunE:    runTrip,
}

func init() {
	tripCmd.Flags().StringVar(&tripFrom, "from", "", "Origin (stop name, stop ID, or address)")
	tripCmd.Flags().StringVar(&tripTo, "to", "", "Destination (stop name, stop ID, or address)")
	tripCmd.Flags().IntVar(&tripNumTrips, "results", 3, "Number of trip alternatives")
	tripCmd.Flags().StringVar(&tripLang, "lang", "en", "Language (sv or en)")
	tripCmd.Flags().IntVar(&tripMaxChanges, "max-changes", -1, "Max number of changes (-1 = unlimited)")
	tripCmd.Flags().StringVar(&tripRouteType, "route-type", "", "Route preference: leasttime, leastinterchange, leastwalking")

	tripCmd.MarkFlagRequired("from")
	tripCmd.MarkFlagRequired("to")

	rootCmd.AddCommand(tripCmd)
}

func runTrip(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	client := api.NewClient()

	// Resolve origin
	originID, err := resolveLocation(ctx, client, tripFrom)
	if err != nil {
		return fmt.Errorf("resolving origin: %w", err)
	}

	// Resolve destination
	destID, err := resolveLocation(ctx, client, tripTo)
	if err != nil {
		return fmt.Errorf("resolving destination: %w", err)
	}

	resp, err := client.PlanTrip(ctx, api.TripOptions{
		OriginID:   originID,
		DestID:     destID,
		NumTrips:   tripNumTrips,
		Language:   tripLang,
		MaxChanges: tripMaxChanges,
		RouteType:  tripRouteType,
	})
	if err != nil {
		return fmt.Errorf("planning trip: %w", err)
	}

	// Check for system errors
	for _, msg := range resp.SystemMessages {
		if msg.Type == "error" {
			return fmt.Errorf("journey planner: %s", msg.Text)
		}
	}

	if jsonOutput {
		return format.JSON(resp.Journeys)
	}

	format.Trips(resp.Journeys)
	return nil
}

func resolveLocation(ctx context.Context, client *api.Client, input string) (string, error) {
	// If it looks like a stop-finder ID (long numeric), use directly
	if strings.HasPrefix(input, "9") && len(input) > 8 {
		return input, nil
	}

	// Try stop-finder to resolve the name/address
	locations, err := client.FindStops(ctx, input)
	if err != nil {
		return "", err
	}
	if len(locations) == 0 {
		// Try with the broader filter (stops + addresses + POI)
		return input, nil // Let the API try to resolve it
	}
	return locations[0].ID, nil
}
