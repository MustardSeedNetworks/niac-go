package protocols

import (
	"encoding/binary"
	"net"
	"strings"

	"github.com/google/gopacket/layers"

	"github.com/krisarmstrong/niac-go/internal/config"
)

// NetBIOS NBSTAT name entry structure.
type nbstatNameEntry struct {
	Name   string
	Suffix byte
	Group  bool
}

// lookupHost looks up IP addresses for a hostname.
func (h *DNSHandler) lookupHost(hostname string, set *dnsRecordSet) []dnsRecord {
	h.mu.RLock()
	defer h.mu.RUnlock()

	hostname = strings.ToLower(strings.TrimSuffix(hostname, "."))

	// Device-specific records
	if set != nil {
		if recs, ok := set.forward[hostname]; ok {
			return recs
		}

		if !strings.Contains(hostname, ".") {
			fullname := hostname + "." + h.domain
			if recs, ok := set.forward[fullname]; ok {
				return recs
			}
		}

		return nil
	}

	// Global records
	if recs, ok := h.records[hostname]; ok {
		return recs
	}

	if !strings.Contains(hostname, ".") {
		fullname := hostname + "." + h.domain
		if recs, ok := h.records[fullname]; ok {
			return recs
		}
	}

	return nil
}

func (h *DNSHandler) lookupPTR(ip net.IP, set *dnsRecordSet) (dnsPTR, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if set != nil {
		rec, ok := set.reverse[ip.String()]

		return rec, ok
	}

	rec, ok := h.ptrRecords[ip.String()]

	return rec, ok
}

func (h *DNSHandler) selectServerDevice(
	devices []*config.Device,
	wantIPv6 bool,
) (*config.Device, net.IP) {
	for _, dev := range devices {
		if !h.deviceHasDNSRecords(dev) {
			continue
		}

		ip := pickIPAddressForDNS(dev, wantIPv6)
		if ip == nil {
			continue
		}

		if len(dev.MACAddress) == 0 {
			continue
		}

		return dev, ip
	}

	return nil, nil
}

func (h *DNSHandler) deviceHasDNSRecords(dev *config.Device) bool {
	if dev == nil {
		return false
	}

	h.mu.RLock()
	_, hasSet := h.deviceRecords[dev]
	hasGlobal := len(h.records) > 0
	h.mu.RUnlock()

	if hasSet {
		return true
	}

	if dev.DNSConfig != nil {
		return true
	}
	// If no device-specific record set exists, fall back to global records
	return dev.DNSConfig == nil && hasGlobal
}

func (h *DNSHandler) getRecordSetForDevice(dev *config.Device) *dnsRecordSet {
	if dev == nil {
		return nil
	}

	h.mu.RLock()
	set := h.deviceRecords[dev]
	h.mu.RUnlock()

	return set
}

func pickIPAddressForDNS(device *config.Device, wantIPv6 bool) net.IP {
	for _, ip := range device.IPAddresses {
		if wantIPv6 {
			if ip.To4() == nil && ip.To16() != nil {
				return ip
			}
		} else if v4 := ip.To4(); v4 != nil {
			return v4
		}
	}

	return nil
}

func serverDeviceIP(device *config.Device, wantIPv6 bool) net.IP {
	if device == nil {
		return nil
	}

	for _, ip := range device.IPAddresses {
		if wantIPv6 {
			if ip.To4() == nil && ip.To16() != nil {
				return ip
			}
		} else if v4 := ip.To4(); v4 != nil {
			return v4
		}
	}

	return nil
}

// buildNBSTATHeader builds the DNS header for an NBSTAT response.
func buildNBSTATHeader(id uint16) []byte {
	header := make([]byte, dnsHeaderSize)
	binary.BigEndian.PutUint16(header[0:2], id)
	flags := uint16(dnsFlagQR | dnsFlagAA)
	binary.BigEndian.PutUint16(header[2:4], flags)
	binary.BigEndian.PutUint16(header[4:6], 1) // QDCOUNT
	binary.BigEndian.PutUint16(header[6:8], 1) // ANCOUNT

	return header
}

// buildNBSTATQuestion builds the question section for an NBSTAT response.
func buildNBSTATQuestion(q layers.DNSQuestion, encodedQName []byte) []byte {
	question := make([]byte, 0, len(encodedQName)+dnsQuestionExtra+1)
	question = append(question, encodedQName...)
	question = append(question, dnsTerminator)
	question = append(question, byte(q.Type>>dnsByteShift), byte(q.Type))
	question = append(question, byte(q.Class>>dnsByteShift), byte(q.Class))

	return question
}

// buildNBSTATAnswer builds the answer section for an NBSTAT response.
func buildNBSTATAnswer(
	q layers.DNSQuestion,
	names []nbstatNameEntry,
	ownerNodeType uint8,
	macAddr net.HardwareAddr,
) []byte {
	rdLength := 1 + nbstatNameEntrySize*len(names) + nbstatMACAndStatsSize
	answer := make([]byte, 0, dnsHeaderSize+rdLength)

	// NAME pointer to question at offset 12 -> 0xC00C
	answer = append(answer, dnsPointerByte, dnsPointerOffset)
	answer = append(answer, byte(q.Type>>dnsByteShift), byte(q.Type))
	answer = append(answer, byte(q.Class>>dnsByteShift), byte(q.Class))
	// TTL = 0
	answer = append(answer, dnsTerminator, dnsTerminator, dnsTerminator, dnsTerminator)
	answer = append(answer, byte(rdLength>>dnsByteShift), byte(rdLength))

	// RDATA: name count + name entries
	answer = append(answer, byte(len(names)))
	answer = appendNBSTATNameEntries(answer, names, ownerNodeType)
	answer = appendNBSTATMACAndStats(answer, macAddr)

	return answer
}

// appendNBSTATNameEntries appends NetBIOS name entries to the answer buffer.
func appendNBSTATNameEntries(answer []byte, names []nbstatNameEntry, ownerNodeType uint8) []byte {
	for _, name := range names {
		rawName := name.Name
		if len(rawName) > netbiosNameLen {
			rawName = rawName[:netbiosNameLen]
		}

		nameBytes := make([]byte, netbiosNameLen)
		copy(nameBytes, rawName)

		for i := len(rawName); i < netbiosNameLen; i++ {
			nameBytes[i] = ' '
		}

		answer = append(answer, nameBytes...)
		answer = append(answer, name.Suffix)

		nameFlags := uint16(dnsFlagAA) | (uint16(ownerNodeType) << nbstatOwnerTypeShift)
		if name.Group {
			nameFlags |= netbiosGroupFlag
		}

		answer = append(answer, byte(nameFlags>>dnsByteShift), byte(nameFlags))
	}

	return answer
}

// appendNBSTATMACAndStats appends MAC address and statistics to the answer buffer.
func appendNBSTATMACAndStats(answer []byte, macAddr net.HardwareAddr) []byte {
	if len(macAddr) == dnsMACOctets {
		answer = append(answer, macAddr...)
	} else {
		answer = append(answer, []byte{0, 0, 0, 0, 0, 0}...)
	}

	answer = append(answer, make([]byte, netbiosStatsSize)...)

	return answer
}

func (h *DNSHandler) buildNBSTATResponse(
	device *config.Device,
	id uint16,
	q layers.DNSQuestion,
) []byte {
	if device == nil || device.NetBIOSConfig == nil {
		return nil
	}

	service := decodeNBSTATService(q.Name)
	if !isNBSTATServiceSupported(service, device) {
		return nil
	}

	names := netbiosNamesForDevice(device)
	if len(names) == 0 {
		return nil
	}

	header := buildNBSTATHeader(id)
	question := buildNBSTATQuestion(q, encodeDNSName(q.Name))
	answer := buildNBSTATAnswer(q, names, netbiosOwnerNodeType(device), device.MACAddress)

	payload := make([]byte, dnsHeaderSize+len(question)+len(answer))
	copy(payload, header)
	copy(payload[dnsHeaderSize:], question)
	copy(payload[dnsHeaderSize+len(question):], answer)

	return payload
}

func encodeDNSName(name []byte) []byte {
	trimmed := strings.TrimSuffix(string(name), ".")
	if trimmed == "" {
		return []byte{0}
	}

	labels := strings.Split(trimmed, ".")
	buf := make([]byte, 0, len(trimmed)+dnsBufPadding)

	for _, label := range labels {
		if label == "" {
			continue
		}

		buf = append(buf, byte(len(label)))
		buf = append(buf, []byte(label)...)
	}

	return buf
}

func decodeNBSTATService(name []byte) string {
	base := strings.TrimSuffix(string(name), ".")
	if idx := strings.IndexByte(base, '.'); idx != -1 {
		base = base[:idx]
	}

	if len(base) < dnsMinLabelParts {
		return ""
	}

	length := len(base) / nbNibbleEncodingFactor

	decoded := make([]byte, 0, length)

	for i := range length {
		hi := base[2*i] - 'A'

		lo := base[2*i+1] - 'A'

		if hi > nbMaxNibbleValue || lo > nbMaxNibbleValue {
			return ""
		}

		decoded = append(decoded, (hi<<nbNibbleShift)|lo)
	}

	return string(decoded)
}

func isNBSTATServiceSupported(service string, device *config.Device) bool {
	if device == nil || device.NetBIOSConfig == nil {
		return false
	}

	workstation := string(append([]byte{'*'}, make([]byte, netbiosNameLen)...))
	masterBrowser := string(
		[]byte{0x01, 0x02, '_', '_', 'M', 'S', 'B', 'R', 'O', 'W', 'S', 'E', '_', '_', 0x02, 0x01},
	)

	if service == workstation {
		return true
	}

	if service == masterBrowser {
		return device.NetBIOSConfig.MsBrowse
	}

	return false
}

func netbiosNamesForDevice(device *config.Device) []nbstatNameEntry {
	if device == nil || device.NetBIOSConfig == nil {
		return nil
	}

	cfg := device.NetBIOSConfig
	names := make([]nbstatNameEntry, 0)

	if len(cfg.Names) > 0 {
		for _, n := range cfg.Names {
			names = append(names, nbstatNameEntry{Name: n.Name, Suffix: n.Suffix, Group: n.Group})
		}

		if cfg.MsBrowse {
			names = append(
				names,
				nbstatNameEntry{Name: "__MSBROWSE__", Suffix: nbNameTypeMSBrowse, Group: true},
			)
		}

		return names
	}

	baseName := cfg.Name
	if baseName == "" {
		baseName = device.Name
	}

	for _, svc := range cfg.Services {
		switch strings.ToLower(svc) {
		case "workstation":
			names = append(
				names,
				nbstatNameEntry{Name: baseName, Suffix: nbNameTypeWorkstation, Group: false},
			)
		case "messenger":
			names = append(
				names,
				nbstatNameEntry{Name: baseName, Suffix: nbNameTypeMessenger, Group: false},
			)
		case "fileserver":
			names = append(
				names,
				nbstatNameEntry{Name: baseName, Suffix: nbNameTypeFileServer, Group: false},
			)
		case "domainmaster":
			names = append(
				names,
				nbstatNameEntry{Name: baseName, Suffix: nbNameTypeDomainMaster, Group: true},
			)
		case "masterbrowser":
			names = append(
				names,
				nbstatNameEntry{Name: baseName, Suffix: nbNameTypeMasterBrowser, Group: true},
			)
		case "browser":
			names = append(
				names,
				nbstatNameEntry{Name: baseName, Suffix: nbNameTypeBrowserElec, Group: true},
			)
		case "msbrowse":
			names = append(
				names,
				nbstatNameEntry{Name: "__MSBROWSE__", Suffix: nbNameTypeMSBrowse, Group: true},
			)
		}
	}

	if cfg.MsBrowse {
		names = append(
			names,
			nbstatNameEntry{Name: "__MSBROWSE__", Suffix: nbNameTypeMSBrowse, Group: true},
		)
	}

	return names
}

func netbiosOwnerNodeType(device *config.Device) uint8 {
	if device == nil || device.NetBIOSConfig == nil {
		return nbNodeTypeB
	}

	switch strings.ToUpper(device.NetBIOSConfig.NodeType) {
	case "P":
		return nbNodeTypeP
	case "M":
		return nbNodeTypeM
	case "H":
		return nbNodeTypeH
	default:
		return nbNodeTypeB
	}
}

// isValidDNSName validates DNS name length per RFC 1035
// SECURITY FIX MEDIUM-2: Prevents malformed DNS responses.
func isValidDNSName(name []byte) bool {
	// RFC 1035: Maximum domain name length is 255 bytes
	if len(name) > dnsMaxNameLen {
		return false
	}

	// Validate individual label lengths (max 63 bytes per label)
	labels := strings.SplitSeq(string(name), ".")
	for label := range labels {
		if len(label) > dnsMaxLabelLen {
			return false
		}
	}

	return true
}
