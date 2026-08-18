/**
 * @fileoverview NIAC — per-page help content for the HelpDrawer's Pages tab.
 * @description What each screen is for, where to go next, and the gotchas
 *              worth knowing before using it. Keyed by route path: a route
 *              with an entry here gets a (?) in its page header that opens
 *              the drawer on this content, and pageHelpCoverage.test.ts holds
 *              every route to having one.
 *
 *              This prose used to live as JSX in pageRegistry. It is data now
 *              so the drawer can list and search it alongside the protocol and
 *              device reference, rather than hiding it behind a per-page panel
 *              nothing else could reach.
 *
 *              Text may carry three inline markers, rendered by PageHelpBody:
 *              `code`, **strong**, and [label](https://example.com). Copy is
 *              English-only, as the rest of the drawer's content is.
 *
 * @copyright 2026 Mustard Seed Networks. All rights reserved.
 * @license Proprietary
 */

/** A labelled term and its explanation — the shape of a "what does X do" list. */
export interface PageHelpTerm {
  term: string;
  description: string;
}

/** One renderable unit of a page's help. PageHelpBody switches on `kind`. */
export type PageHelpBlock =
  | { kind: 'paragraph'; text: string }
  | { kind: 'heading'; text: string }
  | { kind: 'terms'; items: PageHelpTerm[] }
  | { kind: 'tips'; items: string[] };

/** Route path -> that page's help. Titles come from the `pages` namespace. */
export const pageHelp: Record<string, PageHelpBlock[]> = {
  '/': [
    {
      kind: 'paragraph',
      text: 'Real-time view of the running daemon: packet counters per protocol, recent simulation runs, device count, and recent alert state.',
    },
    { kind: 'heading', text: 'Where to go from here' },
    {
      kind: 'terms',
      items: [
        { term: 'Simulation', description: 'to start/stop a run.' },
        { term: 'Running Devices', description: 'to see what the running config produced.' },
        { term: 'Packets', description: 'to watch traffic in real time.' },
        { term: 'Alerts', description: 'to set the threshold and webhook target.' },
      ],
    },
  ],
  '/runtime': [
    {
      kind: 'paragraph',
      text: 'Start, stop, and inspect the simulation. Pick a network interface, point to a YAML config, and the daemon spawns the requested device personas on that interface.',
    },
    { kind: 'heading', text: 'Tips' },
    {
      kind: 'tips',
      items: [
        "**Download YAML** exports the running config — equivalent to the legacy CLI's runtime config dump. Useful for capturing the merged result of imports / template usage and committing it to source control.",
        '`capture_playbacks:` in the YAML auto-starts a PCAP replay alongside the sim. **Traffic** lets you switch the playback file at runtime.',
        'Use `lo0` (macOS) / `lo` (Linux) for safe local testing.',
      ],
    },
  ],
  '/new-simulation': [
    {
      kind: 'paragraph',
      text: "A guided path through the existing pieces of the app: pick a starting config, add/edit devices, review the per-device protocol config, then save. Nothing here is new — each step embeds the same page or endpoint used elsewhere (Simulation's config picker, Device Library, Running Devices' protocol table, the Simulation page's preview, and the config-save endpoint).",
    },
    { kind: 'heading', text: 'Why the simulation starts early' },
    {
      kind: 'paragraph',
      text: 'Device editing only works against the daemon\'s currently-running config, so step 1 starts the simulation on the picked interface before handing off to the device editor — there\'s no separate "load without running" step.',
    },
  ],
  '/devices': [
    {
      kind: 'paragraph',
      text: "What's currently running on the daemon, derived from the YAML the daemon was started with. Read-only.",
    },
    { kind: 'heading', text: 'For editing' },
    {
      kind: 'paragraph',
      text: 'To modify saved device definitions, go to **Devices** (under Library). To swap the running network wholesale, restart the simulation from **Simulation**.',
    },
  ],
  '/segments': [
    {
      kind: 'paragraph',
      text: 'Devices grouped by VLAN segment (ADR 0008), exactly as the active YAML config describes them. A flat config with no `segments:` block still reports one "Untagged" segment containing every device. Read-only.',
    },
    { kind: 'heading', text: 'Where to go from here' },
    {
      kind: 'terms',
      items: [
        { term: 'Running Devices', description: 'shows the same devices as a single flat list.' },
        { term: 'Topology', description: 'shows the links between them.' },
      ],
    },
  ],
  '/device-config': [
    {
      kind: 'paragraph',
      text: "Your library of saved device configurations. Edits here update the YAML stored on disk but don't push to the running simulation — start a new simulation to load the changes.",
    },
    { kind: 'heading', text: 'Common actions' },
    {
      kind: 'terms',
      items: [
        { term: 'Click a device', description: 'open the visual editor.' },
        { term: 'Clone', description: 'duplicate a device with a new hostname.' },
        {
          term: 'Bulk select',
          description: 'checkbox in each row, then bulk-delete from the toolbar.',
        },
      ],
    },
  ],
  '/topology': [
    {
      kind: 'paragraph',
      text: 'Visual graph of the configured topology — devices and the links you declared in the YAML ( `trunk_ports:`, `port_channels:`, etc.). Use this to sanity-check your design before starting the simulation.',
    },
    { kind: 'heading', text: 'Live discovery vs design' },
    {
      kind: 'paragraph',
      text: 'The graph shows the **design** from your YAML; the neighbor table shows which adjacencies actually formed at runtime via CDP / LLDP / EDP / FDP — useful for catching mistyped `remote_device` references.',
    },
    { kind: 'heading', text: 'Export' },
    {
      kind: 'paragraph',
      text: 'Topology can be exported from the export button as a PNG image, a JSON snapshot, or — rendered server-side from the running daemon — Graphviz `.dot` or GraphML for use in Graphviz, yEd, or gephi.',
    },
  ],
  '/automation': [
    {
      kind: 'paragraph',
      text: 'The daemon emits an alert webhook when the total packet count crosses the configured threshold. Useful for catching runaway traffic from a misconfigured device persona.',
    },
    { kind: 'heading', text: 'Webhook security' },
    {
      kind: 'paragraph',
      text: 'Outbound webhook destinations are gated by an SSRF defence: raw private / loopback / link-local IPs are rejected, and if the daemon was started with `--webhook-allowed-host`, only those hostnames are allowed. Set the allowlist in production rather than relying on the implicit IP filter — see [SECURITY.md](https://github.com/MustardSeedNetworks/niac-go/blob/main/SECURITY.md).',
    },
    { kind: 'heading', text: 'Disabling alerts' },
    {
      kind: 'paragraph',
      text: 'Leave the threshold blank or set it to 0 to disable packet alerts entirely.',
    },
  ],
  '/traffic': [
    {
      kind: 'paragraph',
      text: 'Inject controlled interface counter faults into the running simulation to test how upstream monitoring reacts. Drives `POST /api/v1/errors`.',
    },
    { kind: 'heading', text: 'Common faults' },
    {
      kind: 'terms',
      items: [
        { term: 'FCS errors', description: 'increments frame-check and inbound error counters.' },
        {
          term: 'Packet discards',
          description: 'increments inbound and outbound discard counters.',
        },
        {
          term: 'Utilization',
          description: 'advances octet counters at a percentage of interface speed.',
        },
      ],
    },
    {
      kind: 'paragraph',
      text: 'Errors clear when you stop the simulation or use the clear action in this panel.',
    },
  ],
  '/debug': [
    {
      kind: 'paragraph',
      text: 'Live log tail from the daemon (Server-Sent Events from `/api/v1/stream/logs`). Pause to freeze the buffer, filter by protocol or severity, set per-protocol debug levels (0=quiet, 3=verbose).',
    },
    { kind: 'heading', text: 'Per-protocol debug levels' },
    {
      kind: 'paragraph',
      text: "Equivalent of the CLI's `--debug-arp` / `--debug-icmp` / `--debug-snmp` family of flags. Levels are applied to the running stack immediately.",
    },
  ],
  '/packets': [
    { kind: 'heading', text: 'Live capture' },
    {
      kind: 'paragraph',
      text: 'Streams every packet the daemon sees from `/api/v1/stream/packets` — hex + decoded fields, BPF filter, freeze frame, save to PCAP. If the page lags, narrow the BPF filter (e.g. limit to a single protocol or device IP).',
    },
    { kind: 'heading', text: 'PCAP files' },
    {
      kind: 'paragraph',
      text: 'Switch to the **PCAP files** tab to open and inspect a captured file — protocol breakdown, per-conversation stats, full per-packet decode. Equivalent of `niac analyze-pcap` on the CLI.',
    },
  ],
  '/config-diff': [
    { kind: 'paragraph', text: 'Two ways to merge two YAML configs:' },
    {
      kind: 'terms',
      items: [
        {
          term: 'Block-by-block (top of page)',
          description:
            'choose left or right for each diff block. Best when reviewing a small targeted change.',
        },
        {
          term: 'Server-side overlay merge (bottom card)',
          description:
            'same semantics as `niac config merge`: overlay devices REPLACE base devices with the same name; new devices are appended; base-only devices are kept. Best when applying a patch to a base config.',
        },
      ],
    },
  ],
  '/walk-validator': [
    {
      kind: 'paragraph',
      text: 'A "walk file" is an `snmpwalk` capture — a flat list of OID = value lines that NIAC replays via its simulated SNMP agent. This page wraps the same validator the CLI\'s `niac analyze-walk` uses.',
    },
    { kind: 'heading', text: 'What "Validate" reports' },
    {
      kind: 'terms',
      items: [
        { term: 'error', description: "line is malformed enough that the parser can't replay it." },
        {
          term: 'warning',
          description: 'likely-wrong field (suspicious type, encoding mismatch).',
        },
        { term: 'info', description: 'stylistic (e.g. unquoted strings). Replay still works.' },
      ],
    },
    { kind: 'heading', text: 'Auto-fix' },
    {
      kind: 'paragraph',
      text: 'Auto-fix rewrites the walk in place. A `.bak` next to the original is created before the rewrite so you can roll back. Re-run Validate after fixing to confirm.',
    },
  ],
  '/walk-analyzer': [
    {
      kind: 'paragraph',
      text: 'Parses a walk file into device identity, interface inventory, and LLDP/CDP neighbor adjacencies — the same engine as `niac analyze-walk`. Read-only; never modifies the file.',
    },
    { kind: 'heading', text: 'What it extracts' },
    {
      kind: 'terms',
      items: [
        {
          term: 'Device',
          description:
            'sysName, sysDescr, sysObjectID, and sysContact/sysLocation when present (SNMPv2-MIB).',
        },
        {
          term: 'Interfaces',
          description:
            'index, name, type, speed, admin/oper status, and MAC address (IF-MIB / ifXTable).',
        },
        { term: 'Neighbors', description: 'adjacencies discovered via LLDP-MIB or CISCO-CDP-MIB.' },
      ],
    },
  ],
  '/library/walks': [
    {
      kind: 'paragraph',
      text: "Shows every `.walk` file under `~/.niac/library/walks/` (or `/var/lib/niac/library/walks/` on packaged installs). Drop files in directly, or run `niac content install` to fetch the published bundle for this binary's version.",
    },
    {
      kind: 'paragraph',
      text: 'The SNMP section on the Device editor uses the same endpoint to populate its walk picker, so anything that shows up here is immediately selectable when configuring a device.',
    },
  ],
  '/library/pcaps': [
    {
      kind: 'paragraph',
      text: 'Shows every capture under `~/.niac/library/pcaps/`. Same ingress paths as the walk library: drop files directly or use `niac content install`.',
    },
    {
      kind: 'paragraph',
      text: 'The Packets and Traffic pages will reuse this listing for their PCAP pickers — the unified library means a PCAP added here is visible everywhere the daemon needs to pick one without extra plumbing.',
    },
  ],
};
