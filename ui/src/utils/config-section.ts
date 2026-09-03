import { isMap, type Node, parseDocument, type Scalar } from 'yaml';

/**
 * A top-level section's slice of the whole-config YAML.
 *
 * The same reasoning as device-fragment.ts, one level up. The Networks step
 * owns `networks` and `attachments` but the daemon only accepts the whole
 * config, so the edit has to go back into the file it came from. A YAML
 * round-trip of the whole document would reformat every other section and
 * drop the comments operators write, so a section is a *byte range* and
 * everything outside it is copied through untouched.
 */
export interface ConfigSection {
  /** Offset of the section key's line. */
  start: number;
  /** Offset just past the section's last content line. */
  end: number;
}

/** Offset of the start of the line containing offset. */
function lineStart(source: string, offset: number): number {
  const previous = source.lastIndexOf('\n', offset - 1);
  return previous + 1;
}

/**
 * Offset just past the section's last content line.
 *
 * The parser's value-end runs past trailing blank lines; those are the gap
 * before the next section rather than part of this one, so excluding them
 * keeps the spacing an author wrote from closing up on every save.
 */
function blockEnd(source: string, start: number, valueEnd: number): number {
  const nextNewline = source.indexOf('\n', valueEnd);
  let end = nextNewline === -1 ? source.length : nextNewline + 1;
  while (end > start) {
    const lastLineStart = lineStart(source, end - 1);
    if (source.slice(lastLineStart, end).trim() !== '') {
      break;
    }
    end = lastLineStart;
  }
  return end;
}

/**
 * findConfigSection locates one top-level key's block in the whole-config
 * YAML. Returns null when the config does not parse or has no such key — the
 * caller then appends rather than editing a range it guessed at.
 */
export function findConfigSection(configText: string, key: string): ConfigSection | null {
  const doc = parseDocument(configText);
  if (doc.errors.length > 0 || !isMap(doc.contents)) {
    return null;
  }

  for (const item of doc.contents.items) {
    const itemKey = item.key as Scalar | undefined;
    if (itemKey?.value !== key) {
      continue;
    }
    // The key's own range, not the value's: the block starts at `networks:`,
    // not at the first list entry under it.
    const keyRange = (itemKey as Node).range;
    const valueRange = (item.value as Node | null)?.range ?? keyRange;
    if (!keyRange || !valueRange) {
      return null;
    }
    const start = lineStart(configText, keyRange[0]);
    return { start, end: blockEnd(configText, start, valueRange[1]) };
  }
  return null;
}

/**
 * spliceConfigSection replaces a top-level section, or appends it when the
 * config has none. Passing empty `yamlText` removes the section, which is how
 * clearing every network is expressed — writing `networks:` with nothing
 * under it would author an explicit empty list instead of an absent one.
 *
 * Every byte outside the section's range is preserved, including comments.
 */
export function spliceConfigSection(configText: string, key: string, yamlText: string): string {
  const section = findConfigSection(configText, key);
  const body = yamlText.replace(/\n+$/, '');

  if (!section) {
    if (body === '') {
      return configText;
    }
    const separator = configText === '' || configText.endsWith('\n') ? '' : '\n';
    const spacer = configText.trim() === '' ? '' : '\n';
    return `${configText}${separator}${spacer}${body}\n`;
  }

  if (body === '') {
    // Also consume the blank line that followed the section. blockEnd
    // deliberately leaves it out of the range so a *replacement* keeps the
    // author's spacing, but on removal the gap before the section and the gap
    // after it would both survive and stack up into a double blank line.
    let end = section.end;
    while (end < configText.length) {
      const lineEnd = configText.indexOf('\n', end);
      const stop = lineEnd === -1 ? configText.length : lineEnd + 1;
      if (configText.slice(end, stop).trim() !== '') {
        break;
      }
      end = stop;
    }
    return configText.slice(0, section.start) + configText.slice(end);
  }
  return `${configText.slice(0, section.start)}${body}\n${configText.slice(section.end)}`;
}
