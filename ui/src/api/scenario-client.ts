import { deduplicatedGet, requestJsonCamelCase } from './requestCore';

export interface ScenarioSite {
  code: string;
  octet: number;
  location: string;
}

export interface ScenarioCounts {
  siteWanRouters: number;
  firewalls: number;
  coreSwitches: number;
  distributionSwitches: number;
  accessSwitches: number;
  serverSwitches: number;
  accessPointsPerAccess: number;
  workstationsPerAccess: number;
  wirelessControllers: number;
}

export interface ScenarioGenerateRequest {
  sites: ScenarioSite[];
  counts: ScenarioCounts;
  domain: string;
  snmpCommunity: string;
  attachmentName: string;
  endpointProfile?:
    | 'enterprise'
    | 'hospital'
    | 'warehouse'
    | 'manufacturing'
    | 'retail'
    | 'service-provider';
  /** How the access tier is organized. Absent means every access switch
   * dual-homes into the distribution pair; 'ring' closes them into one ring
   * that meets distribution at two points; 'collapsed-core' lands them on the
   * core with no distribution tier; 'chain' daisy-chains them off one uplink. */
  accessLayer?: 'ring' | 'collapsed-core' | 'chain';
  /** Interfaces the pack deliberately runs hot, replacing the generated band.
   * At or above 80% Link-Live raises an interface Warning, which is what makes
   * a guided demo have something to find. */
  congestion?: {
    device: string;
    interface: string;
    inUtilization: number;
    outUtilization: number;
  }[];
}

export interface ScenarioManifest {
  deviceCount: number;
  networkCount: number;
  linkCount: number;
  deviceNamesSha256: string;
  networksSha256: string;
  linksSha256: string;
}

export interface ScenarioGenerateResponse {
  content: string;
  manifest: ScenarioManifest;
}

export interface ScenarioPack {
  id: string;
  version: string;
  manifestVersion: number;
  name: string;
  description: string;
  mapPurpose: 'presentation' | 'stress';
  request: ScenarioGenerateRequest;
  manifest: ScenarioManifest;
}

export interface ScenarioDeviceProfile {
  role: string;
  deviceType: string;
  vendor: string;
  model: string;
  platform: string;
  software: string;
  sysObjectId: string;
  walkName?: string;
  interfaceCount?: number;
  supportedSnmpData?: string[];
  interfaces?: Array<{
    name: string;
    type: string;
    mtu?: number;
    speed?: number;
    adminStatus?: string;
    operStatus?: string;
  }>;
  source?: string;
}

export const enterpriseScenarioRequest = (): ScenarioGenerateRequest => ({
  sites: [
    { code: 'COS', octet: 240, location: 'Colorado Springs, CO' },
    { code: 'EVT', octet: 241, location: 'Everett, WA' },
    { code: 'EHV', octet: 242, location: 'Eindhoven, Netherlands' },
    { code: 'LON', octet: 243, location: 'London, UK' },
  ],
  counts: {
    siteWanRouters: 2,
    firewalls: 2,
    coreSwitches: 2,
    distributionSwitches: 4,
    accessSwitches: 16,
    serverSwitches: 2,
    accessPointsPerAccess: 2,
    workstationsPerAccess: 4,
    wirelessControllers: 2,
  },
  domain: 'demo.lab',
  snmpCommunity: 'NetAllyDemo',
  attachmentName: 'cyberscope',
  endpointProfile: 'enterprise',
});

const validDomain = (domain: string) =>
  domain.length > 0 &&
  domain.length <= 237 &&
  domain.trim() === domain &&
  domain.split('.').every((label) => /^[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?$/.test(label));

const utf8Length = (value: string) => new TextEncoder().encode(value).length;

export const isScenarioRequestValid = (request: ScenarioGenerateRequest) => {
  if (request.sites.length < 1 || request.sites.length > 4 || !validDomain(request.domain)) {
    return false;
  }
  if (!request.snmpCommunity.trim() || utf8Length(request.snmpCommunity) > 255) return false;
  if (!request.attachmentName.trim() || utf8Length(request.attachmentName) > 64) return false;
  if (
    request.endpointProfile !== undefined &&
    ![
      'enterprise',
      'hospital',
      'warehouse',
      'manufacturing',
      'retail',
      'service-provider',
    ].includes(request.endpointProfile)
  ) {
    return false;
  }

  const codes = new Set<string>();
  const octets = new Set<number>();
  for (const site of request.sites) {
    if (!/^[A-Z][A-Z0-9]{1,7}$/.test(site.code) || codes.has(site.code)) return false;
    if (
      !Number.isInteger(site.octet) ||
      site.octet < 1 ||
      site.octet > 253 ||
      octets.has(site.octet)
    ) {
      return false;
    }
    if (!site.location.trim() || utf8Length(site.location) > 128) return false;
    codes.add(site.code);
    octets.add(site.octet);
  }

  const counts = request.counts;
  if (!Object.values(counts).every(Number.isInteger)) return false;
  return (
    counts.siteWanRouters === counts.firewalls &&
    counts.siteWanRouters === counts.coreSwitches &&
    counts.siteWanRouters >= 1 &&
    counts.siteWanRouters <= 2 &&
    counts.distributionSwitches >= 2 &&
    counts.distributionSwitches <= 8 &&
    counts.distributionSwitches % 2 === 0 &&
    counts.accessSwitches >= 1 &&
    counts.accessSwitches <= 20 &&
    counts.serverSwitches >= 1 &&
    counts.serverSwitches <= 8 &&
    counts.accessPointsPerAccess >= 0 &&
    counts.accessPointsPerAccess <= 9 &&
    counts.workstationsPerAccess >= 0 &&
    counts.workstationsPerAccess <= 39 &&
    counts.wirelessControllers >= 0 &&
    counts.wirelessControllers <= 8 &&
    counts.accessSwitches * counts.accessPointsPerAccess <= 154 &&
    counts.accessSwitches * counts.workstationsPerAccess <= 79
  );
};

export const fetchScenarioProfiles = () =>
  deduplicatedGet<ScenarioDeviceProfile[]>('/api/v1/scenario/profiles');

export const fetchScenarioPacks = () => deduplicatedGet<ScenarioPack[]>('/api/v1/scenario/packs');

export const generateScenario = (payload: ScenarioGenerateRequest) =>
  requestJsonCamelCase<ScenarioGenerateResponse>('/api/v1/scenario/generate', payload, {
    method: 'POST',
  });
