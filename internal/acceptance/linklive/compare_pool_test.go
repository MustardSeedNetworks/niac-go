package linklive_test

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/acceptance/linklive"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// R-A2: on the 2026-10-14 captures every pack failed the attachment-pool ports
// its testers sat on (GigabitEthernet1/0/43 upwards on the first access
// switch): authored down, rendered Up, because placing a tester brings its
// pool port up.

func TestCompareAcceptsTesterOnPoolPort(t *testing.T) {
	cases := []struct {
		name     string
		pool     bool
		observed string
		want     bool
	}{
		{"pool port with a tester", true, "Up", false},
		{"pool port without a tester", true, "Down", false},
		{"other port authored down", false, "Up", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			authored := authoredPair(t)
			authored.Devices[0].Interfaces[0].Status = "down"
			authored.Devices[0].Interfaces[0].AttachmentPool = tc.pool
			observed := observedPair("Switch", "Full", "100 Gb")
			observed.Hosts[0].Interfaces[0].Interface.Status = tc.observed

			got := false
			for _, finding := range linklive.Compare(authored, observed) {
				got = got || finding.Kind == linklive.FindingInterfaceStatusConflict
			}
			if got != tc.want {
				t.Fatalf("status finding = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFromConfigMarksEveryPackPoolPort(t *testing.T) {
	for _, pack := range scenario.Packs() {
		t.Run(pack.ID, func(t *testing.T) {
			result, err := scenario.Generate(pack.Request)
			if err != nil {
				t.Fatal(err)
			}
			marked := markedPoolPorts(linklive.FromConfig(result.Config))
			pool := authoredPoolPorts(result.Config.Attachments)
			for port := range pool {
				if !marked[port] {
					t.Errorf("%s is not marked as a pool port", port)
				}
			}
			if len(pool) == 0 || len(marked) != len(pool) {
				t.Fatalf("pool ports = %d, marked = %d", len(pool), len(marked))
			}
		})
	}
}

func markedPoolPorts(snapshot linklive.AuthoredSnapshot) map[string]bool {
	marked := make(map[string]bool)
	for _, device := range snapshot.Devices {
		for _, iface := range device.Interfaces {
			if iface.AttachmentPool {
				marked[device.Name+" "+iface.Name] = true
			}
		}
	}
	return marked
}

func authoredPoolPorts(attachments []config.LogicalAttachment) map[string]bool {
	pool := make(map[string]bool)
	for _, attachment := range attachments {
		for _, at := range attachment.At {
			for _, port := range at.Ports {
				pool[at.Device+" "+port] = true
			}
		}
	}
	return pool
}
