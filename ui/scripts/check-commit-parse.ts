// release-please drops any commit its conventional-commit parser rejects, and
// says so only at debug level. b13b73e0 lost its changelog entry to a body line
// with nested parentheses, and v0.96.0 stalled until a BEGIN_COMMIT_OVERRIDE
// (niac-go#2389). The release-notes guard catches the short changelog after the
// fact; this runs release-please's own parsing over the commits a merge-queue
// group would land, so the commit is caught before it reaches main.
//
// release-please also replaces a commit's message with the text after
// BEGIN_COMMIT_OVERRIDE in its merged PR's body, paired or not (niac-go#2509:
// #2495 named the marker in prose and its commit was dropped from 0.108.3).
// Each commit is therefore parsed together with its PR's body, and a marker
// with no END_COMMIT_OVERRIDE after it is refused outright.
//
// The release-please and parser versions come from the lockfile of the
// release-please-action commit that release-please.yml pins, so a bump there
// moves this check with it.
//
// Usage: GITHUB_REPOSITORY=owner/name GITHUB_TOKEN=… \
//          node ui/scripts/check-commit-parse.ts <base-sha> <head-sha>
import { spawnSync } from 'node:child_process';
import { mkdtempSync, readFileSync, rmSync } from 'node:fs';
import { createRequire } from 'node:module';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { pathToFileURL } from 'node:url';

const workflowPath = join(import.meta.dirname, '../../.github/workflows/release-please.yml');

interface Logger {
  error: (message: string) => void;
  warn: (message: string) => void;
  info: (message: string) => void;
  debug: (message: string) => void;
  trace: (message: string) => void;
}

// The fields of release-please's PullRequest that its commit parsing reads.
export interface PullRequest {
  number: number;
  body: string;
}

export interface RawCommit {
  sha: string;
  message: string;
  files: string[];
  pullRequest?: PullRequest;
}

export interface ReleasePlease {
  parseConventionalCommits: (commits: RawCommit[], logger: Logger) => unknown[];
  parser: (message: string) => unknown;
}

export interface Versions {
  releasePlease: string;
  parser: string;
}

export interface Rejection {
  sha: string;
  subject: string;
  errors: string[];
  line?: { number: number; text: string };
}

export function actionPin(workflow: string): string {
  const pin = /googleapis\/release-please-action@([0-9a-f]{40})/.exec(workflow);
  if (!pin) {
    throw new Error('release-please.yml does not pin googleapis/release-please-action to a SHA');
  }
  return pin[1];
}

export function versionsFromLock(lockText: string): Versions {
  const lock = JSON.parse(lockText) as { packages?: Record<string, { version?: string }> };
  const version = (name: string): string => {
    const found = lock.packages?.[`node_modules/${name}`]?.version;
    if (!found) {
      throw new Error(`release-please-action lockfile has no ${name}`);
    }
    return found;
  };
  return {
    releasePlease: version('release-please'),
    parser: version('@conventional-commits/parser'),
  };
}

// A squash commit's subject ends with its PR number; that PR's body is what
// release-please will read the override from.
export function pullRequestNumber(message: string): number | undefined {
  const found = /\(#(\d+)\)$/.exec(message.split('\n')[0]);
  return found ? Number(found[1]) : undefined;
}

export async function fetchPullRequest(
  repository: string,
  number: number,
  token: string,
): Promise<PullRequest> {
  const response = await fetch(`https://api.github.com/repos/${repository}/pulls/${number}`, {
    headers: { accept: 'application/vnd.github+json', authorization: `Bearer ${token}` },
  });
  if (!response.ok) {
    throw new Error(`fetching ${repository}#${number}: HTTP ${response.status}`);
  }
  const pull = (await response.json()) as { body: string | null };
  return { number, body: pull.body ?? '' };
}

// The message release-please parses for a commit: release-please's own
// preprocessCommitMessage, restated only to quote the rejected line.
export function effectiveMessage(commit: RawCommit): string {
  const override = (commit.pullRequest?.body.split('BEGIN_COMMIT_OVERRIDE')[1] ?? '')
    .split('END_COMMIT_OVERRIDE')[0]
    .trim();
  return override || commit.message;
}

export function unpairedOverride(body: string): boolean {
  const after = body.split('BEGIN_COMMIT_OVERRIDE');
  return after.length > 1 && !after[1].includes('END_COMMIT_OVERRIDE');
}

export async function pinnedVersions(): Promise<Versions> {
  const pin = actionPin(readFileSync(workflowPath, 'utf8'));
  const response = await fetch(
    `https://raw.githubusercontent.com/googleapis/release-please-action/${pin}/package-lock.json`,
  );
  if (!response.ok) {
    throw new Error(`fetching the release-please-action lockfile: HTTP ${response.status}`);
  }
  return versionsFromLock(await response.text());
}

export function installReleasePlease(versions: Versions, prefix: string): ReleasePlease {
  const result = spawnSync(
    'npm',
    [
      'install',
      '--prefix',
      prefix,
      '--no-save',
      '--ignore-scripts',
      '--no-audit',
      '--no-fund',
      '--loglevel=error',
      `release-please@${versions.releasePlease}`,
      `@conventional-commits/parser@${versions.parser}`,
    ],
    { encoding: 'utf8' },
  );
  if (result.status !== 0) {
    throw new Error(`npm install failed:\n${result.stderr}`);
  }
  const load = createRequire(join(prefix, 'package.json'));
  const commit = load('release-please/build/src/commit') as Pick<
    ReleasePlease,
    'parseConventionalCommits'
  >;
  const parser = load('@conventional-commits/parser') as Pick<ReleasePlease, 'parser'>;
  return { parseConventionalCommits: commit.parseConventionalCommits, parser: parser.parser };
}

export function commitsBetween(base: string, head: string, cwd: string): RawCommit[] {
  const result = spawnSync('git', ['log', '--format=%H%x00%B%x1e', `${base}..${head}`], {
    cwd,
    encoding: 'utf8',
  });
  if (result.status !== 0) {
    throw new Error(`git log ${base}..${head} failed:\n${result.stderr}`);
  }
  return result.stdout
    .split('\x1e')
    .map((record) => record.replace(/^\n/, ''))
    .filter((record) => record.length > 0)
    .map((record) => {
      const [sha, message] = record.split('\x00');
      return { sha, message, files: [] };
    });
}

// release-please splits a message into parts and drops each part it cannot
// parse, so a commit can lose a nested entry while its header survives. Every
// logged parse error is a loss. The parser's position is quoted only when it
// refers to the whole message, which it does unless a nested part failed.
export function rejections(commits: RawCommit[], releasePlease: ReleasePlease): Rejection[] {
  const found: Rejection[] = [];
  for (const commit of commits) {
    const errors: string[] = [];
    if (commit.pullRequest && unpairedOverride(commit.pullRequest.body)) {
      errors.push(
        `#${commit.pullRequest.number}'s body has BEGIN_COMMIT_OVERRIDE with no ` +
          'END_COMMIT_OVERRIDE after it, so release-please replaces this message with ' +
          'the rest of the body',
      );
    }
    const quiet = (): void => {};
    const logger: Logger = {
      error: quiet,
      warn: quiet,
      info: quiet,
      trace: quiet,
      debug: (message) => {
        if (message.startsWith('error message: ')) {
          errors.push(message.slice('error message: '.length));
        }
      },
    };
    releasePlease.parseConventionalCommits([commit], logger);
    if (errors.length === 0) {
      continue;
    }
    const rejection: Rejection = {
      sha: commit.sha,
      subject: commit.message.split('\n')[0],
      errors,
    };
    const message = effectiveMessage(commit);
    try {
      releasePlease.parser(message);
    } catch (error) {
      const at = / at (\d+):\d+/.exec(String(error));
      if (at) {
        const number = Number(at[1]);
        rejection.line = { number, text: message.split('\n')[number - 1] ?? '' };
      }
    }
    found.push(rejection);
  }
  return found;
}

export function report(found: Rejection[]): string {
  return found
    .map((rejection) => {
      const lines = [`${rejection.sha.slice(0, 8)} ${rejection.subject}`];
      for (const error of rejection.errors) {
        lines.push(`  release-please drops it: ${error}`);
      }
      if (rejection.line) {
        lines.push(`  line ${rejection.line.number}: ${rejection.line.text}`);
      }
      return lines.join('\n');
    })
    .join('\n');
}

async function main(): Promise<number> {
  const [base, head] = process.argv.slice(2);
  if (!base || !head) {
    process.stderr.write('usage: check-commit-parse.ts <base-sha> <head-sha>\n');
    return 2;
  }
  const repository = process.env.GITHUB_REPOSITORY;
  const token = process.env.GITHUB_TOKEN;
  if (!repository || !token) {
    process.stderr.write('GITHUB_REPOSITORY and GITHUB_TOKEN must be set\n');
    return 2;
  }
  const versions = await pinnedVersions();
  const prefix = mkdtempSync(join(tmpdir(), 'niac-commit-parse-'));
  try {
    const commits = commitsBetween(base, head, process.cwd());
    for (const commit of commits) {
      const number = pullRequestNumber(commit.message);
      if (number !== undefined) {
        commit.pullRequest = await fetchPullRequest(repository, number, token);
      }
    }
    const found = rejections(commits, installReleasePlease(versions, prefix));
    process.stdout.write(
      `release-please ${versions.releasePlease}, parser ${versions.parser}: ` +
        `${commits.length} commit(s) checked\n`,
    );
    if (found.length === 0) {
      return 0;
    }
    process.stderr.write(
      `${report(found)}\n\nThe squash message is built from the PR's commit messages, ` +
        'or from its BEGIN_COMMIT_OVERRIDE block when the body has one. Reword the ' +
        'rejected line there, or pair the marker, and queue the PR again.\n',
    );
    return 1;
  } finally {
    rmSync(prefix, { recursive: true, force: true });
  }
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  process.exitCode = await main();
}
