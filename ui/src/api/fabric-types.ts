import type { SimulationRequest } from './api-response-types';

export type AttachmentMode = 'direct' | 'access' | 'trunk';

/**
 * One physical binding the operator approved when starting the daemon
 * (--attachment-policy). A start outside this set is refused with
 * attachment_policy_denied, so the picker offers these rather than free input.
 *
 * accessVlan carries the single approved VLAN in access mode; allowedVlans
 * carries the trunk's approved tag set. Direct mode carries neither.
 */
export interface AttachmentPolicy {
  interface: string;
  mode: AttachmentMode;
  accessVlan?: number;
  allowedVlans?: number[];
}

export interface AttachmentPoliciesResponse {
  policies: AttachmentPolicy[] | null;
}

/**
 * The attachments a prepared scenario declares. `routed` is false for a flat
 * scenario, which binds no attachment at all and so must not be asked for one.
 */
export interface SimulationAttachments {
  routed: boolean;
  attachments: string[] | null;
}

export type SimulationPreflightRequest = SimulationRequest & {
  attachment: string;
  attachmentMode: AttachmentMode;
};

export interface FabricBinding {
  attachment: string;
  interface: string;
  mode: 'direct' | 'access' | 'trunk';
  physicalVlan?: number;
  network: string;
  wireTagged: boolean;
}

export interface FabricNetwork {
  name: string;
  prefix: string;
  virtualVlan?: number;
}

export interface FabricInterface {
  device: string;
  name: string;
  network: string;
  address: string;
}

export interface FabricRoute {
  device: string;
  destination: string;
  via: string;
  connected: boolean;
}

export interface FabricDhcpScope {
  device: string;
  network: string;
  start: string;
  end: string;
  router?: string;
}

export interface FabricDiagnostic {
  code: string;
  field: string;
  message: string;
}

/**
 * The daemon now seeds these collections so they marshal as `[]` (see
 * internal/fabric.NewTopology). They are still typed nullable because a Go nil
 * slice renders as `null`, and asserting non-null here is what let D6 ship: the
 * success branch did `topology.networks.length` and crashed the wizard on a
 * preflight that had *passed*. Keep the nullability so the compiler forces a
 * guard at every read, including against an older daemon.
 */
export interface SimulationPreflightReport {
  safe: boolean;
  topology: {
    binding: FabricBinding;
    networks: FabricNetwork[] | null;
    interfaces: FabricInterface[] | null;
    routes: FabricRoute[] | null;
    dhcpScopes: FabricDhcpScope[] | null;
  };
  diagnostics: FabricDiagnostic[] | null;
}

export interface SimulationFabricStatus {
  topology: SimulationPreflightReport['topology'];
  forwarded: number;
  drops: number;
  received: number;
  transmitted: number;
}
