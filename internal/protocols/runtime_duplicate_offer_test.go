package protocols

import (
	"encoding/json"
	"net"
	"net/netip"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestRestoredDuplicateOfferPreservesCanonicalOwnership(t *testing.T) {
	original, _ := isolationPair(t)
	if err := original.SetDeviceAddressFault(
		"a",
		devicestate.FaultDuplicateDHCPOffer,
		netip.MustParseAddr("10.0.0.3"),
	); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(original.ExportDeviceStates())
	if err != nil {
		t.Fatal(err)
	}
	var persisted map[string]devicestate.State
	if err = json.Unmarshal(encoded, &persisted); err != nil {
		t.Fatal(err)
	}
	recovered, cfg := isolationPair(t)
	if err = recovered.RestoreDeviceStates(persisted); err != nil {
		t.Fatal(err)
	}
	client := net.HardwareAddr{2, 0, 0, 0, 1, 1}
	sendIsolationDHCP(t, recovered, dhcpDiscover(client))
	assertDuplicateOffers(t, recovered, map[string]string{"10.0.0.2": "10.0.0.3", "10.0.0.3": "10.0.0.110"})
	owners := recovered.devicesForStateIPv4(0, net.ParseIP("10.0.0.3"))
	if len(owners) != 1 || owners[0] != &cfg.Devices[1] {
		t.Fatalf("restored fault changed canonical ownership: %v", owners)
	}
	if len(recovered.dhcpHandlers[&cfg.Devices[0]].leases) != 0 {
		t.Fatal("restored conflict created an ordinary lease")
	}
	if err = recovered.ClearDeviceFault("a", devicestate.FaultDuplicateDHCPOffer); err != nil {
		t.Fatal(err)
	}
	sendIsolationDHCP(t, recovered, dhcpDiscover(client))
	assertDuplicateOffers(t, recovered, map[string]string{"10.0.0.2": "10.0.0.100", "10.0.0.3": "10.0.0.110"})
}
