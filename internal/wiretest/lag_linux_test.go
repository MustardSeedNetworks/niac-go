//go:build linux && integration

package wiretest_test

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/gosnmp/gosnmp"
)

// P5-9: a manager finds a LAG by its ifType and finds what the LAG carries in
// ifStackTable. Without both, a port-channel is a topology edge and nothing a
// poller can see. The member ports and an unbundled port are asserted too,
// because an agent that typed every port 161 would pass the aggregate alone.
const (
	oidIfType        = ".1.3.6.1.2.1.2.2.1.3"
	oidIfStackStatus = ".1.3.6.1.2.1.31.1.2.1.3"
	ifTypeEthernet   = 6
	ifTypeLAG        = 161
	stackRowActive   = 1
)

func TestPortChannelReportsAsALAGOverItsMembersOnTheWire(t *testing.T) {
	authored := startAuthoredFile(t, "lag-wire.yaml", "wiretest-lag")
	client := dialDevice(t, authored, "wire-lag-sw-01")

	aggregate := interfaceIndex(t, client, "port-channel1")
	members := []string{interfaceIndex(t, client, "Ethernet1/1"), interfaceIndex(t, client, "Ethernet1/2")}
	uplink := interfaceIndex(t, client, "Ethernet1/48")

	wantType := map[string]int{
		aggregate:  ifTypeLAG,
		members[0]: ifTypeEthernet,
		members[1]: ifTypeEthernet,
		uplink:     ifTypeEthernet,
	}
	for _, index := range slices.Sorted(maps.Keys(wantType)) {
		got, err := client.Get([]string{oidIfType + "." + index})
		if err != nil || len(got.Variables) != 1 {
			t.Fatalf("GET ifType.%s: %v", index, err)
		}
		if value := gosnmp.ToBigInt(got.Variables[0].Value).Int64(); value != int64(wantType[index]) {
			t.Errorf("ifType.%s = %d, want %d", index, value, wantType[index])
		}
	}

	want := []string{
		aggregate + "." + members[0], aggregate + "." + members[1],
		"0." + aggregate, members[0] + ".0", members[1] + ".0",
		"0." + uplink, uplink + ".0",
	}
	slices.Sort(want)
	var got []string
	for _, pdu := range walkSubtree(t, client, oidIfStackStatus) {
		row := strings.TrimPrefix(pdu.Name, oidIfStackStatus+".")
		if value := gosnmp.ToBigInt(pdu.Value).Int64(); value != stackRowActive {
			t.Errorf("ifStackStatus.%s = %d, want active", row, value)
		}
		got = append(got, row)
	}
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Fatalf("ifStackTable rows = %v, want %v", got, want)
	}
	t.Logf("port-channel1 = ifIndex %s (ifType %d) over %v; ifStackTable %v",
		aggregate, ifTypeLAG, members, got)
}
