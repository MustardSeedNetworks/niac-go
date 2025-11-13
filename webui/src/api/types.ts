export interface StackStatsResponse {
  timestamp: string;
  interface: string;
  version: string;
  device_count: number;
  stack: {
    packets_sent: number;
    packets_received: number;
    arp_requests: number;
    arp_replies: number;
    icmp_requests: number;
    icmp_replies: number;
    dns_queries: number;
    dhcp_requests: number;
    snmp_queries: number;
    errors: number;
  };
}

export interface DeviceSummary {
  name: string;
  type: string;
  ips: string[];
  protocols: string[];
}

export interface HistoryRecord {
  id: number;
  started_at: string;
  duration: number;
  interface: string;
  config_name: string;
  device_count: number;
  packets_sent: number;
  packets_received: number;
  errors: number;
}

export interface NeighborRecord {
  Protocol: string;
  LocalDevice: string;
  RemoteDevice: string;
  RemotePort: string;
  RemoteChassisID: string;
  Description: string;
  Capabilities: string[];
  ManagementAddress: string;
  LastSeen: string;
  TTL: number;
}
