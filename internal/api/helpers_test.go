package api

import (
	"testing"
	"time"

	"github.com/glundgren93/sl-cli/internal/model"
)

func TestDistanceKm(t *testing.T) {
	tests := []struct {
		name     string
		lat1     float64
		lon1     float64
		lat2     float64
		lon2     float64
		wantMin  float64
		wantMax  float64
	}{
		{
			name:    "same point",
			lat1:    59.3121, lon1: 18.0643,
			lat2:    59.3121, lon2: 18.0643,
			wantMin: 0, wantMax: 0.001,
		},
		{
			name:    "Magnus Ladulåsgatan to Medborgarplatsen",
			lat1:    59.3121, lon1: 18.0643,
			lat2:    59.3143, lon2: 18.0734,
			wantMin: 0.3, wantMax: 0.6,
		},
		{
			name:    "Magnus Ladulåsgatan to T-Centralen",
			lat1:    59.3121, lon1: 18.0643,
			lat2:    59.3310, lon2: 18.0593,
			wantMin: 1.5, wantMax: 2.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DistanceKm(tt.lat1, tt.lon1, tt.lat2, tt.lon2)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("DistanceKm() = %f, want between %f and %f", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestFindNearestSites(t *testing.T) {
	sites := []model.Site{
		{ID: 1, Name: "Close", Lat: 59.3122, Lon: 18.0644},
		{ID: 2, Name: "Medium", Lat: 59.3140, Lon: 18.0700},
		{ID: 3, Name: "Far", Lat: 59.3300, Lon: 18.0600},
		{ID: 4, Name: "Too far", Lat: 59.4000, Lon: 18.0000},
	}

	results := FindNearestSites(sites, 59.3121, 18.0643, 0.5)

	if len(results) != 2 {
		t.Fatalf("expected 2 results within 500m, got %d", len(results))
	}
	if results[0].Site.Name != "Close" {
		t.Errorf("closest should be 'Close', got %q", results[0].Site.Name)
	}
	if results[1].Site.Name != "Medium" {
		t.Errorf("second should be 'Medium', got %q", results[1].Site.Name)
	}

	// Verify sorted by distance
	if results[0].DistanceKm > results[1].DistanceKm {
		t.Error("results should be sorted by distance ascending")
	}
}

func TestFindNearestSites_EmptyRadius(t *testing.T) {
	sites := []model.Site{
		{ID: 1, Name: "Far", Lat: 60.0, Lon: 18.0},
	}

	results := FindNearestSites(sites, 59.3121, 18.0643, 0.1)
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestParseDepartures(t *testing.T) {
	deps := []model.Departure{
		{
			Destination: "Henriksdalsberget",
			Direction:   "Henriksdalsberget",
			Display:     "5 min",
			State:       "EXPECTED",
			Scheduled:   time.Now().Add(5 * time.Minute).Format("2006-01-02T15:04:05"),
			Expected:    time.Now().Add(5 * time.Minute).Format("2006-01-02T15:04:05"),
			Line: &model.Line{
				Designation:   "55",
				TransportMode: "BUS",
				GroupOfLines:  "",
			},
			StopArea: &model.StopArea{
				Name: "Timmermansgränd",
				Type: "BUSTERM",
			},
			StopPoint: &model.StopPoint{
				Name:        "Timmermansgränd",
				Designation: "A",
			},
		},
	}

	parsed := ParseDepartures(deps)

	if len(parsed) != 1 {
		t.Fatalf("expected 1 parsed departure, got %d", len(parsed))
	}

	pd := parsed[0]
	if pd.Line != "55" {
		t.Errorf("line = %q, want 55", pd.Line)
	}
	if pd.TransportMode != "BUS" {
		t.Errorf("transport mode = %q, want BUS", pd.TransportMode)
	}
	if pd.Destination != "Henriksdalsberget" {
		t.Errorf("destination = %q, want Henriksdalsberget", pd.Destination)
	}
	if pd.StopArea != "Timmermansgränd" {
		t.Errorf("stop area = %q, want Timmermansgränd", pd.StopArea)
	}
	if pd.Platform != "A" {
		t.Errorf("platform = %q, want A", pd.Platform)
	}
	// MinutesLeft should be around 5
	if pd.MinutesLeft < 4 || pd.MinutesLeft > 6 {
		t.Errorf("minutes left = %d, want ~5", pd.MinutesLeft)
	}
}

func TestParseDepartures_PastTime(t *testing.T) {
	deps := []model.Departure{
		{
			Destination: "Test",
			State:       "ATSTOP",
			Scheduled:   time.Now().Add(-2 * time.Minute).Format("2006-01-02T15:04:05"),
			Expected:    time.Now().Add(-1 * time.Minute).Format("2006-01-02T15:04:05"),
			Line:        &model.Line{Designation: "1", TransportMode: "BUS"},
		},
	}

	parsed := ParseDepartures(deps)
	if parsed[0].MinutesLeft != 0 {
		t.Errorf("past departure should have 0 minutes left, got %d", parsed[0].MinutesLeft)
	}
}

func TestFilterByTransportMode(t *testing.T) {
	deps := []model.ParsedDeparture{
		{Line: "55", TransportMode: "BUS"},
		{Line: "17", TransportMode: "METRO"},
		{Line: "43", TransportMode: "TRAIN"},
		{Line: "66", TransportMode: "BUS"},
	}

	buses := FilterByTransportMode(deps, "BUS")
	if len(buses) != 2 {
		t.Errorf("expected 2 buses, got %d", len(buses))
	}

	metros := FilterByTransportMode(deps, "METRO")
	if len(metros) != 1 {
		t.Errorf("expected 1 metro, got %d", len(metros))
	}

	// Case insensitive
	trains := FilterByTransportMode(deps, "train")
	if len(trains) != 1 {
		t.Errorf("expected 1 train (case insensitive), got %d", len(trains))
	}

	// Empty filter returns all
	all := FilterByTransportMode(deps, "")
	if len(all) != 4 {
		t.Errorf("empty filter should return all, got %d", len(all))
	}
}

func TestFilterByLine(t *testing.T) {
	deps := []model.ParsedDeparture{
		{Line: "55", TransportMode: "BUS"},
		{Line: "17", TransportMode: "METRO"},
		{Line: "55", TransportMode: "BUS"},
	}

	filtered := FilterByLine(deps, "55")
	if len(filtered) != 2 {
		t.Errorf("expected 2 line-55 deps, got %d", len(filtered))
	}

	// Case insensitive
	filtered = FilterByLine(deps, "17")
	if len(filtered) != 1 {
		t.Errorf("expected 1 line-17, got %d", len(filtered))
	}

	// No match
	filtered = FilterByLine(deps, "999")
	if len(filtered) != 0 {
		t.Errorf("expected 0 results for non-existent line, got %d", len(filtered))
	}
}

func TestFilterByDirection(t *testing.T) {
	deps := []model.ParsedDeparture{
		{Line: "55", Destination: "Henriksdalsberget", Direction: "Henriksdalsberget"},
		{Line: "55", Destination: "Tanto", Direction: "Tanto"},
		{Line: "55", Destination: "Finnberget", Direction: "Henriksdalsberget"},
	}

	filtered := FilterByDirection(deps, "Tanto")
	if len(filtered) != 1 {
		t.Errorf("expected 1 Tanto departure, got %d", len(filtered))
	}

	// Substring match on destination
	filtered = FilterByDirection(deps, "berg")
	if len(filtered) != 2 {
		t.Errorf("expected 2 '*berg*' departures, got %d", len(filtered))
	}
}
