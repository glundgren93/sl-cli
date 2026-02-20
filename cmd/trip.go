package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/glundgren93/sl-cli/internal/api"
	"github.com/glundgren93/sl-cli/internal/format"
	"github.com/glundgren93/sl-cli/internal/model"
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
	Long: `Plan a trip from A to B. Accepts stop names, stop IDs, or street addresses.

Examples:
  sl trip --from "Medborgarplatsen" --to "T-Centralen"
  sl trip --from "Magnus Ladulåsgatan 7" --to "Stureplan"
  sl trip --from "Drottninggatan 45" --to "Arlanda" --results 5
  sl trip --from "Medborgarplatsen" --to "T-Centralen" --json`,
	Aliases: []string{"plan", "route"},
	RunE:    runTrip,
}

func init() {
	tripCmd.Flags().StringVar(&tripFrom, "from", "", "Origin (stop name, address, or stop ID)")
	tripCmd.Flags().StringVar(&tripTo, "to", "", "Destination (stop name, address, or stop ID)")
	tripCmd.Flags().IntVar(&tripNumTrips, "results", 3, "Number of trip alternatives")
	tripCmd.Flags().StringVar(&tripLang, "lang", "en", "Language (sv or en)")
	tripCmd.Flags().IntVar(&tripMaxChanges, "max-changes", -1, "Max number of changes (-1 = unlimited)")
	tripCmd.Flags().StringVar(&tripRouteType, "route-type", "", "Route preference: leasttime, leastinterchange, leastwalking")

	tripCmd.MarkFlagRequired("from")
	tripCmd.MarkFlagRequired("to")

	rootCmd.AddCommand(tripCmd)
}

// tripResult wraps journey results with metadata for JSON output.
type tripResult struct {
	From     string              `json:"from"`
	To       string              `json:"to"`
	Journeys []model.JourneyTrip `json:"journeys"`
}

func runTrip(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	client := api.NewClient()

	originID, originName, err := resolveLocation(ctx, client, tripFrom)
	if err != nil {
		return fmt.Errorf("resolving origin: %w", err)
	}

	destID, destName, err := resolveLocation(ctx, client, tripTo)
	if err != nil {
		return fmt.Errorf("resolving destination: %w", err)
	}

	if !jsonOutput {
		fmt.Fprintf(os.Stderr, "📍 %s → %s\n\n", originName, destName)
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

	for _, msg := range resp.SystemMessages {
		if msg.Type == "error" {
			return fmt.Errorf("journey planner: %s", msg.Text)
		}
	}

	if jsonOutput {
		return format.JSON(tripResult{
			From:     originName,
			To:       destName,
			Journeys: resp.Journeys,
		})
	}

	format.Trips(resp.Journeys)
	return nil
}

// resolveLocation resolves a user input (name, address, or ID) to a journey planner location ID.
func resolveLocation(ctx context.Context, client *api.Client, input string) (id string, name string, err error) {
	// If it looks like a stop-finder ID (long numeric starting with 9), use directly
	if strings.HasPrefix(input, "9") && len(input) > 8 {
		return input, input, nil
	}

	// Try broad search (stops + addresses + POI) so both "Medborgarplatsen" and
	// "Magnus Ladulåsgatan 7" work
	locations, err := client.FindAddress(ctx, input)
	if err != nil {
		return "", "", err
	}

	if len(locations) > 0 {
		loc := locations[0]
		displayName := loc.Name
		if loc.DisassembledName != "" && loc.DisassembledName != loc.Name {
			displayName = loc.DisassembledName
		}
		return loc.ID, displayName, nil
	}

	// Fallback: let the journey planner try to resolve it
	return input, input, nil
}
