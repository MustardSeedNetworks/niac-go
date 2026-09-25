package oui_test

import (
	"net"
	"strings"
	"sync"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/oui"
)

func TestParseAndLookup(t *testing.T) {
	registry, err := oui.Parse(strings.NewReader(`
00-00-0C   (hex)        Cisco Systems, Inc
00000C     (base 16)    Cisco Systems, Inc
00-05-85   (hex)        Juniper Networks
000585     (base 16)    Juniper Networks
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	vendor, ok := registry.Lookup(net.HardwareAddr{0x00, 0x05, 0x85, 0xaa, 0xbb, 0xcc})
	if !ok || vendor != "Juniper Networks" {
		t.Fatalf("Lookup() = %q, %v", vendor, ok)
	}
}

func TestAllocateUsesVendorPrefixDeterministically(t *testing.T) {
	registry, err := oui.Parse(strings.NewReader(`
E8-0A-B9   (hex)        Cisco Systems, Inc
00-00-0C   (hex)        Cisco Systems, Inc
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	mac, err := registry.Allocate("cisco", 0x010203)
	if err != nil {
		t.Fatalf("Allocate() error = %v", err)
	}
	if got := mac.String(); got != "00:00:0c:01:02:03" {
		t.Fatalf("Allocate() = %s", got)
	}
}

func TestEmbeddedRegistryContainsLabVendors(t *testing.T) {
	registry, err := oui.LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded() error = %v", err)
	}
	for _, vendor := range []string{
		"Cisco", "Juniper", "Aruba", "Meraki", "MikroTik", "Palo Alto", "Raspberry Pi",
	} {
		if _, err = registry.Allocate(vendor, 1); err != nil {
			t.Errorf("Allocate(%q) error = %v", vendor, err)
		}
	}
}

func TestAllocateRejectsUnknownVendor(t *testing.T) {
	registry, err := oui.Parse(strings.NewReader("00-00-0C   (hex)        Cisco Systems, Inc\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if _, err = registry.Allocate("not-a-vendor", 1); err == nil {
		t.Fatal("Allocate() error = nil, want unknown vendor error")
	}
}

func TestZeroRegistryRejectsVendorWithoutPanicking(t *testing.T) {
	var registry oui.Registry
	if _, err := registry.Allocate("cisco", 1); err == nil {
		t.Fatal("Allocate() error = nil, want unknown vendor error")
	}
}

func TestAllocateIsSafeForConcurrentReuse(t *testing.T) {
	registry, err := oui.LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded() error = %v", err)
	}

	const allocations = 100
	var waitGroup sync.WaitGroup
	errors := make(chan error, allocations)
	for index := range allocations {
		waitGroup.Go(func() {
			_, allocateErr := registry.Allocate("cisco", uint32(index))
			errors <- allocateErr
		})
	}
	waitGroup.Wait()
	close(errors)
	for allocateErr := range errors {
		if allocateErr != nil {
			t.Errorf("Allocate() error = %v", allocateErr)
		}
	}
}

func BenchmarkAllocateCachedVendor(b *testing.B) {
	registry, err := oui.LoadEmbedded()
	if err != nil {
		b.Fatalf("LoadEmbedded() error = %v", err)
	}
	if _, err = registry.Allocate("cisco", 0); err != nil {
		b.Fatalf("prime Allocate() error = %v", err)
	}
	b.ResetTimer()
	for index := range b.N {
		if _, err = registry.Allocate("cisco", uint32(index&0xffffff)); err != nil {
			b.Fatalf("Allocate() error = %v", err)
		}
	}
}

// A vendor search is a substring match that keeps the lowest prefix, so a short
// name can land inside an unrelated organization's: "apc" first matches
// Adapcom and "axis" Galaxis. A discovery tool that looks the MAC up then files
// the device under that stranger's name.
func TestEmbeddedRegistryAllocatesShortVendorNamesTheirOwnPrefix(t *testing.T) {
	registry, err := oui.LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded() error = %v", err)
	}
	for vendor, want := range map[string]string{
		"apc":  "AMERICAN POWER CONVERSION CORP",
		"axis": "Axis Communications AB",
	} {
		mac, allocateErr := registry.Allocate(vendor, 1)
		if allocateErr != nil {
			t.Fatalf("Allocate(%q) error = %v", vendor, allocateErr)
		}
		if got, _ := registry.Lookup(mac); got != want {
			t.Errorf("Allocate(%q) = %s, registered to %q; want %q", vendor, mac, got, want)
		}
	}
}
