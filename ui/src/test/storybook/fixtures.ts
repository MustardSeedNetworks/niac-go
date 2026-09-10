/**
 * Fixtures shared by the page stories.
 *
 * U6's row asked for "the same fixtures the E2E uses". There are none: the
 * E2E suite runs against the embedded binary with real daemon data and
 * declares its `route.fulfill` bodies inline, per spec (ui/e2e). So the
 * fixtures live here instead, typed against the wire types so a shape that
 * drifts from the API fails `tsc` rather than a story.
 *
 * One scenario runs on one interface with three devices, which is enough for
 * every page's loaded state and small enough to read.
 */
import type { LibraryFileEntry } from '../../api/library-client';
import type {
  AlertConfig,
  ConfigDocument,
  DebugLevelResponse,
  DeviceSummary,
  ErrorInjectionInfo,
  HistoryRecord,
  InterfacesResponse,
  LibraryNetwork,
  NeighborRecord,
  ReplayState,
  SegmentSummary,
  SessionSummary,
  SimulationStatus,
  StackStatsResponse,
  StandaloneCaptureStatus,
  Template,
  TopologyGraph,
  VersionInfo,
} from '../../api/types';

export const SESSION_ID = 'sim-story';

export const sessions: SessionSummary[] = [
  { sessionId: SESSION_ID, interface: 'eth0', configPath: 'office.yaml', deviceCount: 3 },
];

export const simulationStatus: SimulationStatus = {
  sessionId: SESSION_ID,
  selected: true,
  running: true,
  interface: 'eth0',
  attachmentMode: 'direct',
  configPath: 'office.yaml',
  configName: 'office',
  deviceCount: 3,
  startedAt: '2026-09-07T09:00:00Z',
  uptimeSeconds: 3_600,
  sessions: [
    {
      sessionId: SESSION_ID,
      selected: true,
      running: true,
      interface: 'eth0',
      configName: 'office',
      deviceCount: 3,
      uptimeSeconds: 3_600,
    },
  ],
};

/** No scenario running: the state every page shows before a first start. */
export const simulationIdle: SimulationStatus = {
  running: false,
  deviceCount: 0,
  uptimeSeconds: 0,
  sessions: [],
};

const coreSwitch: DeviceSummary = {
  name: 'core-sw-01',
  type: 'switch',
  ips: ['10.0.0.2'],
  protocols: ['snmp', 'lldp', 'stp'],
  mac: 'AA:BB:CC:00:00:01',
  vendor: 'Cisco',
  model: 'C9300',
};

const edgeRouter: DeviceSummary = {
  name: 'edge-rtr-01',
  type: 'router',
  ips: ['10.0.0.1'],
  protocols: ['snmp', 'cdp', 'dhcp'],
  mac: 'AA:BB:CC:00:00:02',
  vendor: 'Cisco',
  model: 'ISR4331',
};

const workstation: DeviceSummary = {
  name: 'ws-4021',
  type: 'workstation',
  ips: ['10.0.0.51'],
  protocols: ['arp', 'netbios'],
  mac: 'AA:BB:CC:00:00:03',
  vendor: 'Dell',
};

export const devices: DeviceSummary[] = [coreSwitch, edgeRouter, workstation];

export const segments: SegmentSummary[] = [
  { vlanTag: 10, devices: [coreSwitch, edgeRouter] },
  { vlanTag: 20, untagged: true, devices: [workstation] },
];

export const stats: StackStatsResponse = {
  timestamp: '2026-09-07T10:00:00Z',
  interface: 'eth0',
  version: '0.95.22',
  deviceCount: 3,
  stack: {
    packetsSent: 18_402,
    packetsReceived: 17_918,
    arpRequests: 640,
    arpReplies: 638,
    icmpRequests: 120,
    icmpReplies: 120,
    dnsQueries: 88,
    dhcpRequests: 12,
    snmpQueries: 9_431,
    errors: 3,
  },
};

export const history: HistoryRecord[] = [
  {
    id: 2,
    startedAt: '2026-09-07T09:00:00Z',
    duration: '1h0m0s',
    interface: 'eth0',
    configName: 'office',
    deviceCount: 3,
    packetsSent: 18_402,
    packetsReceived: 17_918,
    errors: 3,
  },
  {
    id: 1,
    startedAt: '2026-09-06T14:00:00Z',
    duration: '22m14s',
    interface: 'eth0',
    configName: 'hospital',
    deviceCount: 12,
    packetsSent: 4_210,
    packetsReceived: 4_090,
    errors: 0,
  },
];

export const neighbors: NeighborRecord[] = [
  {
    protocol: 'lldp',
    localDevice: 'core-sw-01',
    remoteDevice: 'edge-rtr-01',
    remotePort: 'GigabitEthernet0/0/0',
    remoteChassisId: 'AA:BB:CC:00:00:02',
    description: 'uplink',
    capabilities: ['Router'],
    managementAddress: '10.0.0.1',
    lastSeen: '2026-09-07T09:59:00Z',
    ttl: 120,
  },
];

export const topology: TopologyGraph = {
  nodes: [
    { name: 'core-sw-01', type: 'switch' },
    { name: 'edge-rtr-01', type: 'router' },
    { name: 'ws-4021', type: 'workstation' },
  ],
  links: [
    {
      source: 'core-sw-01',
      target: 'edge-rtr-01',
      label: 'Gi1/0/1 — Gi0/0/0',
      linkType: 'lldp',
      sourceInterface: 'GigabitEthernet1/0/1',
      targetInterface: 'GigabitEthernet0/0/0',
      speed: '1 Gbps',
      duplex: 'full',
      status: 'up',
      utilizationPercent: 12,
    },
    { source: 'core-sw-01', target: 'ws-4021', label: 'Gi1/0/4', linkType: 'fdb', fdbOnly: true },
  ],
};

export const interfaces: InterfacesResponse = {
  interfaces: [
    { name: 'eth0', description: 'primary lab NIC', addresses: ['10.0.0.9/24'] },
    { name: 'eth1', description: 'trunk', addresses: [] },
  ],
};

export const errorTypes: ErrorInjectionInfo = {
  info: 'Injected errors persist until cleared.',
  availableTypes: [
    { type: 'crc', description: 'CRC / FCS errors on the selected interface', valueKind: 'number' },
    { type: 'drop', description: 'Silently discard frames', valueKind: 'number' },
    { type: 'latency', description: 'Delay responses', valueKind: 'number' },
  ],
  targets: [
    {
      device: 'core-sw-01',
      address: '10.0.0.2',
      interfaces: ['GigabitEthernet1/0/1'],
      errorTypes: { 'GigabitEthernet1/0/1': ['crc', 'drop'] },
    },
    {
      device: 'edge-rtr-01',
      address: '10.0.0.1',
      interfaces: ['GigabitEthernet0/0/0'],
      errorTypes: { 'GigabitEthernet0/0/0': ['crc', 'drop'] },
    },
  ],
  activeErrors: { 'core-sw-01': { 'GigabitEthernet1/0/1': { crc: { value: 42 } } } },
};

export const alerts: AlertConfig = { packetsThreshold: 10_000, webhookUrl: '' };

export const version: VersionInfo = { version: '0.95.22' };

export const walkFiles: LibraryFileEntry[] = [
  {
    name: 'cisco-c9300.walk',
    sizeBytes: 84_213,
    modifiedAt: '2026-09-01T10:00:00Z',
    source: 'starter',
    edited: false,
  },
  {
    name: 'aruba-2930f.walk',
    sizeBytes: 51_004,
    modifiedAt: '2026-08-28T10:00:00Z',
    source: 'user',
    edited: true,
  },
];

export const pcapFiles: LibraryFileEntry[] = [
  {
    name: 'office-morning.pcap',
    sizeBytes: 2_401_882,
    modifiedAt: '2026-09-05T08:12:00Z',
    source: 'user',
    edited: false,
  },
];

export const libraryNetworks: LibraryNetwork[] = [
  {
    name: 'office',
    description: 'Three-device office segment',
    useCase: 'demo',
    deviceCount: 3,
    modifiedAt: '2026-09-01T10:00:00Z',
    sizeBytes: 4_096,
    source: 'starter',
    valid: true,
  },
];

export const templates: Template[] = [
  {
    name: 'switch-basic',
    displayName: 'Basic switch',
    description: 'One switch with LLDP and SNMP',
    deviceCount: 1,
    type: 'switch',
    vendor: 'Cisco',
  },
];

export const configDocument: ConfigDocument = {
  path: '/var/lib/niac/office.yaml',
  filename: 'office.yaml',
  modifiedAt: '2026-09-01T10:00:00Z',
  sizeBytes: 4_096,
  deviceCount: 3,
  content: 'devices:\n  - hostname: core-sw-01\n    type: switch\n',
};

export const captureStatus: StandaloneCaptureStatus = { running: false, packets: 0 };

export const replayState: ReplayState = {
  running: false,
  file: '',
  loopMs: 0,
  scale: 1,
  packetsSent: 0,
  bytesSent: 0,
  packetsTotal: 0,
  bytesTotal: 0,
  passes: 0,
  packetsFiltered: 0,
};

export const debugLevel: DebugLevelResponse = { level: 'info', defaultLevel: 'info' };
