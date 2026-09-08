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
		{"the LLDP local port table is not a neighbour table", trunked, ".1.0.8802.1.1.2.1.3.7.1.3.1", BucketKept},
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
