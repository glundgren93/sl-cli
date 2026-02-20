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
	bold      = color.New(color.Bold)
	green     = color.New(color.FgGreen, color.Bold)
	yellow    = color.New(color.FgYellow)
	red       = color.New(color.FgRed)
	cyan      = color.New(color.FgCyan)
	dim       = color.New(color.Faint)
	busIcon   = "🚌"
	metroIcon = "🚇"
	trainIcon = "🚆"
	tramIcon  = "🚋"
	shipIcon  = "⛴️"
)

// ModeIcon returns the emoji icon for a transport mode.
func ModeIcon(mode string) string {
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
		icon := ModeIcon(key.mode)
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

// DeviationWarning is a simplified deviation for inline display.
type DeviationWarning struct {
	Line    string
	Header  string
	Details string
	Scope   string
}

// DeviationWarnings prints inline deviation warnings below departures.
func DeviationWarnings(warnings any) {
	// Accept the deviationSummary type from departures command
	type summary struct {
		Line    string `json:"line,omitempty"`
		Header  string `json:"header"`
		Details string `json:"details,omitempty"`
		Scope   string `json:"scope,omitempty"`
	}

	// Use JSON round-trip to convert from the cmd package type
	data, err := json.Marshal(warnings)
	if err != nil {
		return
	}
	var summaries []summary
	if err := json.Unmarshal(data, &summaries); err != nil {
		return
	}

	if len(summaries) == 0 {
		return
	}

	yellow.Printf("⚠️  %d disruption(s) affecting these lines:\n", len(summaries))
	for _, s := range summaries {
		linePrefix := ""
		if s.Line != "" {
			linePrefix = fmt.Sprintf("[Line %s] ", s.Line)
		}
		yellow.Printf("  • %s%s\n", linePrefix, s.Header)
		if s.Details != "" {
			dim.Printf("    %s\n", s.Details)
		}
	}
	fmt.Println()
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
					catLower := strings.ToLower(leg.Transport.Product.CatOutL)
					switch {
					case strings.Contains(catLower, "metro"):
						icon = metroIcon
					case strings.Contains(catLower, "bus"):
						icon = busIcon
					case strings.Contains(catLower, "train"), strings.Contains(catLower, "pendel"):
						icon = trainIcon
					case strings.Contains(catLower, "tram"):
						icon = tramIcon
					}
				}
				fmt.Printf("  %s %s: %s → %s (%s – %s)\n", icon, leg.Transport.Name, origin, dest, depTime, arrTime)
			} else {
				walkMin := leg.Duration / 60
				if walkMin == 0 {
					walkMin = 1
				}
				fmt.Printf("  🚶 Walk: %s → %s (%d min)\n", origin, dest, walkMin)
			}
		}
	}
	fmt.Println()
}

func formatISOTime(isoTime string) string {
	if len(isoTime) >= 16 {
		return isoTime[11:16]
	}
	return isoTime
}

// Lines prints lines in human-readable format.
func Lines(lines []model.Line) {
	if len(lines) == 0 {
		dim.Println("No lines found.")
		return
	}

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
		icon := ModeIcon(mode)
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
