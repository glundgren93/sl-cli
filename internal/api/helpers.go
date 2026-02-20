package api

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/glundgren93/sl-cli/internal/model"
)

const stockholmTZ = "Europe/Stockholm"

// ParseDepartures converts raw departures into agent-friendly parsed departures.
func ParseDepartures(departures []model.Departure) []model.ParsedDeparture {
	loc, _ := time.LoadLocation(stockholmTZ)
	now := time.Now().In(loc)

	var parsed []model.ParsedDeparture
	for _, d := range departures {
		pd := model.ParsedDeparture{
			Destination: d.Destination,
			Direction:   d.Direction,
			Display:     d.Display,
			State:       d.State,
		}

		if d.Line != nil {
			pd.Line = d.Line.Designation
			pd.TransportMode = d.Line.TransportMode
			pd.GroupOfLines = d.Line.GroupOfLines
		}
		if d.StopArea != nil {
			pd.StopArea = d.StopArea.Name
		}
		if d.StopPoint != nil {
			pd.StopPoint = d.StopPoint.Name
			pd.Platform = d.StopPoint.Designation
		}

		// Parse times — SL uses "2006-01-02T15:04:05" (no timezone, local Stockholm time)
		if t, err := time.ParseInLocation("2006-01-02T15:04:05", d.Scheduled, loc); err == nil {
			pd.Scheduled = t
		}
		if t, err := time.ParseInLocation("2006-01-02T15:04:05", d.Expected, loc); err == nil {
			pd.Expected = t
		}

		// Calculate minutes left from expected time (or scheduled if no expected)
		ref := pd.Expected
		if ref.IsZero() {
			ref = pd.Scheduled
		}
		if !ref.IsZero() {
			mins := int(math.Ceil(ref.Sub(now).Minutes()))
			if mins < 0 {
				mins = 0
			}
			pd.MinutesLeft = mins
		}

		parsed = append(parsed, pd)
	}

	return parsed
}

// FilterByTransportMode filters departures by transport mode.
func FilterByTransportMode(deps []model.ParsedDeparture, mode string) []model.ParsedDeparture {
	if mode == "" {
		return deps
	}
	mode = strings.ToUpper(mode)
	var filtered []model.ParsedDeparture
	for _, d := range deps {
		if strings.EqualFold(d.TransportMode, mode) {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

// FilterByLine filters departures by line designation.
func FilterByLine(deps []model.ParsedDeparture, line string) []model.ParsedDeparture {
	if line == "" {
		return deps
	}
	var filtered []model.ParsedDeparture
	for _, d := range deps {
		if strings.EqualFold(d.Line, line) {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

// FilterByDirection filters departures by direction name (substring match).
func FilterByDirection(deps []model.ParsedDeparture, direction string) []model.ParsedDeparture {
	if direction == "" {
		return deps
	}
	direction = strings.ToLower(direction)
	var filtered []model.ParsedDeparture
	for _, d := range deps {
		if strings.Contains(strings.ToLower(d.Destination), direction) ||
			strings.Contains(strings.ToLower(d.Direction), direction) {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

// DistanceKm calculates the Haversine distance between two coordinates in km.
func DistanceKm(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0 // Earth radius in km
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

// FindNearestSites finds sites within a given radius (km) sorted by distance.
func FindNearestSites(sites []model.Site, lat, lon, radiusKm float64) []SiteWithDistance {
	var results []SiteWithDistance
	for _, s := range sites {
		d := DistanceKm(lat, lon, s.Lat, s.Lon)
		if d <= radiusKm {
			results = append(results, SiteWithDistance{Site: s, DistanceKm: d})
		}
	}
	// Sort by distance
	sort.Slice(results, func(i, j int) bool {
		return results[i].DistanceKm < results[j].DistanceKm
	})
	return results
}

// SiteWithDistance is a site with its distance from a reference point.
type SiteWithDistance struct {
	Site       model.Site `json:"site"`
	DistanceKm float64    `json:"distance_km"`
	DistanceM  int        `json:"distance_m"`
}
