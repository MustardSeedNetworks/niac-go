package protocols

import (
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// add_mibs precedence (plan row F8).
//
// initSNMPAgent loads every configured walk first and applies add_mibs after,
// so an add_mibs entry naming an OID the capture already carries replaces it.
// That is the authoring rule -- a walk is the base, add_mibs is the override --
// and docs/design/2026-09-replay-fidelity-contract.md states it. Nothing
// asserted it, so the order was free to flip in either direction: applying
// add_mibs first would have made an override silently do nothing on exactly the
// devices it was written for.
const precedenceWalk = `.1.3.6.1.2.1.1.1.0 = STRING: "from the capture"
.1.3.6.1.2.1.1.6.0 = STRING: "capture location"
`

func TestAddMibsOverrideWalkRows(t *testing.T) {
	dir := t.TempDir()
	walkPath := filepath.Join(dir, "precedence.walk")
	if err := os.WriteFile(walkPath, []byte(precedenceWalk), 0o600); err != nil {
		t.Fatalf("write walk: %v", err)
	}

	cfg := &config.Config{Devices: []config.Device{{
		Name:        "precedence-sw1",
		Type:        "switch",
		MACAddress:  mustTestMAC(t, "02:00:00:00:0f:09"),
		IPAddresses: []net.IP{net.ParseIP("192.0.2.90")},
		SNMPConfig: config.SNMPConfig{
			Community: "public",
			WalkFile:  walkPath,
			WalkFiles: []string{walkPath},
			AddMibs: []config.AddMib{
				// Names a row the walk carries.
				{OID: "1.3.6.1.2.1.1.6.0", Type: "STRING", Value: "fixed(Wiring Closet A)"},
				// Names one it does not.
				{OID: "1.3.6.1.4.1.9.9.9999.2.0", Type: "STRING", Value: "fixed(added)"},
			},
		},
	}}}

	agent := NewStack(nil, cfg, logging.NewDebugConfig(0)).snmpAgents[&cfg.Devices[0]].baseAgent

	overridden, err := agent.HandleGet("1.3.6.1.2.1.1.6.0")
	if err != nil {
		t.Fatalf("get sysLocation: %v", err)
	}
	if got := valueString(overridden.Value); got != "Wiring Closet A" {
		t.Errorf("sysLocation = %q, want the add_mibs value; add_mibs no longer wins over the walk", got)
	}

	added, err := agent.HandleGet("1.3.6.1.4.1.9.9.9999.2.0")
	if err != nil {
		t.Fatalf("get the added OID: %v", err)
	}
	if got := valueString(added.Value); got != "added" {
		t.Errorf("added OID = %q, want %q", got, "added")
	}

	// The row add_mibs did not name is the capture's, untouched.
	kept, err := agent.HandleGet("1.3.6.1.2.1.1.1.0")
	if err != nil {
		t.Fatalf("get sysDescr: %v", err)
	}
	if got := valueString(kept.Value); got != "from the capture" {
		t.Errorf("sysDescr = %q, want the captured value", got)
	}
}

func valueString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []byte:
		return string(typed)
	default:
		return ""
	}
}
