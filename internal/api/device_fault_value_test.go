package api

import "testing"

func TestDeviceFaultRequestUsesCatalogBounds(t *testing.T) {
	for _, tc := range []struct {
		kind  string
		value int
		valid bool
	}{
		{"Latency", 60000, true},
		{"Latency", 60001, false},
		{"Captive Portal", 1, true},
		{"Captive Portal", 2, false},
		{"Captive Portal", 0, true},
		{"Captive Portal", -1, false},
		{"CPU Utilization", 100, true},
		{"CPU Utilization", 101, false},
	} {
		req := errorInjectionRequest{Device: "server-1", ErrorType: tc.kind, Value: new(tc.value)}
		if message := req.validationMessage(); (message == "") != tc.valid {
			t.Errorf("%s value%d: %q, valid=%v", tc.kind, tc.value, message, tc.valid)
		}
	}
}

func TestDeviceFaultCatalogDescribesControls(t *testing.T) {
	want := map[string]struct {
		max  int
		kind string
	}{
		"DHCP No Offer": {100, "toggle"}, "DNS NXDOMAIN": {100, "toggle"},
		"DNS Timeout": {100, "toggle"}, "Latency": {60000, "milliseconds"},
		"CPU Utilization": {100, "percent"}, "Memory Utilization": {100, "percent"},
		"Disk Utilization": {100, "percent"}, "Captive Portal": {1, "toggle"},
	}
	for _, entry := range availableDeviceErrorTypes() {
		if entry.Type == "Duplicate DHCP Offer" {
			if entry.ValueKind != "address" || entry.MaxValue != nil {
				t.Fatalf("address fault has numeric metadata: %+v", entry)
			}
			continue
		}
		value, ok := want[entry.Type]
		if !ok || entry.MaxValue == nil || value.max != *entry.MaxValue || value.kind != entry.ValueKind {
			t.Errorf("unexpected control metadata: %+v", entry)
		}
		delete(want, entry.Type)
	}
	if len(want) != 0 {
		t.Fatalf("missing controls: %v", want)
	}
}
