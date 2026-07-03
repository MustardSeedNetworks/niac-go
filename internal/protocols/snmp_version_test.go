package protocols

import (
	"testing"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
)

// TestSNMPPeekVersion checks raw-datagram version detection used to route v3
// traffic to the USM engine. Each fixture is a real marshalled SNMP message.
func TestSNMPPeekVersion(t *testing.T) {
	marshal := func(v gosnmp.SnmpVersion, sp gosnmp.SnmpV3SecurityParameters) []byte {
		pkt := &gosnmp.SnmpPacket{
			Version:            v,
			Community:          "public",
			PDUType:            gosnmp.GetRequest,
			SecurityModel:      gosnmp.UserSecurityModel,
			SecurityParameters: sp,
			Variables:          []gosnmp.SnmpPDU{{Name: ".1.3.6.1.2.1.1.1.0", Type: gosnmp.Null}},
		}
		raw, err := pkt.MarshalMsg()
		if err != nil {
			t.Fatalf("marshal v%d: %v", v, err)
		}
		return raw
	}

	cases := []struct {
		name string
		raw  []byte
		want gosnmp.SnmpVersion
		ok   bool
	}{
		{"v1", marshal(gosnmp.Version1, nil), gosnmp.Version1, true},
		{"v2c", marshal(gosnmp.Version2c, nil), gosnmp.Version2c, true},
		{"v3", marshal(gosnmp.Version3, &gosnmp.UsmSecurityParameters{UserName: "u"}), gosnmp.Version3, true},
		{"empty", []byte{}, 0, false},
		{"not-sequence", []byte{0x02, 0x01, 0x00}, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := snmpPeekVersion(tc.raw)
			if ok != tc.ok || (ok && got != tc.want) {
				t.Errorf("snmpPeekVersion = (%d,%v), want (%d,%v)", got, ok, tc.want, tc.ok)
			}
		})
	}
}

// TestBuildResponseRejectsSNMPv3 reproduces the live simulator crash: a CyberScope
// (and other discovery tools) send SNMPv3 probes. gosnmp's MarshalMsg dispatches
// a Version3 response to marshalV3, which dereferences the SecurityParameters the
// simulator never populates → SIGSEGV, taking down the whole sim (every device
// goes dark → "0 OIDs"). buildResponse must refuse unsupported versions instead.
func TestBuildResponseRejectsSNMPv3(t *testing.T) {
	h := &SNMPHandler{}
	dev := &config.Device{Name: "sw", Type: "switch", Properties: map[string]string{}}
	agent := snmp.NewAgentWithCommunity(dev, "public", 0)

	req := &gosnmp.SnmpPacket{
		Version:   gosnmp.Version3,
		PDUType:   gosnmp.GetRequest,
		Variables: []gosnmp.SnmpPDU{{Name: ".1.3.6.1.2.1.1.1.0", Type: gosnmp.Null}},
	}

	// Must not panic; must decline to answer.
	payload, err := h.buildResponse(agent, req)
	if err == nil {
		t.Fatal("expected an error for an SNMPv3 request, got nil")
	}
	if payload != nil {
		t.Errorf("expected nil payload for unsupported version, got %d bytes", len(payload))
	}
}

// TestBuildResponseAcceptsV2c is the companion happy path: v2c still marshals.
func TestBuildResponseAcceptsV2c(t *testing.T) {
	h := &SNMPHandler{}
	dev := &config.Device{Name: "sw", Type: "switch", Properties: map[string]string{}}
	agent := snmp.NewAgentWithCommunity(dev, "public", 0)

	req := &gosnmp.SnmpPacket{
		Version:   gosnmp.Version2c,
		Community: "public",
		PDUType:   gosnmp.GetRequest,
		Variables: []gosnmp.SnmpPDU{{Name: ".1.3.6.1.2.1.1.5.0", Type: gosnmp.Null}},
	}

	payload, err := h.buildResponse(agent, req)
	if err != nil {
		t.Fatalf("v2c buildResponse: %v", err)
	}
	if len(payload) == 0 {
		t.Error("expected a non-empty v2c response")
	}
}
