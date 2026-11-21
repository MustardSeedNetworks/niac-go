/**
 * A 543-device scenario pack, shaped like `enterprise-scale`: three sites,
 * each a core pair over a distribution tier over access switches, with the
 * endpoints hanging off the access switches, and the sites joined by WAN links
 * between their cores. Generated rather than checked in so the size is one
 * constant, and deterministic so a failure reproduces.
 */

export interface PackDevice {
  name: string;
  type: string;
  ips: string[];
  protocols: string[];
  mac: string;
  properties: { site: string; role: string };
}

export interface PackLink {
  source: string;
  target: string;
  label: string;
  sourceInterface: string;
  targetInterface: string;
  linkType?: string;
  speed?: string;
  discovered: boolean;
}

export const SCALE_PACK_SITES = ['COS', 'EVT', 'EHV'] as const;
export const SCALE_PACK_DEVICE_COUNT = 543;

const ACCESS_PER_SITE = 12;
const DIST_PER_SITE = 4;
const CORE_PER_SITE = 2;
const ENDPOINT_TYPES = ['ap', 'workstation', 'printer', 'voip-phone'] as const;

function pad(n: number, width = 2): string {
  return String(n).padStart(width, '0');
}

export function buildScalePack(): { devices: PackDevice[]; links: PackLink[] } {
  const devices: PackDevice[] = [];
  const links: PackLink[] = [];
  const perSite = SCALE_PACK_DEVICE_COUNT / SCALE_PACK_SITES.length;
  let serial = 0;

  const add = (site: string, name: string, type: string, role: string, protocols: string[]) => {
    serial += 1;
    devices.push({
      name,
      type,
      ips: [`10.${devices.length % 250}.${Math.floor(devices.length / 250)}.${(serial % 250) + 1}`],
      protocols,
      mac: `02:00:00:${pad(serial >> 8, 2)}:${pad(serial & 0xff, 2)}:01`,
      properties: { site, role },
    });
  };
  const link = (source: string, target: string, linkType?: string, speed = '1000') => {
    links.push({
      source,
      target,
      label: `Gi0/${(links.length % 48) + 1}`,
      sourceInterface: `Gi0/${(links.length % 48) + 1}`,
      targetInterface: `Gi1/${(links.length % 48) + 1}`,
      linkType,
      speed,
      discovered: false,
    });
  };

  for (const site of SCALE_PACK_SITES) {
    const cores = Array.from({ length: CORE_PER_SITE }, (_, i) => `${site}-CORE-${pad(i + 1)}`);
    const dists = Array.from({ length: DIST_PER_SITE }, (_, i) => `${site}-DIST-${pad(i + 1)}`);
    const access = Array.from({ length: ACCESS_PER_SITE }, (_, i) => `${site}-ACC-${pad(i + 1)}`);
    for (const name of cores) add(site, name, 'router', 'core', ['lldp', 'snmp', 'bgp']);
    for (const name of dists)
      add(site, name, 'layer3-switch', 'distribution', ['lldp', 'snmp', 'stp']);
    for (const name of access) add(site, name, 'switch', 'access', ['lldp', 'snmp', 'stp']);

    link(cores[0] as string, cores[1] as string, 'lag', '10000');
    for (const [i, dist] of dists.entries()) {
      link(dist, cores[i % CORE_PER_SITE] as string, 'trunk', '10000');
      link(dist, cores[(i + 1) % CORE_PER_SITE] as string, 'trunk', '10000');
    }
    for (const [i, name] of access.entries()) {
      link(name, dists[i % DIST_PER_SITE] as string, 'trunk', '1000');
    }

    const endpoints = perSite - cores.length - dists.length - access.length;
    for (let i = 0; i < endpoints; i += 1) {
      const type = ENDPOINT_TYPES[i % ENDPOINT_TYPES.length] as string;
      const name = `${site}-${type.slice(0, 3).toUpperCase()}-${pad(i + 1, 3)}`;
      add(site, name, type, 'endpoint', ['lldp']);
      link(name, access[i % ACCESS_PER_SITE] as string, undefined, '100');
      // Access points uplink to two switches. That keeps them out of the leaf
      // packing the layout does for single-homed devices, as in the real pack.
      if (type === 'ap') {
        link(name, access[(i + 1) % ACCESS_PER_SITE] as string, undefined, '100');
      }
    }
  }
  for (let i = 0; i < SCALE_PACK_SITES.length; i += 1) {
    const next = (i + 1) % SCALE_PACK_SITES.length;
    link(`${SCALE_PACK_SITES[i]}-CORE-01`, `${SCALE_PACK_SITES[next]}-CORE-01`, 'trunk', '10000');
  }
  return { devices, links };
}

/** The shapes the two device surfaces read. `/devices` and `/topology` are
 *  per-session runtime routes; `/config/devices` is the authored inventory. */
export function packResponses() {
  const { devices, links } = buildScalePack();
  return {
    runtimeDevices: devices,
    topology: { nodes: devices.map((d) => ({ name: d.name, type: d.type })), links },
    configDevices: {
      devices: devices.map((d) => ({
        hostname: d.name,
        mac: d.mac,
        ip: d.ips[0],
        ips: d.ips,
        type: d.type,
        protocols: d.protocols,
      })),
      totalCount: devices.length,
      configurationLoaded: true,
    },
  };
}
