import { EventEmitter } from 'node:events';
import type { APIRequestContext } from '@playwright/test';
import { beforeEach, expect, it, vi } from 'vitest';
import { withIsolatedDaemon } from '../../e2e/isolated-daemon';

const mocks = vi.hoisted(() => ({ spawn: vi.fn(), rm: vi.fn(), check: vi.fn() }));
vi.mock('node:child_process', () => ({ spawn: mocks.spawn, default: { spawn: mocks.spawn } }));
vi.mock('node:fs/promises', () => {
  const methods = { mkdtemp: async () => '/tmp/niac-fixture-test', rm: mocks.rm };
  return { ...methods, default: methods };
});
vi.mock('@playwright/test', () => ({
  expect: { poll: (callback: () => Promise<boolean>) => ({ toBe: async () => callback() }) },
}));

beforeEach(() => {
  vi.clearAllMocks();
});

it('removes isolated files after a spawn failure without running the check', async () => {
  const child = new EventEmitter();
  Object.assign(child, { exitCode: null, kill: vi.fn() });
  mocks.spawn.mockImplementation(() => {
    queueMicrotask(() => {
      child.emit('error', new Error('spawn failed'));
      child.emit('close');
    });
    return child;
  });
  const request = { get: async () => ({ ok: () => false }) } as unknown as APIRequestContext;
  await expect(withIsolatedDaemon(request, mocks.check)).rejects.toThrow('spawn failed');
  expect(mocks.check).not.toHaveBeenCalled();
  expect(mocks.rm).toHaveBeenCalledWith('/tmp/niac-fixture-test', { recursive: true });
});

it('bounds a stuck shutdown and still removes isolated files', async () => {
  vi.useFakeTimers();
  const child = new EventEmitter();
  const kill = vi.fn((signal: string) => {
    if (signal === 'SIGKILL') child.emit('close');
    return true;
  });
  Object.assign(child, { exitCode: null, signalCode: null, kill });
  mocks.spawn.mockImplementation(() => {
    queueMicrotask(() => child.emit('spawn'));
    return child;
  });
  const request = { get: async () => ({ ok: () => true }) } as unknown as APIRequestContext;
  try {
    const pending = withIsolatedDaemon(request, mocks.check);
    const assertion = expect(pending).rejects.toThrow('shutdown');
    await vi.waitFor(() => expect(kill).toHaveBeenCalledWith('SIGTERM'));
    await vi.advanceTimersByTimeAsync(5000);
    await assertion;
    expect(kill).toHaveBeenCalledWith('SIGKILL');
    expect(mocks.rm).toHaveBeenCalledWith('/tmp/niac-fixture-test', { recursive: true });
  } finally {
    vi.useRealTimers();
  }
});
