package snmp

import "github.com/MustardSeedNetworks/niac-go/internal/config"

func (a *Agent) supportsBridgeLink(port config.TrunkPort) bool {
	return !config.IsRoutedTopologyLink(a.device.Type, port) && a.supportsBridgeTopology()
}
