import { strict as assert } from 'node:assert';
import { spawnSync } from 'node:child_process';
import { mkdtempSync, readFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { after, before, test } from 'node:test';
import {
  actionPin,
  commitsBetween,
  installReleasePlease,
  pinnedVersions,
  pullRequestNumber,
  type ReleasePlease,
  rejections,
  versionsFromLock,
} from './check-commit-parse.ts';

// b13b73e0's message, verbatim: release-please dropped it from v0.96.0.
const droppedMessage = `fix(security): replace G703 nosec suppressions with os.Root confinement (#2384)

Six gosec G703 (path-traversal taint) suppressions were papering over a
real gap: cmd/niac/install_ca.go, internal/acceptance/harness/harness.go,
internal/protocols/snmp/walk.go (x2), tests/fixtures/history/main.go, and
a pre-existing one in cmd/niac/pathutil.go all "cleaned" a caller-named
path with filepath.Clean/Abs before stat/open/write, but Clean only
resolves ".." lexically rather than rejecting it, and neither is one of
gosec's recognized sanitizers (filepath.Base/Rel, path.Base, strconv
parses; analyzers/pathtraversal.go).

Confine every site through os.Root instead, the same pattern already
used elsewhere in this codebase (internal/library, internal/capture,
internal/support/archive.go, cmd/niac/sanitize.go,
cmd/niac-openapi/main.go) for this exact problem. internal/pathconfine
is a new leaf package factoring the repeated
os.OpenRoot(filepath.Dir(p))+filepath.Base(p) shape into Open/Lstat/Stat/
WriteFile helpers; os.Root methods aren't matched by gosec's Sink list
(receiver-based dispatch), and filepath.Base is one of its own
recognized sanitizers, so the calls are both a real fix and gosec-clean
with no suppression. This also closes a real Lstat-then-Open TOCTOU
symlink race in ParseWalkFile that the old code left open.

Fixes #2383
`;

const scratch = mkdtempSync(join(tmpdir(), 'niac-commit-parse-test-'));
const repo = join(scratch, 'repo');
let releasePlease: ReleasePlease;

function git(...args: string[]): string {
  const result = spawnSync('git', args, { cwd: repo, encoding: 'utf8' });
  assert.equal(result.status, 0, result.stderr);
  return result.stdout.trim();
}

function commit(message: string): string {
  git(
    '-c',
    'user.name=t',
    '-c',
    'user.email=t@example.invalid',
    'commit',
    '-q',
    '--allow-empty',
    '-m',
    message,
  );
  return git('rev-parse', 'HEAD');
}

before(async () => {
  releasePlease = installReleasePlease(await pinnedVersions(), join(scratch, 'modules'));
  spawnSync('git', ['init', '-q', repo]);
  commit('chore: base');
});

after(() => rmSync(scratch, { recursive: true, force: true }));

test('a message the parser rejects fails, naming its line', () => {
  const base = git('rev-parse', 'HEAD');
  const dropped = commit(droppedMessage);
  const found = rejections(commitsBetween(base, dropped, repo), releasePlease);

  assert.equal(found.length, 1);
  assert.equal(found[0].sha, dropped);
  assert.match(found[0].errors[0], /unexpected token '\(' at 17:25/);
  assert.deepEqual(found[0].line, {
    number: 17,
    text: 'os.OpenRoot(filepath.Dir(p))+filepath.Base(p) shape into Open/Lstat/Stat/',
  });
});

test('a normal message passes', () => {
  const base = git('rev-parse', 'HEAD');
  const head = commit('fix(api): keep the session (and its lease) alive (#1)\n\nFixes #1\n');
  assert.deepEqual(rejections(commitsBetween(base, head, repo), releasePlease), []);
});

// #2495's body as merged, less its attribution footer: it named the marker in
// prose, unpaired, and release-please dropped its commit from 0.108.3
// (niac-go#2509).
const unpairedBody = readFileSync(join(import.meta.dirname, 'testdata/pr-2495-body.txt'), 'utf8');
const unpairedSubject =
  'ci(release): fail the merge queue on a commit release-please cannot parse (#2495)';

test('an unpaired override marker in the PR body fails', () => {
  const base = git('rev-parse', 'HEAD');
  const head = commit(`${unpairedSubject}\n\nFixes #2389\n`);
  const [only] = commitsBetween(base, head, repo);
  const found = rejections(
    [{ ...only, pullRequest: { number: 2495, body: unpairedBody } }],
    releasePlease,
  );

  assert.equal(found.length, 1);
  assert.match(
    found[0].errors[0],
    /#2495's body has BEGIN_COMMIT_OVERRIDE with no END_COMMIT_OVERRIDE/,
  );
  assert.ok(found[0].errors.length > 1, 'release-please itself rejects the rest of the body');
});

test('a paired override replaces the message release-please parses', () => {
  const base = git('rev-parse', 'HEAD');
  const head = commit(droppedMessage);
  const [only] = commitsBetween(base, head, repo);
  const body =
    'Prose.\n\nBEGIN_COMMIT_OVERRIDE\nfix(security): confine paths with os.Root (#2384)\nEND_COMMIT_OVERRIDE\n';
  assert.deepEqual(
    rejections([{ ...only, pullRequest: { number: 2384, body } }], releasePlease),
    [],
  );
});

test('a paired override that does not parse fails, naming its line', () => {
  const base = git('rev-parse', 'HEAD');
  const head = commit('fix(api): keep the session alive (#1)\n');
  const [only] = commitsBetween(base, head, repo);
  const body = `Prose.\n\nBEGIN_COMMIT_OVERRIDE\n${droppedMessage}END_COMMIT_OVERRIDE\n`;
  const found = rejections([{ ...only, pullRequest: { number: 1, body } }], releasePlease);

  assert.equal(found.length, 1);
  assert.deepEqual(found[0].line, {
    number: 17,
    text: 'os.OpenRoot(filepath.Dir(p))+filepath.Base(p) shape into Open/Lstat/Stat/',
  });
});

test('reads the PR number from a squash subject', () => {
  assert.equal(pullRequestNumber(`${unpairedSubject}\n\nbody (#7)`), 2495);
  assert.equal(pullRequestNumber('chore: base\n'), undefined);
});

test('reads the action pin from the workflow', () => {
  const sha = 'a'.repeat(40);
  assert.equal(actionPin(`uses: googleapis/release-please-action@${sha} # v5.0.0`), sha);
  assert.throws(() => actionPin('uses: googleapis/release-please-action@v5'), /pin/);
});

test('refuses a lockfile without the parser', () => {
  const lock = JSON.stringify({
    packages: { 'node_modules/release-please': { version: '17.6.0' } },
  });
  assert.throws(() => versionsFromLock(lock), /@conventional-commits\/parser/);
});
