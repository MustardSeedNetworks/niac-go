package api

import (
	"strings"
	"testing"
)

func TestValidateHostname(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		wantErr  bool
		errField string
	}{
		// Valid hostnames
		{name: "simple hostname", hostname: "router1", wantErr: false},
		{name: "hostname with numbers", hostname: "router123", wantErr: false},
		{name: "hostname with hyphen", hostname: "core-router", wantErr: false},
		{name: "hostname with dot", hostname: "router.example.com", wantErr: false},
		{name: "single char", hostname: "a", wantErr: false},
		{name: "all numbers", hostname: "123", wantErr: false},
		{name: "uppercase", hostname: "ROUTER", wantErr: false},
		{name: "mixed case", hostname: "CoreRouter1", wantErr: false},

		// Invalid hostnames
		{name: "empty hostname", hostname: "", wantErr: true, errField: "hostname"},
		{name: "IP address", hostname: "192.168.1.1", wantErr: true, errField: "hostname"},
		{name: "IPv6 address", hostname: "::1", wantErr: true, errField: "hostname"},
		{name: "starts with hyphen", hostname: "-router", wantErr: true, errField: "hostname"},
		{name: "ends with hyphen", hostname: "router-", wantErr: true, errField: "hostname"},
		{name: "consecutive dots", hostname: "router..com", wantErr: true, errField: "hostname"},
		{name: "starts with dot", hostname: ".router", wantErr: true, errField: "hostname"},
		{name: "contains underscore", hostname: "core_router", wantErr: true, errField: "hostname"},
		{name: "contains space", hostname: "core router", wantErr: true, errField: "hostname"},
		{name: "contains special char", hostname: "router@home", wantErr: true, errField: "hostname"},
		{name: "label too long", hostname: strings.Repeat("a", 64), wantErr: true, errField: "hostname"},
		{name: "total too long", hostname: strings.Repeat("a", 300), wantErr: true, errField: "hostname"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHostname(tt.hostname)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateHostname(%q) = nil, want error", tt.hostname)
				} else if err.Field != tt.errField {
					t.Errorf("validateHostname(%q) error field = %q, want %q", tt.hostname, err.Field, tt.errField)
				}
			} else {
				if err != nil {
					t.Errorf("validateHostname(%q) = %v, want nil", tt.hostname, err)
				}
			}
		})
	}
}

func TestIsAlphanumeric(t *testing.T) {
	tests := []struct {
		char byte
		want bool
	}{
		{'a', true},
		{'z', true},
		{'A', true},
		{'Z', true},
		{'0', true},
		{'9', true},
		{'-', false},
		{'_', false},
		{'.', false},
		{' ', false},
		{'@', false},
		{'!', false},
	}

	for _, tt := range tests {
		t.Run(string(tt.char), func(t *testing.T) {
			got := isAlphanumeric(tt.char)
			if got != tt.want {
				t.Errorf("isAlphanumeric(%q) = %v, want %v", tt.char, got, tt.want)
			}
		})
	}
}

func TestValidateYAMLInput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "valid YAML", input: "key: value", wantErr: false},
		{name: "valid multiline", input: "devices:\n  - name: router1", wantErr: false},
		{name: "empty input", input: "", wantErr: true},
		{name: "whitespace only", input: "   \n\t  ", wantErr: true},
		{name: "too large", input: strings.Repeat("a", MaxYAMLSize+1), wantErr: true},
		{name: "at limit", input: strings.Repeat("a", MaxYAMLSize), wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateYAMLInput(tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("validateYAMLInput() = nil, want error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("validateYAMLInput() = %v, want nil", err)
			}
		})
	}
}

func TestCheckYAMLDepth(t *testing.T) {
	tests := []struct {
		name    string
		data    any
		depth   int
		wantErr bool
	}{
		{name: "nil data", data: nil, depth: 0, wantErr: false},
		{name: "string", data: "value", depth: 0, wantErr: false},
		{name: "number", data: 42, depth: 0, wantErr: false},
		{name: "flat map", data: map[string]any{"key": "value"}, depth: 0, wantErr: false},
		{name: "flat array", data: []any{"a", "b", "c"}, depth: 0, wantErr: false},
		{
			name:    "nested map",
			data:    map[string]any{"outer": map[string]any{"inner": "value"}},
			depth:   0,
			wantErr: false,
		},
		{
			name:    "nested array",
			data:    []any{[]any{[]any{"deep"}}},
			depth:   0,
			wantErr: false,
		},
		{
			name:    "at max depth",
			data:    "value",
			depth:   MaxYAMLDepth,
			wantErr: false,
		},
		{
			name:    "exceeds max depth",
			data:    "value",
			depth:   MaxYAMLDepth + 1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkYAMLDepth(tt.data, tt.depth)
			if tt.wantErr && err == nil {
				t.Errorf("checkYAMLDepth() = nil, want error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("checkYAMLDepth() = %v, want nil", err)
			}
		})
	}
}

// createNestedMap creates a nested map structure to specified depth
func createNestedMap(depth int) map[string]any {
	if depth == 0 {
		return map[string]any{"leaf": "value"}
	}
	return map[string]any{"nested": createNestedMap(depth - 1)}
}

func TestCheckYAMLDepthNested(t *testing.T) {
	tests := []struct {
		name    string
		depth   int
		wantErr bool
	}{
		{name: "depth 5", depth: 5, wantErr: false},
		{name: "depth 10", depth: 10, wantErr: false},
		{name: "depth 15", depth: 15, wantErr: false},
		{name: "depth 19", depth: 19, wantErr: false},
		{name: "depth 21", depth: 21, wantErr: true},
		{name: "depth 25", depth: 25, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := createNestedMap(tt.depth)
			err := checkYAMLDepth(data, 0)
			if tt.wantErr && err == nil {
				t.Errorf("checkYAMLDepth(depth=%d) = nil, want error", tt.depth)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("checkYAMLDepth(depth=%d) = %v, want nil", tt.depth, err)
			}
		})
	}
}

func TestValidateHostnameEdgeCases(t *testing.T) {
	// Test boundary conditions
	t.Run("exactly max label length", func(t *testing.T) {
		hostname := strings.Repeat("a", maxLabelLen)
		if err := validateHostname(hostname); err != nil {
			t.Errorf("hostname with exactly %d chars should be valid, got %v", maxLabelLen, err)
		}
	})

	t.Run("just over max label length", func(t *testing.T) {
		hostname := strings.Repeat("a", maxLabelLen+1)
		if err := validateHostname(hostname); err == nil {
			t.Errorf("hostname with %d chars should be invalid", maxLabelLen+1)
		}
	})

	t.Run("multiple valid labels", func(t *testing.T) {
		hostname := "router.switch.firewall.server"
		if err := validateHostname(hostname); err != nil {
			t.Errorf("multi-label hostname should be valid, got %v", err)
		}
	})

	t.Run("hyphen in middle", func(t *testing.T) {
		hostname := "core-router-01"
		if err := validateHostname(hostname); err != nil {
			t.Errorf("hyphen in middle should be valid, got %v", err)
		}
	})
}
