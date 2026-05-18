/**
 * themeDeviceColors.ts — niac-specific accent colors for topology.
 * Re-exported through theme.ts.
 *
 * All classes reference niac's semantic device/proto/link tokens declared
 * in src/index.css via @theme:
 *   --color-device-* → utilities text-device-router, bg-device-router, ...
 *   --color-proto-*  → utilities text-proto-arp, bg-proto-arp, ...
 *   --color-link-*   → utilities text-link-1g, text-link-trunk, ...
 *
 * To re-theme: change values in index.css; do not hardcode here.
 */

export const deviceColor = {
  router: {
    icon: 'text-device-router',
    badge: 'bg-device-router/20 text-device-router border border-device-router/30',
    border: 'border-device-router/30',
    bg: 'bg-device-router',
  },
  switch: {
    icon: 'text-device-switch',
    badge: 'bg-device-switch/20 text-device-switch border border-device-switch/30',
    border: 'border-device-switch/30',
    bg: 'bg-device-switch',
  },
  firewall: {
    icon: 'text-device-firewall',
    badge: 'bg-device-firewall/20 text-device-firewall border border-device-firewall/30',
    border: 'border-device-firewall/30',
    bg: 'bg-device-firewall',
  },
  server: {
    icon: 'text-device-server',
    badge: 'bg-device-server/20 text-device-server border border-device-server/30',
    border: 'border-device-server/30',
    bg: 'bg-device-server',
  },
  workstation: {
    icon: 'text-device-workstation',
    badge: 'bg-device-workstation/20 text-device-workstation border border-device-workstation/30',
    border: 'border-device-workstation/30',
    bg: 'bg-device-workstation',
  },
  ap: {
    icon: 'text-device-ap',
    badge: 'bg-device-ap/20 text-device-ap border border-device-ap/30',
    border: 'border-device-ap/30',
    bg: 'bg-device-ap',
  },
  iot: {
    icon: 'text-device-iot',
    badge: 'bg-device-iot/20 text-device-iot border border-device-iot/30',
    border: 'border-device-iot/30',
    bg: 'bg-device-iot',
  },
  unknown: {
    icon: 'text-device-unknown',
    badge: 'bg-device-unknown/20 text-device-unknown border border-device-unknown/30',
    border: 'border-device-unknown/30',
    bg: 'bg-device-unknown',
  },
} as const;

export const protocolColor = {
  arp: { icon: 'text-proto-arp', badge: 'bg-proto-arp/20 text-proto-arp' },
  icmp: { icon: 'text-proto-icmp', badge: 'bg-proto-icmp/20 text-proto-icmp' },
  dns: { icon: 'text-proto-dns', badge: 'bg-proto-dns/20 text-proto-dns' },
  dhcp: { icon: 'text-proto-dhcp', badge: 'bg-proto-dhcp/20 text-proto-dhcp' },
  snmp: { icon: 'text-proto-snmp', badge: 'bg-proto-snmp/20 text-proto-snmp' },
  lldp: { icon: 'text-proto-lldp', badge: 'bg-proto-lldp/20 text-proto-lldp' },
  cdp: { icon: 'text-proto-cdp', badge: 'bg-proto-cdp/20 text-proto-cdp' },
  http: { icon: 'text-proto-http', badge: 'bg-proto-http/20 text-proto-http' },
  tcp: { icon: 'text-proto-tcp', badge: 'bg-proto-tcp/20 text-proto-tcp' },
  udp: { icon: 'text-proto-udp', badge: 'bg-proto-udp/20 text-proto-udp' },
} as const;

export const linkSpeedColor = {
  '10m': 'text-link-10m',
  '100m': 'text-link-100m',
  '1g': 'text-link-1g',
  '10g': 'text-link-10g',
  '25g': 'text-link-25g',
  '40g': 'text-link-40g',
  '100g': 'text-link-100g',
  trunk: 'text-link-trunk',
} as const;
