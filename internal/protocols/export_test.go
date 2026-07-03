package protocols

// export_test.go exposes internal items for use by the external test package (protocols_test).
// This file is only compiled during testing due to the _test.go suffix.

import (
	"net"
	"time"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// ---- Exported internal types for testing ----

// DNSRecord is an exported alias for dnsRecord.
type DNSRecord = dnsRecord

// DNSRecordSet is an exported alias for dnsRecordSet.
type DNSRecordSet = dnsRecordSet

// DNSPTR is an exported alias for dnsPTR.
type DNSPTR = dnsPTR

// ---- ARPHandler exports ----

// ARPHandlerStack returns the handler's stack for testing.
func (h *ARPHandler) ARPHandlerStack() *Stack { return h.stack }

// BuildARPReply exposes buildARPReply for testing.
func (h *ARPHandler) BuildARPReply(
	senderMAC net.HardwareAddr,
	senderIP net.IP,
	targetMAC net.HardwareAddr,
	targetIP net.IP,
) *Packet {
	return h.buildARPReply(senderMAC, senderIP, targetMAC, targetIP)
}

// ---- CDPHandler exports ----

// CDPHandlerStack returns the handler's stack for testing.
func (h *CDPHandler) CDPHandlerStack() *Stack { return h.stack }

// CDPHandlerStopChan returns the stop channel for testing.
func (h *CDPHandler) CDPHandlerStopChan() chan struct{} { return h.stopChan }

// CDPHandlerAdvertiseTicker returns the ticker for testing.
func (h *CDPHandler) CDPHandlerAdvertiseTicker() *time.Ticker { return h.advertiseTicker }

// BuildLLCSNAPHeader exposes buildLLCSNAPHeader for testing.
func (h *CDPHandler) BuildLLCSNAPHeader() []byte { return h.buildLLCSNAPHeader() }

// BuildDeviceIDTLV exposes buildDeviceIDTLV for testing.
func (h *CDPHandler) BuildDeviceIDTLV(device *config.Device) []byte {
	return h.buildDeviceIDTLV(device)
}

// BuildAddressesTLV exposes buildAddressesTLV for testing.
func (h *CDPHandler) BuildAddressesTLV(device *config.Device) []byte {
	return h.buildAddressesTLV(device)
}

// BuildPortIDTLV exposes buildPortIDTLV for testing.
func (h *CDPHandler) BuildPortIDTLV(device *config.Device) []byte {
	return h.buildPortIDTLV(device)
}

// BuildCapabilitiesTLV exposes buildCapabilitiesTLV for testing.
func (h *CDPHandler) BuildCapabilitiesTLV(device *config.Device) []byte {
	return h.buildCapabilitiesTLV(device)
}

// BuildSoftwareVersionTLV exposes buildSoftwareVersionTLV for testing.
func (h *CDPHandler) BuildSoftwareVersionTLV(device *config.Device) []byte {
	return h.buildSoftwareVersionTLV(device)
}

// BuildPlatformTLV exposes buildPlatformTLV for testing.
func (h *CDPHandler) BuildPlatformTLV(device *config.Device) []byte {
	return h.buildPlatformTLV(device)
}

// CDPCalculateChecksum exposes calculateChecksum for testing.
func (h *CDPHandler) CDPCalculateChecksum(data []byte) uint16 { return h.calculateChecksum(data) }

// BuildCDPFrame exposes buildCDPFrame for testing.
func (h *CDPHandler) BuildCDPFrame(device *config.Device) []byte { return h.buildCDPFrame(device) }

// CDPSendAdvertisements exposes sendAdvertisements for testing.
func (h *CDPHandler) CDPSendAdvertisements() { h.sendAdvertisements() }

// ---- DHCPHandler exports ----

// DHCPHandlerStack returns the handler's stack for testing.
func (h *DHCPHandler) DHCPHandlerStack() *Stack { return h.stack }

// DHCPHandlerLeases returns the leases map for testing.
func (h *DHCPHandler) DHCPHandlerLeases() map[string]*DHCPLease { return h.leases }

// DHCPHandlerIPPool returns the IP pool for testing.
func (h *DHCPHandler) DHCPHandlerIPPool() []net.IP { return h.ipPool }

// DHCPHandlerSubnetMask returns the subnet mask for testing.
func (h *DHCPHandler) DHCPHandlerSubnetMask() net.IP { return h.subnetMask }

// DHCPHandlerServerIP returns the server IP for testing.
func (h *DHCPHandler) DHCPHandlerServerIP() net.IP { return h.serverIP }

// DHCPHandlerGateway returns the gateway for testing.
func (h *DHCPHandler) DHCPHandlerGateway() net.IP { return h.gateway }

// DHCPHandlerDNSServers returns the DNS servers for testing.
func (h *DHCPHandler) DHCPHandlerDNSServers() []net.IP { return h.dnsServers }

// DHCPHandlerDomainName returns the domain name for testing.
func (h *DHCPHandler) DHCPHandlerDomainName() string { return h.domainName }

// DHCPHandlerNTPServers returns the NTP servers for testing.
func (h *DHCPHandler) DHCPHandlerNTPServers() []net.IP { return h.ntpServers }

// DHCPHandlerDomainSearch returns the domain search list for testing.
func (h *DHCPHandler) DHCPHandlerDomainSearch() []string { return h.domainSearch }

// DHCPHandlerTFTPServerName returns the TFTP server name for testing.
func (h *DHCPHandler) DHCPHandlerTFTPServerName() string { return h.tftpServerName }

// DHCPHandlerBootfileName returns the bootfile name for testing.
func (h *DHCPHandler) DHCPHandlerBootfileName() string { return h.bootfileName }

// DHCPHandlerPoolStart returns the pool start IP for testing.
func (h *DHCPHandler) DHCPHandlerPoolStart() net.IP { return h.poolStart }

// DHCPHandlerPoolEnd returns the pool end IP for testing.
func (h *DHCPHandler) DHCPHandlerPoolEnd() net.IP { return h.poolEnd }

// GenerateIPPool exposes generateIPPool for testing.
func (h *DHCPHandler) GenerateIPPool(start, end net.IP) ([]net.IP, error) {
	return h.generateIPPool(start, end)
}

// FindAvailableIP exposes findAvailableIP for testing.
func (h *DHCPHandler) FindAvailableIP() net.IP { return h.findAvailableIP() }

// AllocateLease exposes allocateLease for testing.
func (h *DHCPHandler) AllocateLease(
	mac net.HardwareAddr,
	requestedIP net.IP,
	hostname string,
) (*DHCPLease, error) {
	return h.allocateLease(mac, requestedIP, hostname)
}

// IsIPInPool exposes isIPInPool for testing.
func (h *DHCPHandler) IsIPInPool(ip net.IP) bool { return h.isIPInPool(ip) }

// ---- DHCPv6Handler exports ----

// DHCPv6HandlerStack returns the handler's stack for testing.
func (h *DHCPv6Handler) DHCPv6HandlerStack() *Stack { return h.stack }

// DHCPv6HandlerLeases returns the leases map for testing.
func (h *DHCPv6Handler) DHCPv6HandlerLeases() map[string]*DHCPv6Lease { return h.leases }

// DHCPv6HandlerPreferredLifetime returns the preferred lifetime for testing.
func (h *DHCPv6Handler) DHCPv6HandlerPreferredLifetime() time.Duration { return h.preferredLifetime }

// DHCPv6HandlerValidLifetime returns the valid lifetime for testing.
func (h *DHCPv6Handler) DHCPv6HandlerValidLifetime() time.Duration { return h.validLifetime }

// DHCPv6HandlerServerDUID returns the server DUID for testing.
func (h *DHCPv6Handler) DHCPv6HandlerServerDUID() []byte { return h.serverDUID }

// DHCPv6HandlerAddressPool returns the address pool for testing.
func (h *DHCPv6Handler) DHCPv6HandlerAddressPool() []net.IP { return h.addressPool }

// DHCPv6HandlerPrefixPool returns the prefix pool for testing.
func (h *DHCPv6Handler) DHCPv6HandlerPrefixPool() []net.IPNet { return h.prefixPool }

// DHCPv6HandlerDNSServers returns the DNS servers for testing.
func (h *DHCPv6Handler) DHCPv6HandlerDNSServers() []net.IP { return h.dnsServers }

// DHCPv6HandlerDomainList returns the domain list for testing.
func (h *DHCPv6Handler) DHCPv6HandlerDomainList() []string { return h.domainList }

// DHCPv6HandlerSNTPServers returns the SNTP servers for testing.
func (h *DHCPv6Handler) DHCPv6HandlerSNTPServers() []net.IP { return h.sntpServers }

// DHCPv6HandlerNTPServers returns the NTP servers for testing.
func (h *DHCPv6Handler) DHCPv6HandlerNTPServers() []net.IP { return h.ntpServers }

// DHCPv6HandlerSIPServers returns the SIP servers for testing.
func (h *DHCPv6Handler) DHCPv6HandlerSIPServers() []net.IP { return h.sipServers }

// DHCPv6HandlerSIPDomains returns the SIP domains for testing.
func (h *DHCPv6Handler) DHCPv6HandlerSIPDomains() []string { return h.sipDomains }

// DHCPv6AllocateLease exposes allocateLease for testing.
func (h *DHCPv6Handler) DHCPv6AllocateLease(clientDUID []byte, iaid uint32) (*DHCPv6Lease, error) {
	return h.allocateLease(clientDUID, iaid)
}

// ConfirmLease exposes confirmLease for testing.
func (h *DHCPv6Handler) ConfirmLease(clientDUID []byte, iaid uint32) (*DHCPv6Lease, error) {
	return h.confirmLease(clientDUID, iaid)
}

// FindLease exposes findLease for testing.
func (h *DHCPv6Handler) FindLease(clientDUID []byte) *DHCPv6Lease { return h.findLease(clientDUID) }

// RenewLease exposes renewLease for testing.
func (h *DHCPv6Handler) RenewLease(lease *DHCPv6Lease) { h.renewLease(lease) }

// ParseDHCPv6Message exposes parseDHCPv6Message for testing.
func (h *DHCPv6Handler) ParseDHCPv6Message(data []byte) (*DHCPv6Message, error) {
	return h.parseDHCPv6Message(data)
}

// FindOption exposes findOption for testing.
func (h *DHCPv6Handler) FindOption(msg *DHCPv6Message, optionCode uint16) *DHCPv6Option {
	return h.findOption(msg, optionCode)
}

// ExtractClientDUID exposes extractClientDUID for testing.
func (h *DHCPv6Handler) ExtractClientDUID(msg *DHCPv6Message) []byte {
	return h.extractClientDUID(msg)
}

// ExtractIANA exposes extractIANA for testing.
func (h *DHCPv6Handler) ExtractIANA(msg *DHCPv6Message) (uint32, bool) {
	return h.extractIANA(msg)
}

// EncodeDomainList exposes encodeDomainList for testing.
func (h *DHCPv6Handler) EncodeDomainList(domains []string) []byte { return h.encodeDomainList(domains) }

// MessageTypeString exposes messageTypeString for testing.
func (h *DHCPv6Handler) MessageTypeString(msgType uint8) string {
	return h.messageTypeString(msgType)
}

// BuildIANAOption exposes buildIANAOption for testing.
func (h *DHCPv6Handler) BuildIANAOption(lease *DHCPv6Lease) DHCPv6Option {
	return h.buildIANAOption(lease)
}

// BuildIAAddrOption exposes buildIAAddrOption for testing.
func (h *DHCPv6Handler) BuildIAAddrOption(lease *DHCPv6Lease) DHCPv6Option {
	return h.buildIAAddrOption(lease)
}

// ---- DNSHandler exports ----

// DNSHandlerStack returns the handler's stack for testing.
func (h *DNSHandler) DNSHandlerStack() *Stack { return h.stack }

// DNSHandlerRecords returns the records map for testing.
func (h *DNSHandler) DNSHandlerRecords() map[string][]dnsRecord { return h.records }

// DNSHandlerPTRRecords returns the PTR records map for testing.
func (h *DNSHandler) DNSHandlerPTRRecords() map[string]dnsPTR { return h.ptrRecords }

// DNSHandlerDomain returns the domain for testing.
func (h *DNSHandler) DNSHandlerDomain() string { return h.domain }

// LookupHost exposes lookupHost for testing.
func (h *DNSHandler) LookupHost(hostname string, set *dnsRecordSet) []dnsRecord {
	return h.lookupHost(hostname, set)
}

// ResolveQuestions exposes resolveQuestions for testing.
func (h *DNSHandler) ResolveQuestions(
	questions []layers.DNSQuestion,
	set *dnsRecordSet,
	debugLevel int,
	serial int,
) ([]layers.DNSResourceRecord, layers.DNSResponseCode) {
	return h.resolveQuestions(questions, set, debugLevel, serial)
}

// ---- EDPHandler exports ----

// EDPHandlerStack returns the handler's stack for testing.
func (h *EDPHandler) EDPHandlerStack() *Stack { return h.stack }

// EDPHandlerStopChan returns the stop channel for testing.
func (h *EDPHandler) EDPHandlerStopChan() chan struct{} { return h.stopChan }

// EDPHandlerAdvertiseTicker returns the ticker for testing.
func (h *EDPHandler) EDPHandlerAdvertiseTicker() *time.Ticker { return h.advertiseTicker }

// BuildDisplayTLV exposes buildDisplayTLV for testing.
func (h *EDPHandler) BuildDisplayTLV(device *config.Device) []byte { return h.buildDisplayTLV(device) }

// BuildInfoTLV exposes buildInfoTLV for testing.
func (h *EDPHandler) BuildInfoTLV(device *config.Device) []byte { return h.buildInfoTLV(device) }

// BuildNullTLV exposes buildNullTLV for testing.
func (h *EDPHandler) BuildNullTLV() []byte { return h.buildNullTLV() }

// EDPCalculateChecksum exposes calculateChecksum for testing.
func (h *EDPHandler) EDPCalculateChecksum(data []byte) uint16 { return h.calculateChecksum(data) }

// BuildEDPFrame exposes buildEDPFrame for testing.
func (h *EDPHandler) BuildEDPFrame(device *config.Device) []byte { return h.buildEDPFrame(device) }

// EDPSendAdvertisements exposes sendAdvertisements for testing.
func (h *EDPHandler) EDPSendAdvertisements() { h.sendAdvertisements() }

// ---- FDPHandler exports ----

// FDPHandlerStack returns the handler's stack for testing.
func (h *FDPHandler) FDPHandlerStack() *Stack { return h.stack }

// FDPHandlerStopChan returns the stop channel for testing.
func (h *FDPHandler) FDPHandlerStopChan() chan struct{} { return h.stopChan }

// FDPHandlerAdvertiseTicker returns the ticker for testing.
func (h *FDPHandler) FDPHandlerAdvertiseTicker() *time.Ticker { return h.advertiseTicker }

// FDPBuildLLCSNAPHeader exposes buildLLCSNAPHeader for testing.
func (h *FDPHandler) FDPBuildLLCSNAPHeader() []byte { return h.buildLLCSNAPHeader() }

// FDPBuildDeviceIDTLV exposes buildDeviceIDTLV for testing.
func (h *FDPHandler) FDPBuildDeviceIDTLV(device *config.Device) []byte {
	return h.buildDeviceIDTLV(device)
}

// BuildPortTLV exposes buildPortTLV for testing.
func (h *FDPHandler) BuildPortTLV(device *config.Device) []byte { return h.buildPortTLV(device) }

// FDPBuildPlatformTLV exposes buildPlatformTLV for testing.
func (h *FDPHandler) FDPBuildPlatformTLV(device *config.Device) []byte {
	return h.buildPlatformTLV(device)
}

// FDPBuildCapabilitiesTLV exposes buildCapabilitiesTLV for testing.
func (h *FDPHandler) FDPBuildCapabilitiesTLV(device *config.Device) []byte {
	return h.buildCapabilitiesTLV(device)
}

// BuildSoftwareTLV exposes buildSoftwareTLV for testing.
func (h *FDPHandler) BuildSoftwareTLV(device *config.Device) []byte {
	return h.buildSoftwareTLV(device)
}

// BuildIPAddressTLV exposes buildIPAddressTLV for testing.
func (h *FDPHandler) BuildIPAddressTLV(device *config.Device) []byte {
	return h.buildIPAddressTLV(device)
}

// FDPCalculateChecksum exposes calculateChecksum for testing.
func (h *FDPHandler) FDPCalculateChecksum(data []byte) uint16 { return h.calculateChecksum(data) }

// BuildFDPFrame exposes buildFDPFrame for testing.
func (h *FDPHandler) BuildFDPFrame(device *config.Device) []byte { return h.buildFDPFrame(device) }

// FDPSendAdvertisements exposes sendAdvertisements for testing.
func (h *FDPHandler) FDPSendAdvertisements() { h.sendAdvertisements() }

// ---- FTPHandler exports ----

// FTPHandlerStack returns the handler's stack for testing.
func (h *FTPHandler) FTPHandlerStack() *Stack { return h.stack }

// BuildFTPResponse exposes buildFTPResponse for testing.
func (h *FTPHandler) BuildFTPResponse(command string, passive bool, devices []*config.Device) string {
	return h.buildFTPResponse(command, passive, devices)
}

// ---- HTTPHandler exports ----

// HTTPHandlerStack returns the handler's stack for testing.
func (h *HTTPHandler) HTTPHandlerStack() *Stack { return h.stack }

// GenerateResponse exposes generateResponse for testing.
func (h *HTTPHandler) GenerateResponse(request *HTTPRequest, devices []*config.Device) []byte {
	return h.generateResponse(request, devices)
}

// ---- ICMPHandler exports ----

// ICMPHandlerStack returns the handler's stack for testing.
func (h *ICMPHandler) ICMPHandlerStack() *Stack { return h.stack }

// SendEchoReply exposes sendEchoReply for testing.
func (h *ICMPHandler) SendEchoReply(
	srcMAC, dstMAC net.HardwareAddr,
	srcIP, dstIP net.IP,
	id, seq uint16,
	payload []byte,
	device *config.Device,
) error {
	return h.sendEchoReply(srcMAC, dstMAC, srcIP, dstIP, id, seq, payload, device, 0)
}

// ---- ICMPv6Handler exports ----

// GetTypeName exposes getTypeName for testing.
func (h *ICMPv6Handler) GetTypeName(msgType uint8) string { return h.getTypeName(msgType) }

// ---- IPHandler exports ----

// IPHandlerStack returns the handler's stack for testing.
func (h *IPHandler) IPHandlerStack() *Stack { return h.stack }

// ---- LLDPHandler exports ----

// LLDPHandlerStack returns the handler's stack for testing.
func (h *LLDPHandler) LLDPHandlerStack() *Stack { return h.stack }

// LLDPHandlerStopChan returns the stop channel for testing.
func (h *LLDPHandler) LLDPHandlerStopChan() chan struct{} { return h.stopChan }

// LLDPHandlerAdvertiseTicker returns the ticker for testing.
func (h *LLDPHandler) LLDPHandlerAdvertiseTicker() *time.Ticker { return h.advertiseTicker }

// BuildChassisIDTLV exposes buildChassisIDTLV for testing.
func (h *LLDPHandler) BuildChassisIDTLV(device *config.Device) []byte {
	return h.buildChassisIDTLV(device)
}

// LLDPBuildPortIDTLV exposes buildPortIDTLV for testing.
func (h *LLDPHandler) LLDPBuildPortIDTLV(device *config.Device) []byte {
	return h.buildPortIDTLV(device)
}

// BuildTTLTLV exposes buildTTLTLV for testing.
func (h *LLDPHandler) BuildTTLTLV(device *config.Device) []byte { return h.buildTTLTLV(device) }

// BuildPortDescriptionTLV exposes buildPortDescriptionTLV for testing.
func (h *LLDPHandler) BuildPortDescriptionTLV(device *config.Device) []byte {
	return h.buildPortDescriptionTLV(device)
}

// BuildSystemNameTLV exposes buildSystemNameTLV for testing.
func (h *LLDPHandler) BuildSystemNameTLV(device *config.Device) []byte {
	return h.buildSystemNameTLV(device)
}

// BuildSystemDescriptionTLV exposes buildSystemDescriptionTLV for testing.
func (h *LLDPHandler) BuildSystemDescriptionTLV(device *config.Device) []byte {
	return h.buildSystemDescriptionTLV(device)
}

// BuildEndTLV exposes buildEndTLV for testing.
func (h *LLDPHandler) BuildEndTLV() []byte { return h.buildEndTLV() }

// BuildManagementAddressTLV exposes buildManagementAddressTLV for testing.
func (h *LLDPHandler) BuildManagementAddressTLV(device *config.Device) []byte {
	return h.buildManagementAddressTLV(device)
}

// BuildLLDPFrame exposes buildLLDPFrame for testing.
func (h *LLDPHandler) BuildLLDPFrame(device *config.Device) []byte { return h.buildLLDPFrame(device) }

// LLDPSendAdvertisements exposes sendAdvertisements for testing.
func (h *LLDPHandler) LLDPSendAdvertisements() { h.sendAdvertisements() }

// ---- NetBIOSHandler exports ----

// EncodeNetBIOSName exposes encodeNetBIOSName for testing.
func (h *NetBIOSHandler) EncodeNetBIOSName(name string, nameType byte) []byte {
	return h.encodeNetBIOSName(name, nameType)
}

// DecodeNetBIOSName exposes decodeNetBIOSName for testing.
func (h *NetBIOSHandler) DecodeNetBIOSName(data []byte) (string, byte, int) {
	return h.decodeNetBIOSName(data)
}

// ---- STPHandler exports ----

// MakeBridgeID exposes makeBridgeID for testing.
func (h *STPHandler) MakeBridgeID(priority uint16, mac net.HardwareAddr) uint64 {
	return h.makeBridgeID(priority, mac)
}

// STPHandlerBridgePriority returns the bridge priority for testing.
func (h *STPHandler) STPHandlerBridgePriority() uint16 { return h.bridgePriority }

// STPSetDebugLevel exposes SetDebugLevel for testing.
func (h *STPHandler) STPSetDebugLevel(level int) { h.SetDebugLevel(level) }

// STPGetSTPParams exposes getSTPParams for testing.
// Returns bridgePriority, helloTime, maxAge, forwardDelay.
func (h *STPHandler) STPGetSTPParams(device *config.Device) (uint16, uint16, uint16, uint16) {
	p := h.getSTPParams(device)
	return p.bridgePriority, p.helloTime, p.maxAge, p.forwardDelay
}

// ---- DHCP unexported function exports ----

// DHCPMessageTypeString exposes dhcpMessageTypeString for testing.
func (h *DHCPHandler) DHCPMessageTypeString(msgType uint8) string {
	return h.dhcpMessageTypeString(msgType)
}

// DHCPEncodeUint32 exposes encodeUint32 for testing.
func (h *DHCPHandler) DHCPEncodeUint32(val uint32) []byte {
	return h.encodeUint32(val)
}

// DHCPEncodeDomainSearchList exposes encodeDomainSearchList for testing.
func (h *DHCPHandler) DHCPEncodeDomainSearchList(domains []string) ([]byte, error) {
	return h.encodeDomainSearchList(domains)
}

// DHCPMatchStaticLease exposes matchStaticLease for testing.
func (h *DHCPHandler) DHCPMatchStaticLease(mac net.HardwareAddr) net.IP {
	return h.matchStaticLease(mac)
}

// DHCPIsIPLeased exposes isIPLeased for testing.
func (h *DHCPHandler) DHCPIsIPLeased(ip net.IP) bool {
	return h.isIPLeased(ip)
}

// MACMatchesMask exposes macMatchesMask for testing.
func MACMatchesMask(mac, match, mask net.HardwareAddr) bool {
	return macMatchesMask(mac, match, mask)
}

// SplitDomain exposes splitDomain for testing.
func SplitDomain(domain string) []string {
	return splitDomain(domain)
}

// ---- DHCPv6 unexported function exports ----

// DHCPv6SetAddressPool exposes SetAddressPool for testing.
func (h *DHCPv6Handler) DHCPv6SetAddressPool(addresses []net.IP) {
	h.SetAddressPool(addresses)
}

// DHCPv6SetServerConfig exposes SetServerConfig for testing.
func (h *DHCPv6Handler) DHCPv6SetServerConfig(dnsServers []net.IP, domainList []string) {
	h.SetServerConfig(dnsServers, domainList)
}

// DHCPv6SetAdvancedOptions exposes SetAdvancedOptions for testing.
func (h *DHCPv6Handler) DHCPv6SetAdvancedOptions(sntpServers, ntpServers, sipServers []net.IP, sipDomains []string) {
	h.SetAdvancedOptions(sntpServers, ntpServers, sipServers, sipDomains)
}

// ---- DNS unexported function exports ----

// DNSAddRecord exposes AddRecord for testing.
func (h *DNSHandler) DNSAddRecord(hostname string, ip net.IP) {
	h.AddRecord(hostname, ip)
}

// DNSAddRecordWithTTL exposes AddRecordWithTTL for testing.
func (h *DNSHandler) DNSAddRecordWithTTL(hostname string, ip net.IP, ttl uint32, rcode layers.DNSResponseCode) {
	h.AddRecordWithTTL(hostname, ip, ttl, rcode)
}

// DNSSetDomain exposes SetDomain for testing.
func (h *DNSHandler) DNSSetDomain(domain string) {
	h.SetDomain(domain)
}

// DNSLoadDeviceRecords exposes LoadDeviceRecords for testing.
func (h *DNSHandler) DNSLoadDeviceRecords(devices []*config.Device) {
	h.LoadDeviceRecords(devices)
}

// ---- Device table exports ----

// DTRegisterTTL exposes RegisterTTL for testing.
func (dt *DeviceTable) DTRegisterTTL(device *config.Device) {
	dt.RegisterTTL(device)
}

// DTGetDeviceByTTL exposes GetDeviceByTTL for testing.
func (dt *DeviceTable) DTGetDeviceByTTL(ttl int) *config.Device {
	return dt.GetDeviceByTTL(ttl)
}

// DTIncrementTTLCount exposes IncrementTTLCount for testing.
func (dt *DeviceTable) DTIncrementTTLCount(device *config.Device) {
	dt.IncrementTTLCount(device)
}

// DTRegisterForwardingDevice exposes RegisterForwardingDevice for testing.
func (dt *DeviceTable) DTRegisterForwardingDevice(device *config.Device) {
	dt.RegisterForwardingDevice(device)
}

// DTGetForwardingDevices exposes GetForwardingDevices for testing.
func (dt *DeviceTable) DTGetForwardingDevices() []*config.Device {
	return dt.GetForwardingDevices()
}

// GetAllDHCPRelayAgentsAndServersExported exposes GetAllDHCPRelayAgentsAndServers for testing.
func GetAllDHCPRelayAgentsAndServersExported() net.IP {
	return GetAllDHCPRelayAgentsAndServers()
}

// GetAllDHCPServersExported exposes GetAllDHCPServers for testing.
func GetAllDHCPServersExported() net.IP {
	return GetAllDHCPServers()
}

// DHCPv6FindAvailableAddress exposes findAvailableAddress for testing via allocateLease.
// Note: allocateLease is already exported via DHCPv6AllocateLease.
