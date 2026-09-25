/**
 * pillText.test.ts — a pill's text must be legible on its own wash.
 *
 * A pill is a hue at low alpha (`bg-status-error/10`) under text of the same
 * hue. The wash pulls the ground toward the text, so every hue that clears
 * 4.5:1 on a plain surface drops under it on its own wash (.github#74:
 * 3.45–3.71 at /20 on the worst surface). No alpha fixes that, so each pill
 * hue has a `-strong` text token measured against its wash (owner 2026-09-22,
 * UI-FLEET-3), and a pill uses it instead of the bare hue.
 *
 * The grounds here are the five surface tokens. The page body also carries
 * faint brand and gold gradients (index.css), which only Storybook axe sees.
 */
import { readdirSync, readFileSync } from 'node:fs';
import { join, relative } from 'node:path';
import { describe, expect, it } from 'vitest';
import { contrast, palette, type Rgb, rgb, SURFACES, src } from '../test/palette';

// The strongest wash a pill may use. Every site in the tree is at /20 or below.
const MAX_WASH = 0.2;
const STATUS = ['status-success', 'status-warning', 'status-error', 'status-info'];
const PILL_HUES = [...STATUS, 'brand-primary'];

function wash([hr, hg, hb]: Rgb, [gr, gg, gb]: Rgb, alpha: number): Rgb {
  const mix = (h: number, g: number) => Math.round(h * alpha + g * (1 - alpha));
  return [mix(hr, gr), mix(hg, gg), mix(hb, gb)];
}

describe('pill text tokens', () => {
  for (const mode of ['light', 'dark'] as const) {
    const tokens = palette(mode);
    it.each(PILL_HUES)(`${mode}: text-%s-strong is 4.5:1 on its wash over every surface`, (hue) => {
      const text = tokens.get(`${hue}-strong`);
      const base = tokens.get(hue);
      expect(text, `--color-${hue}-strong is not defined in ${mode} mode`).toBeDefined();
      expect(base).toBeDefined();
      const failing = SURFACES.flatMap((surface) => {
        const ground = tokens.get(`surface-${surface}`);
        if (!(text && base && ground)) return [`surface-${surface} missing`];
        for (let alpha = 0.05; alpha <= MAX_WASH + 1e-9; alpha += 0.05) {
          const ratio = contrast(rgb(text), wash(rgb(base), rgb(ground), alpha));
          if (ratio < 4.5) return [`${surface} /${Math.round(alpha * 100)}: ${ratio.toFixed(2)}`];
        }
        return [];
      });
      expect(failing).toEqual([]);
    });
  }
});

// Brand pills share one family: brand-accent or text-accent text on a brand
// wash has the same problem as brand-primary text on it (text-accent is
// brand-primary in light mode, 4.23:1 on its own /20 wash).
const FAMILY: Record<string, string> = {
  'brand-accent': 'brand-primary',
  'text-accent': 'brand-primary',
};
const HUE = `(status-(?:success|warning|error|info)|brand-(?:primary|accent)|text-accent|module-[a-z]+)`;

function sourceFiles(dir: string): string[] {
  return readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const path = join(dir, entry.name);
    if (entry.isDirectory()) return sourceFiles(path);
    return /\.(tsx?|css)$/.test(entry.name) && !/\.test\.tsx?$/.test(entry.name) ? [path] : [];
  });
}

// What the markup scan reads, one unit per entry. A TS line is its own unit.
// A CSS rule is one unit, reported at its opening line, with its wash and
// text declarations spelled as the utilities they are, so a rule that pairs
// `color-mix(... var(--color-status-error) 15%, transparent)` with
// `color: var(--color-status-error)` is held to the same rule as the markup.
function units(file: string): { line: number; text: string }[] {
  const lines = readFileSync(file, 'utf8').split('\n');
  if (!file.endsWith('.css')) return lines.map((text, i) => ({ line: i + 1, text }));
  const css = lines.join('\n');
  const out: { line: number; text: string }[] = [];
  for (const m of css.matchAll(/\{([^{}]*)\}/g)) {
    const text = (m[1] ?? '')
      .replace(
        new RegExp(
          String.raw`background(?:-color)?:\s*color-mix\(in \w+,\s*var\(--color-${HUE}\)\s*(\d+)%,\s*transparent\)`,
          'g',
        ),
        ' bg-$1/$2 ',
      )
      .replace(new RegExp(String.raw`(?<![\w-])color:\s*var\(--color-${HUE}\)`, 'g'), ' text-$1 ');
    out.push({ line: css.slice(0, m.index).split('\n').length, text });
  }
  return out;
}

describe('pill markup', () => {
  it('same-hue text on a wash uses the -strong token', () => {
    const washRe = new RegExp(String.raw`(?<![\w-])(?:[\w-]+:)*bg-${HUE}/(\d+)(?![\w-])`, 'g');
    const bare = new RegExp(String.raw`(?<![\w-])text-${HUE}(?![\w-])`, 'g');
    const offenders: string[] = [];
    for (const file of sourceFiles(src)) {
      for (const { line: at, text: line } of units(file)) {
        for (const [, washHue = '', alpha = ''] of line.matchAll(washRe)) {
          // /50 and up is a fill, whose text is an on-* token.
          if (Number(alpha) >= 50) continue;
          const family = FAMILY[washHue] ?? washHue;
          const where = `${relative(src, file)}:${at}`;
          for (const [, textHue = ''] of line.matchAll(bare)) {
            if ((FAMILY[textHue] ?? textHue) === family) {
              offenders.push(`${where} text-${textHue} on bg-${washHue}/${alpha}`);
            }
          }
          // A -strong token is measured on its own hue's wash only.
          if (washHue !== family && line.includes(`text-${family}-strong`)) {
            offenders.push(`${where} bg-${washHue}/${alpha} is unmeasured; wash ${family}`);
          }
          if (Number(alpha) > MAX_WASH * 100 && line.includes(`text-${family}-strong`)) {
            offenders.push(`${where} bg-${washHue}/${alpha} is stronger than the measured /20`);
          }
        }
      }
    }
    expect([...new Set(offenders)]).toEqual([]);
  });
});
