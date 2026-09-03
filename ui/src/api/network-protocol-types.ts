/**
 * Network Protocol Configuration Types
 * Layer 2/3 protocols (discovery, STP, ICMP, OS fingerprint)
 */

// ============================================================================
// Discovery Protocol Types (LLDP, CDP, EDP, FDP)
// ============================================================================

/**
 * LLDP (Link Layer Discovery Protocol) configuration
 */
export interface LLDPConfig {
  enabled: boolean;
  advertiseInterval?: number;
  ttl?: number;
  chassisIdType?: ChassisIDType;
  systemDescription?: string;
  portDescription?: string;
}

export type ChassisIDType = 'mac' | 'local' | 'network_address';

/**
 * CDP (Cisco Discovery Protocol) configuration
 */
export interface CDPConfig {
  enabled: boolean;
  advertiseInterval?: number;
  holdtime?: number;
  version?: 1 | 2;
  platform?: string;
  softwareVersion?: string;
  portId?: string;
}

/**
 * EDP (Extreme Discovery Protocol) configuration
 */
export interface EDPConfig {
  enabled: boolean;
  advertiseInterval?: number;
  versionString?: string;
  displayString?: string;
}

/**
 * FDP (Foundry Discovery Protocol) configuration
 */
export interface FDPConfig {
  enabled: boolean;
  advertiseInterval?: number;
  holdtime?: number;
  softwareVersion?: string;
  platform?: string;
  portId?: string;
}

// ============================================================================
// Spanning Tree Protocol Types
// ============================================================================

/**
 * STP (Spanning Tree Protocol) configuration
 */
export interface STPConfig {
  enabled: boolean;
  version?: STPVersion;
  bridgePriority?: number;
  helloTime?: number;
  maxAge?: number;
  forwardDelay?: number;
}

export type STPVersion = 'stp' | 'rstp' | 'mstp';

// ============================================================================
// ICMP Configuration Types
// ============================================================================

/**
 * ICMP (v4) configuration
 */
export interface ICMPConfig {
  enabled: boolean;
  ttl?: number;
  rateLimit?: number;
  addressMaskReply?: string;
  routerAdvertisement?: ICMPRouterAdvertisement;
}

export interface ICMPRouterAdvertisement {
  period?: number;
  lifetime?: number;
  routers?: ICMPRouter[];
}

export interface ICMPRouter {
  address: string;
  preference?: number;
}

/**
 * ICMPv6 configuration
 */
export interface ICMPv6Config {
  enabled: boolean;
  hopLimit?: number;
  rateLimit?: number;
  routerAdvertisement?: ICMPv6RouterAdvertisement;
}

export interface ICMPv6RouterAdvertisement {
  period?: number;
  curHopLimit?: number;
  managed?: number;
  other?: number;
  lifetime?: number;
  reachableTime?: number;
  retransTimer?: number;
  mtu?: number;
  prefixInfo?: ICMPv6PrefixInfo[];
}

export interface ICMPv6PrefixInfo {
  prefixLength?: number;
  onlink?: number;
  auto?: number;
  validLifetime?: number;
  preferredLifetime?: number;
  prefix?: string;
}

// ============================================================================
// Miscellaneous Configuration Types
// ============================================================================

/**
 * TTL Configuration for traceroute simulation
 */
export interface TTLConfig {
  ttl: number;
  ip?: string;
  mask?: string;
}

/**
 * OS Fingerprint configuration
 */
export interface OSFingerprintConfig {
  osType?: OSType;
  ttl?: number;
  windowSize?: number;
  windowScale?: number;
  mss?: number;
  sshBanner?: string;
  httpServer?: string;
  ftpBanner?: string;
  smtpBanner?: string;
  telnetBanner?: string;
  dontFragment?: boolean;
}

export type OSType =
  | 'linux'
  | 'windows'
  | 'macos'
  | 'freebsd'
  | 'cisco-ios'
  | 'cisco-nxos'
  | 'juniper-junos'
  | 'arista-eos';

/**
 * iPerf3 Server configuration
 */
export interface IPerf3Config {
  enabled: boolean;
  port?: number;
  maxBandwidthMbps?: number;
  typicalLatencyMs?: number;
  jitterMs?: number;
  packetLossPercent?: number;
  uploadMbps?: number;
  downloadMbps?: number;
}
