// release-please drops any commit its conventional-commit parser rejects, and
// says so only at debug level. b13b73e0 lost its changelog entry to a body line
// with nested parentheses, and v0.96.0 stalled until a BEGIN_COMMIT_OVERRIDE
// (niac-go#2389). The release-notes guard catches the short changelog after the
// fact; this runs release-please's own parsing over the commits a merge-queue
// group would land, so the commit is caught before it reaches main.
//
// The release-please and parser versions come from the lockfile of the
// release-please-action commit that release-please.yml pins, so a bump there
// moves this check with it.
//
// Usage: node ui/scripts/check-commit-parse.ts <base-sha> <head-sha>
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

interface RawCommit {
  sha: string;
  message: string;
  files: string[];
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
    try {
      releasePlease.parser(commit.message);
    } catch (error) {
      const at = / at (\d+):\d+/.exec(String(error));
      if (at) {
        const number = Number(at[1]);
        rejection.line = { number, text: commit.message.split('\n')[number - 1] ?? '' };
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
  const versions = await pinnedVersions();
  const prefix = mkdtempSync(join(tmpdir(), 'niac-commit-parse-'));
  try {
    const commits = commitsBetween(base, head, process.cwd());
    const found = rejections(commits, installReleasePlease(versions, prefix));
    process.stdout.write(
      `release-please ${versions.releasePlease}, parser ${versions.parser}: ` +
        `${commits.length} commit(s) checked\n`,
    );
    if (found.length === 0) {
      return 0;
    }
    process.stderr.write(
      `${report(found)}\n\nThe squash message is built from the PR's commit messages. ` +
        'Reword the rejected line in those commits, push, and queue the PR again.\n',
    );
    return 1;
  } finally {
    rmSync(prefix, { recursive: true, force: true });
  }
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  process.exitCode = await main();
}
