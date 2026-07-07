/**
 * @fileoverview NIAC - Help Content for WebUI
 * @description Authoritative help content for the Network In A Can web UI.
 *              Includes device-type guides, protocol reference, CLI command help,
 *              glossary, keyboard shortcuts, and FAQ. Both technical and
 *              layman-friendly explanations are included.
 *
 *              Structure mirrors stem/ui/src/data/help-content.ts so that both
 *              projects share an analogous shape (TestHelp / Parameter / Metric /
 *              Example / Category). NIAC's "tests" are simulated protocols,
 *              device personas, and CLI subcommands rather than RFC test suites.
 *
 * @copyright 2026 Mustard Seed Networks. All rights reserved.
 * @license Proprietary
 * @maintainer kris.armstrong@icloud.com
 */

// ---------------------------------------------------------------------------
// Core interfaces (parity with stem)
// ---------------------------------------------------------------------------

/**
 * A single help "item" — a simulated device persona, a protocol handler,
 * a CLI subcommand, or a feature. Modeled on stem's `TestHelp` but renamed
 * `HelpItem` because NIAC items aren't tests.
 *
 * For structural parity, `TestHelp` is exported as an alias.
 */
export interface HelpItem {
  id: string;
  name: string;
  /** Standard / namespace (e.g. "RFC 826", "IEEE 802.1AB", "niac CLI"). */
  standard: string;
  /** Category id — must match an entry in `categories`. */
  category: string;
  summary: string;
  techDesc: string;
  laymanDesc: string;
  whenToUse: string;
  whenNotToUse: string;
  parameters: Parameter[];
  /** YAML config keys for protocol / device items; empty for commands. */
  configFields: ConfigField[];
  /** Output metrics or counters relevant to this item. */
  metrics: Metric[];
  examples: Example[];
  tips: string[];
  seeAlso: string[];
}

/** Backwards-compatible alias for stem parity. */
export type TestHelp = HelpItem;

export interface Parameter {
  name: string;
  flag: string;
  type: string;
  defaultValue: string;
  required: boolean;
  techDesc: string;
  laymanDesc: string;
  example: string;
}

export interface ConfigField {
  /** Dotted YAML path, e.g. `lldp.system_name`. */
  path: string;
  type: string;
  defaultValue: string;
  required: boolean;
  description: string;
  example: string;
}

export interface Metric {
  name: string;
  unit: string;
  goodRange: string;
  badMeaning: string;
}

export interface Example {
  desc: string;
  command: string;
  output?: string;
}

export interface Category {
  id: string;
  name: string;
  fullName: string;
  summary: string;
  description: string;
  /** List of HelpItem ids that belong to this category. */
  items: string[];
  whenToUse: string;
}

export interface GlossaryEntry {
  term: string;
  definition: string;
  category: 'protocol' | 'concept' | 'device' | 'niac' | 'security';
}

export interface Shortcut {
  keys: string[];
  description: string;
  category: 'navigation' | 'actions' | 'general' | 'tui';
}

export interface FAQEntry {
  id: string;
  question: string;
  answer: string;
  tags: string[];
}

export interface Feature {
  title: string;
  description: string;
  path: string;
  badge?: string;
}

// ---------------------------------------------------------------------------
// Categories
// ---------------------------------------------------------------------------

export const categories: Record<string, Category> = {
  devices: {
    id: 'devices',
    name: 'Devices',
    fullName: 'Simulated Device Personas',
    summary: 'Pre-defined device archetypes NIAC can pretend to be on the wire.',
    description: `NIAC simulates network devices by responding to wire-level protocols
on a chosen interface. A device persona is a YAML block under \`devices:\` that
combines a MAC address, one or more IP addresses, and a set of enabled
protocol handlers (ARP, ICMP, LLDP, CDP, SNMP, HTTP, etc.). The persona's
\`type\` field selects sensible defaults — e.g. a router advertises router
capabilities in LLDP and CDP, a switch advertises bridge capabilities and runs
STP, an IoT host stays minimal.`,
    items: [
      'device-router',
      'device-switch',
      'device-firewall',
      'device-server',
      'device-workstation',
      'device-ap',
      'device-iot',
      'device-unknown',
    ],
    whenToUse:
      'Building a YAML config from scratch, deciding what each box on the simulated network should look like to a discovery tool.',
  },
  protocols: {
    id: 'protocols',
    name: 'Protocols',
    fullName: 'Wire-Protocol Implementations',
    summary: 'The protocols NIAC speaks for simulated devices.',
    description: `Each simulated device opts into a set of protocol handlers.
Layer 2 protocols (ARP, LLDP, CDP, EDP, FDP, STP) make the device discoverable
to neighbors. Layer 3 (ICMPv4, ICMPv6, IPv4, IPv6) make it reachable.
Application-layer handlers (DHCPv4, DHCPv6, DNS, HTTP, FTP, NetBIOS, SNMP,
iperf3) make it useful to a tool that's poking it. All handlers share a common
shape: a YAML block keyed by the protocol name with \`enabled: true\` and
protocol-specific fields.`,
    items: [
      'proto-arp',
      'proto-icmpv4',
      'proto-icmpv6',
      'proto-ipv4',
      'proto-ipv6',
      'proto-dhcpv4',
      'proto-dhcpv6',
      'proto-dns',
      'proto-lldp',
      'proto-cdp',
      'proto-edp',
      'proto-fdp',
      'proto-stp',
      'proto-snmpv1',
      'proto-snmpv2c',
      'proto-snmpv3',
      'proto-http',
      'proto-tcp',
      'proto-udp',
      'proto-iperf3',
      'proto-netbios',
      'proto-ftp',
    ],
    whenToUse:
      'Deciding which protocol to enable on a device, debugging why a discovery tool isn’t seeing the simulated box, or extending a config to cover a new vendor scenario.',
  },
  topology: {
    id: 'topology',
    name: 'Topology',
    fullName: 'Multi-Device Topology Features',
    summary: 'VLANs, trunks, port-channels, and the topology view.',
    description: `NIAC isn’t just one device per process — a single YAML
config can describe a small network of simulated devices that share a single
host interface. Topology features include VLAN tagging on a per-device basis,
trunk ports carrying multiple VLANs, LACP-bonded port-channels, and a graph
visualization in the web UI that derives links from LLDP/CDP neighbor entries.`,
    items: ['topo-vlan', 'topo-trunk', 'topo-portchannel', 'topo-graph'],
    whenToUse:
      'Modeling something more interesting than a single device — a simulated access switch with three downstream IoT devices, a multi-VLAN core router, etc.',
  },
  commands: {
    id: 'commands',
    name: 'Commands',
    fullName: 'NIAC CLI Subcommands',
    summary: 'Every command the niac binary exposes.',
    description: `NIAC is a single binary with a Cobra-based CLI. Top-level
subcommands cover the full lifecycle: \`init\`/\`generate\`/\`template\` for
creating configs, \`validate\` for checking them, \`daemon\`/\`run\`/\`interactive\`
for executing simulations, \`status\`/\`monitor\`/\`logs\`/\`dump\` for observing a
running daemon, \`inject\` for fault injection, \`neighbors\` for the LLDP/CDP
table, \`analyze-pcap\`/\`analyze-walk\`/\`sanitize\` for offline tools, and
\`topology\`/\`service\`/\`man\`/\`completion\` for housekeeping.`,
    items: [
      'cmd-daemon',
      'cmd-run',
      'cmd-interactive',
      'cmd-init',
      'cmd-generate',
      'cmd-validate',
      'cmd-template',
      'cmd-status',
      'cmd-monitor',
      'cmd-logs',
      'cmd-dump',
      'cmd-inject',
      'cmd-neighbors',
      'cmd-analyze-pcap',
      'cmd-analyze-walk',
      'cmd-sanitize',
      'cmd-topology',
      'cmd-service',
      'cmd-man',
      'cmd-completion',
    ],
    whenToUse:
      'Figuring out what flag does what, building scripts around NIAC, integrating with CI.',
  },
  analysis: {
    id: 'analysis',
    name: 'Analysis',
    fullName: 'Capture & Walk Analysis',
    summary: 'Offline tools that turn pcaps and SNMP walks into NIAC configs.',
    description: `Two analysis pipelines short-circuit the usual write-YAML-from-scratch
loop: \`analyze-pcap\` reads a packet capture and emits protocol counters plus a
draft device list inferred from observed MACs and IPs, and \`analyze-walk\`
reads an SNMP walk file (snmpwalk output) and reconstructs interface
inventories, neighbor tables, and routing relationships from the OIDs it
finds. Both are read-only by default.`,
    items: ['analysis-pcap', 'analysis-walk', 'analysis-sanitize'],
    whenToUse:
      'You have a packet capture or SNMP walk of a real device and want NIAC to roughly mimic it.',
  },
  errors: {
    id: 'errors',
    name: 'Errors',
    fullName: 'Error Injection',
    summary: 'Make a simulated device misbehave to test monitoring tooling.',
    description: `Error injection lets you push a running simulated device into a
faulted state without restarting the daemon. The \`niac inject\` command talks
to the daemon over its IPC socket and tells it to set per-device counters
(FCS errors, packet discards, interface errors) or per-device thresholds
(high CPU, high memory, high disk, high utilization) to specified values.
The simulated SNMP agent then reflects those values, so a monitoring tool
polling the device will see the fault.`,
    items: [
      'err-fcs',
      'err-discards',
      'err-iferrors',
      'err-cpu',
      'err-memory',
      'err-disk',
      'err-util',
    ],
    whenToUse:
      'Validating that an NMS, SIEM, or alerting platform actually reacts to faults, without breaking real hardware.',
  },
} as const;

// ---------------------------------------------------------------------------
// Devices
// ---------------------------------------------------------------------------

const deviceItems: HelpItem[] = [
  {
    id: 'device-router',
    name: 'Router',
    standard: 'NIAC device persona',
    category: 'devices',
    summary: 'Simulates an L3 router with full discovery and management protocols.',
    techDesc:
      'A router persona enables ARP, ICMPv4/v6, LLDP (capabilities: bridge, router), CDP (capabilities: Router, Switch, IGMP), and SNMP by default. Typical configs add multiple IPs to model multiple subnets and enable trap-emission for high CPU/memory.',
    laymanDesc:
      'The "router" type tells NIAC to act like a Cisco-style routing box — it shows up in LLDP and CDP as a router, answers pings on every configured IP, and responds to SNMP polls for the standard router OIDs.',
    whenToUse:
      'Validating routing protocol discovery, NMS template detection for routers, or testing tooling that special-cases L3 devices.',
    whenNotToUse:
      'You only need a thing that responds to ping. Use the `host` or `unknown` type instead — less protocol noise on the wire.',
    parameters: [],
    configFields: [
      {
        path: 'devices[].type',
        type: 'string',
        defaultValue: '"router"',
        required: true,
        description: 'Selects the router persona and its discovery-protocol defaults.',
        example: 'type: router',
      },
      {
        path: 'devices[].ips',
        type: 'string[]',
        defaultValue: '[]',
        required: true,
        description: 'One or more IPv4 / IPv6 addresses the router answers on.',
        example: 'ips: ["10.0.0.1", "192.168.1.1"]',
      },
      {
        path: 'devices[].lldp.capabilities',
        type: 'string[]',
        defaultValue: '["bridge", "router"]',
        required: false,
        description: 'LLDP system capabilities advertised by the router.',
        example: 'capabilities: ["bridge", "router"]',
      },
    ],
    metrics: [],
    examples: [
      {
        desc: 'Minimal router persona',
        command: 'niac template use router router.yaml && sudo niac run en0 router.yaml',
      },
      {
        desc: 'Validate a router config before running it',
        command: 'niac validate router.yaml',
        output: 'OK: 1 device(s), 5 protocol handler(s), 0 warnings',
      },
    ],
    tips: [
      'Use a non-routable MAC OUI (00:1A:2B:…) to avoid colliding with anything real.',
      'Pair with `snmp.traps.enabled: true` to drive trap-receiver testing.',
      'Add at least one IP per VLAN you want the router to "own".',
    ],
    seeAlso: ['device-switch', 'proto-snmpv2c', 'proto-lldp', 'topo-vlan'],
  },
  {
    id: 'device-switch',
    name: 'Switch',
    standard: 'NIAC device persona',
    category: 'devices',
    summary: 'Simulates an L2/L3 switch with STP, VLAN tagging, and discovery.',
    techDesc:
      'A switch persona enables ARP, ICMP, STP (with bridge priority and root selection), LLDP/CDP (capabilities: bridge), and SNMP. Per-port configuration supports VLAN tagging, native VLAN, duplex, and port roles.',
    laymanDesc:
      'The "switch" type makes NIAC act like a managed L2 switch. It runs spanning-tree, tags VLAN traffic, advertises bridge capabilities to neighbors, and answers SNMP for switch-style OIDs (bridge MIB, RFC 2674, etc.).',
    whenToUse:
      'Testing topology discovery, STP-aware tools, switch-template detection in an NMS, or VLAN configurations.',
    whenNotToUse:
      'Validating routing behavior — the switch persona doesn’t do routing; use a router or pair both.',
    parameters: [],
    configFields: [
      {
        path: 'devices[].type',
        type: 'string',
        defaultValue: '"switch"',
        required: true,
        description: 'Selects the switch persona.',
        example: 'type: switch',
      },
      {
        path: 'devices[].stp.bridge_priority',
        type: 'integer',
        defaultValue: '32768',
        required: false,
        description: 'STP bridge priority. Lower wins root election.',
        example: 'bridge_priority: 4096',
      },
      {
        path: 'devices[].lldp.vlan_id',
        type: 'integer',
        defaultValue: '0',
        required: false,
        description: 'Native VLAN advertised in LLDP TLV.',
        example: 'vlan_id: 100',
      },
    ],
    metrics: [],
    examples: [
      {
        desc: 'Spin up a single access switch',
        command: 'niac template use switch switch.yaml && sudo niac run en0 switch.yaml',
      },
      {
        desc: 'Make this switch the STP root',
        command: '# set bridge_priority lower than any other switch in the YAML',
      },
    ],
    tips: [
      'STP root election uses bridge priority then MAC address — mirror real-world tiebreakers.',
      'Set distinct OUIs per vendor when simulating a multi-vendor lab.',
    ],
    seeAlso: ['proto-stp', 'proto-lldp', 'proto-cdp', 'topo-vlan'],
  },
  {
    id: 'device-firewall',
    name: 'Firewall',
    standard: 'NIAC device persona',
    category: 'devices',
    summary: 'A router-like persona with firewall-style SNMP and capabilities.',
    techDesc:
      'NIAC treats firewall as a specialization of router: same L2/L3 protocols, but SNMP sysDescr and LLDP system description default to firewall-vendor strings, and CDP/LLDP advertise host capability rather than router/bridge.',
    laymanDesc:
      'Use the "firewall" type when your tooling does string-matching on sysDescr or asks "is this device a firewall?" Otherwise it behaves like a normal L3 box.',
    whenToUse:
      'NMS detects firewalls via sysDescr regex, or you want a clearly-labeled firewall in the topology view.',
    whenNotToUse:
      'You’re testing actual firewall packet inspection — NIAC doesn’t filter; it just answers protocol queries.',
    parameters: [],
    configFields: [
      {
        path: 'devices[].type',
        type: 'string',
        defaultValue: '"firewall"',
        required: true,
        description: 'Selects firewall persona.',
        example: 'type: firewall',
      },
      {
        path: 'devices[].snmp.sys_descr',
        type: 'string',
        defaultValue: '"NIAC firewall"',
        required: false,
        description:
          'SNMP sysDescr string. Match a real vendor string to drive template detection.',
        example: 'sys_descr: "Palo Alto Networks PA-3220"',
      },
    ],
    metrics: [],
    examples: [
      {
        desc: 'Palo Alto-flavored firewall',
        command: '# set snmp.sys_descr to a PA-series string and lldp.system_description likewise',
      },
      {
        desc: 'Cisco ASA-flavored firewall',
        command: '# set snmp.sys_descr: "Cisco Adaptive Security Appliance Software Version 9.16"',
      },
    ],
    tips: [
      'Real firewalls rarely run CDP — disable it for a more authentic shape.',
      'Set sysObjectID to a vendor-appropriate OID to drive proper template detection.',
    ],
    seeAlso: ['device-router', 'proto-snmpv2c'],
  },
  {
    id: 'device-server',
    name: 'Server',
    standard: 'NIAC device persona',
    category: 'devices',
    summary: 'A general-purpose host with HTTP, SSH banner, and optional SNMP.',
    techDesc:
      'Server persona enables ARP, ICMP, optional HTTP (with custom endpoints), and SNMP with host-style sysDescr. STP and CDP are disabled by default. Common adds: DNS server, FTP banner, NetBIOS for Windows-flavored hosts.',
    laymanDesc:
      'The "server" type makes NIAC act like a Linux or Windows server: pings work, HTTP answers, and management tooling sees a host — not a bridge or router.',
    whenToUse:
      'Simulating application servers, building inventory test fixtures, exercising server-discovery rules.',
    whenNotToUse:
      'You need real HTTP semantics (auth, sessions, full REST). NIAC’s HTTP server returns canned content per path.',
    parameters: [],
    configFields: [
      {
        path: 'devices[].type',
        type: 'string',
        defaultValue: '"server"',
        required: true,
        description: 'Selects server persona.',
        example: 'type: server',
      },
      {
        path: 'devices[].http.endpoints',
        type: 'object[]',
        defaultValue: '[]',
        required: false,
        description: 'Path/content pairs returned by the HTTP handler.',
        example: 'endpoints: [{path: "/", content: "<h1>NIAC</h1>"}]',
      },
    ],
    metrics: [],
    examples: [
      {
        desc: 'Web server with custom homepage',
        command: 'niac template use server server.yaml && sudo niac run en0 server.yaml',
      },
      {
        desc: 'Server with a JSON API',
        command:
          '# set http.endpoints to [{path: "/api/v1/status", content: \'{"ok":true}\', headers: {Content-Type: application/json}}]',
      },
    ],
    tips: [
      'Set `server_header` on the HTTP block to mimic Apache/nginx/IIS.',
      'NetBIOS makes the host show up in Windows network browsing — enable it for Windows-server feel.',
    ],
    seeAlso: ['proto-http', 'proto-netbios', 'proto-ftp'],
  },
  {
    id: 'device-workstation',
    name: 'Workstation',
    standard: 'NIAC device persona',
    category: 'devices',
    summary: 'End-user host: ARP, ICMP, NetBIOS, minimal SNMP.',
    techDesc:
      'Workstation persona enables ARP and ICMP, optionally NetBIOS (for Windows-flavored boxes) and a minimal SNMP read-only agent if needed. STP and CDP are off; LLDP is off by default but can be enabled for "host-only" capability advertisement.',
    laymanDesc:
      'Use "workstation" for desktop / laptop endpoints. Answers ping, shows up in NetBIOS browsing, but doesn’t pretend to be infrastructure.',
    whenToUse: 'Populating a simulated office network with realistic endpoint density.',
    whenNotToUse:
      'You need to drive real client behavior (DHCP DISCOVER, mDNS, etc.) — NIAC is a responder, not an initiator.',
    parameters: [],
    configFields: [
      {
        path: 'devices[].type',
        type: 'string',
        defaultValue: '"workstation"',
        required: true,
        description: 'Selects workstation persona.',
        example: 'type: workstation',
      },
    ],
    metrics: [],
    examples: [
      {
        desc: 'Win10 workstation with NetBIOS',
        command: '# devices[].netbios.enabled: true; devices[].netbios.name: "WORKSTATION-01"',
      },
      {
        desc: 'Mac workstation, no NetBIOS',
        command: '# devices[].netbios.enabled: false (default for workstation)',
      },
    ],
    tips: [
      'Workstations rarely run SNMP — leave it off unless you have an inventory tool that polls them.',
      'A NetBIOS-suffix-03 entry gives the host a Windows-friendly browse name.',
    ],
    seeAlso: ['proto-netbios', 'proto-arp', 'proto-icmpv4'],
  },
  {
    id: 'device-ap',
    name: 'Access Point',
    standard: 'NIAC device persona',
    category: 'devices',
    summary: 'Wi-Fi AP: LLDP-advertised, SNMP for radio OIDs.',
    techDesc:
      'AP persona advertises LLDP capability `wlan-access-point` with optional management-address TLV, runs CDP with `Host` capability, and exposes SNMP under wireless-specific MIBs (CISCO-LWAPP-AP-MIB / IEEE 802.11 MIB) when a walk file is supplied.',
    laymanDesc:
      'The "ap" type makes NIAC look like a thin AP — advertises itself to the network as a Wi-Fi access point and answers wireless-controller polls.',
    whenToUse:
      'Validating wireless-management tooling (Cisco DNA, Aruba Central, etc.) without standing up a real AP.',
    whenNotToUse: 'You’re testing actual Wi-Fi RF behavior — NIAC has no 802.11 radio.',
    parameters: [],
    configFields: [
      {
        path: 'devices[].type',
        type: 'string',
        defaultValue: '"ap"',
        required: true,
        description: 'Selects access-point persona.',
        example: 'type: ap',
      },
      {
        path: 'devices[].lldp.capabilities',
        type: 'string[]',
        defaultValue: '["wlan-access-point"]',
        required: false,
        description: 'LLDP capability TLV.',
        example: 'capabilities: ["wlan-access-point"]',
      },
    ],
    metrics: [],
    examples: [
      {
        desc: 'Cisco Aironet 9120 AP',
        command: 'niac template use ap ap.yaml && sudo niac run en0 ap.yaml',
      },
      {
        desc: 'AP with SNMP walk reflecting real radio counters',
        command: '# devices[].snmp.walk_file: examples/device_walks/aironet-9120.walk',
      },
    ],
    tips: [
      'Use a wireless OUI prefix (e.g. 00:1C:2D for Cisco) to look authentic.',
      'Set `power_draw` in CDP to a sensible PoE value (e.g. 15.4 for 802.3af).',
    ],
    seeAlso: ['proto-lldp', 'proto-snmpv2c'],
  },
  {
    id: 'device-iot',
    name: 'IoT / Basic Host',
    standard: 'NIAC device persona',
    category: 'devices',
    summary: 'Bare-minimum endpoint: ARP, ICMP, optional tiny HTTP.',
    techDesc:
      'The `host` type (alias `iot` in templates) enables only ARP and ICMP by default. Useful for modeling sensor-class endpoints. Optionally enable HTTP with a single endpoint to mimic an IoT device’s telemetry endpoint, or a stripped LLDP TLV set for tools that expect every endpoint to announce itself.',
    laymanDesc:
      'The "iot" template is for cameras, sensors, badge readers — things that mostly just answer ping and maybe serve a tiny JSON API.',
    whenToUse:
      'Building dense networks of stub endpoints, or stress-testing tools that have to scan many low-talk devices.',
    whenNotToUse:
      'You need the device to actively report (CoAP, MQTT, syslog) — NIAC is a passive responder.',
    parameters: [],
    configFields: [
      {
        path: 'devices[].type',
        type: 'string',
        defaultValue: '"host"',
        required: true,
        description: 'Bare-minimum host persona; templates name this `iot`.',
        example: 'type: host',
      },
    ],
    metrics: [],
    examples: [
      {
        desc: 'Temperature sensor with /data endpoint',
        command: 'niac template use iot sensor.yaml && sudo niac run en0 sensor.yaml',
      },
      {
        desc: 'Fifty bare endpoints for scanner stress test',
        command:
          'niac config generate scan-targets.yaml  # accept defaults, set device count to 50',
      },
    ],
    tips: [
      'Skip SNMP — most IoT devices don’t run it and silence on UDP/161 is more realistic.',
      'Use station-only LLDP capability if you enable LLDP at all.',
    ],
    seeAlso: ['proto-arp', 'proto-icmpv4', 'proto-http'],
  },
  {
    id: 'device-unknown',
    name: 'Unknown',
    standard: 'NIAC device persona',
    category: 'devices',
    summary: 'Catch-all device with no opinionated defaults.',
    techDesc:
      'The `unknown` type disables all type-driven defaults. Every protocol must be explicitly enabled. Use it when you want full control or when modeling something that doesn’t fit standard archetypes (e.g. an industrial PLC, a smart TV, a printer).',
    laymanDesc:
      'Pick "unknown" when none of the other types feel right and you want to opt in to each protocol by hand.',
    whenToUse: 'You’re modeling something specialized and want zero implicit behavior.',
    whenNotToUse: 'You want sensible defaults — use a more specific type.',
    parameters: [],
    configFields: [
      {
        path: 'devices[].type',
        type: 'string',
        defaultValue: '"unknown"',
        required: true,
        description: 'No-defaults persona.',
        example: 'type: unknown',
      },
    ],
    metrics: [],
    examples: [
      {
        desc: 'Network printer',
        command: '# type: unknown, enable arp + icmp + snmp + http + ftp + netbios',
      },
      {
        desc: 'PLC / industrial controller',
        command: '# type: unknown, enable arp + icmp + minimal SNMP',
      },
    ],
    tips: [
      'If you find yourself enabling the same 5 protocols repeatedly with `unknown`, consider proposing a new persona type upstream.',
    ],
    seeAlso: ['device-iot', 'device-server'],
  },
];

// ---------------------------------------------------------------------------
// Protocols
// ---------------------------------------------------------------------------

const protocolItems: HelpItem[] = [
  {
    id: 'proto-arp',
    name: 'ARP',
    standard: 'RFC 826',
    category: 'protocols',
    summary: 'Maps IPv4 addresses to MAC addresses on the local link.',
    techDesc:
      'NIAC answers ARP Requests (opcode 1) for any IPv4 address listed in the device’s `ips` and replies with the device’s `mac`. Gratuitous ARP is emitted at startup for each IP to seed neighbor caches. Both Ethernet II and 802.1Q-tagged ARP are handled when VLANs are configured.',
    laymanDesc:
      'ARP is how computers find each other’s hardware (MAC) addresses on a LAN. NIAC answers ARP "who-has X" questions for every IP you give it, so the rest of the network knows where to send packets.',
    whenToUse: 'Always. Without ARP, no IPv4 traffic reaches the simulated device.',
    whenNotToUse: 'You’re running an IPv6-only device — then ICMPv6 ND replaces ARP.',
    parameters: [],
    configFields: [
      {
        path: 'devices[].arp.enabled',
        type: 'boolean',
        defaultValue: 'true',
        required: false,
        description: 'Enable ARP responder.',
        example: 'arp:\n  enabled: true',
      },
      {
        path: 'devices[].arp.gratuitous',
        type: 'boolean',
        defaultValue: 'true',
        required: false,
        description: 'Emit gratuitous ARP for each IP at startup.',
        example: 'arp:\n  gratuitous: false',
      },
    ],
    metrics: [
      {
        name: 'arp_requests_received',
        unit: 'count',
        goodRange: '> 0',
        badMeaning: 'Nobody asked — the device may be unreachable.',
      },
      {
        name: 'arp_replies_sent',
        unit: 'count',
        goodRange: 'matches requests for own IPs',
        badMeaning: 'Mismatch indicates handler bug.',
      },
    ],
    examples: [
      {
        desc: 'Verify ARP works from another host',
        command: 'arp -n 10.0.0.1   # after sudo niac run en0 router.yaml',
        output: '10.0.0.1   ether   00:1A:2B:3C:4D:01   C   en0',
      },
      {
        desc: 'Disable gratuitous ARP (quieter startup)',
        command: '# devices[].arp.gratuitous: false',
      },
    ],
    tips: [
      'If `ping` from a neighbor times out, run `arpscan` to verify NIAC’s reply lands.',
      'On macOS test interfaces, ARP entries time out fast — keep traffic flowing or pin them.',
    ],
    seeAlso: ['proto-icmpv4', 'proto-icmpv6'],
  },
  {
    id: 'proto-icmpv4',
    name: 'ICMPv4',
    standard: 'RFC 792',
    category: 'protocols',
    summary: 'Answers IPv4 ping (Echo Request).',
    techDesc:
      'NIAC handles ICMPv4 Echo Request (type 8) on every configured IPv4 address and emits Echo Reply (type 0) with matching identifier, sequence, and payload. Other ICMP types are dropped silently.',
    laymanDesc:
      'ICMP is how `ping` works — enabling this makes the simulated device answer pings on every IP you gave it.',
    whenToUse: 'Almost always — ping is the universal "is it up?" check.',
    whenNotToUse: 'You want to specifically test how a tool reacts to a non-pingable device.',
    parameters: [],
    configFields: [
      {
        path: 'devices[].icmp.enabled',
        type: 'boolean',
        defaultValue: 'true',
        required: false,
        description: 'Enable ICMPv4 responder.',
        example: 'icmp:\n  enabled: true',
      },
    ],
    metrics: [
      {
        name: 'icmp_echo_requests',
        unit: 'count',
        goodRange: '> 0 if pinged',
        badMeaning: '0 = not being polled or interface down.',
      },
    ],
    examples: [
      {
        desc: 'Ping a NIAC device',
        command: 'ping -c 3 10.0.0.1',
        output: '64 bytes from 10.0.0.1: icmp_seq=1 ttl=64 time=0.3 ms',
      },
    ],
    tips: [
      'TTL is set to the host kernel’s default; if you need a specific TTL value, raise an issue.',
    ],
    seeAlso: ['proto-icmpv6', 'proto-arp'],
  },
  {
    id: 'proto-icmpv6',
    name: 'ICMPv6',
    standard: 'RFC 4443 / RFC 4861',
    category: 'protocols',
    summary: 'IPv6 Echo + Neighbor Discovery (NS/NA).',
    techDesc:
      'Handles ICMPv6 Echo Request (type 128) and the Neighbor Discovery suite: Neighbor Solicitation (135), Neighbor Advertisement (136), and Router Solicitation/Advertisement (133/134) where the device persona is a router. Replies use the correct multicast/unicast addresses and include the right options (target link-layer address).',
    laymanDesc:
      'ICMPv6 is ICMP plus the IPv6 version of ARP, all rolled into one. Enable it for any device with an IPv6 address.',
    whenToUse: 'Any device with an IPv6 address — IPv6 stacks won’t reach the device without it.',
    whenNotToUse: 'Pure IPv4-only deployments.',
    parameters: [],
    configFields: [
      {
        path: 'devices[].icmpv6.enabled',
        type: 'boolean',
        defaultValue: 'auto (on if any IPv6 IP)',
        required: false,
        description: 'Enable ICMPv6 + ND responder.',
        example: 'icmpv6:\n  enabled: true',
      },
      {
        path: 'devices[].icmpv6.router_advertise',
        type: 'boolean',
        defaultValue: 'false',
        required: false,
        description: 'On router persona, emit periodic Router Advertisements.',
        example: 'icmpv6:\n  router_advertise: true',
      },
    ],
    metrics: [
      {
        name: 'icmpv6_ns_received',
        unit: 'count',
        goodRange: '> 0 on active link',
        badMeaning: 'Nobody trying to resolve the device.',
      },
    ],
    examples: [
      {
        desc: 'Ping an IPv6 address',
        command: 'ping6 -c 3 fd00::1',
      },
      {
        desc: 'Force NS/NA exchange',
        command: 'ndp -n fd00::1',
      },
    ],
    tips: [
      'IPv6 link-local addresses are auto-derived from the MAC; don’t list them in `ips` explicitly.',
    ],
    seeAlso: ['proto-ipv6', 'proto-arp'],
  },
  {
    id: 'proto-ipv4',
    name: 'IPv4',
    standard: 'RFC 791',
    category: 'protocols',
    summary: 'IPv4 stack underlying ARP/ICMPv4/UDP/TCP.',
    techDesc:
      'IPv4 isn’t a per-device toggle — it’s active for any device with at least one IPv4 IP. NIAC handles fragmentation, checksums, and header validation per RFC 791. Devices ignore traffic addressed to IPs they don’t own (no promiscuous forwarding).',
    laymanDesc:
      'This is the basic plumbing of the internet. NIAC sets it up automatically whenever you give a device an IPv4 address.',
    whenToUse: 'Always for IPv4 devices — implicit.',
    whenNotToUse: 'N/A.',
    parameters: [],
    configFields: [
      {
        path: 'devices[].ips',
        type: 'string[]',
        defaultValue: '[]',
        required: true,
        description: 'IPv4 / IPv6 addresses for this device.',
        example: 'ips: ["10.0.0.1", "fd00::1"]',
      },
    ],
    metrics: [],
    examples: [
      {
        desc: 'Inspect IPv4 traffic with tcpdump',
        command: 'sudo tcpdump -i en0 -nn ip and host 10.0.0.1',
      },
    ],
    tips: ['NIAC drops fragments that overlap or fail validation — fragments aren’t the bug.'],
    seeAlso: ['proto-ipv6', 'proto-arp'],
  },
  {
    id: 'proto-ipv6',
    name: 'IPv6',
    standard: 'RFC 8200',
    category: 'protocols',
    summary: 'IPv6 stack underlying ICMPv6/UDP/TCP.',
    techDesc:
      'Activated automatically for any device with an IPv6 address in `ips`. NIAC handles extension headers, hop limits, and IPv6-specific UDP/TCP checksums. Link-local addresses are auto-derived from the device MAC using EUI-64.',
    laymanDesc:
      'The IPv6 equivalent of the IPv4 stack — turns on automatically when you list an IPv6 address.',
    whenToUse: 'Dual-stack or IPv6-only configs.',
    whenNotToUse: 'IPv4-only.',
    parameters: [],
    configFields: [],
    metrics: [],
    examples: [
      {
        desc: 'Dual-stack device',
        command: '# ips: ["192.168.1.10", "2001:db8::10"]',
      },
    ],
    tips: [
      'Global IPv6 addresses listed must be reachable on the host’s interface — NIAC doesn’t do SLAAC by itself.',
    ],
    seeAlso: ['proto-ipv4', 'proto-icmpv6'],
  },
  {
    id: 'proto-dhcpv4',
    name: 'DHCPv4',
    standard: 'RFC 2131',
    category: 'protocols',
    summary: 'DHCPv4 server emulation with pools and options.',
    techDesc:
      'NIAC answers DHCPv4 DISCOVER, REQUEST, INFORM, RELEASE, and DECLINE messages on UDP/67. Each device can host one or more pools defined by subnet, range, lease time, and DHCP options (gateway, DNS, NTP, vendor-specific, option 82 relay-agent info). Leases are kept in memory and persist for the daemon’s lifetime.',
    laymanDesc:
      'Makes the simulated device act like a DHCP server: real clients on the wire can ask it for an IP address and it will hand one out.',
    whenToUse:
      'Testing DHCP-snooping switches, DHCP-fingerprinting NACs, or PXE-boot environments.',
    whenNotToUse:
      'You only want a quiet device — a misconfigured DHCP server can break real clients on the same LAN.',
    parameters: [],
    configFields: [
      {
        path: 'devices[].dhcp.enabled',
        type: 'boolean',
        defaultValue: 'false',
        required: true,
        description: 'Enable DHCPv4 server on this device.',
        example: 'dhcp:\n  enabled: true',
      },
      {
        path: 'devices[].dhcp.pools[].subnet',
        type: 'string (CIDR)',
        defaultValue: '',
        required: true,
        description: 'Subnet to serve.',
        example: 'subnet: "192.168.1.0/24"',
      },
      {
        path: 'devices[].dhcp.pools[].range',
        type: '[string, string]',
        defaultValue: '',
        required: true,
        description: 'Start / end of the address pool.',
        example: 'range: ["192.168.1.100", "192.168.1.200"]',
      },
      {
        path: 'devices[].dhcp.pools[].lease_time',
        type: 'duration',
        defaultValue: '"24h"',
        required: false,
        description: 'Lease duration.',
        example: 'lease_time: "8h"',
      },
      {
        path: 'devices[].dhcp.pools[].options',
        type: 'object',
        defaultValue: '{}',
        required: false,
        description: 'DHCP options: router, dns_servers, domain_name, ntp_servers, etc.',
        example: 'options:\n  router: 192.168.1.1\n  dns_servers: [8.8.8.8, 1.1.1.1]',
      },
    ],
    metrics: [
      {
        name: 'dhcp_discover_received',
        unit: 'count',
        goodRange: '> 0',
        badMeaning: 'No clients are asking — wrong VLAN or broadcast domain.',
      },
      {
        name: 'dhcp_ack_sent',
        unit: 'count',
        goodRange: 'matches REQUEST',
        badMeaning: 'Pool exhaustion.',
      },
    ],
    examples: [
      {
        desc: 'Minimal DHCP server',
        command: '# devices[].dhcp.pools[0]: {subnet: 10.0.0.0/24, range: [10.0.0.50, 10.0.0.99]}',
      },
      {
        desc: 'DHCP with PXE options',
        command: '# options:\n#   tftp_server: 10.0.0.250\n#   boot_filename: pxelinux.0',
      },
    ],
    tips: [
      'On a shared LAN, prefer using `--dry-run` or running on `lo0` to avoid handing leases to coworkers’ laptops.',
      'Option 82 (relay-agent) requires the server to be the relay target — set `relay_agent` on the device.',
    ],
    seeAlso: ['proto-dhcpv6', 'proto-dns'],
  },
  {
    id: 'proto-dhcpv6',
    name: 'DHCPv6',
    standard: 'RFC 8415',
    category: 'protocols',
    summary: 'DHCPv6 stateful + stateless server.',
    techDesc:
      'Answers DHCPv6 SOLICIT, REQUEST, RENEW, REBIND, CONFIRM, RELEASE, DECLINE, INFORMATION-REQUEST on UDP/547. Supports stateful (IA_NA, IA_TA, IA_PD) and stateless (options-only) modes.',
    laymanDesc: 'The IPv6 version of DHCP. Lets you simulate a DHCPv6 server.',
    whenToUse: 'IPv6 environments that don’t use SLAAC, or PD (prefix delegation) testing.',
    whenNotToUse: 'IPv4-only or SLAAC-only IPv6.',
    parameters: [],
    configFields: [
      {
        path: 'devices[].dhcpv6.enabled',
        type: 'boolean',
        defaultValue: 'false',
        required: true,
        description: 'Enable DHCPv6 server.',
        example: 'dhcpv6:\n  enabled: true',
      },
      {
        path: 'devices[].dhcpv6.mode',
        type: 'string',
        defaultValue: '"stateful"',
        required: false,
        description: 'stateful or stateless.',
        example: 'mode: stateless',
      },
    ],
    metrics: [],
    examples: [
      {
        desc: 'Stateful DHCPv6 server',
        command: '# devices[].dhcpv6: {enabled: true, mode: stateful, pools: [...]}',
      },
    ],
    tips: ['DHCPv6 listens on UDP/547, not UDP/67 — don’t conflate them.'],
    seeAlso: ['proto-dhcpv4', 'proto-icmpv6'],
  },
  {
    id: 'proto-dns',
    name: 'DNS',
    standard: 'RFC 1035',
    category: 'protocols',
    summary: 'Authoritative DNS server with A, AAAA, PTR, CNAME, MX, TXT, SRV, NS, SOA.',
    techDesc:
      'Answers DNS queries on UDP/53 (and optionally TCP/53 for large responses). NIAC serves only zones explicitly configured — it does not recurse or forward. Supports A, AAAA, PTR, CNAME, MX, TXT, SRV, NS, SOA records and DNSSEC-unsigned responses.',
    laymanDesc:
      'Makes the device act like a small DNS server with whatever hostnames you tell it about.',
    whenToUse:
      'Spoofing internal hostnames for tooling tests, building reverse-PTR setups, or hosting an authoritative test zone.',
    whenNotToUse: 'You need a full recursive resolver — NIAC isn’t one.',
    parameters: [],
    configFields: [
      {
        path: 'devices[].dns.enabled',
        type: 'boolean',
        defaultValue: 'false',
        required: true,
        description: 'Enable DNS server.',
        example: 'dns:\n  enabled: true',
      },
      {
        path: 'devices[].dns.zones',
        type: 'object[]',
        defaultValue: '[]',
        required: true,
        description: 'Zones served by this DNS server.',
        example:
          'zones:\n  - name: "example.com"\n    records:\n      - {name: "www", type: A, value: "10.0.0.10"}',
      },
    ],
    metrics: [
      {
        name: 'dns_queries_received',
        unit: 'count',
        goodRange: '> 0 if used',
        badMeaning: 'No clients querying — wrong server address.',
      },
      {
        name: 'dns_responses_sent',
        unit: 'count',
        goodRange: 'matches queries',
        badMeaning: 'Records missing or zone unknown.',
      },
    ],
    examples: [
      {
        desc: 'Query the simulated DNS',
        command: 'dig @10.0.0.10 www.example.com',
        output: 'www.example.com.  300  IN  A  10.0.0.10',
      },
      {
        desc: 'PTR (reverse) lookup',
        command: 'dig @10.0.0.10 -x 10.0.0.10',
      },
    ],
    tips: [
      'NIAC returns NXDOMAIN for any name outside configured zones — add a wildcard if you need a catch-all.',
      'For DNSSEC, sign the zone with an external tool and serve the signed records.',
    ],
    seeAlso: ['proto-dhcpv4', 'proto-netbios'],
  },
  {
    id: 'proto-lldp',
    name: 'LLDP',
    standard: 'IEEE 802.1AB',
    category: 'protocols',
    summary: 'Link Layer Discovery Protocol — vendor-neutral L2 discovery.',
    techDesc:
      'Emits LLDPDUs every 30s (configurable) to the LLDP multicast destination (01:80:c2:00:00:0e). Includes the mandatory TLVs (chassis ID, port ID, TTL, end-of-LLDPDU) plus configured optional TLVs (system name, system description, system capabilities, management address, VLAN ID, port description).',
    laymanDesc:
      'LLDP is the standard way devices tell each other "hi, I’m Bob and I’m a router." Enable it so discovery tools can see the simulated device.',
    whenToUse:
      'Any time you want the device to appear in a neighbor table on a connected switch or in an NMS discovery sweep.',
    whenNotToUse: 'The downstream tool only understands CDP — use CDP instead (or both).',
    parameters: [],
    configFields: [
      {
        path: 'devices[].lldp.enabled',
        type: 'boolean',
        defaultValue: 'false',
        required: true,
        description: 'Enable LLDP transmitter.',
        example: 'lldp:\n  enabled: true',
      },
      {
        path: 'devices[].lldp.system_name',
        type: 'string',
        defaultValue: 'device name',
        required: false,
        description: 'System name TLV.',
        example: 'system_name: "core-router-01"',
      },
      {
        path: 'devices[].lldp.system_description',
        type: 'string',
        defaultValue: '""',
        required: false,
        description: 'System description TLV.',
        example: 'system_description: "Cisco IOS XE 17.3"',
      },
      {
        path: 'devices[].lldp.chassis_id',
        type: 'string (MAC)',
        defaultValue: 'device MAC',
        required: false,
        description: 'Chassis ID TLV.',
        example: 'chassis_id: "00:1A:2B:3C:4D:01"',
      },
      {
        path: 'devices[].lldp.port_id',
        type: 'string',
        defaultValue: 'interface name',
        required: false,
        description: 'Port ID TLV.',
        example: 'port_id: "GigabitEthernet0/1"',
      },
      {
        path: 'devices[].lldp.ttl',
        type: 'integer (seconds)',
        defaultValue: '120',
        required: false,
        description: 'TLV TTL.',
        example: 'ttl: 120',
      },
      {
        path: 'devices[].lldp.capabilities',
        type: 'string[]',
        defaultValue: '[]',
        required: false,
        description:
          'Capabilities TLV: bridge, router, repeater, wlan-access-point, telephone, docsis-cable-device, station-only, other.',
        example: 'capabilities: ["bridge", "router"]',
      },
      {
        path: 'devices[].lldp.management_addresses',
        type: 'string[]',
        defaultValue: '[]',
        required: false,
        description: 'Management addresses TLV.',
        example: 'management_addresses: ["10.0.0.1"]',
      },
    ],
    metrics: [
      {
        name: 'lldp_pdus_sent',
        unit: 'count',
        goodRange: '~120/hour at 30s interval',
        badMeaning: 'Stopped — handler failure.',
      },
      {
        name: 'lldp_pdus_received',
        unit: 'count',
        goodRange: 'depends on neighbors',
        badMeaning: '0 = nothing else on the wire speaks LLDP.',
      },
    ],
    examples: [
      {
        desc: 'View NIAC’s neighbor table',
        command: 'niac neighbors',
        output: '00:11:22:33:44:55  switch-1  GigabitEthernet1/0/1  Bridge  10.0.0.50',
      },
      {
        desc: 'Watch neighbors in real time',
        command: 'niac neighbors watch',
      },
    ],
    tips: [
      'TLV ordering matters for some old vendors — NIAC sticks to the IEEE-recommended order.',
      'Chassis ID is usually the device MAC; matching them keeps tooling happy.',
    ],
    seeAlso: ['proto-cdp', 'proto-edp', 'proto-fdp', 'cmd-neighbors'],
  },
  {
    id: 'proto-cdp',
    name: 'CDP',
    standard: 'Cisco proprietary',
    category: 'protocols',
    summary: 'Cisco Discovery Protocol — Cisco’s LLDP equivalent.',
    techDesc:
      'Emits CDP frames every 60s to 01:00:0c:cc:cc:cc with SNAP encapsulation. Includes Device ID, Address, Port ID, Capabilities, Software Version, Platform, Native VLAN, Duplex, Power Available/Draw, and Management Address TLVs.',
    laymanDesc:
      'CDP is what Cisco gear uses to talk to other Cisco gear. Enable it for any device pretending to be Cisco.',
    whenToUse: 'Simulating Cisco devices for a CDP-aware discovery tool.',
    whenNotToUse: 'Non-Cisco shop — LLDP is the better choice.',
    parameters: [],
    configFields: [
      {
        path: 'devices[].cdp.enabled',
        type: 'boolean',
        defaultValue: 'false',
        required: true,
        description: 'Enable CDP transmitter.',
        example: 'cdp:\n  enabled: true',
      },
      {
        path: 'devices[].cdp.device_id',
        type: 'string',
        defaultValue: 'device name',
        required: false,
        description: 'Device ID TLV (DNS-style name).',
        example: 'device_id: "core-router-01"',
      },
      {
        path: 'devices[].cdp.platform',
        type: 'string',
        defaultValue: '""',
        required: false,
        description: 'Platform string.',
        example: 'platform: "Cisco ISR4331"',
      },
      {
        path: 'devices[].cdp.software_version',
        type: 'string',
        defaultValue: '""',
        required: false,
        description: 'Software version (IOS-style banner).',
        example: 'software_version: "IOS XE 17.3.5"',
      },
      {
        path: 'devices[].cdp.capabilities',
        type: 'string[]',
        defaultValue: '[]',
        required: false,
        description: 'Capabilities: Router, Switch, IGMP, Host, Repeater, Bridge.',
        example: 'capabilities: ["Router", "Switch"]',
      },
    ],
    metrics: [],
    examples: [
      {
        desc: 'On a real Cisco switch nearby',
        command: 'show cdp neighbors detail',
      },
    ],
    tips: [
      'CDP and LLDP can coexist on the same device — enable both for max compatibility.',
      'Power-draw fields matter only when the upstream device is PoE-aware.',
    ],
    seeAlso: ['proto-lldp', 'proto-edp', 'proto-fdp'],
  },
  {
    id: 'proto-edp',
    name: 'EDP',
    standard: 'Extreme Networks proprietary',
    category: 'protocols',
    summary: 'Extreme Discovery Protocol.',
    techDesc:
      'Extreme’s pre-LLDP discovery protocol. Emits at 60s intervals with TLVs for VLAN, MAC, slot/port, version.',
    laymanDesc: 'Like CDP, but for Extreme Networks switches.',
    whenToUse: 'Simulating Extreme Networks devices.',
    whenNotToUse: 'Non-Extreme shop.',
    parameters: [],
    configFields: [
      {
        path: 'devices[].edp.enabled',
        type: 'boolean',
        defaultValue: 'false',
        required: true,
        description: 'Enable EDP.',
        example: 'edp:\n  enabled: true',
      },
    ],
    metrics: [],
    examples: [
      {
        desc: 'Extreme-flavored device',
        command: '# devices[].edp.enabled: true; vendor: extreme',
      },
    ],
    tips: ['EDP and LLDP coexist; if simulating modern Extreme gear use both.'],
    seeAlso: ['proto-lldp', 'proto-cdp', 'proto-fdp'],
  },
  {
    id: 'proto-fdp',
    name: 'FDP',
    standard: 'Foundry/Brocade/Ruckus proprietary',
    category: 'protocols',
    summary: 'Foundry Discovery Protocol.',
    techDesc:
      'Foundry’s discovery protocol — wire-similar to CDP but with Foundry-specific OUI and TLV codes. Used by old Brocade ICX and Ruckus switches.',
    laymanDesc: 'Like CDP but for Foundry / Brocade / Ruckus gear.',
    whenToUse: 'Foundry/Brocade/Ruckus simulation.',
    whenNotToUse: 'Other vendors.',
    parameters: [],
    configFields: [
      {
        path: 'devices[].fdp.enabled',
        type: 'boolean',
        defaultValue: 'false',
        required: true,
        description: 'Enable FDP.',
        example: 'fdp:\n  enabled: true',
      },
    ],
    metrics: [],
    examples: [
      {
        desc: 'Brocade ICX-flavored device',
        command: '# devices[].fdp.enabled: true; cdp.enabled: false',
      },
    ],
    tips: ['Modern Ruckus and Brocade gear runs LLDP too; turn that on for max compatibility.'],
    seeAlso: ['proto-lldp', 'proto-cdp', 'proto-edp'],
  },
  {
    id: 'proto-stp',
    name: 'STP / RSTP / MSTP',
    standard: 'IEEE 802.1D / 802.1w / 802.1s',
    category: 'protocols',
    summary: 'Spanning Tree Protocol — loop prevention BPDUs.',
    techDesc:
      'Emits BPDUs every hello-time (2s by default) to 01:80:c2:00:00:00. Supports classic STP (1D), RSTP (1w), and MSTP (1s) variants. Bridge priority + bridge ID drive root election; per-port roles (root, designated, alternate, backup) are derived.',
    laymanDesc:
      'STP is what switches use to keep loops out of the network. NIAC can pretend to participate in STP so an STP-aware monitor sees the simulated switch.',
    whenToUse: 'Simulating switches in an STP-aware lab.',
    whenNotToUse:
      'On endpoints — endpoints don’t speak STP. On a real prod LAN where you don’t want NIAC influencing the topology.',
    parameters: [],
    configFields: [
      {
        path: 'devices[].stp.enabled',
        type: 'boolean',
        defaultValue: 'false',
        required: true,
        description: 'Enable STP.',
        example: 'stp:\n  enabled: true',
      },
      {
        path: 'devices[].stp.mode',
        type: 'string',
        defaultValue: '"rstp"',
        required: false,
        description: 'stp | rstp | mstp.',
        example: 'mode: rstp',
      },
      {
        path: 'devices[].stp.bridge_priority',
        type: 'integer',
        defaultValue: '32768',
        required: false,
        description: 'Bridge priority. Lower wins root election. Must be a multiple of 4096.',
        example: 'bridge_priority: 4096',
      },
    ],
    metrics: [],
    examples: [
      {
        desc: 'Make this device root bridge',
        command: '# devices[].stp.bridge_priority: 0',
      },
    ],
    tips: [
      'BPDUs use a specific multicast destination — if the upstream switch has BPDU-guard, it will err-disable the port.',
      'Use sub-second hellos only when targeting RSTP-aware monitoring.',
    ],
    seeAlso: ['device-switch'],
  },
  {
    id: 'proto-snmpv1',
    name: 'SNMPv1',
    standard: 'RFC 1157',
    category: 'protocols',
    summary: 'Original SNMP — community-string auth, GET/GETNEXT/SET/TRAP.',
    techDesc:
      'Handles SNMPv1 PDUs (GetRequest, GetNextRequest, GetResponse, SetRequest, Trap). Listens on UDP/161; traps go to UDP/162. v1 uses 32-bit counters and lacks Inform.',
    laymanDesc: 'The oldest version of SNMP. Some legacy gear still uses it.',
    whenToUse: 'Testing old NMS tooling that hard-codes v1.',
    whenNotToUse: 'Modern stacks — prefer v2c or v3.',
    parameters: [],
    configFields: [
      {
        path: 'devices[].snmp.version',
        type: 'string',
        defaultValue: '"v2c"',
        required: false,
        description: 'v1 | v2c | v3.',
        example: 'version: v1',
      },
    ],
    metrics: [
      {
        name: 'snmp_get_received',
        unit: 'count',
        goodRange: '> 0 if polled',
        badMeaning: 'No one polling.',
      },
    ],
    examples: [
      {
        desc: 'snmpget over v1',
        command: 'snmpget -v 1 -c public 10.0.0.1 1.3.6.1.2.1.1.1.0',
      },
    ],
    tips: ['v1 traps are unauthenticated and lossy — use only for tests that explicitly need v1.'],
    seeAlso: ['proto-snmpv2c', 'proto-snmpv3'],
  },
  {
    id: 'proto-snmpv2c',
    name: 'SNMPv2c',
    standard: 'RFC 3416 / 3418',
    category: 'protocols',
    summary: 'Community-string SNMP with GetBulk, Inform, 64-bit counters.',
    techDesc:
      'Handles v2c PDUs (GetRequest, GetNextRequest, GetBulkRequest, GetResponse, SetRequest, Trap, Inform). Walk-file mode: a single text file of `OID = TYPE: VALUE` lines is loaded at startup, and GET/GETNEXT/GETBULK walk it.',
    laymanDesc:
      'The most common SNMP version. Uses a shared community string ("public" is the default but insecure).',
    whenToUse: 'Almost always. Default SNMP version in NIAC.',
    whenNotToUse: 'You need authenticated / encrypted access — use v3.',
    parameters: [],
    configFields: [
      {
        path: 'devices[].snmp.enabled',
        type: 'boolean',
        defaultValue: 'false',
        required: true,
        description: 'Enable SNMP agent.',
        example: 'snmp:\n  enabled: true',
      },
      {
        path: 'devices[].snmp.community',
        type: 'string',
        defaultValue: '"public"',
        required: false,
        description: 'Read-only community string.',
        example: 'community: "public"',
      },
      {
        path: 'devices[].snmp.walk_file',
        type: 'path',
        defaultValue: '""',
        required: false,
        description: 'Path to an snmpwalk-format text file driving GET/GETNEXT.',
        example: 'walk_file: "examples/device_walks/cisco-3850.walk"',
      },
      {
        path: 'devices[].snmp.sys_descr',
        type: 'string',
        defaultValue: '""',
        required: false,
        description: 'sysDescr.0 value.',
        example: 'sys_descr: "Cisco IOS XE Software"',
      },
      {
        path: 'devices[].snmp.traps.enabled',
        type: 'boolean',
        defaultValue: 'false',
        required: false,
        description: 'Emit traps to configured receivers.',
        example: 'traps:\n  enabled: true',
      },
      {
        path: 'devices[].snmp.traps.receivers',
        type: 'string[]',
        defaultValue: '[]',
        required: false,
        description: 'Trap destinations (host:port).',
        example: 'receivers: ["10.0.0.100:162"]',
      },
    ],
    metrics: [],
    examples: [
      {
        desc: 'Walk the device',
        command: 'snmpwalk -v 2c -c public 10.0.0.1',
      },
      {
        desc: 'Polled by a real NMS',
        command: 'snmpget -v 2c -c public 10.0.0.1 sysDescr.0',
        output: 'SNMPv2-MIB::sysDescr.0 = STRING: Cisco IOS XE Software',
      },
    ],
    tips: [
      'Walk files from real devices give the most authentic responses — see `examples/device_walks/`.',
      'Use distinct community strings per device when simulating segregated environments.',
    ],
    seeAlso: ['proto-snmpv3', 'analysis-walk', 'cmd-analyze-walk'],
  },
  {
    id: 'proto-snmpv3',
    name: 'SNMPv3',
    standard: 'RFC 3414 / 3826',
    category: 'protocols',
    summary: 'SNMP with authentication and encryption.',
    techDesc:
      'Handles SNMPv3 USM (User-Based Security Model) with MD5/SHA/SHA-224/SHA-256/SHA-384/SHA-512 auth and DES/3DES/AES-128/AES-192/AES-256 priv. Configure one or more users with their auth and priv passphrases. Engine ID is auto-derived per device.',
    laymanDesc: 'The secure version of SNMP — with usernames, passwords, and optional encryption.',
    whenToUse: 'Production-style testing where v2c’s plaintext community is unacceptable.',
    whenNotToUse: 'You just want to scrape a few OIDs quickly — v2c is simpler.',
    parameters: [],
    configFields: [
      {
        path: 'devices[].snmp.v3_users',
        type: 'object[]',
        defaultValue: '[]',
        required: false,
        description: 'List of v3 users.',
        example:
          'v3_users:\n  - name: admin\n    auth_protocol: SHA\n    auth_passphrase: "S3cret!"\n    priv_protocol: AES128\n    priv_passphrase: "Priv4te!"',
      },
    ],
    metrics: [],
    examples: [
      {
        desc: 'v3 walk with auth+priv',
        command: 'snmpwalk -v 3 -u admin -a SHA -A S3cret! -x AES -X Priv4te! -l authPriv 10.0.0.1',
      },
    ],
    tips: [
      'Passphrases must be 8+ chars — v3 spec enforces it.',
      'NIAC accepts authNoPriv too — leave priv_protocol empty.',
    ],
    seeAlso: ['proto-snmpv2c'],
  },
  {
    id: 'proto-http',
    name: 'HTTP',
    standard: 'RFC 9110',
    category: 'protocols',
    summary: 'Tiny HTTP/1.1 responder with configurable endpoints.',
    techDesc:
      'Listens on TCP/80 (configurable). Returns canned content per path. Supports custom Server header, response status, and per-endpoint headers. No HTTPS — TLS termination is out of scope.',
    laymanDesc:
      'Makes the device act like a tiny web server. Give it paths and content — anything else returns 404.',
    whenToUse: 'Simulating IoT device UIs, server homepage banners, REST API stubs.',
    whenNotToUse: 'You need real auth, sessions, or HTTPS.',
    parameters: [],
    configFields: [
      {
        path: 'devices[].http.enabled',
        type: 'boolean',
        defaultValue: 'false',
        required: true,
        description: 'Enable HTTP server.',
        example: 'http:\n  enabled: true',
      },
      {
        path: 'devices[].http.port',
        type: 'integer',
        defaultValue: '80',
        required: false,
        description: 'TCP port to listen on.',
        example: 'port: 8080',
      },
      {
        path: 'devices[].http.server_header',
        type: 'string',
        defaultValue: '"NIAC/1.0"',
        required: false,
        description: 'Server: header value.',
        example: 'server_header: "Apache/2.4.41"',
      },
      {
        path: 'devices[].http.endpoints',
        type: 'object[]',
        defaultValue: '[]',
        required: false,
        description: 'Endpoint list: path + content + headers + status.',
        example:
          'endpoints:\n  - {path: "/", content: "<h1>Hi</h1>"}\n  - {path: "/api", content: \'{"ok":true}\', headers: {Content-Type: application/json}}',
      },
    ],
    metrics: [
      {
        name: 'http_requests_received',
        unit: 'count',
        goodRange: '> 0 if probed',
        badMeaning: 'Nothing reaching the responder.',
      },
    ],
    examples: [
      {
        desc: 'Curl the device',
        command: 'curl http://10.0.0.10/',
      },
      {
        desc: 'Mimic an nginx server',
        command: '# http.server_header: "nginx/1.24.0"',
      },
    ],
    tips: [
      'NIAC’s HTTP is intentionally minimal — if you need a real server, use nginx and put it behind NIAC’s other handlers.',
    ],
    seeAlso: ['proto-tcp', 'proto-ftp'],
  },
  {
    id: 'proto-tcp',
    name: 'TCP',
    standard: 'RFC 9293',
    category: 'protocols',
    summary: 'TCP stack underlying HTTP/FTP/iperf3/SSH-banner.',
    techDesc:
      'NIAC implements enough TCP (SYN/SYN-ACK/ACK, data, FIN, RST) to host its application-layer responders. Window scaling, SACK, and ECN are handled. Multiple simultaneous connections per device are supported.',
    laymanDesc:
      'TCP is the transport under HTTP and friends. It’s on automatically whenever you enable HTTP, FTP, iperf3, etc.',
    whenToUse: 'Implicit for any TCP-based handler.',
    whenNotToUse: 'N/A.',
    parameters: [],
    configFields: [],
    metrics: [],
    examples: [
      {
        desc: 'TCP connect test',
        command: 'nc -v 10.0.0.10 80',
      },
    ],
    tips: ['If a TCP handshake fails, suspect ARP / IP / firewall first — not NIAC’s TCP.'],
    seeAlso: ['proto-udp', 'proto-http'],
  },
  {
    id: 'proto-udp',
    name: 'UDP',
    standard: 'RFC 768',
    category: 'protocols',
    summary: 'UDP transport for DHCP/DNS/SNMP/NetBIOS.',
    techDesc:
      'Implicit transport for all UDP-based handlers. NIAC validates UDP checksums on receive and computes them on send.',
    laymanDesc: 'Connectionless transport. Implicit for many handlers.',
    whenToUse: 'Implicit.',
    whenNotToUse: 'N/A.',
    parameters: [],
    configFields: [],
    metrics: [],
    examples: [
      {
        desc: 'UDP probe',
        command: 'nc -u -v 10.0.0.10 53',
      },
    ],
    tips: [],
    seeAlso: ['proto-tcp'],
  },
  {
    id: 'proto-iperf3',
    name: 'iperf3',
    standard: 'iperf3 protocol',
    category: 'protocols',
    summary: 'iperf3 server emulation for throughput probes.',
    techDesc:
      'Listens on TCP/5201 (configurable) and speaks enough iperf3 control-channel JSON to accept connections, run a timed transfer, and return summary stats. Useful for tools that probe bandwidth as part of inventory.',
    laymanDesc: 'Makes the device respond to iperf3 throughput tests.',
    whenToUse: 'Tools that auto-probe iperf3 servers for capacity reporting.',
    whenNotToUse: 'Real throughput measurement — NIAC’s iperf3 emulation reports synthetic values.',
    parameters: [],
    configFields: [
      {
        path: 'devices[].iperf3.enabled',
        type: 'boolean',
        defaultValue: 'false',
        required: true,
        description: 'Enable iperf3 server.',
        example: 'iperf3:\n  enabled: true',
      },
    ],
    metrics: [],
    examples: [
      {
        desc: 'Run iperf3 client against NIAC',
        command: 'iperf3 -c 10.0.0.10',
      },
    ],
    tips: ['NIAC’s reported throughput is configurable per device but isn’t a real measurement.'],
    seeAlso: ['proto-tcp'],
  },
  {
    id: 'proto-netbios',
    name: 'NetBIOS',
    standard: 'RFC 1001 / 1002',
    category: 'protocols',
    summary: 'NetBIOS name service + datagram.',
    techDesc:
      'Listens on UDP/137 (NetBIOS Name Service) and UDP/138 (Datagram). Answers NBSTAT queries with configured names and suffixes (00 = workstation, 03 = messenger, 20 = file server). No SMB — that’s out of scope.',
    laymanDesc:
      'Old-school Windows name resolution. Enable it for Windows-flavored hosts you want browsable on the LAN.',
    whenToUse:
      'Simulating Windows endpoints / file servers for inventory tools that still rely on NBSTAT.',
    whenNotToUse: 'Modern AD environments — prefer DNS.',
    parameters: [],
    configFields: [
      {
        path: 'devices[].netbios.enabled',
        type: 'boolean',
        defaultValue: 'false',
        required: true,
        description: 'Enable NetBIOS responder.',
        example: 'netbios:\n  enabled: true',
      },
      {
        path: 'devices[].netbios.name',
        type: 'string',
        defaultValue: '""',
        required: false,
        description: 'Computer name (uppercase, ≤ 15 chars).',
        example: 'name: "WORKSTATION-01"',
      },
    ],
    metrics: [],
    examples: [
      {
        desc: 'Query NetBIOS name',
        command: 'nmblookup -A 10.0.0.50',
      },
    ],
    tips: ['NetBIOS names truncate at 15 chars — keep them short.'],
    seeAlso: ['proto-dns'],
  },
  {
    id: 'proto-ftp',
    name: 'FTP',
    standard: 'RFC 959',
    category: 'protocols',
    summary: 'FTP control-channel banner + minimal command set.',
    techDesc:
      'Listens on TCP/21. Greets clients with a configurable banner and accepts USER / PASS / SYST / FEAT / QUIT — enough to satisfy banner-grabbing scanners. Data connections aren’t supported.',
    laymanDesc:
      'Makes the device greet anything connecting to TCP/21 with a configurable FTP banner. Useful for tools that fingerprint by banner.',
    whenToUse: 'Banner-grabbing inventory tools, vulnerability scanner test fixtures.',
    whenNotToUse: 'Real file transfer — NIAC’s FTP is a stub.',
    parameters: [],
    configFields: [
      {
        path: 'devices[].ftp.enabled',
        type: 'boolean',
        defaultValue: 'false',
        required: true,
        description: 'Enable FTP banner responder.',
        example: 'ftp:\n  enabled: true',
      },
      {
        path: 'devices[].ftp.banner',
        type: 'string',
        defaultValue: '"220 NIAC FTP ready"',
        required: false,
        description: '220-greeting banner.',
        example: 'banner: "220 (vsFTPd 3.0.3)"',
      },
    ],
    metrics: [],
    examples: [
      {
        desc: 'Grab the banner',
        command: 'nc -v 10.0.0.10 21',
        output: '220 (vsFTPd 3.0.3)',
      },
    ],
    tips: [
      'Match a real banner string (vsftpd, ProFTPD, IIS FTP) to trigger correct fingerprinting.',
    ],
    seeAlso: ['proto-http'],
  },
];

// ---------------------------------------------------------------------------
// Topology
// ---------------------------------------------------------------------------

const topologyItems: HelpItem[] = [
  {
    id: 'topo-vlan',
    name: 'VLAN Tagging',
    standard: 'IEEE 802.1Q',
    category: 'topology',
    summary: 'Per-device 802.1Q VLAN tag.',
    techDesc:
      'Each device may declare a `vlan` field. NIAC adds an 802.1Q tag (TPID 0x8100) to all egress frames and accepts ingress frames carrying that tag. Multiple devices on the same NIAC instance can sit on different VLANs over a trunked host interface.',
    laymanDesc:
      'Lets you put different simulated devices on different VLANs over the same physical port.',
    whenToUse:
      'Simulating a VLAN’d network. The host interface must be configured as a trunk on the upstream switch.',
    whenNotToUse: 'Single flat network — just leave the VLAN field unset.',
    parameters: [],
    configFields: [
      {
        path: 'devices[].vlan',
        type: 'integer (1-4094)',
        defaultValue: '0',
        required: false,
        description: 'IEEE 802.1Q VLAN tag.',
        example: 'vlan: 100',
      },
    ],
    metrics: [],
    examples: [
      {
        desc: 'Three devices, three VLANs',
        command:
          '# devices: [\n#   {name: a, vlan: 10, …},\n#   {name: b, vlan: 20, …},\n#   {name: c, vlan: 30, …}\n# ]',
      },
    ],
    tips: [
      'Upstream port must be a trunk allowing every VLAN you use.',
      'Native VLAN traffic stays untagged — omit the field or set vlan: 0.',
    ],
    seeAlso: ['topo-trunk', 'device-switch'],
  },
  {
    id: 'topo-trunk',
    name: 'Trunk Ports',
    standard: 'IEEE 802.1Q',
    category: 'topology',
    summary: 'Multiple VLANs over a single host interface.',
    techDesc:
      'NIAC treats the host interface as an 802.1Q trunk implicitly when any device has a `vlan` field. Native VLAN (untagged) and tagged VLANs interleave naturally on the wire.',
    laymanDesc: 'Lets one interface carry many VLANs — exactly like a real switch uplink.',
    whenToUse: 'Multi-VLAN configs.',
    whenNotToUse: 'Single VLAN.',
    parameters: [],
    configFields: [],
    metrics: [],
    examples: [
      {
        desc: 'Configure the host port as a trunk',
        command:
          '# On the upstream Cisco switch:\n# switchport mode trunk\n# switchport trunk allowed vlan 10,20,30',
      },
    ],
    tips: [
      'If the upstream port is in access mode, all tagged traffic is dropped — fix the switch config first.',
    ],
    seeAlso: ['topo-vlan'],
  },
  {
    id: 'topo-portchannel',
    name: 'Port-Channel (LACP)',
    standard: 'IEEE 802.3ad / 802.1AX',
    category: 'topology',
    summary: 'LACPDU emission for bonded uplink simulation.',
    techDesc:
      'NIAC can emit LACPDUs on configured devices, advertising itself as part of a port-channel. Useful when a real upstream switch expects to see LACP partners on its bonded ports.',
    laymanDesc: 'Makes NIAC participate in a Link Aggregation Group (LAG) with an upstream switch.',
    whenToUse: 'Validating that an upstream switch accepts NIAC as an LACP partner.',
    whenNotToUse: 'Single-link setups.',
    parameters: [],
    configFields: [
      {
        path: 'devices[].lacp.enabled',
        type: 'boolean',
        defaultValue: 'false',
        required: true,
        description: 'Enable LACPDU emission.',
        example: 'lacp:\n  enabled: true',
      },
      {
        path: 'devices[].lacp.system_id',
        type: 'string (MAC)',
        defaultValue: 'device MAC',
        required: false,
        description: 'LACP system ID.',
        example: 'system_id: "00:1A:2B:3C:4D:01"',
      },
    ],
    metrics: [],
    examples: [
      {
        desc: 'LACP-active device',
        command: '# devices[].lacp: {enabled: true, mode: active}',
      },
    ],
    tips: ['LACP needs the host interface in bond / team mode — OS-level config is on you.'],
    seeAlso: ['topo-trunk'],
  },
  {
    id: 'topo-graph',
    name: 'Topology View',
    standard: 'NIAC web UI',
    category: 'topology',
    summary: 'Derives a graph from LLDP/CDP neighbor entries.',
    techDesc:
      'The /topology page in the web UI builds a node-link graph: simulated devices are nodes, and edges are inferred from LLDP/CDP neighbor entries (both received and emitted). The graph updates over SSE as new neighbors are discovered.',
    laymanDesc: 'A live network diagram of the simulated topology.',
    whenToUse: 'Visualizing what the simulation looks like to a discovery tool.',
    whenNotToUse: 'You only run a single device with no neighbors.',
    parameters: [],
    configFields: [],
    metrics: [],
    examples: [
      {
        desc: 'Open the topology view',
        command: '# open http://localhost:8080/topology in a browser',
      },
    ],
    tips: [
      'Use vendor-distinct OUIs and capability TLVs to make the graph colorful and informative.',
    ],
    seeAlso: ['proto-lldp', 'proto-cdp', 'cmd-neighbors'],
  },
];

// ---------------------------------------------------------------------------
// Commands
// ---------------------------------------------------------------------------

const commandItems: HelpItem[] = [
  {
    id: 'cmd-daemon',
    name: 'niac daemon',
    standard: 'niac CLI',
    category: 'commands',
    summary: 'Run NIAC as a long-lived background process with web UI.',
    techDesc:
      'Starts the simulation runtime and the embedded web UI. The daemon owns the host interface, opens raw sockets, and exposes a REST + SSE API on `--api-listen`. Configuration is hot-reloadable via the web UI; the daemon itself doesn’t consume a YAML file at startup — it loads them on demand from the API.',
    laymanDesc:
      'The "always on" mode. Start it once, then drive everything (start/stop simulations, edit configs, inject errors) from a browser.',
    whenToUse: 'Lab desktops, persistent test rigs, anything you want to leave running.',
    whenNotToUse: 'One-shot scripted scenarios — use `niac run`.',
    parameters: [
      {
        name: 'API listen',
        flag: '--api-listen',
        type: 'string (host:port)',
        defaultValue: ':8080',
        required: false,
        techDesc: 'Address for the REST API and web UI.',
        laymanDesc: 'Where the daemon listens for browser / API connections.',
        example: '--api-listen :8080',
      },
      {
        name: 'API token',
        flag: 'NIAC_API_TOKEN env',
        type: 'string',
        defaultValue: '',
        required: false,
        techDesc:
          'Bearer token required for API access. Prefer env var; the deprecated `--api-token` flag leaks via `ps`.',
        laymanDesc: 'A shared secret that protects the API from random LAN visitors.',
        example: 'NIAC_API_TOKEN=$(openssl rand -hex 32) niac daemon',
      },
      {
        name: 'Storage path',
        flag: '--storage-path',
        type: 'path',
        defaultValue: '~/.niac/niac.db',
        required: false,
        techDesc: 'SQLite DB where run history is persisted.',
        laymanDesc: 'Where the daemon stores history between restarts.',
        example: '--storage-path /var/lib/niac/niac.db',
      },
    ],
    configFields: [],
    metrics: [],
    examples: [
      {
        desc: 'Local daemon on default port',
        command: 'sudo NIAC_API_TOKEN=secret niac daemon',
      },
      {
        desc: 'Run as a systemd service',
        command: 'sudo systemctl start niac && journalctl -fu niac',
      },
    ],
    tips: [
      'Always set NIAC_API_TOKEN — the daemon will run without it but the API is then unauthenticated.',
      'Use `niac dump` to grab a pcap from a running daemon without restarting.',
    ],
    seeAlso: ['cmd-run', 'cmd-monitor', 'cmd-status'],
  },
  {
    id: 'cmd-run',
    name: 'niac run',
    standard: 'niac CLI',
    category: 'commands',
    summary: 'Run a single simulation from a YAML config.',
    techDesc:
      'Binds to `<interface>`, loads `<config-file>`, starts all configured device handlers, and runs until Ctrl-C. No web UI — use the daemon for that. Equivalent to passing the args positionally in legacy mode.',
    laymanDesc: 'One config, one interface, until you stop it. Like a foreground script.',
    whenToUse: 'CI jobs, quick "does this config even work" checks.',
    whenNotToUse: 'You want the web UI — use `niac daemon`.',
    parameters: [
      {
        name: 'interface',
        flag: '<positional 1>',
        type: 'string',
        defaultValue: '',
        required: true,
        techDesc: 'Host interface NIAC binds to.',
        laymanDesc: 'The network port the simulation runs on.',
        example: 'en0',
      },
      {
        name: 'config-file',
        flag: '<positional 2>',
        type: 'path',
        defaultValue: '',
        required: true,
        techDesc: 'Path to a YAML config.',
        laymanDesc: 'The YAML file describing the simulated network.',
        example: 'examples/complete-kitchen-sink.yaml',
      },
      {
        name: 'dry-run',
        flag: '--dry-run',
        type: 'bool',
        defaultValue: 'false',
        required: false,
        techDesc: 'Validate the config and exit without binding sockets.',
        laymanDesc: 'Check the file without actually starting the sim.',
        example: 'niac --dry-run lo0 my.yaml',
      },
    ],
    configFields: [],
    metrics: [],
    examples: [
      {
        desc: 'Run a router config on en0',
        command: 'sudo niac run en0 examples/vendors/cisco-network.yaml',
      },
      {
        desc: 'Validate without binding',
        command: 'niac run --dry-run lo0 my.yaml',
      },
    ],
    tips: [
      'Use `lo0` (loopback) for any test that doesn’t need to be seen by other machines — no root needed on macOS.',
      'Ctrl-C cleans up sockets and ARP-table entries before exiting.',
    ],
    seeAlso: ['cmd-daemon', 'cmd-interactive', 'cmd-validate'],
  },
  {
    id: 'cmd-interactive',
    name: 'niac interactive',
    standard: 'niac CLI',
    category: 'commands',
    summary: 'Run a simulation with a terminal UI (TUI).',
    techDesc:
      'Like `niac run` but with a Bubble Tea TUI: counters per protocol, recent log lines, neighbor table, error-injection panel. Same args as `niac run`.',
    laymanDesc: 'Same as `run`, but with a fancy terminal dashboard instead of plain stdout.',
    whenToUse: 'Live demos, debug sessions, anything you want to watch in a terminal.',
    whenNotToUse: 'Headless CI — use `niac run`.',
    parameters: [],
    configFields: [],
    metrics: [],
    examples: [
      {
        desc: 'Interactive demo',
        command: 'sudo niac interactive en0 examples/complete-kitchen-sink.yaml',
      },
    ],
    tips: [
      'Press `?` inside the TUI for a key-binding overlay.',
      'Use a wide terminal (≥ 120 cols) for the full layout.',
    ],
    seeAlso: ['cmd-run', 'cmd-monitor'],
  },
  {
    id: 'cmd-init',
    name: 'niac init',
    standard: 'niac CLI',
    category: 'commands',
    summary: 'Interactive wizard — picks a template and customizes it.',
    techDesc:
      'Walks the user through template selection (router / switch / ap / iot / server), persona naming, IP assignment, and writes a starting YAML. Faster than `generate` for "I just need something to run".',
    laymanDesc: 'A friendly question-and-answer wizard that produces a working config file.',
    whenToUse: 'First-time users, quick demos.',
    whenNotToUse: 'You already know exactly what you want — hand-write YAML.',
    parameters: [
      {
        name: 'output-file',
        flag: '<positional>',
        type: 'path',
        defaultValue: 'niac.yaml',
        required: false,
        techDesc: 'Where the generated config is written.',
        laymanDesc: 'Output file name.',
        example: 'niac init my-net.yaml',
      },
    ],
    configFields: [],
    metrics: [],
    examples: [
      {
        desc: 'First-time setup',
        command: 'niac init',
      },
      {
        desc: 'Init into a named file',
        command: 'niac init lab-network.yaml',
      },
    ],
    tips: ['You can always run it again — it doesn’t overwrite without prompting.'],
    seeAlso: ['cmd-generate', 'cmd-template', 'cmd-validate'],
  },
  {
    id: 'cmd-generate',
    name: 'niac config generate',
    standard: 'niac CLI',
    category: 'commands',
    summary: 'More detailed interactive generator.',
    techDesc:
      'Subcommand under `niac config`. Asks more questions than `init`: per-device protocol toggles, SNMP community strings, DHCP pools, DNS records. Produces a fully populated YAML.',
    laymanDesc: 'A longer wizard that builds a more complete config.',
    whenToUse: 'You want every protocol option dialed in via prompts.',
    whenNotToUse: 'Quick "give me a router" — use `init`.',
    parameters: [],
    configFields: [],
    metrics: [],
    examples: [
      {
        desc: 'Detailed wizard',
        command: 'niac config generate detailed.yaml',
      },
    ],
    tips: ['Pipe answers from a file for repeatable generation in CI.'],
    seeAlso: ['cmd-init', 'cmd-template'],
  },
  {
    id: 'cmd-validate',
    name: 'niac validate',
    standard: 'niac CLI',
    category: 'commands',
    summary: 'Static validation of a YAML config.',
    techDesc:
      'Parses the config, checks the schema, validates MAC/IP formats, looks for VLAN conflicts, checks that referenced walk files exist, and reports warnings + errors. Exit code 0 = clean, 1 = errors, 2 = warnings (with --strict).',
    laymanDesc: 'Catches typos and missing files before you run the simulation.',
    whenToUse: 'Pre-commit hook, CI pipeline, anywhere you commit YAML.',
    whenNotToUse: 'Never — always validate.',
    parameters: [
      {
        name: 'strict',
        flag: '--strict',
        type: 'bool',
        defaultValue: 'false',
        required: false,
        techDesc: 'Treat warnings as errors.',
        laymanDesc: 'Fail the run on any warning, not just errors.',
        example: 'niac validate --strict my.yaml',
      },
    ],
    configFields: [],
    metrics: [],
    examples: [
      {
        desc: 'Basic validation',
        command: 'niac validate examples/complete-kitchen-sink.yaml',
        output: 'OK: 5 device(s), 17 protocol handler(s), 0 warnings',
      },
      {
        desc: 'CI gate — warnings = failure',
        command: 'niac validate --strict configs/*.yaml',
      },
    ],
    tips: ['Pair with `niac config diff` to catch regressions when editing a config.'],
    seeAlso: ['cmd-run', 'cmd-init'],
  },
  {
    id: 'cmd-template',
    name: 'niac template',
    standard: 'niac CLI',
    category: 'commands',
    summary: 'Manage built-in YAML templates.',
    techDesc:
      'Subcommands: `template list` (show available templates), `template use <name> <output>` (copy a template into the cwd), `template show <name>` (cat to stdout).',
    laymanDesc: 'Browse the built-in starter configs and copy one into your project.',
    whenToUse: 'Spinning up a new simulation.',
    whenNotToUse: 'You already have a config.',
    parameters: [],
    configFields: [],
    metrics: [],
    examples: [
      {
        desc: 'List templates',
        command: 'niac template list',
        output: 'router  switch  ap  iot  server  minimal  complete',
      },
      {
        desc: 'Copy router template locally',
        command: 'niac template use router router.yaml',
      },
    ],
    tips: ['Templates are versioned with the binary — update NIAC, update the starting configs.'],
    seeAlso: ['cmd-init'],
  },
  {
    id: 'cmd-status',
    name: 'niac status',
    standard: 'niac CLI',
    category: 'commands',
    summary: 'Show running-daemon state.',
    techDesc:
      'Connects to the daemon’s IPC socket and prints: uptime, active interface, loaded config name, device count, active error injections, total packets RX/TX, per-protocol counters.',
    laymanDesc: 'Quick "is it running and what’s it doing?" check.',
    whenToUse: 'Sanity-check from a shell script.',
    whenNotToUse: 'You’re already in the web UI.',
    parameters: [
      {
        name: 'JSON output',
        flag: '--json',
        type: 'bool',
        defaultValue: 'false',
        required: false,
        techDesc: 'Emit machine-readable JSON instead of a table.',
        laymanDesc: 'Get JSON instead of a pretty table.',
        example: 'niac status --json | jq .uptime',
      },
    ],
    configFields: [],
    metrics: [],
    examples: [
      {
        desc: 'Default status',
        command: 'niac status',
      },
      {
        desc: 'JSON status',
        command: 'niac status --json',
      },
    ],
    tips: ['Pair with `niac monitor` for a live feed.'],
    seeAlso: ['cmd-monitor', 'cmd-daemon'],
  },
  {
    id: 'cmd-monitor',
    name: 'niac monitor',
    standard: 'niac CLI',
    category: 'commands',
    summary: 'Stream live counters from a running daemon.',
    techDesc:
      'Subscribes over IPC and prints stats every `--interval` (default 1s) in table / json / csv formats. Like `top` but for the NIAC simulation.',
    laymanDesc: 'Watch the simulation’s packet counters live.',
    whenToUse: 'Tailing a running daemon, scripting metric collection.',
    whenNotToUse: 'You want long-term metrics — use Prometheus via `--metrics-listen`.',
    parameters: [
      {
        name: 'Format',
        flag: '--format',
        type: 'string',
        defaultValue: 'table',
        required: false,
        techDesc: 'table | json | csv.',
        laymanDesc: 'How the data is printed.',
        example: '--format json',
      },
      {
        name: 'Interval',
        flag: '--interval',
        type: 'duration',
        defaultValue: '1s',
        required: false,
        techDesc: 'Refresh interval.',
        laymanDesc: 'How often to print.',
        example: '--interval 5s',
      },
    ],
    configFields: [],
    metrics: [],
    examples: [
      {
        desc: 'Default monitor',
        command: 'niac monitor',
      },
      {
        desc: 'JSON for jq pipelines',
        command: 'niac monitor --format json | jq -r .packets_rx',
      },
      {
        desc: 'CSV every 5s',
        command: 'niac monitor --format csv --interval 5s > metrics.csv',
      },
    ],
    tips: [
      'CSV output prints a header once, then a row per interval — great for spreadsheets.',
      'Ctrl-C exits cleanly with proper terminal restoration.',
    ],
    seeAlso: ['cmd-status'],
  },
  {
    id: 'cmd-logs',
    name: 'niac logs',
    standard: 'niac CLI',
    category: 'commands',
    summary: 'View or stream daemon logs.',
    techDesc:
      'Two modes: `niac logs` shows recent log lines; `niac logs tail` streams new lines via SSE. Supports filtering by per-protocol log level.',
    laymanDesc: 'Like `tail -f` for the simulation.',
    whenToUse: 'Debugging a misbehaving handler.',
    whenNotToUse: 'You’re already in the web UI Logs page.',
    parameters: [
      {
        name: 'Lines',
        flag: '-n',
        type: 'integer',
        defaultValue: '100',
        required: false,
        techDesc: 'Number of recent lines to show.',
        laymanDesc: 'How many old lines to print.',
        example: '-n 1000',
      },
    ],
    configFields: [],
    metrics: [],
    examples: [
      {
        desc: 'Last 100 lines',
        command: 'niac logs',
      },
      {
        desc: 'Live tail',
        command: 'niac logs tail',
      },
    ],
    tips: ['Filter SNMP-only debug noise from the web UI Logs page — it has per-protocol toggles.'],
    seeAlso: ['cmd-daemon', 'cmd-monitor'],
  },
  {
    id: 'cmd-dump',
    name: 'niac dump',
    standard: 'niac CLI',
    category: 'commands',
    summary: 'Dump captured packets from a running daemon.',
    techDesc:
      'Asks the daemon to flush its in-memory rolling capture buffer to a pcap file or stdout. Useful for grabbing the last N seconds of traffic without restarting tcpdump.',
    laymanDesc: 'Save the recent packet traffic to a pcap file you can open in Wireshark.',
    whenToUse: 'You see something weird in the logs and want a packet capture, after the fact.',
    whenNotToUse: 'You knew you’d want a capture before starting — run tcpdump in parallel.',
    parameters: [
      {
        name: 'Output',
        flag: '-o, --output',
        type: 'path',
        defaultValue: 'dump.pcap',
        required: false,
        techDesc: 'pcap output path. `-` for stdout.',
        laymanDesc: 'Where to write the capture.',
        example: '-o /tmp/recent.pcap',
      },
      {
        name: 'BPF filter',
        flag: '--filter',
        type: 'string',
        defaultValue: '',
        required: false,
        techDesc: 'Standard BPF filter expression applied during dump.',
        laymanDesc: 'Only save matching packets.',
        example: '--filter "arp or icmp"',
      },
    ],
    configFields: [],
    metrics: [],
    examples: [
      {
        desc: 'Quick dump',
        command: 'niac dump -o /tmp/r.pcap',
      },
      {
        desc: 'Filtered dump piped to tshark',
        command: 'niac dump --filter "udp and port 161" -o - | tshark -r -',
      },
    ],
    tips: [
      'Capture buffer is bounded — increase the daemon’s capture size if you need more history.',
    ],
    seeAlso: ['cmd-analyze-pcap'],
  },
  {
    id: 'cmd-inject',
    name: 'niac inject',
    standard: 'niac CLI',
    category: 'commands',
    summary: 'Inject error conditions on a simulated device.',
    techDesc:
      'Sends an injection request over the daemon IPC socket. Subcommands: bare `inject <device> <error-type> <value>` (set), `inject list` (show active), `inject clear` (remove). Value is 0-100 (percentage or count, depending on error type). 0 clears.',
    laymanDesc:
      'Tell a simulated device to start reporting an error — high CPU, FCS errors, link discards, etc. — without restarting it.',
    whenToUse: 'Validating that monitoring tooling fires alerts.',
    whenNotToUse: 'You want a static "bad" config — set the value in YAML instead.',
    parameters: [
      {
        name: 'device',
        flag: '<positional>',
        type: 'string',
        defaultValue: '',
        required: true,
        techDesc: 'Device name from the running config.',
        laymanDesc: 'Which simulated device to break.',
        example: 'router-1',
      },
      {
        name: 'error-type',
        flag: '<positional>',
        type: 'string',
        defaultValue: '',
        required: true,
        techDesc:
          'fcs_errors | packet_discards | interface_errors | high_cpu | high_memory | high_disk | high_utilization.',
        laymanDesc: 'What kind of error to inject.',
        example: 'high_cpu',
      },
      {
        name: 'value',
        flag: '<positional>',
        type: 'integer (0-100)',
        defaultValue: '',
        required: true,
        techDesc: '0 = clear; 1-100 = percentage / count.',
        laymanDesc: 'How "bad" — 0 means stop, 100 means max.',
        example: '85',
      },
    ],
    configFields: [],
    metrics: [],
    examples: [
      {
        desc: 'Simulate high CPU on switch-2',
        command: 'niac inject switch-2 high_cpu 85',
      },
      {
        desc: 'Inject 50% FCS errors',
        command: 'niac inject router-1 fcs_errors 50',
      },
      {
        desc: 'Clear all injections',
        command: 'niac inject clear --all',
      },
      {
        desc: 'List active injections',
        command: 'niac inject list',
      },
    ],
    tips: [
      'Each injection bumps the SNMP counter polled by your NMS — verify by snmpwalking the device.',
      'Combine multiple injections to model compound faults (high CPU AND high memory).',
    ],
    seeAlso: ['err-cpu', 'err-fcs', 'err-discards', 'err-iferrors'],
  },
  {
    id: 'cmd-neighbors',
    name: 'niac neighbors',
    standard: 'niac CLI',
    category: 'commands',
    summary: 'Show LLDP/CDP neighbor table.',
    techDesc:
      'Prints the daemon’s discovered-neighbor table (devices NIAC has heard from on the wire). `watch` subcommand re-prints every second.',
    laymanDesc: 'Show every other device on the wire that NIAC has learned about via LLDP or CDP.',
    whenToUse: 'Verifying NIAC sees the upstream switch correctly.',
    whenNotToUse: 'You only care about NIAC’s outgoing advertisements.',
    parameters: [],
    configFields: [],
    metrics: [],
    examples: [
      {
        desc: 'Current neighbors',
        command: 'niac neighbors',
      },
      {
        desc: 'Watch mode',
        command: 'niac neighbors watch',
      },
    ],
    tips: ['Empty table on a known-good link usually means LLDP is disabled upstream.'],
    seeAlso: ['proto-lldp', 'proto-cdp', 'topo-graph'],
  },
  {
    id: 'cmd-analyze-pcap',
    name: 'niac analyze-pcap',
    standard: 'niac CLI',
    category: 'commands',
    summary: 'Summarize a pcap by protocol.',
    techDesc:
      'Reads a pcap file (libpcap format) and emits per-protocol counts: how many ARP / ICMP / DNS / SNMP / DHCP / LLDP frames, plus per-device frame counts grouped by MAC.',
    laymanDesc:
      'Open a pcap and get a quick "what’s in here?" summary without firing up Wireshark.',
    whenToUse: 'First-pass exploration of a capture, building a NIAC config from one.',
    whenNotToUse: 'Deep packet-level debugging — use Wireshark.',
    parameters: [
      {
        name: 'pcap-file',
        flag: '<positional>',
        type: 'path',
        defaultValue: '',
        required: true,
        techDesc: 'pcap file to analyze.',
        laymanDesc: 'The capture file.',
        example: 'capture.pcap',
      },
    ],
    configFields: [],
    metrics: [],
    examples: [
      {
        desc: 'Summarize a capture',
        command: 'niac analyze-pcap captures/office.pcap',
        output: 'ARP: 240   ICMP: 12   DNS: 88   SNMP: 4   LLDP: 30',
      },
    ],
    tips: [
      'Filter with tcpdump first if the file is huge — NIAC reads the whole thing into memory.',
    ],
    seeAlso: ['cmd-dump', 'analysis-pcap'],
  },
  {
    id: 'cmd-analyze-walk',
    name: 'niac analyze-walk',
    standard: 'niac CLI',
    category: 'commands',
    summary: 'Reconstruct a device from an SNMP walk.',
    techDesc:
      'Parses an snmpwalk output file (OID = TYPE: VALUE lines) and extracts: sysDescr / sysName / sysLocation / sysContact, ifTable (interface names, speeds, status), lldpRemTable (neighbors), ipAddrTable (IPs). Emits a draft YAML device block.',
    laymanDesc: 'Take a `snmpwalk` of a real device and produce a YAML config that mimics it.',
    whenToUse: 'You have an SNMP walk and want NIAC to look just like that device.',
    whenNotToUse: 'No SNMP available — use `analyze-pcap` instead.',
    parameters: [
      {
        name: 'walk-file',
        flag: '<positional>',
        type: 'path',
        defaultValue: '',
        required: true,
        techDesc: 'Path to snmpwalk-format text file.',
        laymanDesc: 'The walk file.',
        example: 'cisco-3850.walk',
      },
    ],
    configFields: [],
    metrics: [],
    examples: [
      {
        desc: 'Build a device config from a walk',
        command: 'niac analyze-walk cisco-3850.walk > device.yaml',
      },
    ],
    tips: [
      'Walks should be in `OID = TYPE: VALUE` format (default Net-SNMP output).',
      'Combine with `niac sanitize` to scrub identifying info before sharing.',
    ],
    seeAlso: ['analysis-walk', 'cmd-sanitize'],
  },
  {
    id: 'cmd-sanitize',
    name: 'niac sanitize',
    standard: 'niac CLI',
    category: 'commands',
    summary: 'Scrub identifying info from walks and configs.',
    techDesc:
      'Reads a walk or YAML and replaces real MACs / IPs / serial numbers / location strings with placeholders. Idempotent. Useful for sharing reproducer configs without leaking real-network data.',
    laymanDesc: 'Remove identifying info (real IPs, MACs, hostnames) so the file is safe to share.',
    whenToUse: 'Before posting a config / walk to a bug tracker or public repo.',
    whenNotToUse: 'You’re sharing inside a trusted team and want the originals.',
    parameters: [],
    configFields: [],
    metrics: [],
    examples: [
      {
        desc: 'Sanitize a walk file',
        command: 'niac sanitize cisco-3850.walk > cisco-3850-sanitized.walk',
      },
    ],
    tips: ['Eyeball the output before sharing — sanitizers can miss vendor-specific identifiers.'],
    seeAlso: ['cmd-analyze-walk', 'analysis-sanitize'],
  },
  {
    id: 'cmd-topology',
    name: 'niac topology',
    standard: 'niac CLI',
    category: 'commands',
    summary: 'Render the topology graph as text / JSON.',
    techDesc:
      'Produces a Graphviz DOT, ASCII art, or JSON representation of the running daemon’s topology.',
    laymanDesc: 'Like the web UI topology view, but in your terminal.',
    whenToUse: 'Headless reports, snapshots into documentation.',
    whenNotToUse: 'You’re already in the browser — use the web UI page.',
    parameters: [
      {
        name: 'Format',
        flag: '--format',
        type: 'string',
        defaultValue: 'ascii',
        required: false,
        techDesc: 'ascii | dot | json.',
        laymanDesc: 'Output format.',
        example: '--format dot',
      },
    ],
    configFields: [],
    metrics: [],
    examples: [
      {
        desc: 'ASCII topology',
        command: 'niac topology',
      },
      {
        desc: 'Graphviz dot',
        command: 'niac topology --format dot | dot -Tpng -o topo.png',
      },
    ],
    tips: ['Pipe the dot output into Graphviz for nicer diagrams than the ASCII fallback.'],
    seeAlso: ['topo-graph'],
  },
  {
    id: 'cmd-service',
    name: 'niac service',
    standard: 'niac CLI',
    category: 'commands',
    summary: 'Windows-service lifecycle commands.',
    techDesc:
      'Subcommands `install | uninstall | start | stop | status` for managing NIAC as a Windows service. On Linux/macOS, prefer the systemd / launchd units shipped in `deploy/`.',
    laymanDesc: 'Install NIAC as a Windows background service.',
    whenToUse: 'Windows hosts you want running NIAC at boot.',
    whenNotToUse: 'Linux — use systemd.',
    parameters: [],
    configFields: [],
    metrics: [],
    examples: [
      {
        desc: 'Install as service',
        command: 'niac service install',
      },
      {
        desc: 'Check status',
        command: 'niac service status',
      },
    ],
    tips: ['Service runs as LocalSystem by default — it has rights to open raw sockets.'],
    seeAlso: ['cmd-daemon'],
  },
  {
    id: 'cmd-man',
    name: 'niac man',
    standard: 'niac CLI',
    category: 'commands',
    summary: 'Generate man pages.',
    techDesc: 'Writes Unix man pages for every subcommand into a target directory.',
    laymanDesc: 'Produce manpages for the niac binary.',
    whenToUse: 'Packaging NIAC into a .deb / .rpm / Homebrew formula.',
    whenNotToUse: 'Day-to-day use — the binary self-documents via `--help`.',
    parameters: [
      {
        name: 'Output dir',
        flag: '--output',
        type: 'path',
        defaultValue: './man',
        required: false,
        techDesc: 'Directory to write man pages into.',
        laymanDesc: 'Where the manpages go.',
        example: '--output /usr/local/share/man/man1',
      },
    ],
    configFields: [],
    metrics: [],
    examples: [
      {
        desc: 'Generate locally',
        command: 'niac man --output ./man',
      },
    ],
    tips: ['Section 1 (user commands) is used; configure your packager to install to section 1.'],
    seeAlso: ['cmd-completion'],
  },
  {
    id: 'cmd-completion',
    name: 'niac completion',
    standard: 'niac CLI',
    category: 'commands',
    summary: 'Generate shell-completion scripts.',
    techDesc: 'Emits completion for bash, zsh, fish, or powershell. Writes to stdout.',
    laymanDesc: 'Get tab-completion for niac commands in your shell.',
    whenToUse: 'Once, when setting up a dev shell.',
    whenNotToUse: 'In CI.',
    parameters: [
      {
        name: 'shell',
        flag: '<positional>',
        type: 'string',
        defaultValue: '',
        required: true,
        techDesc: 'bash | zsh | fish | powershell.',
        laymanDesc: 'Which shell.',
        example: 'zsh',
      },
    ],
    configFields: [],
    metrics: [],
    examples: [
      {
        desc: 'Bash completion',
        command: 'niac completion bash | sudo tee /etc/bash_completion.d/niac',
      },
      {
        desc: 'Zsh completion',
        command: 'niac completion zsh > ~/.zsh/completions/_niac',
      },
    ],
    tips: [
      'Source the file from your .bashrc / .zshrc once; afterward, tab-completion just works.',
    ],
    seeAlso: ['cmd-man'],
  },
];

// ---------------------------------------------------------------------------
// Analysis
// ---------------------------------------------------------------------------

const analysisItems: HelpItem[] = [
  {
    id: 'analysis-pcap',
    name: 'PCAP Analysis',
    standard: 'niac analyze-pcap',
    category: 'analysis',
    summary: 'Counts and per-host summaries from a capture file.',
    techDesc:
      'The analysis backend reads libpcap files, decodes Ethernet/802.1Q/IPv4/IPv6/TCP/UDP and the application protocols NIAC understands, then emits per-protocol counters and per-MAC frame totals. Optionally produces a draft YAML config inferred from the observed hosts.',
    laymanDesc: 'Open a pcap and see what protocols and devices are in it.',
    whenToUse: 'You have a capture from a real network and want to know what’s on it.',
    whenNotToUse: 'You want full packet decoding — use Wireshark.',
    parameters: [],
    configFields: [],
    metrics: [],
    examples: [
      {
        desc: 'CLI analysis',
        command: 'niac analyze-pcap office.pcap',
      },
      {
        desc: 'Web UI: Packets page → Analyze tab',
        command: '# open http://localhost:8080/packets',
      },
    ],
    tips: ['The web UI Packets page can analyze recordings stored in the daemon’s capture buffer.'],
    seeAlso: ['cmd-analyze-pcap', 'cmd-dump'],
  },
  {
    id: 'analysis-walk',
    name: 'Walk Analysis',
    standard: 'niac analyze-walk',
    category: 'analysis',
    summary: 'Build a NIAC device from an SNMP walk.',
    techDesc:
      'Parses snmpwalk OID-by-OID output, identifies system / interface / neighbor MIBs, and reconstructs a device persona including sysDescr, ifTable, lldpRemTable, ipAddrTable.',
    laymanDesc: 'Convert a real device’s SNMP walk into a config NIAC can simulate.',
    whenToUse: 'Migrating real-device behavior into NIAC for offline testing.',
    whenNotToUse: 'No SNMP access to the original — use `analyze-pcap` instead.',
    parameters: [],
    configFields: [],
    metrics: [],
    examples: [
      {
        desc: 'Walk → YAML',
        command: 'niac analyze-walk cisco-3850.walk > cisco-3850.yaml',
      },
      {
        desc: 'Web UI → SNMP Walks',
        command: '# open http://localhost:8080/walk-validator',
      },
    ],
    tips: ['Web UI version also validates and auto-fixes malformed walk files.'],
    seeAlso: ['cmd-analyze-walk', 'proto-snmpv2c'],
  },
  {
    id: 'analysis-sanitize',
    name: 'Sanitize',
    standard: 'niac sanitize',
    category: 'analysis',
    summary: 'Strip identifying data from walks and configs.',
    techDesc:
      'Replaces MAC addresses, IP addresses, hostnames, serial numbers, location strings, and contact strings with placeholder values. Designed to make a config / walk shareable without revealing real-network details.',
    laymanDesc: 'Scrub a file so you can share it publicly.',
    whenToUse: 'Posting reproducer files to bug trackers, public repos, or external vendors.',
    whenNotToUse: 'Keep originals when working internally.',
    parameters: [],
    configFields: [],
    metrics: [],
    examples: [
      {
        desc: 'Sanitize a walk',
        command: 'niac sanitize router.walk > router-sanitized.walk',
      },
    ],
    tips: [
      'Eye-check the output — the sanitizer is conservative and may miss vendor-specific identifiers.',
    ],
    seeAlso: ['cmd-sanitize', 'analysis-walk'],
  },
];

// ---------------------------------------------------------------------------
// Errors (injection details)
// ---------------------------------------------------------------------------

const errorItems: HelpItem[] = [
  {
    id: 'err-fcs',
    name: 'FCS Errors',
    standard: 'niac inject fcs_errors',
    category: 'errors',
    summary: 'Frame Check Sequence (CRC) error counter.',
    techDesc:
      'Injection increments the device’s simulated ifInErrors and dot3StatsFCSErrors counters at the injected rate. Polling SNMP-derived dashboards see the counter climb.',
    laymanDesc:
      'Tells the simulated device to report broken Ethernet frames — the most common "bad cable" indicator.',
    whenToUse: 'Validating that a monitoring system actually catches Layer-2 link errors.',
    whenNotToUse: 'You want a one-shot spike — inject the value, wait, then inject 0.',
    parameters: [],
    configFields: [],
    metrics: [
      {
        name: 'ifInErrors',
        unit: 'count',
        goodRange: '0',
        badMeaning: 'CRC mismatches — typically bad cable or transceiver.',
      },
    ],
    examples: [
      {
        desc: 'Inject 50% FCS errors',
        command: 'niac inject router-1 fcs_errors 50',
      },
      {
        desc: 'Clear',
        command: 'niac inject router-1 fcs_errors 0',
      },
    ],
    tips: ['Pair with `high_utilization` to mimic a saturating link that’s also breaking.'],
    seeAlso: ['cmd-inject', 'err-discards'],
  },
  {
    id: 'err-discards',
    name: 'Packet Discards',
    standard: 'niac inject packet_discards',
    category: 'errors',
    summary: 'Input/output packet discard counter.',
    techDesc:
      'Increments ifInDiscards and ifOutDiscards at the injected rate. Indicates the device is dropping frames — typically queue overflow.',
    laymanDesc: 'Simulates dropped packets due to congestion or buffer overflow.',
    whenToUse: 'Testing alerting on packet-loss thresholds.',
    whenNotToUse:
      'You’re trying to model actual congestion behavior — NIAC just counters, it doesn’t actually drop.',
    parameters: [],
    configFields: [],
    metrics: [
      {
        name: 'ifInDiscards',
        unit: 'count',
        goodRange: '0',
        badMeaning: 'Frames dropped at ingress — buffer pressure.',
      },
      {
        name: 'ifOutDiscards',
        unit: 'count',
        goodRange: '0',
        badMeaning: 'Frames dropped at egress — queue overflow.',
      },
    ],
    examples: [
      {
        desc: 'Inject discards',
        command: 'niac inject switch-2 packet_discards 25',
      },
    ],
    tips: ['Combine with iperf3 throughput claims to simulate realistic load + loss.'],
    seeAlso: ['cmd-inject', 'err-fcs'],
  },
  {
    id: 'err-iferrors',
    name: 'Interface Errors',
    standard: 'niac inject interface_errors',
    category: 'errors',
    summary: 'Generic interface-error counter.',
    techDesc:
      'Increments ifInErrors / ifOutErrors without specifying a particular cause. Useful when the NMS only checks the top-level counter.',
    laymanDesc: 'Generic "interface is having problems" counter.',
    whenToUse: 'Testing detection of any-error conditions.',
    whenNotToUse: 'You need a specific error type — use fcs_errors or packet_discards instead.',
    parameters: [],
    configFields: [],
    metrics: [],
    examples: [
      {
        desc: 'Inject',
        command: 'niac inject router-1 interface_errors 15',
      },
    ],
    tips: [],
    seeAlso: ['err-fcs', 'err-discards'],
  },
  {
    id: 'err-cpu',
    name: 'High CPU',
    standard: 'niac inject high_cpu',
    category: 'errors',
    summary: 'CPU-utilization gauge value.',
    techDesc:
      'Sets the simulated device’s CPU gauge to the given percentage. The value is reflected in SNMP polls of cpmCPUTotal5secRev (Cisco) and equivalents on other vendor MIBs.',
    laymanDesc: 'Tells the device to pretend it’s overloaded.',
    whenToUse: 'Drives CPU-threshold alerting tests.',
    whenNotToUse:
      'You actually want the niac process to peg CPU — this is just a counter, not a load generator.',
    parameters: [],
    configFields: [],
    metrics: [
      {
        name: 'cpuLoad',
        unit: '%',
        goodRange: '< 80',
        badMeaning: '> 80% sustained — control-plane saturation risk.',
      },
    ],
    examples: [
      {
        desc: '85% CPU',
        command: 'niac inject router-1 high_cpu 85',
      },
    ],
    tips: ['Many NMS systems trigger only above 80%; test both above and below your threshold.'],
    seeAlso: ['err-memory', 'cmd-inject'],
  },
  {
    id: 'err-memory',
    name: 'High Memory',
    standard: 'niac inject high_memory',
    category: 'errors',
    summary: 'Memory-utilization gauge.',
    techDesc:
      'Sets the memory-utilization gauge polled by SNMP. Mirrors ciscoMemoryPoolUsed proportional to total.',
    laymanDesc: 'Tells the device to pretend it’s low on memory.',
    whenToUse: 'Memory-threshold alerting tests.',
    whenNotToUse: 'You want real memory pressure — NIAC doesn’t allocate just because you asked.',
    parameters: [],
    configFields: [],
    metrics: [],
    examples: [
      {
        desc: '92% memory',
        command: 'niac inject server-1 high_memory 92',
      },
    ],
    tips: [],
    seeAlso: ['err-cpu', 'err-disk'],
  },
  {
    id: 'err-disk',
    name: 'High Disk',
    standard: 'niac inject high_disk',
    category: 'errors',
    summary: 'Disk-utilization gauge.',
    techDesc:
      'Reflects in hrStorageUsed / hrStorageAllocationUnits combination. Useful for tools that watch host-resource MIB for full disks.',
    laymanDesc: 'Tells the simulated device its disk is full.',
    whenToUse: 'Storage-alerting tests.',
    whenNotToUse: 'Non-host-resource MIB consumers.',
    parameters: [],
    configFields: [],
    metrics: [],
    examples: [
      {
        desc: '95% disk',
        command: 'niac inject server-1 high_disk 95',
      },
    ],
    tips: [],
    seeAlso: ['err-memory'],
  },
  {
    id: 'err-util',
    name: 'High Utilization',
    standard: 'niac inject high_utilization',
    category: 'errors',
    summary: 'Interface-bandwidth-utilization gauge.',
    techDesc:
      'Drives ifHCInOctets / ifHCOutOctets velocity to a level consistent with the injected percentage of the interface’s speed (ifSpeed). Polling an NMS over time will show the link saturating.',
    laymanDesc: 'Tells the device to pretend its uplink is busy.',
    whenToUse: 'Bandwidth-threshold alerting tests.',
    whenNotToUse: 'You need actual saturation — use iperf3 client traffic instead.',
    parameters: [],
    configFields: [],
    metrics: [],
    examples: [
      {
        desc: '90% link utilization',
        command: 'niac inject switch-2 high_utilization 90',
      },
    ],
    tips: ['Combine with `packet_discards` to mimic a saturating-link congestion scenario.'],
    seeAlso: ['err-discards', 'proto-snmpv2c'],
  },
];

// ---------------------------------------------------------------------------
// All items, indexed
// ---------------------------------------------------------------------------

export const items: HelpItem[] = [
  ...deviceItems,
  ...protocolItems,
  ...topologyItems,
  ...commandItems,
  ...analysisItems,
  ...errorItems,
];

export const itemsById: Record<string, HelpItem> = Object.fromEntries(
  items.map((it) => [it.id, it]),
);

// ---------------------------------------------------------------------------
// Features (top-level UI navigation)
// ---------------------------------------------------------------------------

export const features: Feature[] = [
  {
    title: 'Dashboard',
    description: 'Live counters, run snapshots, and alert state for the active NIAC stack.',
    path: '/',
  },
  {
    title: 'Simulation',
    description: 'Pick a network, choose an interface, start or stop the daemon.',
    path: '/runtime',
  },
  {
    title: 'Running Devices',
    description: 'Read-only view of the devices the daemon is currently simulating.',
    path: '/devices',
  },
  {
    title: 'Devices Library',
    description: 'Reusable device definitions. Edit, clone, delete the library on disk.',
    path: '/device-config',
  },
  {
    title: 'Topology',
    description: 'Visual graph of the configured network plus the live neighbor table.',
    path: '/topology',
  },
  {
    title: 'Traffic',
    description: 'Inject controlled errors into the simulation and replay captured PCAPs.',
    path: '/traffic',
  },
  {
    title: 'Logs',
    description: 'Live log stream from the daemon, with debug-level controls.',
    path: '/debug',
  },
  {
    title: 'Packets',
    description: 'Live packets crossing the simulation — hex view, BPF filter, save to PCAP.',
    path: '/packets',
  },
  {
    title: 'SNMP Walks',
    description: 'Validate and auto-fix SNMP walk files used by the simulated SNMP agents.',
    path: '/walk-validator',
  },
  {
    title: 'Compare & Merge',
    description: 'Compare two YAML network configs side-by-side and merge changes between them.',
    path: '/config-diff',
  },
];

// ---------------------------------------------------------------------------
// Glossary
// ---------------------------------------------------------------------------

export const glossary: GlossaryEntry[] = [
  // Protocols
  {
    term: 'ARP',
    definition:
      'Address Resolution Protocol (RFC 826). Maps IPv4 addresses to MAC addresses on a local Ethernet link.',
    category: 'protocol',
  },
  {
    term: 'CDP',
    definition:
      'Cisco Discovery Protocol. Cisco’s proprietary L2 discovery protocol; emits frames every 60s announcing device identity, capabilities, and platform.',
    category: 'protocol',
  },
  {
    term: 'DHCP',
    definition:
      'Dynamic Host Configuration Protocol (RFC 2131). Hands out IP addresses, gateways, DNS servers, and other network parameters to clients.',
    category: 'protocol',
  },
  {
    term: 'DHCPv6',
    definition:
      'IPv6 variant of DHCP (RFC 8415). Two modes: stateful (assigns addresses) and stateless (options-only, complementing SLAAC).',
    category: 'protocol',
  },
  {
    term: 'DISCOVER',
    definition:
      'First DHCPv4 message: client broadcasts "I need an IP." Followed by OFFER, REQUEST, ACK.',
    category: 'protocol',
  },
  {
    term: 'OFFER',
    definition: 'Second DHCPv4 message: server responds to DISCOVER with a candidate lease.',
    category: 'protocol',
  },
  {
    term: 'REQUEST',
    definition: 'Third DHCPv4 message: client confirms it wants the offered lease.',
    category: 'protocol',
  },
  {
    term: 'ACK',
    definition: 'Fourth DHCPv4 message: server commits the lease.',
    category: 'protocol',
  },
  {
    term: 'DNS',
    definition:
      'Domain Name System (RFC 1035). Translates names to IPs (A, AAAA records) and reverse (PTR).',
    category: 'protocol',
  },
  {
    term: 'A record',
    definition: 'DNS record mapping a name to an IPv4 address.',
    category: 'protocol',
  },
  {
    term: 'AAAA',
    definition: 'DNS record mapping a name to an IPv6 address ("quad-A").',
    category: 'protocol',
  },
  { term: 'PTR', definition: 'DNS reverse-lookup record — IP to name.', category: 'protocol' },
  {
    term: 'Recursive resolver',
    definition:
      'A DNS server that chases queries down through the hierarchy on the client’s behalf. NIAC is NOT one of these.',
    category: 'protocol',
  },
  {
    term: 'Authoritative server',
    definition:
      'A DNS server that holds the actual records for a zone. NIAC IS one of these (for configured zones).',
    category: 'protocol',
  },
  {
    term: 'EDP',
    definition:
      'Extreme Discovery Protocol — Extreme Networks’ proprietary L2 discovery, pre-dating LLDP.',
    category: 'protocol',
  },
  {
    term: 'FDP',
    definition: 'Foundry Discovery Protocol — Foundry/Brocade/Ruckus proprietary discovery.',
    category: 'protocol',
  },
  {
    term: 'FTP',
    definition:
      'File Transfer Protocol (RFC 959). NIAC implements only the banner / handshake, not real file transfer.',
    category: 'protocol',
  },
  {
    term: 'HTTP',
    definition:
      'Hypertext Transfer Protocol. NIAC ships a minimal HTTP/1.1 responder with configurable endpoints.',
    category: 'protocol',
  },
  {
    term: 'ICMP',
    definition: 'Internet Control Message Protocol — ping, traceroute, error reports.',
    category: 'protocol',
  },
  {
    term: 'ICMPv6',
    definition:
      'IPv6 version of ICMP. Includes Neighbor Discovery (ND), which replaces ARP for v6.',
    category: 'protocol',
  },
  {
    term: 'iperf3',
    definition:
      'Network throughput-measurement protocol. NIAC speaks the control channel but does not perform real throughput tests.',
    category: 'protocol',
  },
  {
    term: 'LACP',
    definition:
      'Link Aggregation Control Protocol (IEEE 802.1AX). Lets multiple physical links act as one logical link.',
    category: 'protocol',
  },
  {
    term: 'LLDP',
    definition:
      'Link Layer Discovery Protocol (IEEE 802.1AB). Vendor-neutral L2 discovery — the standard CDP replacement.',
    category: 'protocol',
  },
  {
    term: 'mDNS',
    definition:
      'Multicast DNS (RFC 6762). Used by Bonjour / Avahi for local-network service discovery. NIAC doesn’t implement mDNS today.',
    category: 'protocol',
  },
  {
    term: 'NetBIOS',
    definition:
      'Legacy Windows name/service protocol (RFC 1001/1002). NIAC answers NBSTAT name queries.',
    category: 'protocol',
  },
  {
    term: 'SNMP',
    definition: 'Simple Network Management Protocol. Query and monitor device state via OIDs.',
    category: 'protocol',
  },
  {
    term: 'SNMPv1',
    definition: 'Original SNMP (RFC 1157). Community-string auth, 32-bit counters, lossy traps.',
    category: 'protocol',
  },
  {
    term: 'SNMPv2c',
    definition:
      'SNMP v2 with community-string auth (RFC 3416/3418). Adds GetBulk and 64-bit counters.',
    category: 'protocol',
  },
  {
    term: 'SNMPv3',
    definition: 'SNMP with USM-based authentication and optional encryption (RFC 3414/3826).',
    category: 'protocol',
  },
  {
    term: 'STP / RSTP / MSTP',
    definition:
      'Spanning Tree Protocol family (IEEE 802.1D / 802.1w / 802.1s). Loop prevention via BPDU exchange.',
    category: 'protocol',
  },
  {
    term: 'TCP',
    definition:
      'Transmission Control Protocol — reliable, ordered transport. Underlies HTTP, FTP, etc.',
    category: 'protocol',
  },
  {
    term: 'UDP',
    definition:
      'User Datagram Protocol — lightweight, connectionless transport. Underlies DNS, SNMP, DHCP.',
    category: 'protocol',
  },

  // Concepts
  {
    term: 'Bridge',
    definition:
      'A device that forwards frames between two or more LAN segments at Layer 2. Switches are multi-port bridges.',
    category: 'concept',
  },
  {
    term: 'Broadcast domain',
    definition:
      'The set of devices that receive each other’s broadcast frames. A VLAN defines a broadcast domain.',
    category: 'concept',
  },
  {
    term: 'BPDU',
    definition:
      'Bridge Protocol Data Unit — the frames exchanged by STP to negotiate a loop-free topology.',
    category: 'concept',
  },
  {
    term: 'BPF',
    definition:
      'Berkeley Packet Filter. Compact expressions used by tcpdump, NIAC, etc. to select packets (`tcp port 80 and host 10.0.0.1`).',
    category: 'concept',
  },
  {
    term: 'Capability TLV',
    definition:
      'In LLDP/CDP, a TLV announcing what the device is (router, bridge, AP, station-only).',
    category: 'concept',
  },
  {
    term: 'Chassis ID',
    definition: 'In LLDP, a unique identifier for the device chassis — usually its MAC address.',
    category: 'concept',
  },
  {
    term: 'Community string',
    definition:
      'The shared secret used to authenticate SNMPv1/v2c requests. "public" is the historical default and is insecure.',
    category: 'concept',
  },
  {
    term: 'Default gateway',
    definition: 'The router an end-host sends off-subnet traffic to.',
    category: 'concept',
  },
  {
    term: 'EUI-64',
    definition:
      'A method to generate a 64-bit interface ID from a 48-bit MAC, used to autoconfigure IPv6 link-local addresses.',
    category: 'concept',
  },
  {
    term: 'Frame',
    definition: 'A Layer-2 packet — the unit Ethernet works in.',
    category: 'concept',
  },
  {
    term: 'Gateway',
    definition: 'Loose term for a router that connects different networks.',
    category: 'concept',
  },
  {
    term: 'Inform',
    definition:
      'An SNMP "trap with ack" — the receiver must acknowledge receipt. Adds reliability.',
    category: 'concept',
  },
  {
    term: 'IPC socket',
    definition:
      'Unix-domain socket used by the niac CLI to talk to the running daemon. Default path `/tmp/niac.sock`.',
    category: 'concept',
  },
  {
    term: 'Jumbo frame',
    definition:
      'Ethernet frame larger than the standard 1500-byte MTU — commonly 9000 bytes for storage networks.',
    category: 'concept',
  },
  {
    term: 'Lease',
    definition:
      'A DHCP-assigned IP plus duration. When the lease expires, the client must renew or release it.',
    category: 'concept',
  },
  {
    term: 'MAC address',
    definition:
      'Media Access Control address. 48-bit unique-per-NIC identifier formatted aa:bb:cc:dd:ee:ff.',
    category: 'concept',
  },
  {
    term: 'MIB',
    definition: 'Management Information Base. The schema describing what SNMP OIDs mean.',
    category: 'concept',
  },
  {
    term: 'MTU',
    definition:
      'Maximum Transmission Unit — the largest payload an interface can carry in one frame (default 1500 on Ethernet).',
    category: 'concept',
  },
  {
    term: 'Multicast',
    definition:
      'Frames sent to a group address — one sender, many receivers. LLDP, CDP, STP, mDNS all use multicast.',
    category: 'concept',
  },
  {
    term: 'OID',
    definition:
      'Object Identifier. A dotted-numeric path through the SNMP MIB tree (e.g. 1.3.6.1.2.1.1.1.0 = sysDescr.0).',
    category: 'concept',
  },
  {
    term: 'Option 82',
    definition:
      'DHCP relay-agent information option. Lets a DHCP relay tell the server which port the client showed up on.',
    category: 'concept',
  },
  {
    term: 'OUI',
    definition:
      'Organizationally Unique Identifier — the first 24 bits of a MAC, identifying the vendor.',
    category: 'concept',
  },
  {
    term: 'PCAP',
    definition:
      'Packet Capture file format (libpcap). Stores raw packets for offline analysis (Wireshark, NIAC analyze-pcap).',
    category: 'concept',
  },
  {
    term: 'PoE',
    definition:
      'Power over Ethernet (IEEE 802.3af/at/bt). Carries power to devices over the data cable.',
    category: 'concept',
  },
  {
    term: 'Port-channel',
    definition:
      'A bundle of physical interfaces presented as one logical interface. Negotiated with LACP.',
    category: 'concept',
  },
  {
    term: 'Raw socket',
    definition:
      'A socket that delivers L2 or L3 frames without OS protocol processing. NIAC needs raw sockets, which usually requires root.',
    category: 'concept',
  },
  {
    term: 'Relay agent',
    definition:
      'A device that forwards DHCP requests across subnets (because DHCP DISCOVER is broadcast and broadcasts don’t cross routers).',
    category: 'concept',
  },
  {
    term: 'sysDescr',
    definition:
      'SNMP sysDescr.0 — the device’s self-description string. Tools fingerprint vendors from this.',
    category: 'concept',
  },
  {
    term: 'Trap',
    definition:
      'An unsolicited SNMP message from a device to a configured receiver. Inform = trap with acknowledgement.',
    category: 'concept',
  },
  {
    term: 'TLV',
    definition:
      'Type-Length-Value. The encoding LLDP / CDP / DHCP options use — a tag, a length, then the data.',
    category: 'concept',
  },
  {
    term: 'Trunk',
    definition: 'A port carrying multiple VLANs, with each frame tagged using 802.1Q.',
    category: 'concept',
  },
  {
    term: 'TTL',
    definition:
      'Time-To-Live — a value that limits how long a packet or LLDP advertisement is considered valid.',
    category: 'concept',
  },
  {
    term: 'VLAN',
    definition:
      'Virtual LAN (IEEE 802.1Q). A way to partition one physical network into multiple broadcast domains.',
    category: 'concept',
  },
  {
    term: 'Walk',
    definition:
      'An SNMP walk — the act of GETNEXT-ing through every OID under a subtree. Also: the file you get by saving its output.',
    category: 'concept',
  },

  // Devices
  {
    term: 'Access Point (AP)',
    definition: 'A wireless device that bridges Wi-Fi clients onto a wired LAN.',
    category: 'device',
  },
  {
    term: 'Firewall',
    definition: 'A device that enforces a security policy on traffic between zones.',
    category: 'device',
  },
  {
    term: 'Host',
    definition: 'A general-purpose end-system on the network — PC, server, IoT sensor.',
    category: 'device',
  },
  {
    term: 'NMS',
    definition:
      'Network Management System — a platform that polls / monitors devices (SolarWinds, LibreNMS, Zabbix, etc.).',
    category: 'device',
  },
  {
    term: 'Router',
    definition: 'A device that forwards packets between networks at Layer 3.',
    category: 'device',
  },
  {
    term: 'Switch',
    definition: 'A multi-port bridge — forwards Ethernet frames between ports based on MAC tables.',
    category: 'device',
  },
  { term: 'Workstation', definition: 'An end-user host — desktop, laptop.', category: 'device' },

  // NIAC-specific
  {
    term: 'Device simulator',
    definition:
      'NIAC, in one phrase — a tool that pretends to be one or more network devices on a real interface.',
    category: 'niac',
  },
  {
    term: 'Daemon mode',
    definition:
      'NIAC running long-lived, with the embedded web UI active. Started with `niac daemon`.',
    category: 'niac',
  },
  {
    term: 'Run mode',
    definition:
      'NIAC running one config to exit, foreground. Started with `niac run <iface> <config>`.',
    category: 'niac',
  },
  {
    term: 'Interactive mode',
    definition:
      'NIAC running with a terminal UI dashboard. Started with `niac interactive <iface> <config>`.',
    category: 'niac',
  },
  {
    term: 'Topology config',
    definition: 'The YAML file that describes the simulated network: devices, protocols, VLANs.',
    category: 'niac',
  },
  {
    term: 'Error injection',
    definition: 'Forcing a simulated device into a faulted state at runtime via `niac inject`.',
    category: 'niac',
  },
  {
    term: 'Walk file',
    definition:
      'A text file holding SNMP `OID = TYPE: VALUE` lines, used to drive NIAC’s SNMP agent for realistic responses.',
    category: 'niac',
  },
  {
    term: 'Capture buffer',
    definition: 'The daemon’s in-memory rolling pcap. Use `niac dump` to flush it to disk.',
    category: 'niac',
  },
  {
    term: 'Template',
    definition:
      'A built-in starter YAML config for a common device type. List with `niac template list`.',
    category: 'niac',
  },
  {
    term: 'Persona',
    definition:
      'A device-type setting (router / switch / firewall / server / ap / iot / unknown) that applies sensible defaults.',
    category: 'niac',
  },

  // Security
  {
    term: 'API token',
    definition:
      'Bearer token gating the NIAC daemon’s REST API. Set via NIAC_API_TOKEN environment variable.',
    category: 'security',
  },
  {
    term: 'Raw-socket privilege',
    definition:
      'The OS permission needed to open Layer-2 / Layer-3 raw sockets. Typically root, or CAP_NET_RAW on Linux.',
    category: 'security',
  },
];

// ---------------------------------------------------------------------------
// Keyboard shortcuts
// ---------------------------------------------------------------------------

export const shortcuts: Shortcut[] = [
  // General (web UI)
  { keys: ['Esc'], description: 'Close drawer or modal', category: 'general' },
  { keys: ['?'], description: 'Open help', category: 'general' },
  { keys: ['/', ','], description: 'Open settings', category: 'general' },
  { keys: ['Shift', '?'], description: 'Keyboard cheat sheet', category: 'general' },

  // Navigation
  { keys: ['g', 'h'], description: 'Go to Command Center (Dashboard)', category: 'navigation' },
  { keys: ['g', 'r'], description: 'Go to Runtime Control', category: 'navigation' },
  { keys: ['g', 'd'], description: 'Go to Running Devices', category: 'navigation' },
  { keys: ['g', 'e'], description: 'Go to Device Editor', category: 'navigation' },
  { keys: ['g', 't'], description: 'Go to Topology', category: 'navigation' },
  { keys: ['g', 'l'], description: 'Go to Logs', category: 'navigation' },
  { keys: ['g', 'p'], description: 'Go to Packets', category: 'navigation' },
  { keys: ['g', 'w'], description: 'Go to SNMP Walks', category: 'navigation' },
  { keys: ['g', 'i'], description: 'Go to Traffic / Injection', category: 'navigation' },
  { keys: ['g', 'c'], description: 'Go to Compare & Merge', category: 'navigation' },

  // Actions
  { keys: ['Ctrl', 'k'], description: 'Quick search / command palette', category: 'actions' },
  { keys: ['Ctrl', 's'], description: 'Save current configuration', category: 'actions' },
  { keys: ['Ctrl', 'Enter'], description: 'Run / execute action', category: 'actions' },
  { keys: ['Ctrl', 'r'], description: 'Reload from disk', category: 'actions' },
  { keys: ['Ctrl', '.'], description: 'Toggle dark / light theme', category: 'actions' },
  { keys: ['Ctrl', 'b'], description: 'Toggle sidebar', category: 'actions' },

  // TUI (`niac interactive`)
  { keys: ['q'], description: 'Quit interactive mode', category: 'tui' },
  { keys: ['?'], description: 'TUI help overlay', category: 'tui' },
  { keys: ['Tab'], description: 'Cycle through panels', category: 'tui' },
  { keys: ['Shift', 'Tab'], description: 'Cycle backwards', category: 'tui' },
  { keys: ['n'], description: 'Show neighbors panel', category: 'tui' },
  { keys: ['l'], description: 'Show logs panel', category: 'tui' },
  { keys: ['s'], description: 'Show statistics panel', category: 'tui' },
  { keys: ['i'], description: 'Open inject prompt', category: 'tui' },
  { keys: ['/'], description: 'Filter log lines', category: 'tui' },
  { keys: ['j', 'k'], description: 'Scroll down / up (vim-style)', category: 'tui' },
  { keys: ['g', 'g'], description: 'Jump to top', category: 'tui' },
  { keys: ['G'], description: 'Jump to bottom', category: 'tui' },
  { keys: ['Ctrl', 'l'], description: 'Clear screen / redraw', category: 'tui' },
  { keys: ['Ctrl', 'c'], description: 'Force quit', category: 'tui' },
];

// ---------------------------------------------------------------------------
// FAQ
// ---------------------------------------------------------------------------

export const faq: FAQEntry[] = [
  {
    id: 'faq-switch-4-ports',
    question: 'How do I simulate a switch with 4 ports?',
    answer:
      'NIAC simulates devices at the IP / MAC level, not per physical port. To model a 4-port switch with downstream hosts, define one device of `type: switch` plus four host devices (`type: workstation` or `type: host`) all sharing the same host interface. Use VLANs (`devices[].vlan`) to put each on its own broadcast domain if you need port isolation.',
    tags: ['switch', 'topology', 'vlan'],
  },
  {
    id: 'faq-without-root',
    question: 'Can I run NIAC without root?',
    answer:
      'Not really. NIAC needs raw sockets to read Ethernet / IP frames directly. On Linux you can grant CAP_NET_RAW with `sudo setcap cap_net_raw+ep ./niac` so it runs as a normal user. On macOS you must `sudo niac …` (no capabilities system). On loopback (`lo0`) some tests work without privileges, but you’ll be limited to ICMP and certain UDP probes.',
    tags: ['security', 'permissions', 'linux', 'macos'],
  },
  {
    id: 'faq-daemon-vs-run',
    question: 'What’s the difference between daemon and run modes?',
    answer:
      '`niac run <iface> <config>` is a one-shot foreground process: it starts the simulation, runs until Ctrl-C, then exits. No web UI. Good for CI. `niac daemon` is a long-lived background process with the embedded web UI. Configs are loaded / unloaded via the REST API or web UI; the daemon outlives any single simulation. Good for lab desktops or persistent test rigs.',
    tags: ['daemon', 'run', 'modes', 'webui'],
  },
  {
    id: 'faq-inject-packet-loss',
    question: 'How do I inject packet loss?',
    answer:
      'Use `niac inject <device> packet_discards <0-100>` against a running daemon. The simulated device’s ifInDiscards / ifOutDiscards counters increment at the injected rate, so monitoring tooling sees realistic packet drops. To clear, run the same command with value 0, or `niac inject clear --device <name>`.',
    tags: ['inject', 'loss', 'errors'],
  },
  {
    id: 'faq-capture-pcap',
    question: 'How do I capture and analyze a pcap?',
    answer:
      'Three options. (1) Run tcpdump on the host interface in parallel: `sudo tcpdump -i en0 -w out.pcap`. (2) Use the daemon’s rolling capture buffer: `niac dump -o out.pcap` flushes the last N seconds. (3) In the web UI, open /packets and click "Save to PCAP". Once you have the file, run `niac analyze-pcap out.pcap` for a protocol summary.',
    tags: ['pcap', 'capture', 'analyze'],
  },
  {
    id: 'faq-yaml-schema',
    question: 'How do I use the YAML schema?',
    answer:
      'NIAC publishes its config schema at `/api/schema/network` from a running daemon, and the source ships at `docs/schemas/niac.schema.json`. Point VS Code / IntelliJ / yaml-language-server at that file and you’ll get autocomplete + validation as you type. The schema is regenerated by `make schema` and is enforced in CI.',
    tags: ['yaml', 'schema', 'editor'],
  },
  {
    id: 'faq-raw-sockets',
    question: 'Why do I need raw sockets?',
    answer:
      'NIAC operates below the OS’s normal IP stack — it reads and writes raw Ethernet frames so it can answer ARP, run STP, emit LLDP, etc. Normal sockets (TCP/UDP) only let you talk above L4. Raw-socket access is privileged on every modern OS; that’s why NIAC needs root or CAP_NET_RAW.',
    tags: ['security', 'sockets', 'permissions'],
  },
  {
    id: 'faq-loopback',
    question: 'Can I test NIAC on loopback?',
    answer:
      'Yes, run with `lo0` (macOS / BSD) or `lo` (Linux). The simulation runs end-to-end on the same host — useful for unit tests and demos. Most protocols work; some link-layer features (LACP, STP) make less sense because loopback isn’t a real bridge. Use `niac --dry-run lo0 my.yaml` to just validate without binding.',
    tags: ['loopback', 'testing'],
  },
  {
    id: 'faq-validate',
    question: 'How do I validate my YAML before running?',
    answer:
      'Run `niac validate path/to/config.yaml`. It parses the file, checks the schema, validates MAC / IP formats, looks for VLAN conflicts, confirms walk files exist, and reports warnings and errors. Use `--strict` to treat warnings as errors — useful in CI.',
    tags: ['validate', 'yaml', 'ci'],
  },
  {
    id: 'faq-snmp-walk',
    question: 'How do I make NIAC respond like a specific vendor’s device?',
    answer:
      'Run `snmpwalk -v 2c -c public REAL.DEVICE.IP` against the real device to produce a walk file, then set `devices[].snmp.walk_file: path/to/walk` in your YAML. NIAC will serve every OID the walk contains. Use `niac sanitize` first if you plan to share the walk publicly.',
    tags: ['snmp', 'walk', 'vendor'],
  },
  {
    id: 'faq-trap-receivers',
    question: 'Can NIAC send SNMP traps?',
    answer:
      'Yes. Configure `devices[].snmp.traps.enabled: true` and list receivers under `devices[].snmp.traps.receivers: ["10.0.0.100:162"]`. Add trap-condition blocks (`high_cpu`, `high_memory`, etc.) with thresholds and intervals. NIAC will emit traps whenever the configured (or injected) condition crosses the threshold.',
    tags: ['snmp', 'traps'],
  },
  {
    id: 'faq-multi-device',
    question: 'How many devices can one NIAC instance simulate?',
    answer:
      'Practically, dozens to a few hundred. Each device adds protocol handlers and timers, plus an entry in the daemon’s neighbor table. The bottleneck is usually the host CPU when SNMP polling is heavy or LLDP frequencies are short. For thousands of endpoints, split across multiple NIAC processes on different interfaces.',
    tags: ['scale', 'performance'],
  },
  {
    id: 'faq-deploy-prod',
    question: 'Should I run NIAC in production?',
    answer:
      'NIAC is a test tool, not production infrastructure. It’s safe to run on lab networks, dev VMs, and isolated subnets. Do NOT plug a NIAC instance into a live production LAN — STP participation, DHCP server emulation, and ARP advertisements can cause real outages. If you need production-style isolation, use a separate VLAN or air-gapped lab.',
    tags: ['production', 'safety'],
  },
  {
    id: 'faq-config-diff',
    question: 'How do I compare two configs?',
    answer:
      'In the web UI, open Compare & Merge (/config-diff), drop two YAML files into the panes, and see a side-by-side diff. On the CLI, run `niac config diff a.yaml b.yaml`. Both surface the same logical view: identical / added / removed / modified per device and per protocol block.',
    tags: ['diff', 'config'],
  },
  {
    id: 'faq-merge-configs',
    question: 'Can I merge two configs?',
    answer:
      'Yes — `niac config merge <base> <overlay> <output>` (CLI) or use the merge action in the web UI Compare & Merge page. The overlay’s fields win over the base’s. Useful for layering environment-specific tweaks on top of a shared baseline.',
    tags: ['merge', 'config'],
  },
  {
    id: 'faq-systemd',
    question: 'Is there a systemd service file?',
    answer:
      'Yes. The .deb and .rpm packages install `/etc/systemd/system/niac.service`. Enable with `sudo systemctl enable --now niac`. The service runs `niac daemon` as root and reads `NIAC_API_TOKEN` from `/etc/default/niac` (Debian) or `/etc/sysconfig/niac` (RPM).',
    tags: ['systemd', 'packaging'],
  },
  {
    id: 'faq-prometheus',
    question: 'How do I scrape NIAC metrics with Prometheus?',
    answer:
      'Start the daemon with `--metrics-listen :9100` (or any host:port). NIAC exposes a Prometheus-format /metrics endpoint with per-protocol counters, per-device gauges, and runtime stats. Point your Prometheus scrape config at it.',
    tags: ['prometheus', 'metrics', 'observability'],
  },
  {
    id: 'faq-update',
    question: 'How do I update NIAC?',
    answer:
      'Linux: `sudo apt install niac` or `sudo dnf upgrade niac` against the project’s package repo. macOS: `brew upgrade niac` if installed via the Homebrew tap, else download a .pkg from GitHub releases. Always re-run `niac validate` against your configs after an update — schema can evolve between minor versions.',
    tags: ['install', 'upgrade'],
  },
  {
    id: 'faq-walk-files',
    question: 'Where do I get realistic SNMP walk files?',
    answer:
      'Three sources. (1) Walk a real device you own with `snmpwalk -v 2c -c public DEVICE > device.walk`. (2) Use the bundled examples in `examples/device_walks/`. (3) Community uploads to the NIAC content library (`niac content list`).',
    tags: ['snmp', 'walk', 'examples'],
  },
  {
    id: 'faq-ports',
    question: 'What ports does NIAC use?',
    answer:
      'On the wire, whichever simulated protocols are enabled (UDP/53 DNS, UDP/67-68 DHCP, UDP/137-138 NetBIOS, UDP/161-162 SNMP, TCP/21 FTP, TCP/80 HTTP, TCP/5201 iperf3, multicast for LLDP/CDP/STP). Host services: 8080 by default for the web UI / REST API, plus whatever you pass to `--metrics-listen` for Prometheus.',
    tags: ['ports', 'network'],
  },
  {
    id: 'faq-multi-ip',
    question: 'Can one device have multiple IPs?',
    answer:
      'Yes. `devices[].ips` is a list — add as many IPv4 and IPv6 addresses as you want. NIAC will ARP / ND for all of them and answer ICMP on each. Useful for modeling secondary subnets, loopback IPs, and dual-stack.',
    tags: ['ip', 'multi-ip', 'ipv6'],
  },
];

// ---------------------------------------------------------------------------
// Convenience re-exports for the existing HelpDrawer code path
// ---------------------------------------------------------------------------

/** Existing FEATURES export name preserved. */
export const FEATURES = features;
/** Existing GLOSSARY export name preserved. */
export const GLOSSARY = glossary;
/** Existing SHORTCUTS export name preserved. */
export const SHORTCUTS = shortcuts;
