package protocols

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestSyslogClassifiesLinkAndFaultTransitions(t *testing.T) {
	tests := []struct {
		kind      devicestate.EventKind
		up        bool
		priority  int
		messageID string
	}{
		{devicestate.EventInterfaceUpdated, false, 132, "LINK_DOWN"},
		{devicestate.EventInterfaceUpdated, true, 133, "LINK_UP"},
		{devicestate.EventFaultUpdated, false, 132, "FAULT_UPDATED"},
		{devicestate.EventDeviceFaultUpdated, false, 132, "FAULT_UPDATED"},
		{devicestate.EventFaultCleared, false, 133, "FAULT_CLEARED"},
		{devicestate.EventDeviceFaultCleared, false, 133, "FAULT_CLEARED"},
	}
	for _, tt := range tests {
		t.Run(string(tt.kind)+tt.messageID, func(t *testing.T) {
			event := devicestate.Event{
				Kind:    tt.kind,
				Version: 42,
				Target:  "Gi0/1",
				Timestamp: time.Date(
					2026,
					time.September,
					10,
					7,
					0,
					0,
					123456789,
					time.FixedZone("test", -5*60*60),
				),
				Interface:         &devicestate.Interface{OperUp: tt.up},
				PreviousInterface: &devicestate.Interface{OperUp: !tt.up},
			}
			want := fmt.Sprintf(
				"<%d>1 2026-09-10T12:00:00.123456Z COS-CORE-SW01 niac - %s - version=42 kind=%s target=\"Gi0/1\"",
				tt.priority,
				tt.messageID,
				tt.kind,
			)
			if got := formatSyslog("COS-CORE-SW01", event); got != want {
				t.Fatalf("syslog = %q, want %q", got, want)
			}
		})
	}
}

func TestSyslogIgnoresUnrelatedStateChanges(t *testing.T) {
	kinds := []devicestate.EventKind{
		devicestate.EventNetworkInstalled, devicestate.EventIdentityUpdated,
		devicestate.EventStartupSaved, devicestate.EventStartupReloaded,
		devicestate.EventStartupErased, devicestate.EventAuthoredReset,
		devicestate.EventCheckpointSaved, devicestate.EventCheckpointRestored,
		devicestate.EventVLANUpdated, devicestate.EventRouterUpdated, devicestate.EventRouteUpdated,
		devicestate.EventInterfaceUpdated, "unknown",
	}
	for _, kind := range kinds {
		t.Run(string(kind), func(t *testing.T) {
			if got := formatSyslog("edge-1", devicestate.Event{Kind: kind}); got != "" {
				t.Fatalf("unrelated event produced syslog: %q", got)
			}
		})
	}
	event := devicestate.Event{
		Kind:              devicestate.EventInterfaceUpdated,
		Interface:         &devicestate.Interface{OperUp: true, Description: "new description"},
		PreviousInterface: &devicestate.Interface{OperUp: true},
	}
	if got := formatSyslog("edge-1", event); got != "" {
		t.Fatalf("description-only event produced syslog: %q", got)
	}
}

func TestSyslogTargetIsASCIIAndCannotInjectLines(t *testing.T) {
	event := devicestate.Event{Kind: devicestate.EventFaultCleared, Target: "port\"\\]\r\n医疗"}
	message := formatSyslog("edge-1", event)
	if !strings.HasSuffix(message, `target="port\"\\]\r\n\u533b\u7597"`) {
		t.Fatalf("target is not safely quoted: %q", message)
	}
	for _, char := range message {
		if char < ' ' || char > '~' {
			t.Fatalf("syslog contains non-printing or non-ASCII character: %q", message)
		}
	}
}
