package protocols

import (
	"encoding/json"
	"testing"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestRestoredFaultChangesFirstDNSResponse(t *testing.T) {
	original, _, device := newFaultedDNSHandler(t)
	if err := original.SetDeviceFault(device.Name, devicestate.FaultDNSNXDomain, 1); err != nil {
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
	recovered, handler, restoredDevice := newFaultedDNSHandler(t)
	if err = recovered.RestoreDeviceStates(persisted); err != nil {
		t.Fatal(err)
	}
	query := &layers.DNS{ID: 42, Questions: []layers.DNSQuestion{{
		Name: []byte("host.example."), Type: layers.DNSTypeA, Class: layers.DNSClassIN,
	}}}
	response := handler.buildDNSResponse(query, restoredDevice, 0, 1)
	if response.ResponseCode != layers.DNSResponseCodeNXDomain || len(response.Answers) != 0 {
		t.Fatalf("first response ignored restored fault: %+v", response)
	}
	if err = recovered.SetDeviceFault(restoredDevice.Name, devicestate.FaultDNSNXDomain, 0); err != nil {
		t.Fatal(err)
	}
	response = handler.buildDNSResponse(query, restoredDevice, 0, 2)
	if response.ResponseCode != layers.DNSResponseCodeNoErr || len(response.Answers) != 1 {
		t.Fatalf("clearing restored fault did not restore healthy answer: %+v", response)
	}
}
