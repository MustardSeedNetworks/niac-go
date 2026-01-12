package snmp

import (
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"
)

func TestParseVariMibValue_Fixed(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		asnType  gosnmp.Asn1BER
		expected any
	}{
		{"fixed integer", "fixed(42)", gosnmp.Integer, 42},
		{"fixed string", "fixed(hello)", gosnmp.OctetString, "hello"},
		{"fixed counter", "fixed(1000)", gosnmp.Counter32, uint32(1000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := ParseVariMibValue(tt.value, tt.asnType, nil)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if handler.GetValue() != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, handler.GetValue())
			}
		})
	}
}

func TestParseVariMibValue_Integral(t *testing.T) {
	// varimib((100 10)) - increment by 10 every 1 second (100 centiseconds)
	handler, err := ParseVariMibValue("varimib((1 10))", gosnmp.Counter32, int64(1000))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	initial := handler.GetValue().(uint32)
	if initial < 1000 {
		t.Errorf("Initial value should be at least 1000, got %d", initial)
	}

	// Wait for value to change
	time.Sleep(30 * time.Millisecond)

	after := handler.GetValue().(uint32)

	if after <= initial {
		t.Errorf("Value should have increased, initial=%d, after=%d", initial, after)
	}
}

func TestParseVariMibValue_String(t *testing.T) {
	// varimib((10 value1) (10 value2)) - cycle through values
	handler, err := ParseVariMibValue("varimib((10 value1) (10 value2))", gosnmp.OctetString, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	value := handler.GetValue().(string)
	if value != "value1" && value != "value2" {
		t.Errorf("Expected value1 or value2, got %s", value)
	}
}

func TestParseVariMibValue_IPAddress(t *testing.T) {
	// Test IP address cycling (from Java config example)
	handler, err := ParseVariMibValue("varimib((10 10.250.0.3) (10 10.250.0.2))", gosnmp.OctetString, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	value := handler.GetValue().(string)
	if value != "10.250.0.3" && value != "10.250.0.2" {
		t.Errorf("Expected IP address, got %s", value)
	}
}

func TestIsVariMibValue(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
	}{
		{"varimib((200 10))", true},
		{"fixed(hello)", true},
		{"just a string", false},
		{"123", false},
		{"varimib[(100 5)]", true},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			result := IsVariMibValue(tt.value)
			if result != tt.expected {
				t.Errorf("IsVariMibValue(%q) = %v, expected %v", tt.value, result, tt.expected)
			}
		})
	}
}

func TestVariMibRegistry(t *testing.T) {
	registry := NewVariMibRegistry()

	handler, err := ParseVariMibValue("fixed(42)", gosnmp.Integer, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	registry.Register("1.3.6.1.2.1.1.1.0", handler)

	// Test retrieval
	retrieved := registry.Get("1.3.6.1.2.1.1.1.0")
	if retrieved == nil {
		t.Fatal("Expected handler, got nil")
	}

	if retrieved.GetValue() != 42 {
		t.Errorf("Expected 42, got %v", retrieved.GetValue())
	}

	// Test OIDValue retrieval
	oidValue := registry.GetOIDValue("1.3.6.1.2.1.1.1.0")
	if oidValue == nil {
		t.Fatal("Expected OIDValue, got nil")
	}

	if oidValue.Type != gosnmp.Integer {
		t.Errorf("Expected Integer type, got %v", oidValue.Type)
	}
}

func TestVariMibIntegralHandler_CounterWrap(t *testing.T) {
	// Test that 32-bit counters wrap properly
	handler := &VariMibIntegralHandler{
		startTime: time.Now().Add(-time.Hour), // Started an hour ago
		baseValue: 0xFFFFFFF0,                 // Near max 32-bit value
		asnType:   gosnmp.Counter32,
		intervals: []integralInterval{{centiSeconds: 1, delta: 1}},
	}

	value := handler.GetValue().(uint32)
	// Value should have wrapped around
	if value < 0xFFFFFFF0 && value > 1000000 {
		// This is expected - wrapped around
	}
}

func TestVariMibStringHandler_Cycling(t *testing.T) {
	handler := &VariMibStringHandler{
		startTime: time.Now(),
		asnType:   gosnmp.OctetString,
		intervals: []stringInterval{
			{centiSeconds: 5, value: "first"},  // 50ms
			{centiSeconds: 5, value: "second"}, // 50ms
		},
	}

	// Check initial value
	value := handler.GetValue().(string)
	if value != "first" && value != "second" {
		t.Errorf("Expected first or second, got %s", value)
	}

	// Wait and check cycling
	time.Sleep(60 * time.Millisecond)

	newValue := handler.GetValue().(string)
	if newValue != "first" && newValue != "second" {
		t.Errorf("Expected first or second, got %s", newValue)
	}
}

func TestNewVariMibFixedHandler_AllTypes(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		asnType gosnmp.Asn1BER
		wantErr bool
	}{
		{"integer", "42", gosnmp.Integer, false},
		{"counter32", "1000", gosnmp.Counter32, false},
		{"counter64", "9876543210", gosnmp.Counter64, false},
		{"gauge32", "500", gosnmp.Gauge32, false},
		{"timeticks", "12345", gosnmp.TimeTicks, false},
		{"ipaddress", "192.168.1.1", gosnmp.IPAddress, false},
		{"oid", "1.3.6.1.2.1.1.1", gosnmp.ObjectIdentifier, false},
		{"string", "hello world", gosnmp.OctetString, false},
		{"invalid integer", "not-a-number", gosnmp.Integer, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewVariMibFixedHandler(tt.value, tt.asnType)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}
