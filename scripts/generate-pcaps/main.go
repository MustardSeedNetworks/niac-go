// Package main generates example PCAP files for LLDP and CDP discovery protocol testing.
package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
)

// Protocol constants.
const (
	outputDir     = "examples/pcaps"
	lldpEtherType = 0x88CC
	snapLen       = 65535
	lldpInterval  = 30 * time.Second
	cdpInterval   = 60 * time.Second
	mixedInterval = 15 * time.Second
	frameCount    = 5
	mixedDevices  = 3
	dirPerms      = 0o750
	ethHeaderLen  = 14
	minFrameLen   = 60
)

// LLDP TLV type constants (IEEE 802.1AB).
const (
	lldpTLVChassisID = 1
	lldpTLVPortID    = 2
	lldpTLVTTL       = 3
	lldpTLVPortDesc  = 4
	lldpTLVSysName   = 5
	lldpTLVSysDesc   = 6
	lldpTLVMgmtAddr  = 8
	lldpTTLValue     = 120
	lldpTTLBytes     = 2
	lldpChassisMAC   = 4
	lldpPortIfName   = 5
	lldpMgmtAddrLen  = 5
	lldpMgmtIPv4     = 1
	lldpMgmtIfIndex  = 2
	lldpMgmtIfValue  = 1
	lldpTLVShift     = 9
	lldpTLVLenMask   = 0x01FF
	uint16Mask       = 0xFFFF
)

// CDP TLV type constants.
const (
	cdpVersion      = 0x02
	cdpTTLSeconds   = 180
	cdpTLVDeviceID  = 0x0001
	cdpTLVAddresses = 0x0002
	cdpTLVPortID    = 0x0003
	cdpTLVCaps      = 0x0004
	cdpTLVSoftware  = 0x0005
	cdpTLVPlatform  = 0x0006
	cdpTLVHeaderLen = 4
	cdpAddrCountLen = 4
	cdpAddrLenBytes = 2
	cdpIPv4Len      = 4
	cdpCapsValue    = 0x00000029 // Router+Switch+IGMP
	cdpCapBytes     = 4
	cdpProtoNLPID   = 1
	cdpProtoLen     = 1
	cdpProtoIPv4    = 0xCC
	cdpLLCLen       = 8
	cdpChecksumLen  = 4
)

// cdpLLCSNAPHeader returns the 802.2 LLC/SNAP header for CDP.
func cdpLLCSNAPHeader() [cdpLLCLen]byte {
	return [cdpLLCLen]byte{0xAA, 0xAA, 0x03, 0x00, 0x00, 0x0C, 0x20, 0x00}
}

type deviceInfo struct {
	MAC         net.HardwareAddr
	Name        string
	Description string
	PortID      string
	PortDesc    string
	MgmtIP      net.IP
}

func lldpMulticast() net.HardwareAddr {
	return net.HardwareAddr{0x01, 0x80, 0xC2, 0x00, 0x00, 0x0E}
}

func cdpMulticast() net.HardwareAddr {
	return net.HardwareAddr{0x01, 0x00, 0x0C, 0xCC, 0xCC, 0xCC}
}

func main() {
	if err := os.MkdirAll(outputDir, dirPerms); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create output dir: %v\n", err)
		os.Exit(1)
	}

	devices := buildDeviceList()

	if err := generateLLDP(devices); err != nil {
		fmt.Fprintf(os.Stderr, "LLDP generation failed: %v\n", err)
		os.Exit(1)
	}
	log.Println("Generated: examples/pcaps/lldp-discovery.pcap")

	if err := generateCDP(devices); err != nil {
		fmt.Fprintf(os.Stderr, "CDP generation failed: %v\n", err)
		os.Exit(1)
	}
	log.Println("Generated: examples/pcaps/cdp-discovery.pcap")

	if err := generateMixed(devices[:mixedDevices]); err != nil {
		fmt.Fprintf(os.Stderr, "Mixed generation failed: %v\n", err)
		os.Exit(1)
	}
	log.Println("Generated: examples/pcaps/lldp-cdp-mixed.pcap")
}

func buildDeviceList() []deviceInfo {
	return []deviceInfo{
		{
			MAC:         net.HardwareAddr{0x00, 0x1A, 0x2B, 0x3C, 0x4D, 0x01},
			Name:        "core-sw-01",
			Description: "Cisco IOS Software, C9300 Software, Version 17.6.5",
			PortID:      "Gi1/0/1",
			PortDesc:    "GigabitEthernet1/0/1",
			MgmtIP:      net.ParseIP("10.0.0.1"),
		},
		{
			MAC:         net.HardwareAddr{0x00, 0x1A, 0x2B, 0x3C, 0x4D, 0x02},
			Name:        "dist-sw-01",
			Description: "Arista EOS version 4.28.3M",
			PortID:      "Et1/1",
			PortDesc:    "Ethernet1/1",
			MgmtIP:      net.ParseIP("10.0.1.1"),
		},
		{
			MAC:         net.HardwareAddr{0x00, 0x1A, 0x2B, 0x3C, 0x4D, 0x03},
			Name:        "access-sw-01",
			Description: "ExtremeXOS version 30.7.1.4",
			PortID:      "1:1",
			PortDesc:    "Port 1:1",
			MgmtIP:      net.ParseIP("10.0.2.1"),
		},
		{
			MAC:         net.HardwareAddr{0x00, 0x04, 0x96, 0xAB, 0xCD, 0x01},
			Name:        "juniper-ex01",
			Description: "Juniper Networks EX4300-48P",
			PortID:      "ge-0/0/0",
			PortDesc:    "ge-0/0/0.0",
			MgmtIP:      net.ParseIP("10.0.3.1"),
		},
		{
			MAC:         net.HardwareAddr{0x00, 0xE0, 0x52, 0x11, 0x22, 0x01},
			Name:        "dell-n3248-01",
			Description: "Dell EMC Networking N3248TE-ON",
			PortID:      "Te1/0/1",
			PortDesc:    "TenGigabitEthernet1/0/1",
			MgmtIP:      net.ParseIP("10.0.4.1"),
		},
	}
}

func generateLLDP(devices []deviceInfo) error {
	f, err := os.Create(outputDir + "/lldp-discovery.pcap")
	if err != nil {
		return err
	}
	defer f.Close()

	w := pcapgo.NewWriter(f)
	if writeErr := w.WriteFileHeader(snapLen, layers.LinkTypeEthernet); writeErr != nil {
		return writeErr
	}

	baseTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	for i := range frameCount {
		dev := devices[i%len(devices)]
		ts := baseTime.Add(time.Duration(i) * lldpInterval)
		pkt := buildLLDPFrame(dev)
		ci := gopacket.CaptureInfo{
			Timestamp:     ts,
			CaptureLength: len(pkt),
			Length:        len(pkt),
		}
		if writeErr := w.WritePacket(ci, pkt); writeErr != nil {
			return writeErr
		}
	}
	return nil
}

func generateCDP(devices []deviceInfo) error {
	f, err := os.Create(outputDir + "/cdp-discovery.pcap")
	if err != nil {
		return err
	}
	defer f.Close()

	w := pcapgo.NewWriter(f)
	if writeErr := w.WriteFileHeader(snapLen, layers.LinkTypeEthernet); writeErr != nil {
		return writeErr
	}

	baseTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	for i := range frameCount {
		dev := devices[i%len(devices)]
		ts := baseTime.Add(time.Duration(i) * cdpInterval)
		pkt := buildCDPFrame(dev)
		ci := gopacket.CaptureInfo{
			Timestamp:     ts,
			CaptureLength: len(pkt),
			Length:        len(pkt),
		}
		if writeErr := w.WritePacket(ci, pkt); writeErr != nil {
			return writeErr
		}
	}
	return nil
}

func generateMixed(devices []deviceInfo) error {
	f, err := os.Create(outputDir + "/lldp-cdp-mixed.pcap")
	if err != nil {
		return err
	}
	defer f.Close()

	w := pcapgo.NewWriter(f)
	if writeErr := w.WriteFileHeader(snapLen, layers.LinkTypeEthernet); writeErr != nil {
		return writeErr
	}

	baseTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	frameIdx := 0
	for i := range devices {
		dev := devices[i]
		if writeErr := writeFrame(w, buildLLDPFrame(dev), baseTime, frameIdx); writeErr != nil {
			return writeErr
		}
		frameIdx++
		if writeErr := writeFrame(w, buildCDPFrame(dev), baseTime, frameIdx); writeErr != nil {
			return writeErr
		}
		frameIdx++
	}
	return nil
}

func writeFrame(w *pcapgo.Writer, pkt []byte, baseTime time.Time, idx int) error {
	ts := baseTime.Add(time.Duration(idx) * mixedInterval)
	ci := gopacket.CaptureInfo{
		Timestamp:     ts,
		CaptureLength: len(pkt),
		Length:        len(pkt),
	}
	return w.WritePacket(ci, pkt)
}

func buildLLDPFrame(dev deviceInfo) []byte {
	eth := make([]byte, ethHeaderLen)
	copy(eth[0:6], lldpMulticast())
	copy(eth[6:12], dev.MAC)
	binary.BigEndian.PutUint16(eth[12:ethHeaderLen], lldpEtherType)

	var tlvs []byte
	tlvs = appendLLDPChassisID(tlvs, dev.MAC)
	tlvs = appendLLDPPortID(tlvs, dev.PortID)
	tlvs = appendLLDPTTL(tlvs)
	tlvs = append(tlvs, encodeLLDPTLV(lldpTLVSysName, []byte(dev.Name))...)
	tlvs = append(tlvs, encodeLLDPTLV(lldpTLVSysDesc, []byte(dev.Description))...)
	tlvs = append(tlvs, encodeLLDPTLV(lldpTLVPortDesc, []byte(dev.PortDesc))...)
	tlvs = appendLLDPMgmtAddr(tlvs, dev.MgmtIP)
	// End TLV (type 0, length 0)
	tlvs = append(tlvs, 0, 0)

	frame := make([]byte, 0, len(eth)+len(tlvs))
	frame = append(frame, eth...)
	frame = append(frame, tlvs...)
	return padFrame(frame)
}

func appendLLDPChassisID(tlvs []byte, mac net.HardwareAddr) []byte {
	payload := append([]byte{lldpChassisMAC}, mac...)
	return append(tlvs, encodeLLDPTLV(lldpTLVChassisID, payload)...)
}

func appendLLDPPortID(tlvs []byte, portID string) []byte {
	payload := append([]byte{lldpPortIfName}, []byte(portID)...)
	return append(tlvs, encodeLLDPTLV(lldpTLVPortID, payload)...)
}

func appendLLDPTTL(tlvs []byte) []byte {
	payload := make([]byte, lldpTTLBytes)
	binary.BigEndian.PutUint16(payload, lldpTTLValue)
	return append(tlvs, encodeLLDPTLV(lldpTLVTTL, payload)...)
}

func appendLLDPMgmtAddr(tlvs []byte, ip net.IP) []byte {
	if ip == nil {
		return tlvs
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return tlvs
	}
	mgmt := []byte{lldpMgmtAddrLen, lldpMgmtIPv4}
	mgmt = append(mgmt, ip4...)
	mgmt = append(mgmt, lldpMgmtIfIndex)
	mgmt = append(mgmt, 0, 0, 0, lldpMgmtIfValue)
	mgmt = append(mgmt, 0) // OID string length=0
	return append(tlvs, encodeLLDPTLV(lldpTLVMgmtAddr, mgmt)...)
}

func encodeLLDPTLV(tlvType byte, value []byte) []byte {
	length := len(value)
	maskedLen := uint16(length & lldpTLVLenMask) //nolint:gosec // bounded by protocol
	header := uint16(tlvType)<<lldpTLVShift | maskedLen
	buf := make([]byte, lldpTTLBytes+length)
	binary.BigEndian.PutUint16(buf[0:lldpTTLBytes], header)
	copy(buf[lldpTTLBytes:], value)
	return buf
}

func buildCDPFrame(dev deviceInfo) []byte {
	cdpPayload := buildCDPPayload(dev)
	frameLen := cdpLLCLen + len(cdpPayload)

	eth := make([]byte, ethHeaderLen)
	copy(eth[0:6], cdpMulticast())
	copy(eth[6:12], dev.MAC)
	binary.BigEndian.PutUint16(eth[12:ethHeaderLen], uint16(frameLen&uint16Mask)) //nolint:gosec // bounded by MTU

	frame := make([]byte, 0, ethHeaderLen+cdpLLCLen+len(cdpPayload))
	frame = append(frame, eth...)
	snap := cdpLLCSNAPHeader()
	frame = append(frame, snap[:]...)
	frame = append(frame, cdpPayload...)
	return padFrame(frame)
}

func buildCDPPayload(dev deviceInfo) []byte {
	header := []byte{cdpVersion, cdpTTLSeconds, 0x00, 0x00}

	var tlvs []byte
	tlvs = append(tlvs, encodeCDPTLV(cdpTLVDeviceID, []byte(dev.Name))...)
	tlvs = appendCDPAddresses(tlvs, dev.MgmtIP)
	tlvs = append(tlvs, encodeCDPTLV(cdpTLVPortID, []byte(dev.PortDesc))...)
	tlvs = appendCDPCapabilities(tlvs)
	tlvs = append(tlvs, encodeCDPTLV(cdpTLVSoftware, []byte(dev.Description))...)
	tlvs = append(tlvs, encodeCDPTLV(cdpTLVPlatform, []byte(dev.Name))...)

	payload := make([]byte, 0, len(header)+len(tlvs))
	payload = append(payload, header...)
	payload = append(payload, tlvs...)

	csum := cdpChecksum(payload)
	binary.BigEndian.PutUint16(payload[2:cdpChecksumLen], csum)
	return payload
}

func appendCDPAddresses(tlvs []byte, ip net.IP) []byte {
	if ip == nil {
		return tlvs
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return tlvs
	}

	addrPayload := make([]byte, cdpAddrCountLen)
	binary.BigEndian.PutUint32(addrPayload, 1)

	addrEntry := []byte{cdpProtoNLPID, cdpProtoLen, cdpProtoIPv4}
	addrLen := make([]byte, cdpAddrLenBytes)
	binary.BigEndian.PutUint16(addrLen, cdpIPv4Len)
	addrEntry = append(addrEntry, addrLen...)
	addrEntry = append(addrEntry, ip4...)

	combined := make([]byte, 0, len(addrPayload)+len(addrEntry))
	combined = append(combined, addrPayload...)
	combined = append(combined, addrEntry...)
	return append(tlvs, encodeCDPTLV(cdpTLVAddresses, combined)...)
}

func appendCDPCapabilities(tlvs []byte) []byte {
	capPayload := make([]byte, cdpCapBytes)
	binary.BigEndian.PutUint32(capPayload, cdpCapsValue)
	return append(tlvs, encodeCDPTLV(cdpTLVCaps, capPayload)...)
}

func encodeCDPTLV(tlvType uint16, value []byte) []byte {
	length := uint16(cdpTLVHeaderLen + len(value)&uint16Mask) //nolint:gosec // bounded by MTU
	buf := make([]byte, cdpTLVHeaderLen+len(value))
	binary.BigEndian.PutUint16(buf[0:2], tlvType)
	binary.BigEndian.PutUint16(buf[2:cdpTLVHeaderLen], length)
	copy(buf[cdpTLVHeaderLen:], value)
	return buf
}

func cdpChecksum(data []byte) uint16 {
	var sum uint32
	for i := 0; i+1 < len(data); i += 2 {
		sum += uint32(binary.BigEndian.Uint16(data[i : i+2]))
	}
	if len(data)%2 != 0 {
		sum += uint32(data[len(data)-1]) << 8 //nolint:mnd // bit shift for odd-byte checksum
	}
	for sum>>16 != 0 {
		sum = (sum & 0xFFFF) + (sum >> 16) //nolint:mnd // RFC 1071 ones-complement fold
	}
	return ^uint16(sum) //nolint:gosec // ones-complement inversion, no overflow risk
}

func padFrame(frame []byte) []byte {
	for len(frame) < minFrameLen {
		frame = append(frame, 0)
	}
	return frame
}
