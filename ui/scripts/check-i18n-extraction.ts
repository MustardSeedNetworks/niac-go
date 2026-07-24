import { spawnSync } from 'node:child_process';
import { cpSync, existsSync, mkdtempSync, readdirSync, readFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { pathToFileURL } from 'node:url';

const localeRoot = join(import.meta.dirname, '../../internal/i18n/locales');

function jsonFiles(directory: string): string[] {
  return readdirSync(directory).filter((file) => file.endsWith('.json'));
}

export function catalogDrift(expectedDirectory: string, actualDirectory: string): string[] {
  const files = new Set([...jsonFiles(expectedDirectory), ...jsonFiles(actualDirectory)]);
  return [...files]
    .filter((file) => {
      const expected = join(expectedDirectory, file);
      const actual = join(actualDirectory, file);
      return (
        !existsSync(expected) ||
        !existsSync(actual) ||
        readFileSync(expected, 'utf8') !== readFileSync(actual, 'utf8')
      );
    })
    .sort((left, right) => left.localeCompare(right));
}

function main(): void {
  const scratch = mkdtempSync(join(tmpdir(), 'niac-i18n-check-'));
  const scratchLocales = join(scratch, 'locales');

  try {
    cpSync(localeRoot, scratchLocales, { recursive: true });
    const result = spawnSync(
      join(import.meta.dirname, '../node_modules/.bin/i18next-cli'),
      ['extract', '--quiet'],
      {
        cwd: join(import.meta.dirname, '..'),
        env: {
          ...process.env,
          I18NEXT_OUTPUT: join(scratchLocales, '{{language}}', '{{namespace}}.json'),
        },
        encoding: 'utf8',
      },
    );
    if (result.status !== 0) {
      process.stderr.write(result.stderr);
      process.exit(result.status ?? 1);
    }

    const changed = catalogDrift(join(localeRoot, 'en'), join(scratchLocales, 'en'));
    if (changed.length > 0) {
      process.stderr.write(`i18n catalogs are out of date: ${changed.join(', ')}\n`);
      process.exit(1);
    }
  } finally {
    rmSync(scratch, { recursive: true, force: true });
  }
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  main();
}
