package config

// DefaultIPerf3Port is the port an iperf3 server listens on unless authored.
const DefaultIPerf3Port = 5201

// DefaultIPerf3DownloadMbps is the rate a reverse iperf3 test is sent at
// unless authored.
const DefaultIPerf3DownloadMbps = 100

// IPerf3Config holds iPerf3 server emulation configuration for bandwidth testing simulation.
type IPerf3Config struct {
	Enabled      bool    // Whether iPerf3 server is enabled
	Port         uint16  // Listen port (default 5201)
	DownloadMbps float64 // The rate a reverse test is sent at
}
