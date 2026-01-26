/**
 * Device Configuration Types
 * Core device interface and API request/response types
 */

import type {
  CDPConfig,
  EDPConfig,
  FDPConfig,
  ICMPConfig,
  ICMPv6Config,
  IPerf3Config,
  LLDPConfig,
  OSFingerprintConfig,
  STPConfig,
  TrafficConfig,
  TTLConfig,
} from './network-protocol-types';

import type {
  DHCPConfig,
  DHCPv6Config,
  DNSConfig,
  FTPConfig,
  HTTPConfig,
  NetBIOSConfig,
  SNMPAgent,
} from './service-protocol-types';

// Re-export all service protocol types
export type {
  AddMib,
  DHCPConfig,
  DHCPLease,
  DHCPv6Config,
  DHCPv6Pool,
  DNSConfig,
  DNSRecord,
  FdbTableConfig,
  FTPConfig,
  FTPUser,
  HTTPConfig,
  HTTPEndpoint,
  LinkStateTrapConfig,
  MibType,
  NetBIOSConfig,
  NetBIOSName,
  NetBIOSNodeType,
  NetBIOSService,
  SNMPAgent,
  ThresholdTrapConfig,
  TrapConfig,
  TrapTriggerConfig,
} from './service-protocol-types';

// Re-export all network protocol types
export type {
  ARPAnnouncementConfig,
  CDPConfig,
  ChassisIDType,
  EDPConfig,
  FDPConfig,
  ICMPConfig,
  ICMPRouter,
  ICMPRouterAdvertisement,
  ICMPv6Config,
  ICMPv6PrefixInfo,
  ICMPv6RouterAdvertisement,
  IPerf3Config,
  LLDPConfig,
  OSFingerprintConfig,
  OSType,
  PeriodicPingConfig,
  RandomTrafficConfig,
  RandomTrafficPattern,
  STPConfig,
  STPVersion,
  TrafficConfig,
  TTLConfig,
} from './network-protocol-types';

// ============================================================================
// Core Device Types
// ============================================================================

/**
 * Device represents a complete network device configuration
 */
export interface Device {
  hostname: string;
  mac: string;
  ip?: string;
  ips?: string[];
  type?: DeviceType;
  vlan?: number;
  babble?: boolean;
  mapToIp?: string;
  snmpAgent?: SNMPAgent;
  lldp?: LLDPConfig;
  cdp?: CDPConfig;
  edp?: EDPConfig;
  fdp?: FDPConfig;
  stp?: STPConfig;
  dhcp?: DHCPConfig;
  dns?: DNSConfig;
  http?: HTTPConfig;
  ftp?: FTPConfig;
  netbios?: NetBIOSConfig;
  icmp?: ICMPConfig;
  icmpv6?: ICMPv6Config;
  dhcpv6?: DHCPv6Config;
  traffic?: TrafficConfig;
  ttl?: TTLConfig;
  osFingerprint?: OSFingerprintConfig;
  iperf3?: IPerf3Config;
}

export type DeviceType =
  | 'router'
  | 'switch'
  | 'access_point'
  | 'firewall'
  | 'server'
  | 'workstation'
  | 'iot'
  | 'unknown';

// ============================================================================
// Device API Response Types
// ============================================================================

/**
 * Response from GET /api/v1/config/devices
 */
export interface DeviceListResponse {
  devices: Device[];
  total: number;
}

/**
 * Response from GET /api/v1/config/devices/:id
 */
export interface DeviceDetailResponse {
  device: Device;
}

/**
 * Request body for POST /api/v1/config/devices
 */
export interface CreateDeviceRequest {
  device: Device;
}

/**
 * Request body for PUT /api/v1/config/devices/:id
 */
export interface UpdateDeviceRequest {
  device: Partial<Device>;
}

/**
 * Response from device mutations
 */
export interface DeviceMutationResponse {
  success: boolean;
  device?: Device;
  message?: string;
}

/**
 * Request body for POST /api/v1/config/devices/:id/clone
 */
export interface CloneDeviceRequest {
  newHostname: string;
  newMac?: string;
  newIp?: string;
}

// ============================================================================
// JSON Schema Types (for dynamic form generation)
// ============================================================================

export interface JSONSchemaProperty {
  type: string;
  title?: string;
  description?: string;
  default?: unknown;
  enum?: unknown[];
  enumNames?: string[];
  minimum?: number;
  maximum?: number;
  minLength?: number;
  maxLength?: number;
  pattern?: string;
  format?: string;
  items?: JSONSchemaProperty;
  properties?: Record<string, JSONSchemaProperty>;
  required?: string[];
  'ui:widget'?: string;
  'ui:help'?: string;
  'ui:placeholder'?: string;
}

export interface ConfigSchema {
  $schema: string;
  type: string;
  title: string;
  description?: string;
  properties: Record<string, JSONSchemaProperty>;
  required: string[];
  definitions?: Record<string, JSONSchemaProperty>;
}
