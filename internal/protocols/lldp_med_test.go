package protocols_test

import (
	"net"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

// medPhone is the shape of a real IP phone's advertisement: a class III
// endpoint on a tagged voice VLAN, drawing power, with a full inventory.
func medPhone(mac net.HardwareAddr) *config.Device {
	return &config.Device{
		Name:        "clinic-phone-01",
		Type:        "phone",
		MACAddress:  mac,
		IPAddresses: []net.IP{net.ParseIP("10.20.30.40")},
		LLDPConfig: &config.LLDPConfig{
			Enabled: true,
			MED: &config.LLDPMEDConfig{
				DeviceType: "endpoint_class3",
				NetworkPolicies: []config.LLDPMEDNetworkPolicy{
					{Application: "voice", Tagged: true, VLANID: 110, Priority: 5, DSCP: 46},
					{Application: "voice_signaling", Tagged: true, VLANID: 110, Priority: 3, DSCP: 24},
				},
				Power: &config.LLDPMEDPower{
					DeviceType: "pd", Source: "pse", Priority: "high", ValueTenthWatts: 65,
				},
				Inventory: &config.LLDPMEDInventory{
					HardwareRevision: "1.0",
					FirmwareRevision: "SIP88xx.12-8-1",
					SerialNumber:     "FCH2043E0AB",
					Manufacturer:     "Cisco Systems",
					ModelName:        "CP-8841",
				},
			},
		},
	}
}

// An advertisement is only useful if a decoder that is not ours reads it. The
// per-TLV assertions elsewhere cannot catch a TLV that is well-formed alone and
// still derails a decoder walking the chain in order — which is exactly how the
// malformed CDP Address TLV shipped (#1326).
func TestMEDAdvertisementDecodesEndToEnd(t *testing.T) {
	stack := protocols.NewStack(nil, &config.Config{}, logging.NewDebugConfig(0))
	handler := protocols.NewLLDPHandler(stack)

	mac := net.HardwareAddr{0x00, 0x1A, 0x2B, 0x3C, 0x4D, 0x5F}
	payload := handler.BuildLLDPFrame(medPhone(mac))

	packet := gopacket.NewPacket(lldpFrame(t, payload, mac),
		layers.LayerTypeEthernet, gopacket.Default)
	if err := packet.ErrorLayer(); err != nil {
		t.Fatalf("decoding the advertisement failed: %v", err.Error())
	}

	info := packet.Layer(layers.LayerTypeLinkLayerDiscoveryInfo)
	if info == nil {
		t.Fatal("no LLDP info layer: the MED TLVs derailed the decoder")
	}
	decoded, ok := info.(*layers.LinkLayerDiscoveryInfo)
	if !ok {
		t.Fatal("unexpected info layer type")
	}

	tia, err := decoded.DecodeMedia()
	if err != nil {
		t.Fatalf("DecodeMedia: %v", err)
	}

	if got := tia.MediaCapabilities.Class; got != layers.LLDPMediaClassEndpointIII {
		t.Errorf("device class = %v, want endpoint class III", got)
	}
	if !tia.MediaCapabilities.NetworkPolicy {
		t.Error("capabilities do not claim network policy, but policies were sent")
	}
	if !tia.MediaCapabilities.Inventory {
		t.Error("capabilities do not claim inventory, but inventory was sent")
	}
	if !tia.MediaCapabilities.PowerPD {
		t.Error("capabilities do not claim PD power, but a PD power TLV was sent")
	}

	// gopacket keeps one NetworkPolicy field and each TLV overwrites it, so
	// what survives the decode is the last policy sent. Assert on that one, and
	// count the TLVs separately to prove the first was not dropped.
	if got := tia.NetworkPolicy.ApplicationType; got != layers.LLDPappTypeVoiceSignaling {
		t.Errorf("last application type = %v, want voice signaling", got)
	}
	if !tia.NetworkPolicy.Tagged {
		t.Error("signaling policy is not tagged")
	}
	if got := tia.NetworkPolicy.VLANId; got != 110 {
		t.Errorf("signaling VLAN = %d, want 110", got)
	}
	if got := tia.NetworkPolicy.L2Priority; got != 3 {
		t.Errorf("signaling L2 priority = %d, want 3", got)
	}
	if got := tia.NetworkPolicy.DSCPValue; got != 24 {
		t.Errorf("signaling DSCP = %d, want 24", got)
	}
	if got := countMEDSubtype(decoded, 2); got != 2 {
		t.Errorf("%d network-policy TLVs on the wire, want 2", got)
	}

	// The TLV field is in 0.1 W units, which is what the config names and what
	// NIAC writes; gopacket reports it in milliwatts. 65 tenth-watts is the
	// 6.5 W a class-2 phone draws.
	if got := tia.PowerViaMDI.Value; got != 6500 {
		t.Errorf("power value = %d mW, want 6500 (65 tenth-watts on the wire)", got)
	}

	if got := tia.SerialNumber; got != "FCH2043E0AB" {
		t.Errorf("serial number = %q, want FCH2043E0AB", got)
	}
	if got := tia.Manufacturer; got != "Cisco Systems" {
		t.Errorf("manufacturer = %q, want Cisco Systems", got)
	}
	if got := tia.Model; got != "CP-8841" {
		t.Errorf("model = %q, want CP-8841", got)
	}
}

// A switch or router advertises no MED at all, and must keep decoding cleanly.
func TestDeviceWithoutMEDSendsNoMEDTLVs(t *testing.T) {
	stack := protocols.NewStack(nil, &config.Config{}, logging.NewDebugConfig(0))
	handler := protocols.NewLLDPHandler(stack)

	mac := net.HardwareAddr{0x00, 0x1A, 0x2B, 0x3C, 0x4D, 0x60}
	device := &config.Device{
		Name: "core-sw-01", Type: "switch", MACAddress: mac,
		IPAddresses: []net.IP{net.ParseIP("10.20.0.1")},
		LLDPConfig:  &config.LLDPConfig{Enabled: true},
	}

	packet := gopacket.NewPacket(lldpFrame(t, handler.BuildLLDPFrame(device), mac),
		layers.LayerTypeEthernet, gopacket.Default)
	info := packet.Layer(layers.LayerTypeLinkLayerDiscoveryInfo)
	if info == nil {
		t.Fatal("no LLDP info layer")
	}
	decoded, ok := info.(*layers.LinkLayerDiscoveryInfo)
	if !ok {
		t.Fatal("unexpected info layer type")
	}
	if len(decoded.OrgTLVs) != 0 {
		t.Errorf("a device with no MED block sent %d organizational TLVs", len(decoded.OrgTLVs))
	}
}

// An application NIAC does not know is dropped rather than sent as type 0: a
// receiver reads 0 as malformed and may discard every TLV after it.
func TestUnknownApplicationIsNotAdvertised(t *testing.T) {
	stack := protocols.NewStack(nil, &config.Config{}, logging.NewDebugConfig(0))
	handler := protocols.NewLLDPHandler(stack)

	mac := net.HardwareAddr{0x00, 0x1A, 0x2B, 0x3C, 0x4D, 0x61}
	device := medPhone(mac)
	device.LLDPConfig.MED.NetworkPolicies = []config.LLDPMEDNetworkPolicy{
		{Application: "telepathy", VLANID: 10},
	}

	packet := gopacket.NewPacket(lldpFrame(t, handler.BuildLLDPFrame(device), mac),
		layers.LayerTypeEthernet, gopacket.Default)
	if err := packet.ErrorLayer(); err != nil {
		t.Fatalf("an unknown application broke the frame: %v", err.Error())
	}
	info, ok := packet.Layer(layers.LayerTypeLinkLayerDiscoveryInfo).(*layers.LinkLayerDiscoveryInfo)
	if !ok {
		t.Fatal("no LLDP info layer")
	}
	for _, org := range info.OrgTLVs {
		if org.SubType == 2 {
			t.Error("an unknown application was advertised as a network policy")
		}
	}
}

// countMEDSubtype counts the TIA organizationally specific TLVs of one subtype
// that actually reached the wire.
func countMEDSubtype(info *layers.LinkLayerDiscoveryInfo, subtype uint8) int {
	count := 0
	for _, org := range info.OrgTLVs {
		if org.OUI == layers.IEEEOUI(protocols.TIAOUI) && org.SubType == subtype {
			count++
		}
	}

	return count
}
