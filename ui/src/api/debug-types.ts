/**
 * Debug and Logging Types
 * Types for the debug console (live log stream) and the global debug level.
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

/**
 * DebugLevel is the global protocol-stack verbosity. The five values map 1:1
 * onto the backend's numeric ladder (0..4): off, basic, info, verbose, trace.
 * NIAC's protocol handlers read one global level live, so this is the whole
 * contract — there is no per-protocol surface.
 */
export type DebugLevel = 'off' | 'basic' | 'info' | 'verbose' | 'trace';

/** Response from GET/PUT /api/v1/debug/level. */
export interface DebugLevelResponse {
  /** The current global debug level. */
  level: DebugLevel;
  /** The boot default, for a "reset to default" affordance. */
  defaultLevel: DebugLevel;
}

/** Body for PUT /api/v1/debug/level. */
export interface UpdateDebugLevelRequest {
  level: DebugLevel;
}
