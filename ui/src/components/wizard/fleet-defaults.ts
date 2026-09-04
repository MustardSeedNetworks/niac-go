import { isMap, isSeq, parseDocument, type Scalar } from 'yaml';

/** One discovery protocol's fleet-wide default. */
export interface ProtocolDefault {
  enabled?: boolean;
  interval?: number;
}

export type DiscoveryDefaults = Partial<
  Record<'lldp' | 'cdp' | 'edp' | 'fdp', ProtocolDefault | undefined>
>;

export interface CapturePlayback {
  fileName: string;
  loopTime?: number;
  scaleTime?: number;
}

export interface FleetDefaults {
  discoveryProtocols: DiscoveryDefaults;
  capturePlaybacks: CapturePlayback[];
}

const scalar = (node: unknown): string | null => {
  const value = (node as Scalar | undefined)?.value ?? node;
  return typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean'
    ? String(value)
    : null;
};

const numberOrUndefined = (node: unknown): number | undefined => {
  const value = scalar(node);
  return value === null || value === '' ? undefined : Number(value);
};

function readDiscovery(node: unknown): DiscoveryDefaults {
  const defaults: DiscoveryDefaults = {};
  if (!isMap(node)) return defaults;
  for (const protocol of ['lldp', 'cdp', 'edp', 'fdp'] as const) {
    const entry = node.get(protocol);
    if (!isMap(entry)) continue;
    defaults[protocol] = {
      enabled: scalar(entry.get('enabled')) === 'true',
      interval: numberOrUndefined(entry.get('interval')),
    };
  }
  return defaults;
}

function readPlaybacks(node: unknown): CapturePlayback[] {
  if (!isSeq(node)) return [];
  const playbacks: CapturePlayback[] = [];
  for (const item of node.items) {
    if (!isMap(item)) continue;
    const fileName = scalar(item.get('file_name'));
    if (!fileName) continue;
    playbacks.push({
      fileName,
      loopTime: numberOrUndefined(item.get('loop_time')),
      scaleTime: numberOrUndefined(item.get('scale_time')),
    });
  }
  return playbacks;
}

/** parseFleetDefaults reads the two config sections that had no UI control. */
export function parseFleetDefaults(configText: string): FleetDefaults {
  const doc = parseDocument(configText);
  if (doc.errors.length > 0 || !isMap(doc.contents)) {
    return { discoveryProtocols: {}, capturePlaybacks: [] };
  }
  return {
    discoveryProtocols: readDiscovery(doc.get('discovery_protocols')),
    capturePlaybacks: readPlaybacks(doc.get('capture_playbacks')),
  };
}

/** Serializes the discovery defaults, or '' when no protocol is enabled — the
 * caller splices '' as a removal rather than authoring an empty block. */
export function serializeDiscoveryProtocols(defaults: DiscoveryDefaults): string {
  const lines: string[] = [];
  for (const protocol of ['lldp', 'cdp', 'edp', 'fdp'] as const) {
    const entry = defaults[protocol];
    if (!entry?.enabled) continue;
    lines.push(`  ${protocol}:`);
    lines.push('    enabled: true');
    if (entry.interval !== undefined && !Number.isNaN(entry.interval)) {
      lines.push(`    interval: ${entry.interval}`);
    }
  }
  return lines.length === 0 ? '' : `discovery_protocols:\n${lines.join('\n')}\n`;
}

/**
 * Serializes the capture playback list.
 *
 * At most one entry is emitted: the runtime plays exactly one capture, and
 * the loader refuses more than one, so writing a second would author a config
 * that fails validation.
 */
export function serializeCapturePlaybacks(playbacks: readonly CapturePlayback[]): string {
  const first = playbacks[0];
  if (!first || first.fileName.trim() === '') return '';
  const lines = ['capture_playbacks:', `  - file_name: ${first.fileName}`];
  if (first.loopTime !== undefined && !Number.isNaN(first.loopTime)) {
    lines.push(`    loop_time: ${first.loopTime}`);
  }
  if (first.scaleTime !== undefined && !Number.isNaN(first.scaleTime)) {
    lines.push(`    scale_time: ${first.scaleTime}`);
  }
  return `${lines.join('\n')}\n`;
}
