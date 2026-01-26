// PCAP Analysis Module
//
// This module provides PCAP file upload, analysis, and caching functionality.
// The code is organized into the following files:
//
//   - pcap_types.go      - Core data structures (PcapPacket, PcapStats, etc.)
//   - pcap_cache.go      - Memory-limited cache for analysis results
//   - pcap_handlers.go   - HTTP handlers for upload and retrieval
//   - pcap_analysis.go   - Core analysis logic and statistics collection
//   - pcap_layers.go     - Protocol layer parsing (Ethernet, IP, TCP, UDP, etc.)
//   - pcap_validation.go - PCAP file format validation
//
// Security features:
//   - Memory-limited cache to prevent DoS (SECURITY FIX #155)
//   - Secure temp file handling (SECURITY FIX #167)
//   - PCAP structure validation (SECURITY FIX #170)
//   - Error message sanitization (SECURITY FIX #183)
package api
