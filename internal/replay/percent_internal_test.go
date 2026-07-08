package replay

import "testing"

// TestPercentComplete is a white-box test (same package) for the percent
// helper Status() uses to compute ReplayState.PercentComplete. Covers the
// unknown-total case (must not fabricate a percentage), normal progress,
// and the completed/over-100 clamp.
func TestPercentComplete(t *testing.T) {
	tests := []struct {
		name        string
		sent, total uint64
		want        float64
	}{
		{"unknown total returns zero (omitted by json)", 5, 0, 0},
		{"zero progress", 0, 100, 0},
		{"partial progress", 25, 100, 25},
		{"non-round percentage", 1, 3, 33.33},
		{"complete", 100, 100, 100},
		{"sent exceeds total is clamped to 100", 150, 100, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := percentComplete(tt.sent, tt.total)
			if got != tt.want {
				t.Errorf("percentComplete(%d, %d) = %v, want %v", tt.sent, tt.total, got, tt.want)
			}
		})
	}
}
