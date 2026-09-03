import { isMap, isSeq, type Node, parseDocument } from 'yaml';

/**
 * A single device's slice of the whole-config YAML.
 *
 * The detail pane edits one device, but the daemon only accepts the whole
 * config, so the edit has to be put back where it came from. Doing that
 * through a YAML round-trip would reformat the entire file and drop the
 * comments operators write in it, so the fragment is a *byte range* instead:
 * everything outside `[start, end)` is copied through untouched.
 */
export interface DeviceFragment {
  /** The device's YAML, dedented so the editor shows a standalone document. */
  text: string;
  /** Offset of the first character of the device's block (its `- ` marker). */
  start: number;
  /** Offset just past the device's block. */
  end: number;
  /** Indent the list items sit at, restored on splice. */
  indent: string;
}

/** Offset of the start of the line containing offset. */
function lineStart(source: string, offset: number): number {
  const previous = source.lastIndexOf('\n', offset - 1);
  return previous + 1;
}

/**
 * Offset just past the device's last content line. The parser's value-end can
 * run past trailing blank lines; those belong to the gap between devices, not
 * to the device, so they are excluded — otherwise editing a device silently
 * closes up the spacing around it.
 */
function blockEnd(source: string, start: number, valueEnd: number): number {
  const nextNewline = source.indexOf('\n', valueEnd);
  const end = nextNewline === -1 ? source.length : nextNewline + 1;
  return trimTrailingBlankLines(source, start, end);
}

/** Start of the line the next sequence item begins on, or null when this is
 * the last item. */
function nextItemStart(source: string, next: unknown): number | null {
  const range = (next as Node | undefined)?.range;
  return range ? lineStart(source, range[0]) : null;
}

/** Walks `end` back over blank lines, so the gap between two devices belongs
 * to neither of them and editing one does not close up the spacing. */
function trimTrailingBlankLines(source: string, start: number, end: number): number {
  let trimmed = end;
  while (trimmed > start) {
    const lastLineStart = lineStart(source, trimmed - 1);
    if (source.slice(lastLineStart, trimmed).trim() !== '') {
      break;
    }
    trimmed = lastLineStart;
  }
  return trimmed;
}

/**
 * findDeviceFragment locates one device in the whole-config YAML by name.
 * Returns null when the config does not parse, has no `devices` list, or has
 * no device by that name — the caller falls back to whole-config editing
 * rather than showing a pane built on a guess.
 */
export function findDeviceFragment(configText: string, deviceName: string): DeviceFragment | null {
  const doc = parseDocument(configText);
  if (doc.errors.length > 0) {
    return null;
  }
  const devices = doc.get('devices');
  if (!isSeq(devices)) {
    return null;
  }

  for (const [index, item] of devices.items.entries()) {
    if (!isMap(item) || item.get('name') !== deviceName) {
      continue;
    }
    const range = (item as Node).range;
    if (!range) {
      return null;
    }
    const [nodeStart, valueEnd] = range;
    const start = lineStart(configText, nodeStart);

    // Bounded by where the next device starts, not by scanning forward from
    // this one's value-end. A block map's value-end can already sit on the
    // following line, and scanning from there swallowed the next device's
    // `- name:` line whenever the two were adjacent -- which is how the
    // daemon writes every config it saves, since yaml.Marshal puts no blank
    // line between sequence items. Splicing that fragment back deleted the
    // following device and grafted its fields onto this one.
    const nextStart = nextItemStart(configText, devices.items[index + 1]);
    const end =
      nextStart === null
        ? blockEnd(configText, start, valueEnd)
        : trimTrailingBlankLines(configText, start, nextStart);
    const block = configText.slice(start, end);

    // The first line carries the `- ` marker; the rest are indented to line up
    // past it. Both come off so the pane shows a device, not a list item.
    const markerMatch = /^(\s*)-\s+/.exec(block);
    if (!markerMatch) {
      return null;
    }
    const indent = markerMatch[1] ?? '';
    const bodyIndent = ' '.repeat(markerMatch[0].length);
    const text = block
      .split('\n')
      .map((line, index) =>
        index === 0 ? line.slice(markerMatch[0].length) : stripPrefix(line, bodyIndent),
      )
      .join('\n');

    return { text, start, end, indent };
  }
  return null;
}

/** Removes prefix from line when present; blank lines pass through. */
function stripPrefix(line: string, prefix: string): string {
  return line.startsWith(prefix) ? line.slice(prefix.length) : line;
}

/**
 * spliceDeviceFragment puts an edited device back into the whole config,
 * re-indented to the depth the list uses. Every byte outside the fragment's
 * range is preserved exactly, including comments and the spacing between
 * devices.
 */
export function spliceDeviceFragment(
  configText: string,
  fragment: DeviceFragment,
  replacement: string,
): string {
  const marker = `${fragment.indent}- `;
  const bodyIndent = ' '.repeat(marker.length);
  const lines = replacement.replace(/\n$/, '').split('\n');
  const block = lines
    .map((line, index) => {
      if (line === '') {
        return '';
      }
      return index === 0 ? marker + line : bodyIndent + line;
    })
    .join('\n');

  return `${configText.slice(0, fragment.start)}${block}\n${configText.slice(fragment.end)}`;
}
