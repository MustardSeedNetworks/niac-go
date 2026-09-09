import { spawn } from 'node:child_process';
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

const directory = mkdtempSync(join(tmpdir(), 'niac-scope-e2e-'));
const tokenFile = join(directory, 'tokens.json');
const tokens = ['read-only', 'read-write', 'admin'].map((scope) => ({
  scope,
  value: `niac-scope-test-${scope}`,
}));
writeFileSync(tokenFile, JSON.stringify({ tokens }), { mode: 0o600 });
const child = spawn(
  '../niac',
  ['daemon', '--listen', '127.0.0.1:18447', '--storage', 'disabled', '--token-file', tokenFile],
  {
    stdio: 'inherit',
    env: {
      ...process.env,
      NIAC_E2E_DRY_RUN_SIMULATION: '1',
      NIAC_LIBRARY_ROOT: join(directory, 'library'),
      NIAC_CONFIGS_DIR: join(directory, 'configs'),
    },
  },
);
for (const signal of ['SIGINT', 'SIGTERM'] as const) {
  process.on(signal, () => child.kill(signal));
}
child.on('exit', (code) => {
  rmSync(directory, { recursive: true });
  process.exit(code ?? 1);
});
child.on('error', (error) => {
  rmSync(directory, { recursive: true });
  throw error;
});
