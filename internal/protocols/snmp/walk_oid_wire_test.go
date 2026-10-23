package snmp

import "testing"

func TestWalkRejectsObjectIdentifiersThatCannotBeEncoded(t *testing.T) {
	t.Parallel()
	for _, value := range []string{".0", "1", "3.1.1", "1.40.1", "1.3.4294967296.1"} {
		t.Run(value, func(t *testing.T) {
			t.Parallel()
			if entry, err := parseWalkLine(value + " = STRING: unusable key"); err == nil {
				t.Errorf("accepted key: %#v", entry)
			}
			if entry, err := parseWalkLine("1.3.6.1.4.1.99999.1 = OID: " + value); err == nil {
				t.Errorf("accepted value: %#v", entry)
			}
		})
	}
}

func TestWalkRejectsInvalidIPAddressValues(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"", "300.1.1.1", "1.2.3", "2001:db8::1"} {
		t.Run(value, func(t *testing.T) {
			t.Parallel()
			if entry, err := parseWalkLine("1.3.6.1.4.1.99999.1 = IpAddress: " + value); err == nil {
				t.Errorf("accepted invalid address: %#v", entry)
			}
		})
	}
}

func TestWalkPreserves32BitBoundaryValues(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		line string
		want any
	}{
		{"INTEGER: 2147483647", int(2147483647)},
		{"INTEGER: -2147483648", int(-2147483648)},
		{"Gauge32: 4294967295", uint(4294967295)},
		{"Counter32: 4294967295", uint(4294967295)},
		{"Unsigned32: 4294967295", uint(4294967295)},
	} {
		t.Run(tc.line, func(t *testing.T) {
			t.Parallel()
			entry, err := parseWalkLine("1.3.6.1.4.1.99999.1 = " + tc.line)
			if err != nil {
				t.Fatal(err)
			}
			if entry.Value != tc.want {
				t.Fatalf("got %v, want %v", entry.Value, tc.want)
			}
		})
	}
}

func TestWalkTimeticksPreserveValidUnsignedValues(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		value string
		want  uint32
	}{
		{"0", 0},
		{"4294967295", 4294967295},
		{"(12)", 12},
		{"(12) 0:00:00.12", 12},
		{"(9000000) 1 day, 1:00:00.00", 9000000},
		{"(17280000) 2 days, 0:00:00.00", 17280000},
	} {
		t.Run(tc.value, func(t *testing.T) {
			t.Parallel()
			entry, err := parseWalkLine("1.3.6.1.2.1.1.3.0 = Timeticks: " + tc.value)
			if err != nil {
				t.Fatal(err)
			}
			if entry.Value != tc.want {
				t.Fatalf("got %v, want %v", entry.Value, tc.want)
			}
		})
	}
}

func TestWalkTimeticksRejectMalformedValues(t *testing.T) {
	t.Parallel()
	for _, value := range []string{
		"-1", "(-1) 0:00:00.01", "garbage123", "123garbage", "(12) garbage",
		"4294967296", "(4294967296)", "(12", "12)", "(12) 0:99:00.12",
	} {
		t.Run(value, func(t *testing.T) {
			t.Parallel()
			if entry, err := parseWalkLine("1.3.6.1.2.1.1.3.0 = Timeticks: " + value); err == nil {
				t.Fatalf("accepted malformed timeticks: %#v", entry)
			}
		})
	}
}
