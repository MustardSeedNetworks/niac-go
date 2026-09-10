package converter

import "testing"

func TestBehaviorMaskPayload(t *testing.T) {
	for _, test := range []struct {
		name    string
		prefix  *int
		value   *int
		address *string
		valid   bool
	}{
		{name: "default prefix", prefix: new(0), valid: true},
		{name: "host prefix", prefix: new(32), valid: true},
		{name: "missing prefix"},
		{name: "negative prefix", prefix: new(-1)},
		{name: "oversized prefix", prefix: new(33)},
		{name: "numeric payload", prefix: new(24), value: new(1)},
		{name: "address payload", prefix: new(24), address: new("192.0.2.1")},
	} {
		t.Run(test.name, func(t *testing.T) {
			fault := BehaviorFault{
				Device:     "host",
				Interface:  "eth0",
				Type:       "bad_mask",
				PrefixBits: test.prefix,
				Value:      test.value,
				Address:    test.address,
			}
			err := configValidator.Struct(fault)
			if (err == nil) != test.valid {
				t.Fatalf("validation error = %v, want valid %v", err, test.valid)
			}
		})
	}
}

func TestOtherFaultsRejectPrefixPayload(t *testing.T) {
	for _, kind := range []string{"link_down", "duplicate_ip", "duplicate_dhcp_offer"} {
		fault := BehaviorFault{Device: "host", Interface: "eth0", Type: kind, PrefixBits: new(0)}
		if kind == "link_down" {
			fault.Value = new(1)
		} else {
			fault.Address = new("192.0.2.1")
		}
		if err := configValidator.Struct(fault); err == nil {
			t.Fatalf("%s accepted a prefix payload", kind)
		}
	}
}
