import type { SimulationRequest } from './api-response-types';

export type SimulationPreflightRequest = SimulationRequest & {
  attachment: string;
  attachmentMode: 'direct' | 'access';
};

export interface FabricBinding {
  attachment: string;
  interface: string;
  mode: 'direct' | 'access';
  accessVlan?: number;
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
