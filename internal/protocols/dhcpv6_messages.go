package protocols

import (
	"log/slog"
	"net"

	"github.com/krisarmstrong/niac-go/internal/config"
)

// dispatchDHCPv6Message routes the DHCPv6 message to the appropriate handler.
func (h *DHCPv6Handler) dispatchDHCPv6Message(
	msg *DHCPv6Message,
	srcIP, serverIP net.IP,
	serverDevice *config.Device,
	serialNum int,
	debugLevel int,
) {
	switch msg.MessageType {
	case DHCPv6Solicit:
		h.handleSolicit(msg, srcIP, serverIP, serverDevice.MACAddress, serverDevice, serialNum)
	case DHCPv6Request:
		h.handleRequest(msg, srcIP, serverIP, serverDevice.MACAddress, serverDevice, serialNum)
	case DHCPv6Renew:
		h.handleRenew(msg, srcIP, serverIP, serverDevice.MACAddress, serverDevice, serialNum)
	case DHCPv6Rebind:
		h.handleRebind(msg, srcIP, serverIP, serverDevice.MACAddress, serverDevice, serialNum)
	case DHCPv6Release:
		h.handleRelease(msg, serialNum)
	case DHCPv6Decline:
		h.handleDecline(msg, serialNum)
	case DHCPv6InfoRequest:
		h.handleInfoRequest(msg, srcIP, serverIP, serverDevice.MACAddress, serialNum)
	default:
		if debugLevel >= DebugLevelInfo {
			slog.Default().Debug("DHCPv6: Unhandled message type", "type", msg.MessageType, "sn", serialNum)
		}
	}
}

// validateClientIdentity extracts and validates client DUID and IANA from a DHCPv6 message.
// Returns clientDUID, iaid, and true if valid; returns nil, 0, false if validation fails.
// The msgType parameter is used for debug logging (e.g., "Solicit", "Request").
func (h *DHCPv6Handler) validateClientIdentity(msg *DHCPv6Message, msgType string, sn int) ([]byte, uint32, bool) {
	logger := slog.Default()
	debugLevel := h.stack.GetDebugLevel()

	clientDUID := h.extractClientDUID(msg)
	if clientDUID == nil {
		if debugLevel >= DebugLevelInfo {
			logger.Debug("DHCPv6: missing client DUID", "msgType", msgType, "sn", sn)
		}

		return nil, 0, false
	}

	iaid, hasIANA := h.extractIANA(msg)
	if !hasIANA {
		if debugLevel >= DebugLevelInfo {
			logger.Debug("DHCPv6: missing IANA", "msgType", msgType, "sn", sn)
		}

		return nil, 0, false
	}

	return clientDUID, iaid, true
}

// handleSolicit processes DHCPv6 Solicit message.
func (h *DHCPv6Handler) handleSolicit(
	msg *DHCPv6Message,
	clientIP, serverIP net.IP,
	serverMAC net.HardwareAddr,
	device *config.Device,
	sn int,
) {
	h.handleLeaseMessage(
		msg,
		"Solicit",
		sn,
		clientIP,
		serverIP,
		serverMAC,
		device,
		h.allocateLease,
		h.sendAdvertise,
		"dhcp_offers",
		"DHCPv6: Failed to allocate address",
		"DHCPv6: Failed to send Advertise",
		"DHCPv6: Sent Advertise",
	)
}

// handleRequest processes DHCPv6 Request message.
func (h *DHCPv6Handler) handleRequest(
	msg *DHCPv6Message,
	clientIP, serverIP net.IP,
	serverMAC net.HardwareAddr,
	device *config.Device,
	sn int,
) {
	h.handleLeaseMessage(
		msg,
		"Request",
		sn,
		clientIP,
		serverIP,
		serverMAC,
		device,
		h.confirmLease,
		h.sendReply,
		"dhcp_acks",
		"DHCPv6: Failed to confirm lease",
		"DHCPv6: Failed to send Reply",
		"DHCPv6: Sent Reply",
	)
}

func (h *DHCPv6Handler) handleLeaseMessage(
	msg *DHCPv6Message,
	msgType string,
	sn int,
	clientIP, serverIP net.IP,
	serverMAC net.HardwareAddr,
	device *config.Device,
	leaseFn func([]byte, uint32) (*DHCPv6Lease, error),
	sendFn func(*DHCPv6Message, *DHCPv6Lease, net.IP, net.IP, net.HardwareAddr, *config.Device) error,
	successStat string,
	leaseErrMsg string,
	sendErrMsg string,
	successMsg string,
) {
	logger := slog.Default()
	debugLevel := h.stack.GetDebugLevel()

	clientDUID, iaid, ok := h.validateClientIdentity(msg, msgType, sn)
	if !ok {
		return
	}

	lease, err := leaseFn(clientDUID, iaid)
	if err != nil {
		if debugLevel >= 1 {
			logger.Info(leaseErrMsg, "error", err, "sn", sn)
		}

		return
	}

	sendErr := sendFn(msg, lease, clientIP, serverIP, serverMAC, device)
	if sendErr != nil {
		if debugLevel >= 1 {
			logger.Info(sendErrMsg, "error", sendErr, "sn", sn)
		}
	} else {
		h.stack.IncrementStat(successStat)

		if debugLevel >= DebugLevelInfo {
			logger.Debug(successMsg, "address", lease.Address, "sn", sn)
		}
	}
}

// handleRenew processes DHCPv6 Renew message.
func (h *DHCPv6Handler) handleRenew(
	msg *DHCPv6Message,
	clientIP, serverIP net.IP,
	serverMAC net.HardwareAddr,
	device *config.Device,
	sn int,
) {
	logger := slog.Default()
	debugLevel := h.stack.GetDebugLevel()

	clientDUID := h.extractClientDUID(msg)
	if clientDUID == nil {
		return
	}

	lease := h.findLease(clientDUID)
	if lease == nil {
		if debugLevel >= DebugLevelInfo {
			logger.Debug("DHCPv6: Renew for unknown lease", "sn", sn)
		}

		return
	}

	// Renew lease
	h.renewLease(lease)

	err := h.sendReply(msg, lease, clientIP, serverIP, serverMAC, device)
	if err != nil {
		if debugLevel >= 1 {
			logger.Info("DHCPv6: Failed to send Renew Reply", "error", err, "sn", sn)
		}
	} else if debugLevel >= DebugLevelInfo {
		logger.Debug("DHCPv6: Renewed lease", "address", lease.Address, "sn", sn)
	}
}

// handleRebind processes DHCPv6 Rebind message.
func (h *DHCPv6Handler) handleRebind(
	msg *DHCPv6Message,
	clientIP, serverIP net.IP,
	serverMAC net.HardwareAddr,
	device *config.Device,
	sn int,
) {
	// Rebind is similar to Renew but without server ID check
	h.handleRenew(msg, clientIP, serverIP, serverMAC, device, sn)
}

// handleRelease processes DHCPv6 Release message.
func (h *DHCPv6Handler) handleRelease(msg *DHCPv6Message, sn int) {
	logger := slog.Default()
	debugLevel := h.stack.GetDebugLevel()

	clientDUID := h.extractClientDUID(msg)
	if clientDUID == nil {
		return
	}

	h.mu.Lock()
	delete(h.leases, duidString(clientDUID))
	h.mu.Unlock()

	if debugLevel >= DebugLevelInfo {
		logger.Debug("DHCPv6: Released lease", "sn", sn)
	}
}

// handleDecline processes DHCPv6 Decline message.
func (h *DHCPv6Handler) handleDecline(msg *DHCPv6Message, sn int) {
	logger := slog.Default()
	debugLevel := h.stack.GetDebugLevel()

	clientDUID := h.extractClientDUID(msg)
	if clientDUID == nil {
		return
	}

	// Mark address as declined (don't reassign immediately)
	h.mu.Lock()
	delete(h.leases, duidString(clientDUID))
	h.mu.Unlock()

	if debugLevel >= DebugLevelInfo {
		logger.Debug("DHCPv6: Address declined", "sn", sn)
	}
}

// handleInfoRequest processes DHCPv6 Information-Request message.
func (h *DHCPv6Handler) handleInfoRequest(
	msg *DHCPv6Message,
	clientIP, serverIP net.IP,
	serverMAC net.HardwareAddr,
	sn int,
) {
	logger := slog.Default()
	debugLevel := h.stack.GetDebugLevel()

	// Send Reply with configuration info (DNS, domain, etc.) but no address
	err := h.sendInfoReply(msg, clientIP, serverIP, serverMAC, nil)
	if err != nil {
		if debugLevel >= 1 {
			logger.Info("DHCPv6: Failed to send Info Reply", "error", err, "sn", sn)
		}
	} else if debugLevel >= DebugLevelInfo {
		logger.Debug("DHCPv6: Sent Info Reply", "sn", sn)
	}
}
