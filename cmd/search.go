package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/glundgren/sl-cli/internal/api"
	"github.com/glundgren/sl-cli/internal/format"
	"github.com/spf13/cobra"
)

var searchLimit int

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for stops by name",
	Long: `Search for stops/stations by name. Returns matching sites with their IDs.

Examples:
  sl search Medborgarplatsen
  sl search "Stockholm City"
  sl search Slussen --json`,
	Aliases: []string{"find", "s"},
	Args:    cobra.MinimumNArgs(1),
	RunE:    runSearch,
}

func init() {
	searchCmd.Flags().IntVar(&searchLimit, "limit", 20, "Max results")
	rootCmd.AddCommand(searchCmd)
}

type siteResult struct {
	ID   int     `json:"id"`
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
}

func runSearch(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	client := api.NewClient()
	query := strings.Join(args, " ")

	sites, err := client.GetSitesCached(ctx)
	if err != nil {
		return fmt.Errorf("fetching sites: %w", err)
	}

	queryLower := strings.ToLower(query)
	seen := make(map[int]bool)
	var results []siteResult

	for _, s := range sites {
		if seen[s.ID] {
			continue
		}

		matched := strings.Contains(strings.ToLower(s.Name), queryLower)
		if !matched {
			for _, alias := range s.Aliases {
				if strings.Contains(strings.ToLower(alias), queryLower) {
					matched = true
					break
				}
			}
		}

		if matched {
			seen[s.ID] = true
			results = append(results, siteResult{
				ID:   s.ID,
				Name: s.Name,
				Lat:  s.Lat,
				Lon:  s.Lon,
			})
		}
	}

	if searchLimit > 0 && len(results) > searchLimit {
		results = results[:searchLimit]
	}

	if jsonOutput {
		return format.JSON(results)
	}

	if len(results) == 0 {
		fmt.Printf("No stops found matching %q\n", query)
		return nil
	}

	fmt.Printf("Found %d stop(s) matching %q\n", len(results), query)
	fmt.Println(strings.Repeat("─", 60))
	for i, s := range results {
		fmt.Printf("  %d. %-35s (id:%d)\n", i+1, s.Name, s.ID)
	}
	fmt.Println()
	return nil
}
