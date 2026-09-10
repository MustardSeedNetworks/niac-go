package protocols

import (
	"log/slog"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

// offerSuppressed reports whether an armed no-offer fault makes this server
// silent. The DISCOVER is consumed and no OFFER goes out, which is what a
// client sees when a scope is exhausted or the server is wedged. The lease
// pool is left untouched, so clearing the fault resumes normal service.
func (h *DHCPHandler) offerSuppressed(
	serverDevice *config.Device,
	info *dhcpPacketInfo,
	serialNum, debugLevel int,
) bool {
	if !h.stack.deviceFaultActive(serverDevice, devicestate.FaultDHCPNoOffer) {
		return false
	}

	if debugLevel >= DebugLevelInfo {
		slog.Default().Debug("DHCP: Offer suppressed by fault",
			"device", serverDevice.Name, "mac", info.dhcp.ClientHWAddr, "sn", serialNum)
	}

	return true
}
