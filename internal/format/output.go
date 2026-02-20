package format

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/glundgren/sl-cli/internal/api"
	"github.com/glundgren/sl-cli/internal/model"
)

var (
	bold    = color.New(color.Bold)
	green   = color.New(color.FgGreen, color.Bold)
	yellow  = color.New(color.FgYellow)
	red     = color.New(color.FgRed)
	cyan    = color.New(color.FgCyan)
	dim     = color.New(color.Faint)
	busIcon = "🚌"
	metroIcon = "🚇"
	trainIcon = "🚆"
	tramIcon  = "🚋"
	shipIcon  = "⛴️"
)

func modeIcon(mode string) string {
	switch strings.ToUpper(mode) {
	case "BUS":
		return busIcon
	case "METRO":
		return metroIcon
	case "TRAIN":
		return trainIcon
	case "TRAM":
		return tramIcon
	case "SHIP", "FERRY":
		return shipIcon
	default:
		return "🚏"
	}
}

// JSON outputs any value as formatted JSON.
func JSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// Departures prints departures in human-readable format.
func Departures(deps []model.ParsedDeparture, stopName string) {
	if len(deps) == 0 {
		dim.Println("No departures found.")
		return
	}

	bold.Printf("📍 %s\n", stopName)
	fmt.Println(strings.Repeat("─", 60))

	// Group by transport mode + line
	type lineKey struct {
		mode string
		line string
	}
	groups := make(map[lineKey][]model.ParsedDeparture)
	var order []lineKey

	for _, d := range deps {
		key := lineKey{d.TransportMode, d.Line}
		if _, exists := groups[key]; !exists {
			order = append(order, key)
		}
		groups[key] = append(groups[key], d)
	}

	for _, key := range order {
		lineDeps := groups[key]
		icon := modeIcon(key.mode)
		bold.Printf("\n%s Line %s", icon, key.line)
		if lineDeps[0].GroupOfLines != "" {
			dim.Printf(" (%s)", lineDeps[0].GroupOfLines)
		}
		fmt.Println()

		for _, d := range lineDeps {
			timeStr := formatTime(d)
			stateStr := formatState(d.State)
			platform := ""
			if d.Platform != "" {
				platform = dim.Sprintf(" [plat %s]", d.Platform)
			}
			fmt.Printf("  → %-25s %s %s%s\n", d.Destination, timeStr, stateStr, platform)
		}
	}
	fmt.Println()
}

func formatTime(d model.ParsedDeparture) string {
	if d.Display == "Nu" || d.MinutesLeft == 0 {
		return green.Sprint("NOW")
	}
	if d.MinutesLeft <= 5 {
		return yellow.Sprintf("%d min", d.MinutesLeft)
	}
	return cyan.Sprintf("%d min", d.MinutesLeft)
}

func formatState(state string) string {
	switch state {
	case "ATSTOP":
		return green.Sprint("● at stop")
	case "EXPECTED":
		return ""
	case "CANCELLED":
		return red.Sprint("✗ cancelled")
	default:
		return dim.Sprint(state)
	}
}

// NearbyStops prints nearby stops in human-readable format.
func NearbyStops(stops []api.SiteWithDistance) {
	if len(stops) == 0 {
		dim.Println("No stops found nearby.")
		return
	}

	bold.Println("📍 Nearby stops")
	fmt.Println(strings.Repeat("─", 60))

	for i, s := range stops {
		distStr := fmt.Sprintf("%dm", int(s.DistanceKm*1000))
		bold.Printf("  %d. ", i+1)
		fmt.Printf("%-35s ", s.Site.Name)
		cyan.Printf("%-8s", distStr)
		dim.Printf(" (id:%d)\n", s.Site.ID)
	}
	fmt.Println()
}

// Deviations prints deviations in human-readable format.
func Deviations(devs []model.Deviation) {
	if len(devs) == 0 {
		green.Println("✓ No deviations found.")
		return
	}

	bold.Printf("⚠️  %d deviation(s)\n", len(devs))
	fmt.Println(strings.Repeat("─", 60))

	for _, d := range devs {
		for _, msg := range d.MessageVariants {
			if msg.Language != "sv" && msg.Language != "en" {
				continue
			}
			yellow.Printf("\n  %s\n", msg.Header)
			if msg.ScopeAlias != "" {
				dim.Printf("  Affects: %s\n", msg.ScopeAlias)
			}
			if msg.Details != "" {
				// Truncate long details
				details := msg.Details
				if len(details) > 200 {
					details = details[:200] + "..."
				}
				fmt.Printf("  %s\n", details)
			}
		}
	}
	fmt.Println()
}

// Trips prints journey plans in human-readable format.
func Trips(journeys []model.JourneyTrip) {
	if len(journeys) == 0 {
		dim.Println("No routes found.")
		return
	}

	bold.Printf("🗺️  %d route(s) found\n", len(journeys))
	fmt.Println(strings.Repeat("─", 60))

	for i, j := range journeys {
		durationMin := j.TripRtDuration / 60
		if durationMin == 0 {
			durationMin = j.TripDuration / 60
		}
		bold.Printf("\nRoute %d", i+1)
		cyan.Printf(" — %d min", durationMin)
		if j.Interchanges > 0 {
			dim.Printf(" (%d change(s))", j.Interchanges)
		}
		fmt.Println()

		for _, leg := range j.Legs {
			origin := "?"
			dest := "?"
			depTime := ""
			arrTime := ""

			if leg.Origin != nil {
				origin = leg.Origin.Name
				if t := leg.Origin.DepartureTimeEstimated; t != "" {
					depTime = formatISOTime(t)
				} else if t := leg.Origin.DepartureTimePlanned; t != "" {
					depTime = formatISOTime(t)
				}
			}
			if leg.Destination != nil {
				dest = leg.Destination.Name
				if t := leg.Destination.ArrivalTimeEstimated; t != "" {
					arrTime = formatISOTime(t)
				} else if t := leg.Destination.ArrivalTimePlanned; t != "" {
					arrTime = formatISOTime(t)
				}
			}

			if leg.Transport != nil && leg.Transport.Name != "" {
				icon := "🚏"
				if leg.Transport.Product != nil {
					switch {
					case strings.Contains(strings.ToLower(leg.Transport.Product.CatOutL), "metro"):
						icon = metroIcon
					case strings.Contains(strings.ToLower(leg.Transport.Product.CatOutL), "bus"):
						icon = busIcon
					case strings.Contains(strings.ToLower(leg.Transport.Product.CatOutL), "train"),
						strings.Contains(strings.ToLower(leg.Transport.Product.CatOutL), "pendel"):
						icon = trainIcon
					case strings.Contains(strings.ToLower(leg.Transport.Product.CatOutL), "tram"):
						icon = tramIcon
					}
				}
				fmt.Printf("  %s %s %s → %s (%s – %s)\n", icon, leg.Transport.Name, origin, dest, depTime, arrTime)
			} else {
				fmt.Printf("  🚶 Walk %s → %s (%d min)\n", origin, dest, leg.Duration/60)
			}
		}
	}
	fmt.Println()
}

func formatISOTime(isoTime string) string {
	// Parse ISO 8601 with timezone offset
	layouts := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05+01:00",
	}
	for _, layout := range layouts {
		if t, err := fmt.Sscanf(isoTime, layout); err == nil && t > 0 {
			break
		}
	}
	// Just extract HH:MM
	if len(isoTime) >= 16 {
		return isoTime[11:16]
	}
	return isoTime
}

// Sites prints sites in human-readable format (for search results).
func Sites(sites []model.Site) {
	if len(sites) == 0 {
		dim.Println("No sites found.")
		return
	}

	bold.Printf("Found %d site(s)\n", len(sites))
	fmt.Println(strings.Repeat("─", 60))

	for i, s := range sites {
		bold.Printf("  %d. ", i+1)
		fmt.Printf("%-35s", s.Name)
		dim.Printf(" (id:%d, lat:%.4f, lon:%.4f)\n", s.ID, s.Lat, s.Lon)
	}
	fmt.Println()
}

// Lines prints lines in human-readable format.
func Lines(lines []model.Line) {
	if len(lines) == 0 {
		dim.Println("No lines found.")
		return
	}

	// Group by transport mode
	groups := make(map[string][]model.Line)
	var modes []string
	for _, l := range lines {
		if _, exists := groups[l.TransportMode]; !exists {
			modes = append(modes, l.TransportMode)
		}
		groups[l.TransportMode] = append(groups[l.TransportMode], l)
	}

	bold.Printf("Found %d line(s)\n", len(lines))
	fmt.Println(strings.Repeat("─", 60))

	for _, mode := range modes {
		icon := modeIcon(mode)
		bold.Printf("\n%s %s\n", icon, mode)
		modeLines := groups[mode]
		lineDesigs := make([]string, 0, len(modeLines))
		for _, l := range modeLines {
			lineDesigs = append(lineDesigs, l.Designation)
		}
		fmt.Printf("  %s\n", strings.Join(lineDesigs, ", "))
	}
	fmt.Println()
}
