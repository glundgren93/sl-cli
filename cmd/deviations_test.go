package cmd

import (
	"testing"

	"github.com/glundgren93/sl-cli/internal/model"
)

func TestFilterDeviationsByLine(t *testing.T) {
	devs := []model.Deviation{
		{
			DeviationCaseID: 1,
			MessageVariants: []model.MessageVariant{{Header: "Bus 55 delayed", Language: "en"}},
			Scope: &model.DeviationScope{
				Lines: []model.Line{{Designation: "55", TransportMode: "BUS"}},
			},
		},
		{
			DeviationCaseID: 2,
			MessageVariants: []model.MessageVariant{{Header: "Metro 17 works", Language: "en"}},
			Scope: &model.DeviationScope{
				Lines: []model.Line{{Designation: "17", TransportMode: "METRO"}},
			},
		},
		{
			DeviationCaseID: 3,
			MessageVariants: []model.MessageVariant{{Header: "Train 43 issue", Language: "en"}},
			Scope: &model.DeviationScope{
				Lines: []model.Line{{Designation: "43", TransportMode: "TRAIN"}},
			},
		},
		{
			DeviationCaseID: 4,
			Scope:           nil, // nil scope should be skipped
		},
	}

	// Filter for line 55
	filtered := filterDeviationsByLine(devs, []string{"55"})
	if len(filtered) != 1 {
		t.Fatalf("expected 1 deviation for line 55, got %d", len(filtered))
	}
	if filtered[0].DeviationCaseID != 1 {
		t.Errorf("expected deviation case 1, got %d", filtered[0].DeviationCaseID)
	}

	// Filter for multiple lines
	filtered = filterDeviationsByLine(devs, []string{"55", "43"})
	if len(filtered) != 2 {
		t.Fatalf("expected 2 deviations for lines 55+43, got %d", len(filtered))
	}

	// Case insensitive (though line numbers are usually numeric)
	filtered = filterDeviationsByLine(devs, []string{"17"})
	if len(filtered) != 1 {
		t.Errorf("expected 1 deviation for line 17, got %d", len(filtered))
	}

	// Non-existent line
	filtered = filterDeviationsByLine(devs, []string{"999"})
	if len(filtered) != 0 {
		t.Errorf("expected 0 deviations for line 999, got %d", len(filtered))
	}

	// Empty designations
	filtered = filterDeviationsByLine(devs, []string{})
	if len(filtered) != 0 {
		t.Errorf("expected 0 deviations for empty filter, got %d", len(filtered))
	}
}
