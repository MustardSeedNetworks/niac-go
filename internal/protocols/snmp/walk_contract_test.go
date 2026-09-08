package snmp

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// TestWalkContractClassify covers each signed substitution, in both the
// authored and unauthored shape where the substitution is conditional, and
// with and without the leading dot walk files carry.
func TestWalkContractClassify(t *testing.T) {
	bare := WalkContract{authoredIfIndexes: map[string]struct{}{}}
	authored := WalkContract{
		authoredSysDescr:    true,
		authoredSysContact:  true,
		authoredSysLocation: true,
		authoredIfIndexes:   map[string]struct{}{},
	}
	trunked := WalkContract{ownsTopology: true, authoredIfIndexes: map[string]struct{}{}}
	addressed := WalkContract{
		authoredMAC:         true,
		authoredEthernetMAC: true,
		authoredIfIndexes:   map[string]struct{}{},
	}
	longMAC := WalkContract{authoredMAC: true, authoredIfIndexes: map[string]struct{}{}}

	cases := []struct {
		name     string
		contract WalkContract
		oid      string
		want     Bucket
	}{
		{"sysName is always authored", bare, ".1.3.6.1.2.1.1.5.0", BucketAuthored},
		{"sysName without the dot", bare, "1.3.6.1.2.1.1.5.0", BucketAuthored},
		{"unauthored sysDescr keeps the capture", bare, ".1.3.6.1.2.1.1.1.0", BucketKept},
		{"authored sysDescr wins", authored, ".1.3.6.1.2.1.1.1.0", BucketAuthored},
		{"unauthored sysContact keeps the capture", bare, ".1.3.6.1.2.1.1.4.0", BucketKept},
		{"authored sysContact wins", authored, ".1.3.6.1.2.1.1.4.0", BucketAuthored},
		{"unauthored sysLocation keeps the capture", bare, ".1.3.6.1.2.1.1.6.0", BucketKept},
		{"authored sysLocation wins", authored, ".1.3.6.1.2.1.1.6.0", BucketAuthored},
		{"sysObjectID is never substituted", authored, ".1.3.6.1.2.1.1.2.0", BucketKept},

		{"ip group is live", bare, ".1.3.6.1.2.1.4.3.0", BucketLive},
		{"icmp group is live", bare, ".1.3.6.1.2.1.5.1.0", BucketLive},
		{"tcp group is live", bare, ".1.3.6.1.2.1.6.10.0", BucketLive},
		{"udp group is live", bare, ".1.3.6.1.2.1.7.1.0", BucketLive},
		{"egp group is live", bare, ".1.3.6.1.2.1.8.1.0", BucketLive},
		{"snmp group is the agent's own", bare, ".1.3.6.1.2.1.11.1.0", BucketLive},
		{"a sibling of the ip root is not live", bare, ".1.3.6.1.2.1.44.1.0", BucketKept},

		{"LLDP neighbours drop under trunk_ports", trunked, ".1.0.8802.1.1.2.1.4.1.1.9.1.1", BucketTopology},
		{"LLDP neighbour root", trunked, "1.0.8802.1.1.2.1.4", BucketTopology},
		{"CDP cache drops under trunk_ports", trunked, ".1.3.6.1.4.1.9.9.23.1.2.1.1.6.1.1", BucketTopology},
		{"bridge FDB drops under trunk_ports", trunked, ".1.3.6.1.2.1.17.4.3.1.2.1.2.3.4.5.6", BucketTopology},
		{"VLAN FDB counters drop under trunk_ports", trunked, ".1.3.6.1.2.1.17.7.1.2.1.1.2.210", BucketTopology},
		{"VLAN FDB drops under trunk_ports", trunked, ".1.3.6.1.2.1.17.7.1.2.2.1.2.210.1.2.3.4.5.6", BucketTopology},
		{"without trunk_ports the walk's neighbours stand", bare, ".1.0.8802.1.1.2.1.4.1.1.9.1.1", BucketKept},
		{"the LLDP local port table goes with them", trunked, ".1.0.8802.1.1.2.1.3.7.1.3.1", BucketTopology},
		{"without trunk_ports the walk's local port table stands", bare, ".1.0.8802.1.1.2.1.3.7.1.3.1", BucketKept},
		{"the LLDP chassis ID is identity, not the port table", bare, ".1.0.8802.1.1.2.1.3.2.0", BucketKept},

		{"ifPhysAddress takes the authored MAC", addressed, ".1.3.6.1.2.1.2.2.1.6.1", BucketAuthored},
		{"the bridge address takes the authored MAC", addressed, ".1.3.6.1.2.1.17.1.1.0", BucketAuthored},
		{"the LLDP chassis ID takes the authored MAC", addressed, ".1.0.8802.1.1.2.1.3.2.0", BucketAuthored},
		{"the serial number is derived from it", addressed, ".1.3.6.1.2.1.47.1.1.1.1.11.1", BucketAuthored},
		{"a device with no MAC keeps the captured address", bare, ".1.3.6.1.2.1.2.2.1.6.1", BucketKept},
		{"a device with no MAC keeps the captured serial", bare, ".1.3.6.1.2.1.47.1.1.1.1.11.1", BucketKept},
		{"a neighbouring ENTITY-MIB column is untouched", addressed, ".1.3.6.1.2.1.47.1.1.1.1.10.1", BucketKept},
		{"a non-Ethernet MAC cannot stand in for an address", longMAC, ".1.3.6.1.2.1.2.2.1.6.1", BucketKept},
		{"but it still seeds the serial", longMAC, ".1.3.6.1.2.1.47.1.1.1.1.11.1", BucketAuthored},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := testCase.contract.Classify(testCase.oid); got != testCase.want {
				t.Errorf("Classify(%s) = %s, want %s", testCase.oid, got, testCase.want)
			}
		})
	}
}

// TestWalkContractClassifiesAuthoredInterfaceRows: the interface substitution
// applies only to rows the scenario authored, and splits within one row —
// configuration columns come from the scenario, counters from this run.
func TestWalkContractClassifiesAuthoredInterfaceRows(t *testing.T) {
	contract := WalkContract{authoredIfIndexes: map[string]struct{}{"1": {}}}

	cases := []struct {
		name string
		oid  string
		want Bucket
	}{
		{"ifSpeed on an authored row", ifSpeed + ".1", BucketAuthored},
		{"ifAlias on an authored row", ifAlias + ".1", BucketAuthored},
		{"ifOperStatus on an authored row", ifOperStatus + ".1", BucketAuthored},
		{"ifInOctets on an authored row", ifInOctets + ".1", BucketLive},
		{"ifHCOutOctets on an authored row", ifHCOutOctets + ".1", BucketLive},
		{"ifDescr is never overwritten", ifDescr + ".1", BucketKept},
		{"ifSpeed on a row nobody authored", ifSpeed + ".7", BucketKept},
		{"ifInOctets on a row nobody authored", ifInOctets + ".7", BucketKept},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := contract.Classify(testCase.oid); got != testCase.want {
				t.Errorf("Classify(%s) = %s, want %s", testCase.oid, got, testCase.want)
			}
		})
	}
}

// TestWalkContractDropsFromWalk: only the substitutions that do not need the
// walk's own IF-MIB indexes are decided during the load. The authored
// interface rows are loaded and then overwritten, so the loader keeps them.
func TestWalkContractDropsFromWalk(t *testing.T) {
	contract := WalkContract{
		authoredSysDescr:  true,
		ownsTopology:      true,
		authoredIfIndexes: map[string]struct{}{"1": {}},
	}

	drops := []string{
		".1.3.6.1.2.1.1.5.0",            // sysName
		".1.3.6.1.2.1.1.1.0",            // authored sysDescr
		".1.3.6.1.2.1.11.1.0",           // snmp group
		".1.3.6.1.2.1.4.3.0",            // ip group
		".1.0.8802.1.1.2.1.4.1.1.9.1.1", // LLDP neighbour under trunk_ports
	}
	keeps := []string{
		ifSpeed + ".1",       // authored, but overwritten after the load
		ifInOctets + ".1",    // live, but registered after the load
		".1.3.6.1.2.1.1.4.0", // sysContact this device never authored
		ifDescr + ".1",       // ordinary walk content
	}

	for _, oid := range drops {
		if !contract.DropsFromWalk(oid) {
			t.Errorf("DropsFromWalk(%s) = false, want true", oid)
		}
	}
	for _, oid := range keeps {
		if contract.DropsFromWalk(oid) {
			t.Errorf("DropsFromWalk(%s) = true, want false", oid)
		}
	}
}

// TestAgentWalkContractResolvesAuthoredInterfaces drives the contract through
// a real Agent: the interface substitution has to find the ifIndex the MIB
// gave each authored interface, and must not claim rows the scenario never
// authored.
func TestAgentWalkContractResolvesAuthoredInterfaces(t *testing.T) {
	device := createTestDevice()
	device.Interfaces = []config.Interface{{Name: "eth0", Speed: 1000}}
	agent := NewAgent(device, 0)

	contract := agent.WalkContract()
	index, ok := agent.ifIndexForInterface("eth0")
	if !ok {
		t.Fatal("the agent gave the authored interface no ifIndex")
	}
	if _, known := contract.authoredIfIndexes[index]; !known {
		t.Fatalf("contract missed authored ifIndex %s, has %v", index, contract.authoredIfIndexes)
	}
	if got := contract.Classify(ifSpeed + "." + index); got != BucketAuthored {
		t.Errorf("Classify(ifSpeed.%s) = %s, want %s", index, got, BucketAuthored)
	}
	if got := contract.Classify(ifInOctets + "." + index); got != BucketLive {
		t.Errorf("Classify(ifInOctets.%s) = %s, want %s", index, got, BucketLive)
	}
	if got := contract.Classify(ifSpeed + ".99"); got != BucketKept {
		t.Errorf("Classify(ifSpeed.99) = %s, want %s", got, BucketKept)
	}
}

// TestWalkContractClassifiesBridgePortRows: finding 1 of the F0 audit,
// narrowed by the owner on 2026-09-08 to substitution 5's authored-ifIndex
// scope or substitution 4's trunk_ports gate. A bridge port outside both
// arrives byte-identical.
func TestWalkContractClassifiesBridgePortRows(t *testing.T) {
	authored := WalkContract{
		authoredIfIndexes:   map[string]struct{}{"1": {}},
		authoredBridgePorts: map[string]struct{}{"1": {}},
	}
	trunked := WalkContract{ownsTopology: true, authoredIfIndexes: map[string]struct{}{}}

	cases := []struct {
		name     string
		contract WalkContract
		oid      string
		want     Bucket
	}{
		{"the port index restates the scenario", authored, dot1dTpPort + ".1", BucketAuthored},
		{"max info is the device's MTU", authored, dot1dTpPortMaxInfo + ".1", BucketAuthored},
		{"in frames are this run's", authored, dot1dTpPortInFrames + ".1", BucketLive},
		{"out frames are this run's", authored, dot1dTpPortOutFrames + ".1", BucketLive},
		{"in discards are this run's", authored, dot1dTpPortInDiscards + ".1", BucketLive},
		{"an undeclared port keeps the capture", authored, dot1dTpPortMaxInfo + ".9", BucketKept},
		{"trunk_ports covers every port", trunked, dot1dTpPortMaxInfo + ".9", BucketAuthored},
		{"and its counters with them", trunked, dot1dTpPortInFrames + ".9", BucketLive},
		{"the base port table is not the tp port table", authored, dot1dBasePortIfIndex + ".1", BucketKept},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := testCase.contract.Classify(testCase.oid); got != testCase.want {
				t.Errorf("Classify(%s) = %s, want %s", testCase.oid, got, testCase.want)
			}
		})
	}
}

// TestAgentWalkContractResolvesAuthoredBridgePorts: the bridge port index is
// not the ifIndex, so the contract has to walk dot1dBasePortIfIndex to learn
// which ports the scenario's interfaces sit behind. Those rows come from the
// walk, so the contract must be built after the load.
func TestAgentWalkContractResolvesAuthoredBridgePorts(t *testing.T) {
	agent := loadBridgePortWalk(t, func(device *config.Device) {
		device.Interfaces = []config.Interface{{Name: "GigabitEthernet0/2"}}
	})

	contract := agent.WalkContract()
	if _, authored := contract.authoredBridgePorts["2"]; !authored {
		t.Errorf("bridge port 2 not resolved from the authored interface: %v", contract.authoredBridgePorts)
	}
	if _, authored := contract.authoredBridgePorts["1"]; authored {
		t.Error("bridge port 1 claimed for an interface the scenario never authored")
	}
}
