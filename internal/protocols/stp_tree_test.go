package protocols

import (
	"bytes"
	"encoding/binary"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

const (
	dot1dStpDesignatedRootOID = "1.3.6.1.2.1.17.2.5.0"
	dot1dStpRootCostOID       = "1.3.6.1.2.1.17.2.6.0"
	dot1dStpRootPortOID       = "1.3.6.1.2.1.17.2.7.0"
	dot1dStpPortDesigBridge   = "1.3.6.1.2.1.17.2.15.1.8."
	dot1dStpPortDesigCost     = "1.3.6.1.2.1.17.2.15.1.7."
)

func stpBridge(name, mac string, priority uint16, trunks ...config.TrunkPort) config.Device {
	device := config.Device{
		Name: name, Type: "switch", MACAddress: mustMAC(mac),
		STPConfig:  &config.STPConfig{Enabled: true, BridgePriority: priority},
		TrunkPorts: trunks,
		SNMPConfig: config.SNMPConfig{Community: "public"},
	}
	for _, trunk := range trunks {
		device.Interfaces = append(device.Interfaces, config.Interface{Name: trunk.Interface})
	}
	return device
}

func trunkTo(local, remote, remoteInterface string) config.TrunkPort {
	return config.TrunkPort{Interface: local, RemoteDevice: remote, RemoteInterface: remoteInterface}
}

// campusBridges is a core, two distribution switches and an access switch
// dual-homed to both, the shape a pack builds; only the core's priority is
// lowered, as a network engineer would.
func campusBridges() []config.Device {
	return []config.Device{
		stpBridge("access", "02:00:00:00:00:01", 0,
			trunkTo("Te1/1/1", "dist2", "Te1/0/1"), trunkTo("Te1/1/2", "dist1", "Te1/0/1")),
		stpBridge("dist2", "02:00:00:00:00:03", 0, trunkTo("Hu1/0/1", "core", "Hu0/0/2")),
		stpBridge("dist1", "02:00:00:00:00:02", 0, trunkTo("Hu1/0/1", "core", "Hu0/0/1")),
		stpBridge("core", "02:00:00:00:00:09", 24576),
	}
}

func TestElectSpanningTree(t *testing.T) {
	type want struct {
		root, upstream string
		cost           uint32
		rootInterface  string
	}
	tests := []struct {
		name    string
		devices []config.Device
		want    map[string]want
	}{
		{
			name:    "the lowest priority is root and the rest take their cheapest path to it",
			devices: campusBridges(),
			want: map[string]want{
				"core":  {root: "core"},
				"dist1": {root: "core", upstream: "core", cost: 4, rootInterface: "Hu1/0/1"},
				"dist2": {root: "core", upstream: "core", cost: 4, rootInterface: "Hu1/0/1"},
				// Equal cost through both: the lower upstream bridge ID wins.
				"access": {root: "core", upstream: "dist1", cost: 8, rootInterface: "Te1/1/2"},
			},
		},
		{
			name: "equal priorities fall to the lower MAC, and a trunk authored on one end is a link",
			devices: []config.Device{
				stpBridge("high", "02:00:00:00:00:0f", 0),
				stpBridge("low", "02:00:00:00:00:01", 0, trunkTo("Gi0/1", "high", "Gi0/2")),
			},
			want: map[string]want{
				"low":  {root: "low"},
				"high": {root: "low", upstream: "low", cost: 4, rootInterface: "Gi0/2"},
			},
		},
		{
			name: "unlinked bridges, FDB-only trunks and trunks to non-STP devices each stand alone",
			devices: []config.Device{
				stpBridge("east", "02:00:00:00:00:01", 0,
					config.TrunkPort{Interface: "Gi0/1", RemoteDevice: "west", RemoteInterface: "Gi0/1", FDBOnly: true},
					trunkTo("Gi0/2", "router", "Gi0/0")),
				stpBridge("west", "02:00:00:00:00:02", 0),
				{Name: "router", Type: "router", MACAddress: mustMAC("02:00:00:00:00:03")},
			},
			want: map[string]want{"east": {root: "east"}, "west": {root: "west"}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			byName := make(map[string]*config.Device)
			for i := range test.devices {
				byName[test.devices[i].Name] = &test.devices[i]
			}
			positions := electSpanningTree(test.devices)
			if len(positions) != len(test.want) {
				t.Fatalf("placed %d bridges, want %d", len(positions), len(test.want))
			}
			for name, expected := range test.want {
				got := positions[byName[name]]
				wanted := stpPosition{
					root: stpBridgeID(byName[expected.root]), cost: expected.cost,
					rootInterface: expected.rootInterface,
				}
				if expected.upstream != "" {
					wanted.upstream = stpBridgeID(byName[expected.upstream])
				}
				if got != wanted {
					t.Errorf("%s = %+v, want %+v", name, got, wanted)
				}
			}
		})
	}
}

func TestConfigBPDUCarriesElectedTree(t *testing.T) {
	cfg := &config.Config{Devices: campusBridges()}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	access, core := &cfg.Devices[0], &cfg.Devices[3]

	if err := stack.stpHandler.SendConfigBPDU(access); err != nil {
		t.Fatal(err)
	}
	frame := (<-stack.sendQueue).Buffer
	// Ethernet (14) + LLC (3) + protocol, version, type and flags (5).
	const rootOffset = 22
	root := binary.BigEndian.Uint64(frame[rootOffset:])
	cost := binary.BigEndian.Uint32(frame[rootOffset+8:])
	bridge := binary.BigEndian.Uint64(frame[rootOffset+12:])
	if root != stpBridgeID(core) || cost != 8 || bridge != stpBridgeID(access) {
		t.Errorf("BPDU root=%016x cost=%d bridge=%016x, want root %016x cost 8 bridge %016x",
			root, cost, bridge, stpBridgeID(core), stpBridgeID(access))
	}
}

func TestSpanningTreeReachesDot1dStp(t *testing.T) {
	cfg := &config.Config{Devices: campusBridges()}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	agent := stack.snmpAgents[&cfg.Devices[0]].baseAgent
	get := func(oid string) any {
		t.Helper()
		value, err := agent.HandleGet(oid)
		if err != nil {
			t.Fatalf("get %s: %v", oid, err)
		}
		return value.Value
	}

	coreID := binary.BigEndian.AppendUint64(nil, stpBridgeID(&cfg.Devices[3]))
	if root, ok := get(dot1dStpDesignatedRootOID).([]byte); !ok || !bytes.Equal(root, coreID) {
		t.Errorf("dot1dStpDesignatedRoot = %x, want the core %x", root, coreID)
	}
	if cost := get(dot1dStpRootCostOID); cost != 8 {
		t.Errorf("dot1dStpRootCost = %v, want 8", cost)
	}
	port, ok := get(dot1dStpRootPortOID).(int)
	if !ok || port == 0 {
		t.Fatalf("dot1dStpRootPort = %v, want the bridge port of the uplink to dist1", port)
	}
	suffix := binary.BigEndian.AppendUint64(nil, stpBridgeID(&cfg.Devices[2]))
	if bridge, _ := get(dot1dStpPortDesigBridge + strconv.Itoa(port)).([]byte); !bytes.Equal(bridge, suffix) {
		t.Errorf("root port designated bridge = %x, want dist1 %x", bridge, suffix)
	}
	if cost := get(dot1dStpPortDesigCost + strconv.Itoa(port)); cost != 4 {
		t.Errorf("root port designated cost = %v, want dist1's 4", cost)
	}
}

func TestSTPOriginationFollowsHelloTime(t *testing.T) {
	devices := []config.Device{
		stpBridge("fast", "02:00:00:00:00:01", 0),
		stpBridge("slow", "02:00:00:00:00:02", 0),
		stpBridge("off", "02:00:00:00:00:03", 0),
		{Name: "unauthored", Type: "switch", MACAddress: mustMAC("02:00:00:00:00:04")},
	}
	devices[1].STPConfig.HelloTime = 4
	devices[2].STPConfig.Enabled = false
	stack := NewStack(nil, &config.Config{Devices: devices}, logging.NewDebugConfig(0))

	start := time.Now()
	var due map[*config.Device]time.Time
	for _, step := range []struct {
		after time.Duration
		want  []string
	}{
		{0, []string{"fast", "slow"}},
		{time.Second, nil},
		{2 * time.Second, []string{"fast"}},
		{3 * time.Second, nil},
		{4 * time.Second, []string{"fast", "slow"}},
	} {
		due = stack.stpHandler.sendDueBPDUs(start.Add(step.after), due)
		var got []string
		for len(stack.sendQueue) > 0 {
			got = append(got, (<-stack.sendQueue).Device.(*config.Device).Name)
		}
		if !sameNames(got, step.want) {
			t.Errorf("at +%v sent %v, want %v", step.after, got, step.want)
		}
	}
}

// Ticks arrive a few milliseconds late or early. Scheduling from the send time
// turned a late tick followed by an early one into a skipped hello, a 3 s gap
// on the wire.
func TestSTPOriginationIgnoresTickJitter(t *testing.T) {
	stack := NewStack(nil, &config.Config{Devices: []config.Device{
		stpBridge("switch", "02:00:00:00:00:01", 0),
	}}, logging.NewDebugConfig(0))

	start := time.Now()
	var due map[*config.Device]time.Time
	for _, after := range []time.Duration{
		0, 2*time.Second + 10*time.Millisecond, 4*time.Second + 2*time.Millisecond, 6 * time.Second,
	} {
		due = stack.stpHandler.sendDueBPDUs(start.Add(after), due)
		if len(stack.sendQueue) != 1 {
			t.Fatalf("at +%v queued %d BPDUs, want the one due", after, len(stack.sendQueue))
		}
		<-stack.sendQueue
	}
}

// A BPDU on the wire, the stack's own among them, must not change what the
// simulated bridges originate: its timers are in 1/256 s.
func TestReceivedBPDUKeepsOriginationTimers(t *testing.T) {
	cfg := &config.Config{Devices: campusBridges()}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	access := &cfg.Devices[0]
	if err := stack.stpHandler.SendConfigBPDU(access); err != nil {
		t.Fatal(err)
	}
	own := <-stack.sendQueue
	stack.stpHandler.HandlePacket(&Packet{Buffer: own.Buffer, Length: own.Length})

	_, hello, maxAge, forwardDelay := stack.stpHandler.STPGetSTPParams(access)
	if hello != DefaultHelloTime || maxAge != DefaultMaxAge || forwardDelay != DefaultForwardDelay {
		t.Errorf("after a received BPDU: hello=%d maxAge=%d forwardDelay=%d, want the defaults %d/%d/%d",
			hello, maxAge, forwardDelay, DefaultHelloTime, DefaultMaxAge, DefaultForwardDelay)
	}
}

func sameNames(got, want []string) bool {
	slices.Sort(got)
	slices.Sort(want)
	return slices.Equal(got, want)
}
