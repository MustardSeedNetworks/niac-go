package protocols

import (
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
)

type snmpAgentGroup struct {
	agents map[string]*snmp.Agent
}

func newSnmpAgentGroup() *snmpAgentGroup {
	return &snmpAgentGroup{agents: make(map[string]*snmp.Agent)}
}

func (g *snmpAgentGroup) Get(community string) *snmp.Agent {
	if g == nil {
		return nil
	}

	return g.agents[community]
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

	agent := snmp.NewAgentWithCommunity(device, community, debugLevel)
	g.agents[community] = agent

	return agent
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
