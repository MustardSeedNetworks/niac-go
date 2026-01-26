/**
 * Debug and Logging Types
 * Types for debug console, protocol debugging, and log entries
 */

export type LogLevel = 'ERROR' | 'WARN' | 'INFO' | 'DEBUG';

export type Protocol =
  | 'ARP'
  | 'ICMP'
  | 'DNS'
  | 'TCP'
  | 'UDP'
  | 'SNMP'
  | 'DHCP'
  | 'LLDP'
  | 'CDP'
  | 'HTTP'
  | 'HTTPS'
  | 'SSH'
  | 'TELNET';

export interface LogEntry {
  id: string;
  timestamp: string;
  level: LogLevel;
  protocol: Protocol | string;
  message: string;
  source?: string;
  details?: Record<string, unknown>;
}

export type DebugLevel = 'OFF' | 'ERROR' | 'WARN' | 'INFO' | 'DEBUG' | 'TRACE';

export type DebugProtocol =
  | 'SNMP'
  | 'LLDP'
  | 'CDP'
  | 'STP'
  | 'LACP'
  | 'OSPF'
  | 'BGP'
  | 'EIGRP'
  | 'RIP'
  | 'ISIS'
  | 'VRRP'
  | 'HSRP'
  | 'GLBP'
  | 'BFD'
  | 'MPLS'
  | 'PIM'
  | 'IGMP'
  | 'MSDP'
  | 'NetFlow';

export type ProtocolCategory =
  | 'discovery'
  | 'switching'
  | 'routing'
  | 'redundancy'
  | 'multicast'
  | 'monitoring';

export interface ProtocolDebugConfig {
  protocol: DebugProtocol;
  level: DebugLevel;
  category: ProtocolCategory;
}

export interface ProtocolDebugLevelsResponse {
  protocols: ProtocolDebugConfig[];
  defaultLevel: DebugLevel;
}

export interface UpdateProtocolDebugLevelRequest {
  protocol: DebugProtocol;
  level: DebugLevel;
}

export interface UpdateProtocolDebugLevelsRequest {
  protocols: UpdateProtocolDebugLevelRequest[];
}

export interface ResetProtocolDebugLevelsResponse {
  success: boolean;
  message: string;
  protocols: ProtocolDebugConfig[];
}
