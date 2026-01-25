package api

import (
	"sort"
	"strings"

	"github.com/google/gopacket/layers"
)

// getTCPFlags extracts TCP flag names from a TCP layer.
func getTCPFlags(tcp *layers.TCP) string {
	var flags []string
	if tcp.SYN {
		flags = append(flags, "SYN")
	}

	if tcp.ACK {
		flags = append(flags, "ACK")
	}

	if tcp.FIN {
		flags = append(flags, "FIN")
	}

	if tcp.RST {
		flags = append(flags, "RST")
	}

	if tcp.PSH {
		flags = append(flags, "PSH")
	}

	if tcp.URG {
		flags = append(flags, "URG")
	}

	if len(flags) == 0 {
		return "none"
	}

	return strings.Join(flags, ",")
}

// getProtocolByPort returns a well-known protocol name based on port number.
func getProtocolByPort(srcPort, dstPort int, baseProto string) string {
	ports := map[int]string{
		20:   "FTP-DATA",
		21:   "FTP",
		22:   "SSH",
		23:   "Telnet",
		25:   "SMTP",
		53:   "DNS",
		67:   "DHCP",
		68:   "DHCP",
		80:   "HTTP",
		110:  "POP3",
		123:  "NTP",
		137:  "NetBIOS",
		138:  "NetBIOS",
		139:  "NetBIOS",
		143:  "IMAP",
		161:  "SNMP",
		162:  "SNMP-Trap",
		443:  "HTTPS",
		445:  "SMB",
		514:  "Syslog",
		993:  "IMAPS",
		995:  "POP3S",
		3306: "MySQL",
		3389: "RDP",
		5432: "PostgreSQL",
		6379: "Redis",
		8080: "HTTP-Alt",
		8443: "HTTPS-Alt",
	}

	if name, ok := ports[dstPort]; ok {
		return name
	}

	if name, ok := ports[srcPort]; ok {
		return name
	}

	return baseProto
}

// isHexString checks if a string contains only valid hexadecimal characters.
func isHexString(s string) bool {
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}

// topNFromMap returns the top N entries from a string->int map, sorted by value descending.
func topNFromMap(m map[string]int, n int) []PcapIPCount {
	type kv struct {
		key   string
		value int
	}

	var sorted []kv
	for k, v := range m {
		sorted = append(sorted, kv{k, v})
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].value > sorted[j].value
	})

	result := make([]PcapIPCount, 0, n)
	for i := 0; i < len(sorted) && i < n; i++ {
		result = append(result, PcapIPCount{
			IP:    sorted[i].key,
			Count: sorted[i].value,
		})
	}

	return result
}

// estimateResultSize calculates approximate memory size of an analysis result.
func estimateResultSize(result *PcapAnalysisResult) int64 {
	if result == nil {
		return 0
	}

	// Base struct size
	size := int64(resultBaseOverhead) // Approximate fixed overhead

	// Add filename and ID sizes
	size += int64(len(result.Filename) + len(result.ID))

	// Add packet sizes (approximate)
	for _, pkt := range result.Packets {
		size += int64(packetBaseOverhead) // Base packet struct overhead
		size += int64(len(pkt.ID) + len(pkt.Timestamp) + len(pkt.SourceIP))
		size += int64(len(pkt.DestIP) + len(pkt.Protocol) + len(pkt.Info))
		size += int64(len(pkt.RawData)) // RawData is the main contributor
		// Headers map estimate
		for k, v := range pkt.Headers {
			size += int64(len(k) + headerValueEstimate) // key + approximate value size
			if s, ok := v.(string); ok {
				size += int64(len(s))
			}
		}
	}

	// Stats size (relatively small)
	size += int64(statsOverhead) // Approximate stats overhead

	return size
}
