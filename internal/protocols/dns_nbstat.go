package protocols

import (
	"encoding/binary"
	"fmt"
	"net"
	"strings"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/safeconv"
)

func (h *DNSHandler) handleNBSTATQuery(
	_ *Packet,
	ipLayer *layers.IPv4,
	udpLayer *layers.UDP,
	serverDevice *config.Device,
	id uint16,
	q layers.DNSQuestion,
	packet gopacket.Packet,
) error {
	payload := h.buildNBSTATResponse(serverDevice, id, q)
	if len(payload) == 0 {
		return nil
	}

	ethLayer := packet.Layer(layers.LayerTypeEthernet)

	var dstMAC net.HardwareAddr

	if eth, ok := ethLayer.(*layers.Ethernet); ok {
		dstMAC = eth.SrcMAC
	}

	return h.stack.udpHandler.SendUDP(
		serverDeviceIP(serverDevice, false),
		ipLayer.SrcIP,
		dnsPort,
		uint16(udpLayer.SrcPort),
		payload,
		[]byte(serverDevice.MACAddress),
		[]byte(dstMAC),
	)
}

func (h *DNSHandler) handleNBSTATQueryV6(
	_ *Packet,
	ipv6 *layers.IPv6,
	udpLayer *layers.UDP,
	serverDevice *config.Device,
	id uint16,
	q layers.DNSQuestion,
	packet gopacket.Packet,
) error {
	payload := h.buildNBSTATResponse(serverDevice, id, q)
	if len(payload) == 0 {
		return nil
	}

	ethLayer := packet.Layer(layers.LayerTypeEthernet)

	var dstMAC net.HardwareAddr

	if eth, ok := ethLayer.(*layers.Ethernet); ok {
		dstMAC = eth.SrcMAC
	}

	return h.sendDNSPayloadV6(
		payload,
		serverDeviceIP(serverDevice, true),
		ipv6.SrcIP,
		serverDevice.MACAddress,
		dstMAC,
		udpLayer.SrcPort,
	)
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

func (h *DNSHandler) sendDNSPayloadV6(
	payload []byte,
	srcIP, dstIP net.IP,
	srcMAC, dstMAC net.HardwareAddr,
	dstPort layers.UDPPort,
) error {
	udp := &layers.UDP{
		SrcPort: dnsPort,
		DstPort: dstPort,
	}
	ip := &layers.IPv6{
		Version:      dnsIPv6Version,
		TrafficClass: 0,
		FlowLabel:    0,
		NextHeader:   layers.IPProtocolUDP,
		HopLimit:     dnsIPv6HopLimit,
		SrcIP:        srcIP,
		DstIP:        dstIP,
	}
	eth := &layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetTypeIPv6,
	}
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}

	_ = udp.SetNetworkLayerForChecksum(ip) // error is non-critical for simulation
	err := gopacket.SerializeLayers(buf, opts, eth, ip, udp, gopacket.Payload(payload))
	if err != nil {
		return fmt.Errorf("failed to serialize DNS/IPv6 response: %w", err)
	}

	return h.stack.SendRawPacket(buf.Bytes())
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
	question = append(question, byte(q.Type>>dnsByteShift), safeconv.ByteFromUint16(uint16(q.Type)))
	question = append(question, byte(q.Class>>dnsByteShift), safeconv.ByteFromUint16(uint16(q.Class)))

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
	answer = append(answer, byte(q.Type>>dnsByteShift), safeconv.ByteFromUint16(uint16(q.Type)))
	answer = append(answer, byte(q.Class>>dnsByteShift), safeconv.ByteFromUint16(uint16(q.Class)))
	// TTL = 0
	answer = append(answer, dnsTerminator, dnsTerminator, dnsTerminator, dnsTerminator)
	answer = append(answer, safeconv.Byte(rdLength>>dnsByteShift), safeconv.Byte(rdLength))

	// RDATA: name count + name entries
	answer = append(answer, safeconv.Byte(len(names)))
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

		answer = append(answer, byte(nameFlags>>dnsByteShift), safeconv.ByteFromUint16(nameFlags))
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

		buf = append(buf, safeconv.Byte(len(label)))
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

type nbstatNameEntry struct {
	Name   string
	Suffix byte
	Group  bool
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
