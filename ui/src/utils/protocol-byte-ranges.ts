import type { ProtocolLayer } from './protocol-layers';

export interface PacketByteRange {
  layer: string;
  field: string;
  start: number;
  end: number;
}

const layerNames: Record<string, string> = {
  ethernet: 'Ethernet II',
  dot1q: '802.1Q',
  ipv4: 'Internet Protocol Version 4 (IPv4)',
  ipv6: 'Internet Protocol Version 6 (IPv6)',
  tcp: 'Transmission Control Protocol (TCP)',
  udp: 'User Datagram Protocol (UDP)',
  icmp: 'Internet Control Message Protocol (ICMP)',
};

export function applyByteRanges(layers: ProtocolLayer[], ranges: PacketByteRange[] = []): void {
  for (const range of ranges) {
    const name = layerNames[range.layer] ?? range.layer;
    let layer = layers.find((item) => item.name === name);
    if (!layer) {
      layer = { name, fields: [] };
      layers.push(layer);
    }
    if (!range.field) {
      layer.byteEnd = range.end;
      continue;
    }
    const field = layer.fields.find((item) => item.name === range.field);
    if (field) {
      field.byteStart = range.start;
      field.byteEnd = range.end;
    } else if (range.field === 'Options') {
      layer.fields.push({
        name: range.field,
        value: `${range.end - range.start} bytes`,
        byteStart: range.start,
        byteEnd: range.end,
      });
    }
  }
}
