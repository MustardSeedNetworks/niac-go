import { describe, expect, it } from 'vitest';
import { newDraftName } from './wizard-draft-name';

describe('newDraftName', () => {
  it('stays unique when two authors start within the same millisecond', () => {
    const now = new Date('2026-09-22T12:34:56.789Z');

    expect(newDraftName(now)).not.toBe(newDraftName(now));
  });
});
