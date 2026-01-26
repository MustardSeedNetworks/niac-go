package api

// ============================================================================
// PCAP Analysis Types
// ============================================================================

// PcapPacket represents a single packet from a PCAP file.
type PcapPacket struct {
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

// PcapTimeRange represents the time span of a capture.
type PcapTimeRange struct {
	Start      string `json:"start"`
	End        string `json:"end"`
	DurationMs int64  `json:"durationMs"`
}

// PcapIPCount represents an IP address with its packet count.
type PcapIPCount struct {
	IP    string `json:"ip"`
	Count int    `json:"count"`
}

// PcapStats provides statistics about the analyzed PCAP.
type PcapStats struct {
	TotalPackets    int            `json:"totalPackets"`
	TotalBytes      int64          `json:"totalBytes"`
	TimeRange       PcapTimeRange  `json:"timeRange"`
	Protocols       map[string]int `json:"protocols"`
	TopSources      []PcapIPCount  `json:"topSources"`
	TopDestinations []PcapIPCount  `json:"topDestinations"`
}

// PcapAnalysisResult is the complete analysis of a PCAP file.
type PcapAnalysisResult struct {
	ID        string       `json:"id"`
	Filename  string       `json:"filename"`
	FileSize  int64        `json:"fileSize"`
	Packets   []PcapPacket `json:"packets"`
	Stats     PcapStats    `json:"stats"`
	CreatedAt string       `json:"createdAt"`
}

// PcapUploadRequest is the request body for uploading a PCAP.
type PcapUploadRequest struct {
	Filename string `json:"filename"`
	Data     string `json:"data"` // Base64 encoded PCAP data
}

// PcapUploadResponse is returned after successful upload.
type PcapUploadResponse struct {
	Success    bool   `json:"success"`
	AnalysisID string `json:"analysisId"`
	Message    string `json:"message"`
}
