package cmd

import (
	"testing"

	"github.com/glundgren93/sl-cli/internal/format"
	"github.com/glundgren93/sl-cli/internal/model"
)

func TestExtractLines(t *testing.T) {
	parsed := []model.ParsedDeparture{
		{Line: "55", TransportMode: "BUS", GroupOfLines: "", Destination: "Tanto"},
		{Line: "55", TransportMode: "BUS", GroupOfLines: "", Destination: "Henriksdalsberget"},
		{Line: "55", TransportMode: "BUS", GroupOfLines: "", Destination: "Tanto"}, // duplicate dest
		{Line: "43", TransportMode: "TRAIN", GroupOfLines: "Pendeltåg", Destination: "Bålsta"},
		{Line: "43", TransportMode: "TRAIN", GroupOfLines: "Pendeltåg", Destination: "Västerhaninge"},
		{Line: "17", TransportMode: "METRO", GroupOfLines: "Gröna linjen", Destination: "Åkeshov"},
	}

	lines := extractLines(parsed)

	if len(lines) != 3 {
		t.Fatalf("expected 3 unique lines, got %d", len(lines))
	}

	// Verify line 55
	assertLine(t, lines[0], "55", "BUS", 2) // Tanto + Henriksdalsberget (deduped)

	// Verify line 43
	assertLine(t, lines[1], "43", "TRAIN", 2) // Bålsta + Västerhaninge

	// Verify line 17
	assertLine(t, lines[2], "17", "METRO", 1) // Åkeshov
}

func TestExtractLines_Empty(t *testing.T) {
	lines := extractLines(nil)
	if len(lines) != 0 {
		t.Errorf("expected 0 lines from nil input, got %d", len(lines))
	}

	lines = extractLines([]model.ParsedDeparture{})
	if len(lines) != 0 {
		t.Errorf("expected 0 lines from empty input, got %d", len(lines))
	}
}

func assertLine(t *testing.T, line format.StopInfoLine, designation, mode string, minDests int) {
	t.Helper()
	if line.Designation != designation {
		t.Errorf("expected designation %q, got %q", designation, line.Designation)
	}
	if line.TransportMode != mode {
		t.Errorf("expected mode %q, got %q", mode, line.TransportMode)
	}
	if len(line.Destinations) < minDests {
		t.Errorf("expected at least %d destinations for line %s, got %d: %v",
			minDests, designation, len(line.Destinations), line.Destinations)
	}
}
