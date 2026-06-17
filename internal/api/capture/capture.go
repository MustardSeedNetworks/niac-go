// Package capture holds the PCAP analysis engine extracted from the
// internal/api monolith (ADR-0006). It parses libpcap/pcapng byte buffers
// into structured packet summaries, converts RFC 1761 snoop captures, and
// caps an in-memory result cache — all with no dependency on the api
// transport layer, which composes it inward.
//
// The package is named capture (not pcap) to avoid colliding with
// github.com/gopacket/gopacket/pcap, mirroring the tlsutil-vs-crypto/tls
// precedent in the sibling repos.
package capture

// Packet represents a single packet from a PCAP file.
type Packet struct {
	ID         string         `json:"id"`
	Number     int            `json:"number"`
	Timestamp  string         `json:"timestamp"`
	SourceIP   string         `json:"sourceIP"`
	DestIP     string         `json:"destIP"`
	SourcePort *int           `json:"sourcePort,omitempty"`
	DestPort   *int           `json:"destPort,omitempty"`
	Protocol   string         `json:"protocol"`
	Length     int            `json:"length"`
	Info       string         `json:"info"`
	RawData    string         `json:"rawData,omitempty"`
	Headers    map[string]any `json:"headers,omitempty"`
}

// TimeRange represents the time span of a capture.
type TimeRange struct {
	Start      string `json:"start"`
	End        string `json:"end"`
	DurationMs int64  `json:"durationMs"`
}

// IPCount represents an IP address with its packet count.
type IPCount struct {
	IP    string `json:"ip"`
	Count int    `json:"count"`
}

// Stats provides statistics about the analyzed PCAP.
type Stats struct {
	TotalPackets    int            `json:"totalPackets"`
	TotalBytes      int64          `json:"totalBytes"`
	TimeRange       TimeRange      `json:"timeRange"`
	Protocols       map[string]int `json:"protocols"`
	TopSources      []IPCount      `json:"topSources"`
	TopDestinations []IPCount      `json:"topDestinations"`
}

// AnalysisResult is the complete analysis of a PCAP file.
type AnalysisResult struct {
	ID        string   `json:"id"`
	Filename  string   `json:"filename"`
	FileSize  int64    `json:"fileSize"`
	Packets   []Packet `json:"packets"`
	Stats     Stats    `json:"stats"`
	CreatedAt string   `json:"createdAt"`
}
