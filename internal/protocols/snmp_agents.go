package protocols

import (
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
)

type snmpAgentGroup struct {
	agents    map[string]*snmp.Agent
	telemetry *snmp.ProtocolTelemetry
	state     *devicestate.Store
	baseAgent *snmp.Agent
	// v3 is the device's SNMPv3 authoritative engine (nil when v3 is disabled).
	v3 *snmp.V3Engine
	// v3Agent is the MIB-backing agent used to answer v3 scoped PDUs (the base
	// community agent); v3 has no community, so one agent serves all users.
	v3Agent *snmp.Agent
}

func newSnmpAgentGroup(state *devicestate.Store) *snmpAgentGroup {
	return &snmpAgentGroup{
		agents:    make(map[string]*snmp.Agent),
		telemetry: snmp.NewProtocolTelemetry(),
		state:     state,
	}
}

func (g *snmpAgentGroup) Get(community string) *snmp.Agent {
	if g == nil {
		return nil
	}

	return g.agents[community]
}

func (g *snmpAgentGroup) observer() *snmp.Agent {
	if g == nil {
		return nil
	}
	if g.v3Agent != nil {
		return g.v3Agent
	}
	for _, agent := range g.agents {
		return agent
	}
	return nil
}

func (g *snmpAgentGroup) interfaceIndex(name string) (int, bool) {
	if g == nil || g.baseAgent == nil {
		return 0, false
	}
	return g.baseAgent.InterfaceIndex(name)
}

func (g *snmpAgentGroup) Ensure(community string, device *config.Device, debugLevel int) *snmp.Agent {
	if g == nil {
		return nil
	}

	community = strings.TrimSpace(community)
	if community == "" {
		community = config.DefaultSNMPCommunity
	}

	if agent, ok := g.agents[community]; ok {
		return agent
	}

	agent := snmp.NewAgentWithState(device, g.state, snmp.AgentOptions{
		Community:  community,
		DebugLevel: debugLevel,
		Telemetry:  g.telemetry,
	})
	g.agents[community] = agent

	return agent
}

// ReindexAll eagerly builds every agent's sorted OID index. Called once after a
// device's MIBs are fully loaded (walk files, AddMib, topology) so the sort is
// paid at startup instead of lazily on the first GetNext — where it would run on
// the stack's single decode goroutine and stall a fresh multi-device discovery.
func (g *snmpAgentGroup) ReindexAll() {
	if g == nil {
		return
	}

	for _, agent := range g.agents {
		agent.Reindex()
	}
}

// SynthesizePeerTopologyAll fills each agent's downstream bridge FDB from the
// resolved fleet MACs (see Agent.SynthesizePeerTopology). Every community agent
// shares the device's topology, so all get the entries.
func (g *snmpAgentGroup) SynthesizePeerTopologyAll(resolve snmp.PeerMACResolver) {
	if g == nil {
		return
	}

	for _, agent := range g.agents {
		agent.SynthesizePeerTopology(resolve)
	}
}

func (g *snmpAgentGroup) Communities() []string {
	if g == nil {
		return nil
	}

	result := make([]string, 0, len(g.agents))
	for comm := range g.agents {
		result = append(result, comm)
	}

	return result
}
