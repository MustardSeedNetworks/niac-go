type TagColor = 'blue' | 'green' | 'yellow' | 'purple' | 'gray' | 'red';

const PROTOCOL_COLOR_MAP: Record<string, TagColor> = {
  arp: 'yellow',
  icmp: 'blue',
  dns: 'green',
  tcp: 'purple',
  udp: 'gray',
  http: 'blue',
  https: 'green',
  dhcp: 'yellow',
  ssh: 'red',
  tls: 'green',
};

export function getProtocolColor(protocol: string): TagColor {
  return PROTOCOL_COLOR_MAP[protocol.toLowerCase()] ?? 'gray';
}
