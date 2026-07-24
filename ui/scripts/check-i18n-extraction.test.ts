import { strict as assert } from 'node:assert';
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import test from 'node:test';
import { catalogDrift } from './check-i18n-extraction.ts';

test('reports catalogs created only by the extractor', () => {
  const root = mkdtempSync(join(tmpdir(), 'niac-i18n-drift-test-'));
  const expected = join(root, 'expected');
  const actual = join(root, 'actual');

  try {
    mkdirSync(expected);
    mkdirSync(actual);
    writeFileSync(join(expected, 'common.json'), '{}\n');
    writeFileSync(join(actual, 'common.json'), '{}\n');
    writeFileSync(join(actual, 'rogue.json'), '{}\n');

    assert.deepEqual(catalogDrift(expected, actual), ['rogue.json']);
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});
