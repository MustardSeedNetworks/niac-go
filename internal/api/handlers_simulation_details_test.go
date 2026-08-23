package api

import (
	"errors"
	"testing"
)

// TestPreflightErrorDetails guards D5.
//
// The preflight handler collapsed every failure into the fixed string
// "Simulation preflight failed" and passed nil details, so the one button whose
// job is diagnosis reported nothing. The daemon knew the cause all along — the
// same binary's CLI prints "SNMPv1/v2c requires an explicit community" for the
// identical config — and the log line carried only a count.
func TestPreflightErrorDetails(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want []string
	}{
		{
			name: "aggregated semantic validation errors are split per issue",
			err: errors.New("simulation configuration failed semantic validation: " +
				"SNMPv1/v2c requires an explicit community; SNMPv1/v2c requires an explicit community"),
			want: []string{
				"simulation configuration failed semantic validation: SNMPv1/v2c requires an explicit community",
				"SNMPv1/v2c requires an explicit community",
			},
		},
		{
			name: "newline separated issues are split too",
			err:  errors.New("first problem\nsecond problem"),
			want: []string{"first problem", "second problem"},
		},
		{
			name: "a single error still yields one detail",
			err:  errors.New("unknown attachment"),
			want: []string{"unknown attachment"},
		},
		{
			name: "nil error yields no details",
			err:  nil,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := preflightErrorDetails(tt.err)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d details, want %d: %+v", len(got), len(tt.want), got)
			}
			for i := range tt.want {
				if got[i].Issue != tt.want[i] {
					t.Errorf("detail[%d] = %q, want %q", i, got[i].Issue, tt.want[i])
				}
			}
		})
	}
}
