/**
 * Service Protocol Configuration Types
 * Application-layer service protocols (SNMP, DHCP, DNS, HTTP, FTP, NetBIOS)
 */

// ============================================================================
// SNMP Configuration Types
// ============================================================================

/**
 * SNMP Agent configuration
 */
export interface SNMPAgent {
  community?: string;
  sysName?: string;
  sysDescr?: string;
  sysContact?: string;
  sysLocation?: string;
  walkFile?: string;
  walkFiles?: string[];
  addMibs?: AddMib[];
  accessList?: string[];
  snmpAddr?: string;
  dot1dFdbTable?: FdbTableConfig;
  dot1qFdbTable?: FdbTableConfig;
  traps?: TrapConfig;
}

export interface AddMib {
  oid: string;
  type: MibType;
  value: string;
}

export type MibType =
  | 'STRING'
  | 'INTEGER'
  | 'Counter32'
  | 'Counter64'
  | 'Gauge32'
  | 'TimeTicks'
  | 'OID'
  | 'IpAddress'
  | 'Hex-STRING';

export interface FdbTableConfig {
  port: number;
  vlan: number;
}

export interface TrapConfig {
  enabled: boolean;
  receivers?: string[];
  community?: string;
  coldStart?: TrapTriggerConfig;
  linkState?: LinkStateTrapConfig;
}

export interface TrapTriggerConfig {
  enabled: boolean;
  onStartup?: boolean;
}

export interface LinkStateTrapConfig {
  enabled: boolean;
  linkDown?: boolean;
  linkUp?: boolean;
}

// ============================================================================
// DHCP Configuration Types
// ============================================================================

/**
 * DHCP Server configuration
 */
export interface DHCPConfig {
  subnetMask?: string;
  router?: string;
  domainNameServer?: string;
  serverIdentifier?: string;
  nextServerIp?: string;
  poolStart?: string;
  poolEnd?: string;
  domainName?: string;
  ntpServers?: string[];
  domainSearch?: string[];
  tftpServerName?: string;
  bootfileName?: string;
  vendorSpecific?: string;
  clientLeases?: DHCPLease[];
}

export interface DHCPLease {
  clientIp: string;
  macAddress: string;
  macMask?: string;
}

/**
 * DHCPv6 Server configuration
 */
export interface DHCPv6Config {
  enabled: boolean;
  pools?: DHCPv6Pool[];
  preferredLifetime?: number;
  validLifetime?: number;
  preference?: number;
  dnsServers?: string[];
  domainList?: string[];
  sntpServers?: string[];
  ntpServers?: string[];
  sipServers?: string[];
  sipDomains?: string[];
}

export interface DHCPv6Pool {
  network: string;
  rangeStart?: string;
  rangeEnd?: string;
}

// ============================================================================
// DNS Configuration Types
// ============================================================================

/**
 * DNS Server configuration
 */
export interface DNSConfig {
  forwardRecords?: DNSRecord[];
  reverseRecords?: DNSRecord[];
}

export interface DNSRecord {
  name: string;
  ip: string;
  ttl?: number;
  rcode?: number;
}

// ============================================================================
// HTTP/FTP Configuration Types
// ============================================================================

/**
 * HTTP Server configuration
 */
export interface HTTPConfig {
  enabled: boolean;
  serverName?: string;
  endpoints?: HTTPEndpoint[];
}

export interface HTTPEndpoint {
  path: string;
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE';
  statusCode?: number;
  contentType?: string;
  body?: string;
}

/**
 * FTP Server configuration
 */
export interface FTPConfig {
  enabled: boolean;
  welcomeBanner?: string;
  systemType?: string;
  allowAnonymous?: boolean;
  users?: FTPUser[];
}

export interface FTPUser {
  username: string;
  password: string;
  homeDir?: string;
}

// ============================================================================
// NetBIOS Configuration Types
// ============================================================================

/**
 * NetBIOS Service configuration
 */
export interface NetBIOSConfig {
  enabled: boolean;
  name?: string;
  workgroup?: string;
  nodeType?: NetBIOSNodeType;
  services?: NetBIOSService[];
  ttl?: number;
  msbrowse?: boolean;
  names?: NetBIOSName[];
}

export type NetBIOSNodeType = 'B' | 'P' | 'M' | 'H';
export type NetBIOSService = 'workstation' | 'fileserver' | 'messenger';

export interface NetBIOSName {
  name: string;
  suffix: string;
  group?: boolean;
}
