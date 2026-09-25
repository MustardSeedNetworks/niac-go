/**
 * syntaxText.test.ts — the YAML editor's syntax colours are text, so each
 * one is held to 4.5:1. The editor's background is transparent, so it can
 * sit on any of the five surfaces (niac-go#2192: comments were 2.58:1 and
 * strings 2.49:1 in light mode).
 */
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, it } from 'vitest';
import { contrast, palette, rgb, SURFACES, src } from '../test/palette';

// Every token the editor paints with, read from the editor itself.
const editor = readFileSync(join(src, 'components/config/YamlEditor.tsx'), 'utf8');
const SYNTAX = [
  ...new Set([...editor.matchAll(/var\(--color-syntax-([a-z]+)\)/g)].map(([, name]) => name)),
];

describe('editor syntax tokens', () => {
  it('reads the editor palette', () => {
    expect(SYNTAX).toContain('comment');
  });

  for (const mode of ['light', 'dark'] as const) {
    const tokens = palette(mode);
    it.each(SYNTAX)(`${mode}: syntax-%s is 4.5:1 on every surface`, (name) => {
      const text = tokens.get(`syntax-${name}`);
      expect(text, `--color-syntax-${name} is not defined in ${mode} mode`).toBeDefined();
      const failing = SURFACES.flatMap((surface) => {
        const ground = tokens.get(`surface-${surface}`);
        if (!(text && ground)) return [`surface-${surface} missing`];
        const ratio = contrast(rgb(text), rgb(ground));
        return ratio < 4.5 ? [`${surface}: ${ratio.toFixed(2)}`] : [];
      });
      expect(failing).toEqual([]);
    });
  }
});
