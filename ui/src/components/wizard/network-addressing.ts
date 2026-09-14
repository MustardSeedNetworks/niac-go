import { isMap, isSeq, parseDocument, type Scalar } from 'yaml';

/**
 * The Networks step's view of a config: the routed networks, the attachments
 * that expose them, and which interface each device is addressed on.
 *
 * Read out of the YAML rather than held as separate state, so the step always
 * reflects the draft the author is actually editing -- including one they
 * uploaded or generated rather than built here.
 */
export interface AuthoredNetwork {
  name: string;
  subnet: string;
  virtualVlan?: number;
}

/** A pool of free ports on one device — where a tester actually appears. */
export interface AuthoredAttachmentPool {
  device: string;
  ports: string[];
}

/** One client MAC fixed to one port of the pool. */
export interface AuthoredAttachmentPin {
  mac: string;
  device: string;
  interface: string;
}

export interface AuthoredAttachment {
  name: string;
  /** The network form. Empty when the attachment uses a port pool instead. */
  connect: string;
  at?: AuthoredAttachmentPool;
  pins?: AuthoredAttachmentPin[];
}

/** One port on a device, as the attachment-pool picker needs to see it. */
export interface AuthoredDevicePort {
  name: string;
  /** The port's access VLAN, or null when it carries none or several. */
  vlan: number | null;
  /** True when a trunk port or port-channel already uses it. */
  occupied: boolean;
}

export interface DeviceAddressing {
  device: string;
  /** First interface's name, which is what auto-assign addresses. */
  interfaceName: string | null;
  network: string | null;
  /** Prefix form, e.g. 10.20.0.5/24. */
  address: string | null;
  /** Every port the device declares, for the attachment-pool picker. */
  ports: AuthoredDevicePort[];
}

export interface NetworkModel {
  networks: AuthoredNetwork[];
  attachments: AuthoredAttachment[];
  devices: DeviceAddressing[];
}

const scalar = (node: unknown): string | null => {
  const value = (node as Scalar | undefined)?.value ?? node;
  return typeof value === 'string' || typeof value === 'number' ? String(value) : null;
};

function readNetworks(node: unknown): AuthoredNetwork[] {
  if (!isSeq(node)) return [];
  const networks: AuthoredNetwork[] = [];
  for (const item of node.items) {
    if (!isMap(item)) continue;
    const name = scalar(item.get('name'));
    if (!name) continue;
    const vlan = scalar(item.get('virtual_vlan'));
    networks.push({
      name,
      subnet: scalar(item.get('subnet')) ?? '',
      ...(vlan ? { virtualVlan: Number(vlan) } : {}),
    });
  }
  return networks;
}

function readAttachments(node: unknown): AuthoredAttachment[] {
  if (!isSeq(node)) return [];
  const attachments: AuthoredAttachment[] = [];
  for (const item of node.items) {
    if (!isMap(item)) continue;
    const name = scalar(item.get('name'));
    if (!name) continue;
    const at = readAttachmentPool(item.get('at'));
    const pins = readAttachmentPins(item.get('pins'));
    attachments.push({
      name,
      connect: scalar(item.get('connect')) ?? '',
      ...(at ? { at } : {}),
      ...(pins.length > 0 ? { pins } : {}),
    });
  }
  return attachments;
}

function readAttachmentPool(node: unknown): AuthoredAttachmentPool | null {
  if (!isMap(node)) return null;
  const device = scalar(node.get('device'));
  if (!device) return null;
  const ports = node.get('ports');
  const names: string[] = [];
  if (isSeq(ports)) {
    for (const port of ports.items) {
      const value = scalar(port);
      if (value) names.push(value);
    }
  }
  return { device, ports: names };
}

function readAttachmentPins(node: unknown): AuthoredAttachmentPin[] {
  if (!isSeq(node)) return [];
  const pins: AuthoredAttachmentPin[] = [];
  for (const item of node.items) {
    if (!isMap(item)) continue;
    const mac = scalar(item.get('mac'));
    const device = scalar(item.get('device'));
    const port = scalar(item.get('interface'));
    if (!mac || !device || !port) continue;
    pins.push({ mac, device, interface: port });
  }
  return pins;
}

function readDeviceAddressing(node: unknown): DeviceAddressing[] {
  if (!isSeq(node)) return [];
  const devices: DeviceAddressing[] = [];
  for (const item of node.items) {
    if (!isMap(item)) continue;
    const device = scalar(item.get('name'));
    if (!device) continue;
    const interfaces = item.get('interfaces');
    const first = isSeq(interfaces) && isMap(interfaces.items[0]) ? interfaces.items[0] : null;
    devices.push({
      device,
      interfaceName: first ? scalar(first.get('name')) : null,
      network: first ? scalar(first.get('network')) : null,
      address: first ? scalar(first.get('address')) : null,
      ports: readDevicePorts(interfaces, item.get('trunk_ports'), item.get('port_channels')),
    });
  }
  return devices;
}

function readDevicePorts(
  interfaces: unknown,
  trunkPorts: unknown,
  portChannels: unknown,
): AuthoredDevicePort[] {
  if (!isSeq(interfaces)) return [];
  const used = new Set<string>();
  if (isSeq(trunkPorts)) {
    for (const trunk of trunkPorts.items) {
      if (!isMap(trunk)) continue;
      const name = scalar(trunk.get('interface'));
      if (name) used.add(name);
    }
  }
  if (isSeq(portChannels)) {
    for (const channel of portChannels.items) {
      if (!isMap(channel)) continue;
      const members = channel.get('members');
      if (!isSeq(members)) continue;
      for (const member of members.items) {
        const name = scalar(member);
        if (name) used.add(name);
      }
    }
  }
  const ports: AuthoredDevicePort[] = [];
  for (const item of interfaces.items) {
    if (!isMap(item)) continue;
    const name = scalar(item.get('name'));
    if (!name) continue;
    ports.push({ name, vlan: accessVlan(item.get('vlans')), occupied: used.has(name) });
  }
  return ports;
}

/** accessVlan is the port's single VLAN, or null -- a port carrying several is
 * a trunk, and the network behind it is not decidable from the port alone. */
function accessVlan(node: unknown): number | null {
  if (!isSeq(node) || node.items.length !== 1) return null;
  const value = Number(scalar(node.items[0]));
  return Number.isInteger(value) && value >= 1 && value <= 4094 ? value : null;
}

/** parseNetworkModel reads the step's slice of a config. Returns empty lists
 * for a config that does not parse: the step then shows nothing rather than a
 * model built on a guess, and the author can still fix the YAML by hand. */
export function parseNetworkModel(configText: string): NetworkModel {
  const doc = parseDocument(configText);
  if (doc.errors.length > 0 || !isMap(doc.contents)) {
    return { networks: [], attachments: [], devices: [] };
  }
  return {
    networks: readNetworks(doc.get('networks')),
    attachments: readAttachments(doc.get('attachments')),
    devices: readDeviceAddressing(doc.get('devices')),
  };
}

/** Serializes the networks section, or '' when there are none — the caller
 * splices '' as a removal rather than authoring an empty list. */
export function serializeNetworks(networks: AuthoredNetwork[]): string {
  if (networks.length === 0) return '';
  const lines = ['networks:'];
  for (const network of networks) {
    lines.push(`  - name: ${network.name}`);
    lines.push(`    subnet: ${network.subnet}`);
    if (network.virtualVlan) lines.push(`    virtual_vlan: ${network.virtualVlan}`);
  }
  return `${lines.join('\n')}\n`;
}

/** Serializes the attachments section, or '' when there are none. */
export function serializeAttachments(attachments: AuthoredAttachment[]): string {
  if (attachments.length === 0) return '';
  const lines = ['attachments:'];
  for (const attachment of attachments) {
    lines.push(`  - name: ${attachment.name}`);
    // Exactly one form reaches the file. An attachment carrying both, or a
    // pool with no ports, is one the daemon refuses -- so the half-written
    // state stays in the editor rather than in the scenario.
    const ports = attachment.at?.ports.filter((port) => port !== '') ?? [];
    if (attachment.at && ports.length > 0) {
      lines.push('    at:');
      lines.push(`      device: ${attachment.at.device}`);
      lines.push('      ports:');
      for (const port of ports) lines.push(`        - ${port}`);
      let pinned = false;
      for (const pin of attachment.pins ?? []) {
        if (!pin.mac || !pin.interface) continue;
        if (!pinned) lines.push('    pins:');
        pinned = true;
        // Quoted: a MAC still being typed can look like a number to YAML
        // ("00" reads back as 0), which silently rewrites the author's input.
        lines.push(`      - mac: "${pin.mac}"`);
        lines.push(`        device: ${pin.device || attachment.at.device}`);
        lines.push(`        interface: ${pin.interface}`);
      }
      continue;
    }
    lines.push(`    connect: ${attachment.connect}`);
  }
  return `${lines.join('\n')}\n`;
}

const ipToInt = (ip: string): number | null => {
  const octets = ip.split('.');
  if (octets.length !== 4) return null;
  let value = 0;
  for (const octet of octets) {
    if (!/^\d{1,3}$/.test(octet)) return null;
    const part = Number(octet);
    if (part > 255) return null;
    value = value * 256 + part;
  }
  return value;
};

const intToIp = (value: number): string =>
  [24, 16, 8, 0].map((shift) => (value >>> shift) & 255).join('.');

/**
 * nextFreeAddress returns the next unused host address in subnet, in prefix
 * form, or null when the subnet is malformed or exhausted.
 *
 * The network and broadcast addresses are skipped, and the prefix length is
 * carried onto the result: the fabric compiler requires an interface address
 * to match its network's prefix length exactly, so a bare address or a /32
 * would be refused at start.
 */
export function nextFreeAddress(subnet: string, taken: readonly string[]): string | null {
  const [base, bitsText] = subnet.split('/');
  const bits = Number(bitsText);
  const baseValue = base ? ipToInt(base) : null;
  if (baseValue === null || !Number.isInteger(bits) || bits < 1 || bits > 30) {
    return null;
  }

  const size = 2 ** (32 - bits);
  const network = baseValue - (baseValue % size);
  const used = new Set(
    taken.map((entry) => entry.split('/')[0]).filter((entry): entry is string => Boolean(entry)),
  );

  // Skip the network address itself; stop before the broadcast address.
  for (let offset = 1; offset < size - 1; offset += 1) {
    const candidate = intToIp(network + offset);
    if (!used.has(candidate)) {
      return `${candidate}/${bits}`;
    }
  }
  return null;
}

/** Every address already spoken for, so auto-assign does not hand one out
 * twice. Bare `ips` entries count: they occupy the address just as an
 * interface prefix does. */
export function takenAddresses(configText: string): string[] {
  const taken: string[] = [];
  const doc = parseDocument(configText);
  if (doc.errors.length > 0 || !isMap(doc.contents)) return taken;
  const devices = doc.get('devices');
  if (!isSeq(devices)) return taken;

  for (const item of devices.items) {
    if (!isMap(item)) continue;
    const ips = item.get('ips');
    if (isSeq(ips)) {
      for (const ip of ips.items) {
        const value = scalar(ip);
        if (value) taken.push(value);
      }
    }
    const interfaces = item.get('interfaces');
    if (!isSeq(interfaces)) continue;
    for (const iface of interfaces.items) {
      if (!isMap(iface)) continue;
      const address = scalar(iface.get('address'));
      if (address) taken.push(address);
    }
  }
  return taken;
}
