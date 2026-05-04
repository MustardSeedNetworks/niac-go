package main

import (
	"slices"
	"testing"
)

func TestIsValidErrorType(t *testing.T) {
	tests := []struct {
		name      string
		errorType string
		valid     bool
	}{
		// Canonical names
		{"FCS Errors canonical", "FCS Errors", true},
		{"Packet Discards canonical", "Packet Discards", true},
		{"Interface Errors canonical", "Interface Errors", true},
		{"High Utilization canonical", "High Utilization", true},
		{"High CPU canonical", "High CPU", true},
		{"High Memory canonical", "High Memory", true},
		{"High Disk canonical", "High Disk", true},
		// Snake_case aliases
		{"fcs_errors alias", "fcs_errors", true},
		{"packet_discards alias", "packet_discards", true},
		{"interface_errors alias", "interface_errors", true},
		{"high_utilization alias", "high_utilization", true},
		{"high_cpu alias", "high_cpu", true},
		{"high_memory alias", "high_memory", true},
		{"high_disk alias", "high_disk", true},
		// Invalid types
		{"empty string", "", false},
		{"random string", "random", false},
		{"partial match", "fcs", false},
		{"wrong case", "FCS_ERRORS", false},
		{"with spaces alias", "fcs errors", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidErrorType(tt.errorType)
			if result != tt.valid {
				t.Errorf("isValidErrorType(%q) = %v, want %v", tt.errorType, result, tt.valid)
			}
		})
	}
}

func TestNormalizeErrorType(t *testing.T) {
	tests := []struct {
		name      string
		errorType string
		expected  string
	}{
		// Aliases should normalize to canonical
		{"fcs_errors", "fcs_errors", "FCS Errors"},
		{"packet_discards", "packet_discards", "Packet Discards"},
		{"interface_errors", "interface_errors", "Interface Errors"},
		{"high_utilization", "high_utilization", "High Utilization"},
		{"high_cpu", "high_cpu", "High CPU"},
		{"high_memory", "high_memory", "High Memory"},
		{"high_disk", "high_disk", "High Disk"},
		// Canonical names should pass through
		{"FCS Errors passthrough", "FCS Errors", "FCS Errors"},
		{"High CPU passthrough", "High CPU", "High CPU"},
		// Unknown types should pass through
		{"unknown passthrough", "unknown_type", "unknown_type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeErrorType(tt.errorType)
			if result != tt.expected {
				t.Errorf("normalizeErrorType(%q) = %q, want %q", tt.errorType, result, tt.expected)
			}
		})
	}
}

func TestValidErrorTypes(t *testing.T) {
	types := validErrorTypes()

	if len(types) == 0 {
		t.Fatal("validErrorTypes() returned empty list")
	}

	expectedTypes := []string{
		"FCS Errors",
		"Packet Discards",
		"Interface Errors",
		"High Utilization",
		"High CPU",
		"High Memory",
		"High Disk",
	}

	if len(types) != len(expectedTypes) {
		t.Errorf("validErrorTypes() returned %d types, want %d", len(types), len(expectedTypes))
	}

	for _, expected := range expectedTypes {
		found := slices.Contains(types, expected)
		if !found {
			t.Errorf("validErrorTypes() missing %q", expected)
		}
	}
}

func TestErrorTypeAliases(t *testing.T) {
	aliases := errorTypeAliases()

	if len(aliases) == 0 {
		t.Fatal("errorTypeAliases() returned empty map")
	}

	// Verify all aliases map to valid canonical names
	validTypes := validErrorTypes()
	for alias, canonical := range aliases {
		found := slices.Contains(validTypes, canonical)
		if !found {
			t.Errorf("alias %q maps to invalid canonical name %q", alias, canonical)
		}
	}

	// Verify expected aliases exist
	expectedAliases := map[string]string{
		"fcs_errors":       "FCS Errors",
		"packet_discards":  "Packet Discards",
		"interface_errors": "Interface Errors",
		"high_utilization": "High Utilization",
		"high_cpu":         "High CPU",
		"high_memory":      "High Memory",
		"high_disk":        "High Disk",
	}

	for alias, expected := range expectedAliases {
		if canonical, ok := aliases[alias]; !ok {
			t.Errorf("missing alias %q", alias)
		} else if canonical != expected {
			t.Errorf("alias %q = %q, want %q", alias, canonical, expected)
		}
	}
}

func TestGetIPCClient(t *testing.T) {
	client := getIPCClient("/tmp/test.sock")
	if client == nil {
		t.Fatal("getIPCClient() returned nil")
	}
}

func TestInjectionStructs(t *testing.T) {
	// Verify Injection struct fields
	inj := Injection{
		Device:    "router-1",
		Interface: "Gi0/1",
		ErrorType: "FCS Errors",
		Value:     50,
	}

	if inj.Device != "router-1" {
		t.Errorf("Device = %q, want %q", inj.Device, "router-1")
	}
	if inj.Interface != "Gi0/1" {
		t.Errorf("Interface = %q, want %q", inj.Interface, "Gi0/1")
	}
	if inj.ErrorType != "FCS Errors" {
		t.Errorf("ErrorType = %q, want %q", inj.ErrorType, "FCS Errors")
	}
	if inj.Value != 50 {
		t.Errorf("Value = %d, want %d", inj.Value, 50)
	}

	// Verify InjectionResponse struct fields
	resp := InjectionResponse{
		Success:    true,
		Message:    "test message",
		Injections: []Injection{inj},
	}

	if !resp.Success {
		t.Error("Expected Success=true")
	}
	if resp.Message != "test message" {
		t.Errorf("Message = %q, want %q", resp.Message, "test message")
	}
	if len(resp.Injections) != 1 {
		t.Errorf("Expected 1 injection, got %d", len(resp.Injections))
	}
}

func TestRunInjectClearValidation(t *testing.T) {
	t.Run("neither device nor all", func(t *testing.T) {
		options := &injectOptions{
			clearDev: "",
			clearAll: false,
		}
		err := runInjectClear(options)
		if err == nil {
			t.Error("Expected error when neither --device nor --all specified")
		}
	})

	t.Run("both device and all", func(t *testing.T) {
		options := &injectOptions{
			clearDev: "router-1",
			clearAll: true,
		}
		err := runInjectClear(options)
		if err == nil {
			t.Error("Expected error when both --device and --all specified")
		}
	})
}

func TestRunInjectArgValidation(t *testing.T) {
	t.Run("wrong number of args", func(t *testing.T) {
		options := &injectOptions{}
		err := runInject([]string{"device"}, options)
		if err == nil {
			t.Error("Expected error for wrong arg count")
		}
	})

	t.Run("invalid error type", func(t *testing.T) {
		options := &injectOptions{}
		err := runInject([]string{"device", "invalid_type", "50"}, options)
		if err == nil {
			t.Error("Expected error for invalid error type")
		}
	})

	t.Run("non-numeric value", func(t *testing.T) {
		options := &injectOptions{}
		err := runInject([]string{"device", "fcs_errors", "abc"}, options)
		if err == nil {
			t.Error("Expected error for non-numeric value")
		}
	})

	t.Run("value out of range negative", func(t *testing.T) {
		options := &injectOptions{}
		err := runInject([]string{"device", "fcs_errors", "-1"}, options)
		if err == nil {
			t.Error("Expected error for negative value")
		}
	})

	t.Run("value out of range over 100", func(t *testing.T) {
		options := &injectOptions{}
		err := runInject([]string{"device", "fcs_errors", "101"}, options)
		if err == nil {
			t.Error("Expected error for value > 100")
		}
	})
}
