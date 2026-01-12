package mibdb

import (
	"testing"
	"time"
)

func TestNewInMemory(t *testing.T) {
	db, err := NewInMemory()
	if err != nil {
		t.Fatalf("Failed to create in-memory database: %v", err)
	}

	defer func() { _ = db.Close() }()

	stats, err := db.Stats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats["oid_names"] != 918 {
		t.Errorf("Expected 918 OID entries, got %d", stats["oid_names"])
	}
}

func TestResolveOIDName(t *testing.T) {
	db, err := NewInMemory()
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	defer func() { _ = db.Close() }()

	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{"numeric OID passthrough", "1.3.6.1.2.1.1.1.0", "1.3.6.1.2.1.1.1.0", false},
		{"numeric OID with dot prefix", ".1.3.6.1.2.1.1.1.0", "1.3.6.1.2.1.1.1.0", false},
		{"sysDescr", "sysDescr.0", "1.3.6.1.2.1.1.1.0", false},
		{"sysUpTime", "sysUpTime.0", "1.3.6.1.2.1.1.3.0", false},
		{"ifDescr with index", "ifDescr.1", "1.3.6.1.2.1.2.2.1.2.1", false},
		{"unknown name", "unknownOID.0", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := db.ResolveOIDName(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for input %s", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if result != tt.expected {
					t.Errorf("Expected %s, got %s", tt.expected, result)
				}
			}
		})
	}
}

func TestGetOIDByName(t *testing.T) {
	db, err := NewInMemory()
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	defer func() { _ = db.Close() }()

	entry, err := db.GetOIDByName("sysDescr")
	if err != nil {
		t.Fatalf("Failed to get OID: %v", err)
	}

	if entry == nil {
		t.Fatal("Expected entry, got nil")
	}

	if entry.OID != "1.3.6.1.2.1.1.1" {
		t.Errorf("Expected OID 1.3.6.1.2.1.1.1, got %s", entry.OID)
	}
}

func TestGetOIDsByPrefix(t *testing.T) {
	db, err := NewInMemory()
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	defer func() { _ = db.Close() }()

	// Get all system MIB OIDs
	entries, err := db.GetOIDsByPrefix("1.3.6.1.2.1.1")
	if err != nil {
		t.Fatalf("Failed to get OIDs by prefix: %v", err)
	}

	if len(entries) < 5 {
		t.Errorf("Expected at least 5 system MIB entries, got %d", len(entries))
	}
}

func TestParseVariMibIntegral(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
		wantErr  bool
	}{
		{"single interval", "varimib[(200 10)]", 1, false},
		{"two intervals", "varimib((2000 10) (4000 -10))", 2, false},
		{"negative delta", "varimib[(100 -5)]", 1, false},
		{"invalid format", "invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseVariMibIntegral(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for input %s", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if len(result.Intervals) != tt.expected {
					t.Errorf("Expected %d intervals, got %d", tt.expected, len(result.Intervals))
				}
			}
		})
	}
}

func TestParseVariMibString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
		wantErr  bool
	}{
		{"single value", "varimib[(42000 10.250.0.3)]", 1, false},
		{"two values", "varimib((42000 10.250.0.3) (42000 10.250.0.2))", 2, false},
		{"invalid format", "invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseVariMibString(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for input %s", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if len(result.Intervals) != tt.expected {
					t.Errorf("Expected %d intervals, got %d", tt.expected, len(result.Intervals))
				}
			}
		})
	}
}

func TestVariMibIntegralHandler(t *testing.T) {
	config := &VariMibIntegralConfig{
		Intervals: []struct {
			CentiSeconds int64
			Delta        int64
		}{
			{1, 10}, // Increment by 10 every 10ms (1 centisecond)
		},
	}

	handler := NewVariMibIntegralHandler(config, 1000)

	// Initial value should be base value
	initial := handler.Value()
	if initial < 1000 {
		t.Errorf("Initial value should be at least base value 1000, got %d", initial)
	}

	// Wait a bit and check value increased (50ms = 5 centiseconds = 5 intervals)
	time.Sleep(50 * time.Millisecond)

	after := handler.Value()
	if after <= initial {
		t.Errorf("Value should have increased after 50ms, initial=%d, after=%d", initial, after)
	}
}

func TestVariMibStringHandler(t *testing.T) {
	config := &VariMibStringConfig{
		Intervals: []struct {
			CentiSeconds int64
			Value        string
		}{
			{10, "value1"}, // 100ms
			{10, "value2"}, // 100ms
		},
	}

	handler := NewVariMibStringHandler(config)

	// Should get one of the values
	value := handler.Value()
	if value != "value1" && value != "value2" {
		t.Errorf("Expected value1 or value2, got %s", value)
	}
}

func TestAddVariMibHandler(t *testing.T) {
	db, err := NewInMemory()
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	defer func() { _ = db.Close() }()

	handler := VariMibHandler{
		OIDPattern:  "1.3.6.1.2.1.2.2.1.10.1",
		HandlerType: "integral",
		Config:      "varimib((200 10))",
	}

	if err := db.AddVariMibHandler(handler); err != nil {
		t.Fatalf("Failed to add handler: %v", err)
	}

	retrieved, err := db.GetVariMibHandler("1.3.6.1.2.1.2.2.1.10.1")
	if err != nil {
		t.Fatalf("Failed to get handler: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected handler, got nil")
	}

	if retrieved.HandlerType != "integral" {
		t.Errorf("Expected handler type 'integral', got %s", retrieved.HandlerType)
	}
}
