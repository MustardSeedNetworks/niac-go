/**
 * Walk Analyzer Types
 *
 * Types for POST /api/v1/walk/analyze — parses an SNMP walk file into
 * device identity, interface inventory, and LLDP/CDP neighbor records.
 * Mirrors internal/walkanalysis.Analysis server-side. Device fields stay
 * concatenated-lowercase (sysname, sysdescr, ...) because the wire JSON
 * has no underscores to convert; interface/neighbor/stats fields are
 * snake_case on the wire and arrive here camelCased by the API client's
 * automatic case conversion.
 */

export interface WalkAnalyzeDevice {
  sysname: string;
  sysdescr: string;
  sysobjectid: string;
  syscontact?: string;
  syslocation?: string;
}

export interface WalkAnalyzeInterface {
  index: number;
  name: string;
  description?: string;
  type: string; // e.g. "ethernetCsmacd", "softwareLoopback"
  speed?: number; // bits per second
  adminStatus?: string; // "up" | "down" | "testing"
  operStatus?: string;
  macAddress?: string; // "AA:BB:CC:DD:EE:FF"
}

export interface WalkAnalyzeNeighbor {
  localInterface: string;
  remoteDevice: string;
  remoteInterface: string;
  protocol: string; // "lldp" | "cdp"
}

export interface WalkAnalyzeStats {
  totalInterfaces: number;
  physicalInterfaces: number;
  logicalInterfaces: number;
  totalNeighbors: number;
}

export interface WalkAnalysis {
  device: WalkAnalyzeDevice;
  interfaces: WalkAnalyzeInterface[];
  neighbors: WalkAnalyzeNeighbor[];
  statistics: WalkAnalyzeStats;
}

export interface WalkAnalyzeResponse {
  success: boolean;
  result?: WalkAnalysis;
}
