import type { SimulationRequest } from './api-response-types';

export type SimulationPreflightRequest = SimulationRequest & {
  attachment: string;
  attachmentMode: 'direct' | 'access' | 'trunk';
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

export interface SimulationPreflightReport {
  safe: boolean;
  topology: {
    binding: FabricBinding;
    networks: FabricNetwork[];
    interfaces: FabricInterface[];
    routes: FabricRoute[];
    dhcpScopes: FabricDhcpScope[];
  };
  diagnostics: FabricDiagnostic[];
}

export interface SimulationFabricStatus {
  topology: SimulationPreflightReport['topology'];
  forwarded: number;
  drops: number;
  received: number;
  transmitted: number;
}
