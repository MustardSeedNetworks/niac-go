package protocols

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/krisarmstrong/niac-go/pkg/config"
)

// DHCPv6 message types (RFC 8415).
const (
	DHCPv6Solicit     = 1
	DHCPv6Advertise   = 2
	DHCPv6Request     = 3
	DHCPv6Confirm     = 4
	DHCPv6Renew       = 5
	DHCPv6Rebind      = 6
	DHCPv6Reply       = 7
	DHCPv6Release     = 8
	DHCPv6Decline     = 9
	DHCPv6Reconfigure = 10
	DHCPv6InfoRequest = 11
	DHCPv6RelayForw   = 12
	DHCPv6RelayRepl   = 13
)

// DHCPv6 option codes (RFC 8415).
const (
	DHCPv6OptClientID               = 1
	DHCPv6OptServerID               = 2
	DHCPv6OptIANA                   = 3 // Identity Association for Non-temporary Addresses
	DHCPv6OptIATA                   = 4 // Identity Association for Temporary Addresses
	DHCPv6OptIAAddr                 = 5 // IA Address
	DHCPv6OptORO                    = 6 // Option Request Option
	DHCPv6OptPreference             = 7
	DHCPv6OptElapsedTime            = 8
	DHCPv6OptRelayMsg               = 9
	DHCPv6OptAuth                   = 11
	DHCPv6OptUnicast                = 12
	DHCPv6OptStatusCode             = 13
	DHCPv6OptRapidCommit            = 14
	DHCPv6OptUserClass              = 15
	DHCPv6OptVendorClass            = 16
	DHCPv6OptVendorOpts             = 17
	DHCPv6OptInterfaceID            = 18
	DHCPv6OptReconfMsg              = 19
	DHCPv6OptReconfAccept           = 20
	DHCPv6OptSIPServers             = 21 // SIP Servers (Domain Name List)
	DHCPv6OptSIPServerAddrs         = 22 // SIP Servers (IPv6 Address List)
	DHCPv6OptDNSServers             = 23
	DHCPv6OptDomainList             = 24
	DHCPv6OptIAPD                   = 25 // Identity Association for Prefix Delegation
	DHCPv6OptIAPrefix               = 26
	DHCPv6OptSNTPServers            = 31 // SNTP Servers
	DHCPv6OptInformationRefreshTime = 32
	DHCPv6OptFQDN                   = 39 // Client FQDN
	DHCPv6OptNTPServer              = 56 // NTP Server
)

// DHCPv6 status codes.
const (
	DHCPv6StatusSuccess      = 0
	DHCPv6StatusUnspecFail   = 1
	DHCPv6StatusNoAddrsAvail = 2
	DHCPv6StatusNoBinding    = 3
	DHCPv6StatusNotOnLink    = 4
	DHCPv6StatusUseMulticast = 5
)

// DHCPv6 DUID types (RFC 8415).
const (
	DUIDTypeLLT = 1 // Link-layer address plus time
	DUIDTypeEN  = 2 // Vendor-assigned unique ID based on Enterprise Number
	DUIDTypeLL  = 3 // Link-layer address
)

// DHCPv6 lease duration (7 days preferred, 30 days valid).
const (
	DefaultPreferredLifetime = 7 * 24 * time.Hour
	DefaultValidLifetime     = 30 * 24 * time.Hour
)

// DHCPv6 ports.
const (
	DHCPv6ServerPort = 547
	DHCPv6ClientPort = 546
)

// DHCPv6 multicast addresses.
var (
	AllDHCPRelayAgentsAndServers = net.ParseIP("ff02::1:2")
	AllDHCPServers               = net.ParseIP("ff05::1:3")
)

// DHCPv6Message represents a DHCPv6 message.
type DHCPv6Message struct {
	MessageType   uint8
	TransactionID [3]byte
	Options       []DHCPv6Option
}

// DHCPv6Option represents a DHCPv6 option.
type DHCPv6Option struct {
	Code   uint16
	Length uint16
	Data   []byte
}

// DHCPv6Lease represents an IPv6 address lease.
type DHCPv6Lease struct {
	Address           net.IP
	Prefix            *net.IPNet // For prefix delegation
	DUID              []byte
	IAID              uint32
	PreferredLifetime time.Time
	ValidLifetime     time.Time
	LastRenewal       time.Time
}

// DHCPv6Handler handles DHCPv6 server functionality.
type DHCPv6Handler struct {
	stack             *Stack
	leases            map[string]*DHCPv6Lease // Key: DUID hex string
	addressPool       []net.IP
	prefixPool        []net.IPNet
	serverDUID        []byte
	preferredLifetime time.Duration
	validLifetime     time.Duration
	dnsServers        []net.IP
	domainList        []string
	sntpServers       []net.IP // Option 31: SNTP servers
	ntpServers        []net.IP // Option 56: NTP servers
	sipServers        []net.IP // Option 22: SIP server addresses
	sipDomains        []string // Option 21: SIP domain names
	mu                sync.RWMutex
}

// NewDHCPv6Handler creates a new DHCPv6 handler.
func NewDHCPv6Handler(stack *Stack) *DHCPv6Handler {
	return &DHCPv6Handler{
		stack:             stack,
		leases:            make(map[string]*DHCPv6Lease),
		addressPool:       make([]net.IP, 0),
		prefixPool:        make([]net.IPNet, 0),
		serverDUID:        generateDUID(),
		preferredLifetime: DefaultPreferredLifetime,
		validLifetime:     DefaultValidLifetime,
	}
}

// Reset clears DHCPv6 leases and cached options.
func (h *DHCPv6Handler) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.leases = make(map[string]*DHCPv6Lease)
	h.addressPool = nil
	h.prefixPool = nil
	h.serverDUID = generateDUID()
	h.preferredLifetime = DefaultPreferredLifetime
	h.validLifetime = DefaultValidLifetime
	h.dnsServers = nil
	h.domainList = nil
	h.sntpServers = nil
	h.ntpServers = nil
	h.sipServers = nil
	h.sipDomains = nil
}

// generateDUID generates a DUID-LL (Link-Layer) for the server.
func generateDUID() []byte {
	// DUID-LL format: Type(2) + HW Type(2) + Link-Layer Address(variable)
	duid := make([]byte, 10)
	binary.BigEndian.PutUint16(duid[0:2], DUIDTypeLL) // DUID-LL
	binary.BigEndian.PutUint16(duid[2:4], 1)          // Ethernet

	// Generate random MAC for server DUID
	mac := make([]byte, 6)
	_, _ = rand.Read(mac)           // crypto/rand read errors will result in zero bytes
	mac[0] = (mac[0] | 0x02) & 0xfe // Set local, clear multicast
	copy(duid[4:10], mac)

	return duid
}

// SetAddressPool configures the DHCPv6 address pool.
func (h *DHCPv6Handler) SetAddressPool(addresses []net.IP) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.addressPool = addresses
}

// SetPrefixPool configures the DHCPv6 prefix delegation pool.
func (h *DHCPv6Handler) SetPrefixPool(prefixes []net.IPNet) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.prefixPool = prefixes
}

// SetServerConfig configures DHCPv6 server parameters.
func (h *DHCPv6Handler) SetServerConfig(dnsServers []net.IP, domainList []string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.dnsServers = dnsServers
	h.domainList = domainList
}

// SetAdvancedOptions configures advanced DHCPv6 options.
func (h *DHCPv6Handler) SetAdvancedOptions(sntpServers, ntpServers, sipServers []net.IP, sipDomains []string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.sntpServers = sntpServers
	h.ntpServers = ntpServers
	h.sipServers = sipServers
	h.sipDomains = sipDomains
}

// HandlePacket processes a DHCPv6 packet.
func (h *DHCPv6Handler) HandlePacket(
	pkt *Packet,
	ipv6Layer *layers.IPv6,
	udpLayer *layers.UDP,
	devices []*config.Device,
) {
	debugLevel := h.stack.GetDebugLevel()

	h.stack.IncrementStat("dhcp_requests")

	// Parse DHCPv6 message
	msg, err := h.parseDHCPv6Message(udpLayer.Payload)
	if err != nil {
		if debugLevel >= 2 {
			slog.Debug("DHCPv6: Failed to parse message", "error", err, "sn", pkt.SerialNumber)
		}

		return
	}

	if debugLevel >= 3 {
		slog.Debug(
			"DHCPv6 message received",
			"type",
			h.messageTypeString(msg.MessageType),
			"srcIP",
			ipv6Layer.SrcIP,
			"sn",
			pkt.SerialNumber,
		)
	}

	// Find server device
	var serverDevice *config.Device

	for _, dev := range devices {
		if len(dev.IPAddresses) > 0 {
			// Check if device has an IPv6 address
			for _, ip := range dev.IPAddresses {
				if ip.To4() == nil && ip.To16() != nil {
					serverDevice = dev

					break
				}
			}

			if serverDevice != nil {
				break
			}
		}
	}

	if serverDevice == nil {
		if debugLevel >= 2 {
			slog.Debug("DHCPv6: No IPv6 server device configured", "sn", pkt.SerialNumber)
		}

		return
	}

	// Get server IPv6 address
	var serverIP net.IP

	for _, ip := range serverDevice.IPAddresses {
		if ip.To4() == nil && ip.To16() != nil {
			serverIP = ip

			break
		}
	}

	// Handle message based on type
	switch msg.MessageType {
	case DHCPv6Solicit:
		h.handleSolicit(msg, ipv6Layer.SrcIP, serverIP, serverDevice.MACAddress, serverDevice, pkt.SerialNumber)

	case DHCPv6Request:
		h.handleRequest(msg, ipv6Layer.SrcIP, serverIP, serverDevice.MACAddress, serverDevice, pkt.SerialNumber)

	case DHCPv6Renew:
		h.handleRenew(msg, ipv6Layer.SrcIP, serverIP, serverDevice.MACAddress, serverDevice, pkt.SerialNumber)

	case DHCPv6Rebind:
		h.handleRebind(msg, ipv6Layer.SrcIP, serverIP, serverDevice.MACAddress, serverDevice, pkt.SerialNumber)

	case DHCPv6Release:
		h.handleRelease(msg, pkt.SerialNumber)

	case DHCPv6Decline:
		h.handleDecline(msg, pkt.SerialNumber)

	case DHCPv6InfoRequest:
		h.handleInfoRequest(msg, ipv6Layer.SrcIP, serverIP, serverDevice.MACAddress, pkt.SerialNumber)

	default:
		if debugLevel >= 2 {
			slog.Debug("DHCPv6: Unhandled message type", "type", msg.MessageType, "sn", pkt.SerialNumber)
		}
	}
}

// parseDHCPv6Message parses a DHCPv6 message from bytes.
func (h *DHCPv6Handler) parseDHCPv6Message(data []byte) (*DHCPv6Message, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("%w: %d bytes", ErrMessageTooShort, len(data))
	}

	msg := &DHCPv6Message{
		MessageType: data[0],
		Options:     make([]DHCPv6Option, 0),
	}
	copy(msg.TransactionID[:], data[1:4])

	// Parse options
	offset := 4
	for offset < len(data) {
		if offset+4 > len(data) {
			break
		}

		opt := DHCPv6Option{
			Code:   binary.BigEndian.Uint16(data[offset : offset+2]),
			Length: binary.BigEndian.Uint16(data[offset+2 : offset+4]),
		}
		offset += 4

		if offset+int(opt.Length) > len(data) {
			return nil, ErrOptionDataExceedsLen
		}

		opt.Data = make([]byte, opt.Length)
		copy(opt.Data, data[offset:offset+int(opt.Length)])
		offset += int(opt.Length)

		msg.Options = append(msg.Options, opt)
	}

	return msg, nil
}

// findOption finds an option by code in the message.
func (h *DHCPv6Handler) findOption(msg *DHCPv6Message, code uint16) *DHCPv6Option {
	for i := range msg.Options {
		if msg.Options[i].Code == code {
			return &msg.Options[i]
		}
	}

	return nil
}

// extractClientDUID extracts the client DUID from the message.
func (h *DHCPv6Handler) extractClientDUID(msg *DHCPv6Message) []byte {
	opt := h.findOption(msg, DHCPv6OptClientID)
	if opt != nil {
		return opt.Data
	}

	return nil
}

// extractIANA extracts IANA option from message.
func (h *DHCPv6Handler) extractIANA(msg *DHCPv6Message) (uint32, bool) {
	opt := h.findOption(msg, DHCPv6OptIANA)
	if opt != nil && len(opt.Data) >= 4 {
		iaid := binary.BigEndian.Uint32(opt.Data[0:4])

		return iaid, true
	}

	return 0, false
}

// duidString converts DUID bytes to hex string for map key.
func duidString(duid []byte) string {
	return hex.EncodeToString(duid)
}

// validateClientIdentity extracts and validates client DUID and IANA from a DHCPv6 message.
// Returns clientDUID, iaid, and true if valid; returns nil, 0, false if validation fails.
// The msgType parameter is used for debug logging (e.g., "Solicit", "Request").
func (h *DHCPv6Handler) validateClientIdentity(msg *DHCPv6Message, msgType string, sn int) ([]byte, uint32, bool) {
	debugLevel := h.stack.GetDebugLevel()

	clientDUID := h.extractClientDUID(msg)
	if clientDUID == nil {
		if debugLevel >= 2 {
			slog.Debug("DHCPv6: missing client DUID", "msgType", msgType, "sn", sn)
		}

		return nil, 0, false
	}

	iaid, hasIANA := h.extractIANA(msg)
	if !hasIANA {
		if debugLevel >= 2 {
			slog.Debug("DHCPv6: missing IANA", "msgType", msgType, "sn", sn)
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
	debugLevel := h.stack.GetDebugLevel()

	clientDUID, iaid, ok := h.validateClientIdentity(msg, "Solicit", sn)
	if !ok {
		return
	}

	// Allocate or find existing lease
	lease, err := h.allocateLease(clientDUID, iaid)
	if err != nil {
		if debugLevel >= 1 {
			slog.Info("DHCPv6: Failed to allocate address", "error", err, "sn", sn)
		}

		return
	}

	// Send Advertise
	if err := h.sendAdvertise(msg, lease, clientIP, serverIP, serverMAC, device); err != nil {
		if debugLevel >= 1 {
			slog.Info("DHCPv6: Failed to send Advertise", "error", err, "sn", sn)
		}
	} else {
		h.stack.IncrementStat("dhcp_offers")

		if debugLevel >= 2 {
			slog.Debug("DHCPv6: Sent Advertise", "address", lease.Address, "sn", sn)
		}
	}
}

// handleRequest processes DHCPv6 Request message.
func (h *DHCPv6Handler) handleRequest(
	msg *DHCPv6Message,
	clientIP, serverIP net.IP,
	serverMAC net.HardwareAddr,
	device *config.Device,
	sn int,
) {
	debugLevel := h.stack.GetDebugLevel()

	clientDUID, iaid, ok := h.validateClientIdentity(msg, "Request", sn)
	if !ok {
		return
	}

	// Confirm or allocate lease
	lease, err := h.confirmLease(clientDUID, iaid)
	if err != nil {
		if debugLevel >= 1 {
			slog.Info("DHCPv6: Failed to confirm lease", "error", err, "sn", sn)
		}

		return
	}

	// Send Reply
	if err := h.sendReply(msg, lease, clientIP, serverIP, serverMAC, device); err != nil {
		if debugLevel >= 1 {
			slog.Info("DHCPv6: Failed to send Reply", "error", err, "sn", sn)
		}
	} else {
		h.stack.IncrementStat("dhcp_acks")

		if debugLevel >= 2 {
			slog.Debug("DHCPv6: Sent Reply", "address", lease.Address, "sn", sn)
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
	debugLevel := h.stack.GetDebugLevel()

	clientDUID := h.extractClientDUID(msg)
	if clientDUID == nil {
		return
	}

	lease := h.findLease(clientDUID)
	if lease == nil {
		if debugLevel >= 2 {
			slog.Debug("DHCPv6: Renew for unknown lease", "sn", sn)
		}

		return
	}

	// Renew lease
	h.renewLease(lease)

	err := h.sendReply(msg, lease, clientIP, serverIP, serverMAC, device)
	if err != nil {
		if debugLevel >= 1 {
			slog.Info("DHCPv6: Failed to send Renew Reply", "error", err, "sn", sn)
		}
	} else if debugLevel >= 2 {
		slog.Debug("DHCPv6: Renewed lease", "address", lease.Address, "sn", sn)
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
	debugLevel := h.stack.GetDebugLevel()

	clientDUID := h.extractClientDUID(msg)
	if clientDUID == nil {
		return
	}

	h.mu.Lock()
	delete(h.leases, duidString(clientDUID))
	h.mu.Unlock()

	if debugLevel >= 2 {
		slog.Debug("DHCPv6: Released lease", "sn", sn)
	}
}

// handleDecline processes DHCPv6 Decline message.
func (h *DHCPv6Handler) handleDecline(msg *DHCPv6Message, sn int) {
	debugLevel := h.stack.GetDebugLevel()

	clientDUID := h.extractClientDUID(msg)
	if clientDUID == nil {
		return
	}

	// Mark address as declined (don't reassign immediately)
	h.mu.Lock()
	delete(h.leases, duidString(clientDUID))
	h.mu.Unlock()

	if debugLevel >= 2 {
		slog.Debug("DHCPv6: Address declined", "sn", sn)
	}
}

// handleInfoRequest processes DHCPv6 Information-Request message.
func (h *DHCPv6Handler) handleInfoRequest(
	msg *DHCPv6Message,
	clientIP, serverIP net.IP,
	serverMAC net.HardwareAddr,
	sn int,
) {
	debugLevel := h.stack.GetDebugLevel()

	// Send Reply with configuration info (DNS, domain, etc.) but no address
	err := h.sendInfoReply(msg, clientIP, serverIP, serverMAC, nil)
	if err != nil {
		if debugLevel >= 1 {
			slog.Info("DHCPv6: Failed to send Info Reply", "error", err, "sn", sn)
		}
	} else if debugLevel >= 2 {
		slog.Debug("DHCPv6: Sent Info Reply", "sn", sn)
	}
}

// allocateLease allocates a new IPv6 address lease.
func (h *DHCPv6Handler) allocateLease(clientDUID []byte, iaid uint32) (*DHCPv6Lease, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	duidKey := duidString(clientDUID)

	// Check if client already has a lease
	if existing, ok := h.leases[duidKey]; ok {
		// Renew existing lease
		h.renewLeaseUnlocked(existing)

		return existing, nil
	}

	// Find available address
	address := h.findAvailableAddress()
	if address == nil {
		return nil, ErrNoAvailableAddresses
	}

	// Create new lease
	now := time.Now()
	lease := &DHCPv6Lease{
		Address:           address,
		DUID:              clientDUID,
		IAID:              iaid,
		PreferredLifetime: now.Add(h.preferredLifetime),
		ValidLifetime:     now.Add(h.validLifetime),
		LastRenewal:       now,
	}

	h.leases[duidKey] = lease

	return lease, nil
}

// confirmLease confirms an existing lease or allocates new one.
func (h *DHCPv6Handler) confirmLease(clientDUID []byte, iaid uint32) (*DHCPv6Lease, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	duidKey := duidString(clientDUID)

	if existing, ok := h.leases[duidKey]; ok {
		h.renewLeaseUnlocked(existing)

		return existing, nil
	}

	// Allocate new lease (unlock then relock via allocateLease)
	h.mu.Unlock()
	lease, err := h.allocateLease(clientDUID, iaid)
	h.mu.Lock()

	return lease, err
}

// findLease finds a lease by client DUID.
func (h *DHCPv6Handler) findLease(clientDUID []byte) *DHCPv6Lease {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if lease, ok := h.leases[duidString(clientDUID)]; ok {
		return lease
	}

	return nil
}

// renewLease renews a lease (with locking).
func (h *DHCPv6Handler) renewLease(lease *DHCPv6Lease) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.renewLeaseUnlocked(lease)
}

// renewLeaseUnlocked renews a lease (without locking).
func (h *DHCPv6Handler) renewLeaseUnlocked(lease *DHCPv6Lease) {
	now := time.Now()
	lease.PreferredLifetime = now.Add(h.preferredLifetime)
	lease.ValidLifetime = now.Add(h.validLifetime)
	lease.LastRenewal = now
}

// findAvailableAddress finds an available IPv6 address.
func (h *DHCPv6Handler) findAvailableAddress() net.IP {
	// Check pool for available address
	for _, addr := range h.addressPool {
		inUse := false

		for _, lease := range h.leases {
			if lease.Address.Equal(addr) && time.Now().Before(lease.ValidLifetime) {
				inUse = true

				break
			}
		}

		if !inUse {
			// Return a copy
			result := make(net.IP, len(addr))
			copy(result, addr)

			return result
		}
	}

	return nil
}

// Continue in next part...

// sendAdvertise sends a DHCPv6 Advertise message.
func (h *DHCPv6Handler) sendAdvertise(
	clientMsg *DHCPv6Message,
	lease *DHCPv6Lease,
	clientIP, serverIP net.IP,
	serverMAC net.HardwareAddr,
	device *config.Device,
) error {
	return h.sendDHCPv6Response(DHCPv6Advertise, clientMsg, lease, clientIP, serverIP, serverMAC, device, false)
}

// sendReply sends a DHCPv6 Reply message.
func (h *DHCPv6Handler) sendReply(
	clientMsg *DHCPv6Message,
	lease *DHCPv6Lease,
	clientIP, serverIP net.IP,
	serverMAC net.HardwareAddr,
	device *config.Device,
) error {
	return h.sendDHCPv6Response(DHCPv6Reply, clientMsg, lease, clientIP, serverIP, serverMAC, device, false)
}

// sendInfoReply sends a DHCPv6 Reply for Information-Request.
func (h *DHCPv6Handler) sendInfoReply(
	clientMsg *DHCPv6Message,
	clientIP, serverIP net.IP,
	serverMAC net.HardwareAddr,
	device *config.Device,
) error {
	return h.sendDHCPv6Response(DHCPv6Reply, clientMsg, nil, clientIP, serverIP, serverMAC, device, true)
}

// sendDHCPv6Response sends a DHCPv6 response message.
func (h *DHCPv6Handler) sendDHCPv6Response(msgType uint8, clientMsg *DHCPv6Message, lease *DHCPv6Lease,
	clientIP, serverIP net.IP, serverMAC net.HardwareAddr, device *config.Device, infoOnly bool,
) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Build response message
	response := &DHCPv6Message{
		MessageType:   msgType,
		TransactionID: clientMsg.TransactionID,
		Options:       make([]DHCPv6Option, 0),
	}

	// Add Server ID
	serverDUIDLen := min(len(h.serverDUID), 65535)

	response.Options = append(response.Options, DHCPv6Option{
		Code:   DHCPv6OptServerID,
		Length: uint16(serverDUIDLen), // #nosec G115 -- serverDUIDLen capped at 65535 above
		Data:   h.serverDUID,
	})

	// Add Client ID (echo from request)
	if clientID := h.findOption(clientMsg, DHCPv6OptClientID); clientID != nil {
		response.Options = append(response.Options, *clientID)
	}

	// Add IA_NA with address (if not info-only)
	if !infoOnly && lease != nil {
		ianaOpt := h.buildIANAOption(lease)
		response.Options = append(response.Options, ianaOpt)
	}

	// Add DNS servers if configured
	if len(h.dnsServers) > 0 {
		dnsData := make([]byte, 0, len(h.dnsServers)*16)
		for _, dns := range h.dnsServers {
			dnsData = append(dnsData, dns.To16()...)
		}

		dnsDataLen := min(len(dnsData), 65535)

		response.Options = append(response.Options, DHCPv6Option{
			Code:   DHCPv6OptDNSServers,
			Length: uint16(dnsDataLen), // #nosec G115 -- dnsDataLen capped at 65535 above
			Data:   dnsData,
		})
	}

	// Add domain search list if configured
	if len(h.domainList) > 0 {
		domainData := h.encodeDomainList(h.domainList)

		domainDataLen := min(len(domainData), 65535)

		response.Options = append(response.Options, DHCPv6Option{
			Code:   DHCPv6OptDomainList,
			Length: uint16(domainDataLen), // #nosec G115 -- domainDataLen capped at 65535 above
			Data:   domainData,
		})
	}

	// Add SNTP servers if configured (Option 31)
	if len(h.sntpServers) > 0 {
		sntpData := make([]byte, 0, len(h.sntpServers)*16)
		for _, sntp := range h.sntpServers {
			sntpData = append(sntpData, sntp.To16()...)
		}

		sntpDataLen := min(len(sntpData), 65535)

		response.Options = append(response.Options, DHCPv6Option{
			Code:   DHCPv6OptSNTPServers,
			Length: uint16(sntpDataLen), // #nosec G115 -- sntpDataLen capped at 65535 above
			Data:   sntpData,
		})
	}

	// Add NTP servers if configured (Option 56)
	if len(h.ntpServers) > 0 {
		ntpData := make([]byte, 0, len(h.ntpServers)*16)
		for _, ntp := range h.ntpServers {
			ntpData = append(ntpData, ntp.To16()...)
		}

		ntpDataLen := min(len(ntpData), 65535)

		response.Options = append(response.Options, DHCPv6Option{
			Code:   DHCPv6OptNTPServer,
			Length: uint16(ntpDataLen), // #nosec G115 -- ntpDataLen capped at 65535 above
			Data:   ntpData,
		})
	}

	// Add SIP server addresses if configured (Option 22)
	if len(h.sipServers) > 0 {
		sipData := make([]byte, 0, len(h.sipServers)*16)
		for _, sip := range h.sipServers {
			sipData = append(sipData, sip.To16()...)
		}

		sipDataLen := min(len(sipData), 65535)

		response.Options = append(response.Options, DHCPv6Option{
			Code:   DHCPv6OptSIPServerAddrs,
			Length: uint16(sipDataLen), // #nosec G115 -- sipDataLen capped at 65535 above
			Data:   sipData,
		})
	}

	// Add SIP domain names if configured (Option 21)
	if len(h.sipDomains) > 0 {
		sipDomainData := h.encodeDomainList(h.sipDomains)

		sipDomainDataLen := min(len(sipDomainData), 65535)

		response.Options = append(response.Options, DHCPv6Option{
			Code:   DHCPv6OptSIPServers,
			Length: uint16(sipDomainDataLen), // #nosec G115 -- sipDomainDataLen capped at 65535 above
			Data:   sipDomainData,
		})
	}

	// Add preference (higher is better, 255 is max)
	preference := uint8(0) // Default to 0 (lowest priority)
	if device != nil && device.DHCPv6Config != nil {
		preference = device.DHCPv6Config.Preference
	}

	response.Options = append(response.Options, DHCPv6Option{
		Code:   DHCPv6OptPreference,
		Length: 1,
		Data:   []byte{preference},
	})

	// Serialize message
	msgBytes := h.serializeDHCPv6Message(response)

	// Build UDP layer
	udp := &layers.UDP{
		SrcPort: DHCPv6ServerPort,
		DstPort: DHCPv6ClientPort,
	}

	// Build IPv6 layer - send to client's link-local or use multicast
	dstIP := clientIP
	dstMAC := net.HardwareAddr{0x33, 0x33, 0x00, 0x01, 0x00, 0x02} // All DHCP relay agents and servers

	// If client IP is link-local, use it directly
	if clientIP.IsLinkLocalUnicast() {
		// Calculate multicast MAC from client IP
		if clientIP.To4() == nil && len(clientIP) == 16 {
			dstMAC = IPv6MulticastToMAC(AllDHCPRelayAgentsAndServers)
		}
	}

	ipv6 := &layers.IPv6{
		Version:    6,
		HopLimit:   64,
		NextHeader: layers.IPProtocolUDP,
		SrcIP:      serverIP,
		DstIP:      dstIP,
	}

	// Build Ethernet layer
	eth := &layers.Ethernet{
		SrcMAC:       serverMAC,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetTypeIPv6,
	}

	// Calculate UDP length and checksum
	msgBytesLen := min(len(msgBytes),
		// 65535 - 8 (UDP header size)
		65527)

	udp.Length = uint16(8 + msgBytesLen) // #nosec G115 -- msgBytesLen capped at 65527 above

	// Serialize packet
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}

	// Set UDP payload
	payload := gopacket.Payload(msgBytes)

	_ = udp.SetNetworkLayerForChecksum(ipv6) // error is non-critical for simulation

	err := gopacket.SerializeLayers(buf, opts, eth, ipv6, udp, payload)
	if err != nil {
		return fmt.Errorf("failed to serialize DHCPv6 response: %w", err)
	}

	// Send packet
	return h.stack.SendRawPacket(buf.Bytes())
}

// buildIANAOption builds an IA_NA option with IA Address.
func (h *DHCPv6Handler) buildIANAOption(lease *DHCPv6Lease) DHCPv6Option {
	// IA_NA option format:
	// IAID (4 bytes) + T1 (4 bytes) + T2 (4 bytes) + IA_NA options
	ianaHeader := make([]byte, 12)

	// IAID
	binary.BigEndian.PutUint32(ianaHeader[0:4], lease.IAID)

	// T1 (renewal time) - 50% of preferred lifetime
	t1 := uint32(h.preferredLifetime.Seconds() / 2)
	binary.BigEndian.PutUint32(ianaHeader[4:8], t1)

	// T2 (rebinding time) - 80% of preferred lifetime
	t2 := uint32(h.preferredLifetime.Seconds() * 4 / 5)
	binary.BigEndian.PutUint32(ianaHeader[8:12], t2)

	// Build IA Address option
	iaAddrOpt := h.buildIAAddrOption(lease)
	ianaHeader = append(ianaHeader, h.serializeOption(iaAddrOpt)...)

	ianaDataLen := min(len(ianaHeader), 65535)

	return DHCPv6Option{
		Code:   DHCPv6OptIANA,
		Length: uint16(ianaDataLen), // #nosec G115 -- ianaDataLen capped at 65535 above
		Data:   ianaHeader,
	}
}

// buildIAAddrOption builds an IA Address option.
func (h *DHCPv6Handler) buildIAAddrOption(lease *DHCPv6Lease) DHCPv6Option {
	// IA Address option format:
	// IPv6 address (16 bytes) + preferred-lifetime (4 bytes) + valid-lifetime (4 bytes)
	iaAddrData := make([]byte, 24)

	// IPv6 address
	copy(iaAddrData[0:16], lease.Address.To16())

	// Preferred lifetime (in seconds)
	preferred := uint32(time.Until(lease.PreferredLifetime).Seconds())
	if time.Now().After(lease.PreferredLifetime) {
		preferred = 0
	}

	binary.BigEndian.PutUint32(iaAddrData[16:20], preferred)

	// Valid lifetime (in seconds)
	valid := uint32(time.Until(lease.ValidLifetime).Seconds())
	if time.Now().After(lease.ValidLifetime) {
		valid = 0
	}

	binary.BigEndian.PutUint32(iaAddrData[20:24], valid)

	return DHCPv6Option{
		Code:   DHCPv6OptIAAddr,
		Length: 24,
		Data:   iaAddrData,
	}
}

// serializeDHCPv6Message serializes a DHCPv6 message to bytes.
func (h *DHCPv6Handler) serializeDHCPv6Message(msg *DHCPv6Message) []byte {
	// Calculate total size
	size := 4 // Message type (1) + Transaction ID (3)
	for _, opt := range msg.Options {
		size += 4 + int(opt.Length) // Code (2) + Length (2) + Data
	}

	buf := make([]byte, size)
	buf[0] = msg.MessageType
	copy(buf[1:4], msg.TransactionID[:])

	// Serialize options
	offset := 4
	for _, opt := range msg.Options {
		offset += h.serializeOptionAt(buf[offset:], opt)
	}

	return buf
}

// serializeOption serializes a single option.
func (h *DHCPv6Handler) serializeOption(opt DHCPv6Option) []byte {
	buf := make([]byte, 4+opt.Length)
	h.serializeOptionAt(buf, opt)

	return buf
}

// serializeOptionAt serializes an option into a buffer.
func (h *DHCPv6Handler) serializeOptionAt(buf []byte, opt DHCPv6Option) int {
	binary.BigEndian.PutUint16(buf[0:2], opt.Code)
	binary.BigEndian.PutUint16(buf[2:4], opt.Length)
	copy(buf[4:], opt.Data)

	return 4 + int(opt.Length)
}

// encodeDomainList encodes domain names for DHCPv6 Domain Search List option.
func (h *DHCPv6Handler) encodeDomainList(domains []string) []byte {
	data := make([]byte, 0)

	for _, domain := range domains {
		// DNS name encoding: length-prefixed labels
		labels := splitDomainLabels(domain)
		for _, label := range labels {
			data = append(data, byte(len(label)))
			data = append(data, []byte(label)...)
		}

		data = append(data, 0) // Null terminator
	}

	return data
}

// splitDomainLabels splits a domain name into labels.
func splitDomainLabels(domain string) []string {
	if domain == "" {
		return []string{}
	}

	labels := make([]string, 0)
	start := 0

	for i := range len(domain) {
		if domain[i] == '.' {
			if i > start {
				labels = append(labels, domain[start:i])
			}

			start = i + 1
		}
	}

	if start < len(domain) {
		labels = append(labels, domain[start:])
	}

	return labels
}

// messageTypeString returns string representation of DHCPv6 message type.
func (h *DHCPv6Handler) messageTypeString(msgType uint8) string {
	switch msgType {
	case DHCPv6Solicit:
		return "SOLICIT"
	case DHCPv6Advertise:
		return "ADVERTISE"
	case DHCPv6Request:
		return "REQUEST"
	case DHCPv6Confirm:
		return "CONFIRM"
	case DHCPv6Renew:
		return "RENEW"
	case DHCPv6Rebind:
		return "REBIND"
	case DHCPv6Reply:
		return "REPLY"
	case DHCPv6Release:
		return "RELEASE"
	case DHCPv6Decline:
		return "DECLINE"
	case DHCPv6Reconfigure:
		return "RECONFIGURE"
	case DHCPv6InfoRequest:
		return "INFORMATION-REQUEST"
	case DHCPv6RelayForw:
		return "RELAY-FORW"
	case DHCPv6RelayRepl:
		return "RELAY-REPL"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", msgType)
	}
}
