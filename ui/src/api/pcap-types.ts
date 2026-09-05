/**
 * PCAP Analyzer Types
 * Types for packet capture analysis
 */

export interface PcapPacket {
  id: string;
  number: number;
  timestamp: string;
  sourceIp: string;
  destIp: string;
  sourcePort?: number;
  destPort?: number;
  protocol: string;
  length: number;
  info: string;
  rawData?: string;
  headers?: Record<string, unknown>;
}

export interface PcapAnalysisResult {
  filename: string;
  fileSize: number;
  packets: PcapPacket[];
  /**
   * The capture held more packets than the analyzer keeps rows for, so
   * `packets` is a prefix of the file. `stats` still describes the whole
   * capture — showing the short list without saying so would make the two
   * disagree with no explanation.
   */
  truncated?: boolean;
  stats: PcapStats;
}

export interface PcapStats {
  totalPackets: number;
  totalBytes: number;
  timeRange: {
    start: string;
    end: string;
    durationMs: number;
  };
  protocols: Record<string, number>;
  topSources: Array<{ ip: string; count: number }>;
  topDestinations: Array<{ ip: string; count: number }>;
}

export interface PcapUploadRequest {
  filename: string;
  data: string; // Base64 encoded PCAP data
}

export interface PcapUploadResponse {
  success: boolean;
  analysisId: string;
  message: string;
}

export interface CaptureFilterResponse {
  active: boolean;
  filter: string;
}
