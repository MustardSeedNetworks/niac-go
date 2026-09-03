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

// Re-export all network protocol types
export type {
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
  STPConfig,
  STPVersion,
  TTLConfig,
} from './network-protocol-types';
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
  TrapConfig,
  TrapTriggerConfig,
} from './service-protocol-types';

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
  interfaces?: string[];
  interfaceDetails?: DeviceInterface[];
  /**
   * Protocols this device speaks, as computed by the server. Present on both
   * the list summary and the detail response. The UI used to re-derive this
   * from the sub-objects below, which the *summary* response omits — so every
   * device rendered as "No protocols" (D9). Read this field.
   */
  protocols?: string[];
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
  ttl?: TTLConfig;
  osFingerprint?: OSFingerprintConfig;
  ssh?: SSHConfig;
  syslog?: SyslogConfig;
  iperf3?: IPerf3Config;
}

export type SSHConfig =
  | { enabled: false; username?: never; passwordEnv?: never }
  | { enabled: true; username: string; passwordEnv: string };

export type SyslogConfig =
  | { enabled: false; receivers?: never }
  | { enabled: true; receivers: [string, ...string[]] };

export interface DeviceInterface {
  name: string;
  speed?: number;
  duplex?: 'full' | 'half' | 'auto' | '';
  adminStatus?: 'up' | 'down' | '';
  operStatus?: 'up' | 'down' | 'testing' | '';
  description?: string;
  vlans?: number[];
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
 * GET /api/v1/config/devices/:id returns the device **flat**, not wrapped.
 * This used to declare `{ device: Device }`, so the editor's `fetchedDevice?.device`
 * was always undefined and the form silently stayed empty (D10). Typed as the
 * device itself so the compiler enforces the real shape.
 *
 * `rawYaml` is the authored document the daemon loaded this device from, and
 * the single-device GET always serializes it. It is what the device editor
 * reads and writes: the camelCase projection above covers 56 of the 223 fields
 * an author can set, so an editor holding it drops the rest on save.
 */
export type DeviceDetailResponse = Device & { rawYaml?: string };

/**
 * Request body for POST /api/v1/config/devices.
 *
 * `hostname` names the device — the daemon takes the name from here, not from
 * the document — and `rawYaml` carries everything else.
 */
export interface CreateDeviceRequest {
  hostname: string;
  rawYaml: string;
}

/**
 * Request body for PUT /api/v1/config/devices/:id. The URL names the device;
 * the body is the authored document.
 */
export interface UpdateDeviceRequest {
  rawYaml: string;
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

/**
 * Request body for DELETE /api/v1/config/devices (batch delete)
 */
export interface DeviceBatchDeleteRequest {
  hostnames: string[];
}

/**
 * Per-hostname outcome of a batch device delete.
 */
export interface DeviceBatchDeleteResult {
  hostname: string;
  success: boolean;
  error?: string;
}

/**
 * Response from DELETE /api/v1/config/devices (batch delete). The request
 * succeeds even when some hostnames fail; per-hostname outcomes are in
 * `results`.
 */
export interface DeviceBatchDeleteResponse {
  results: DeviceBatchDeleteResult[];
  deleted: number;
  failed: number;
}

// ============================================================================
// JSON Schema Types (for dynamic form generation)
// ============================================================================

export interface JSONSchemaProperty {
  type?: string;
  title?: string;
  description?: string;
  default?: unknown;
  enum?: unknown[];
  enumNames?: string[];
  minimum?: number;
  maximum?: number;
  minItems?: number;
  minLength?: number;
  maxLength?: number;
  pattern?: string;
  format?: string;
  items?: JSONSchemaProperty;
  properties?: Record<string, JSONSchemaProperty>;
  required?: string[];
  const?: unknown;
  allOf?: JSONSchemaProperty[];
  if?: JSONSchemaProperty;
  then?: JSONSchemaProperty;
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
