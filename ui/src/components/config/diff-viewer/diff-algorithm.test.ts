/**
 * The config diff had no tests, and it is an LCS implementation — the kind of
 * code where an off-by-one produces a plausible-looking diff rather than a
 * crash. These are characterization tests: they were written against the
 * behaviour as it stood, before the loops were restructured for
 * noUncheckedIndexedAccess, so that the restructure is provably a refactor.
 */
import { describe, expect, it } from 'vitest';
import { computeDiff, computeLcs } from './diff-algorithm';

describe('computeLcs', () => {
  it('returns the common subsequence in order', () => {
    expect(computeLcs(['a', 'b', 'c', 'd'], ['a', 'x', 'c', 'd'])).toEqual(['a', 'c', 'd']);
  });

  it('is empty when nothing is shared', () => {
    expect(computeLcs(['a', 'b'], ['x', 'y'])).toEqual([]);
  });

  it('handles either side being empty', () => {
    expect(computeLcs([], ['a'])).toEqual([]);
    expect(computeLcs(['a'], [])).toEqual([]);
    expect(computeLcs([], [])).toEqual([]);
  });

  it('prefers the longer subsequence, not the first match', () => {
    /* The greedy answer here is ['a']; the longest is ['b','c']. */
    expect(computeLcs(['a', 'b', 'c'], ['b', 'c', 'a'])).toEqual(['b', 'c']);
  });

  it('keeps duplicates that genuinely appear in both', () => {
    expect(computeLcs(['a', 'a', 'b'], ['a', 'a', 'c'])).toEqual(['a', 'a']);
  });
});

describe('computeDiff', () => {
  it('reports identical input as one unchanged block', () => {
    const blocks = computeDiff('one\ntwo', 'one\ntwo');
    expect(blocks).toHaveLength(1);
    expect(blocks[0]?.type).toBe('unchanged');
    expect(blocks[0]?.leftLines.map((l) => l.content)).toEqual(['one', 'two']);
  });

  it('pairs a replaced line as a modified block', () => {
    const blocks = computeDiff('one\ntwo\nthree', 'one\nCHANGED\nthree');
    const modified = blocks.filter((b) => b.type === 'modified');
    expect(modified).toHaveLength(1);
    expect(modified[0]?.leftLines.map((l) => l.content)).toEqual(['two']);
    expect(modified[0]?.rightLines.map((l) => l.content)).toEqual(['CHANGED']);
  });

  it('reports an insertion as added on the right only', () => {
    const blocks = computeDiff('one\nthree', 'one\ntwo\nthree');
    const added = blocks.filter((b) => b.type === 'added');
    expect(added).toHaveLength(1);
    expect(added[0]?.rightLines.map((l) => l.content)).toEqual(['two']);
    expect(added[0]?.leftLines).toEqual([]);
  });

  it('reports a deletion as removed on the left only', () => {
    const blocks = computeDiff('one\ntwo\nthree', 'one\nthree');
    const removed = blocks.filter((b) => b.type === 'removed');
    expect(removed).toHaveLength(1);
    expect(removed[0]?.leftLines.map((l) => l.content)).toEqual(['two']);
    expect(removed[0]?.rightLines).toEqual([]);
  });

  it('numbers lines from one, per side', () => {
    const blocks = computeDiff('a\nb', 'a\nZ');
    const modified = blocks.find((b) => b.type === 'modified');
    expect(modified?.leftLines[0]?.leftLineNumber).toBe(2);
    expect(modified?.rightLines[0]?.rightLineNumber).toBe(2);
  });

  it('handles an empty side without losing the other', () => {
    const added = computeDiff('', 'only');
    expect(added.flatMap((b) => b.rightLines.map((l) => l.content))).toContain('only');

    const removed = computeDiff('only', '');
    expect(removed.flatMap((b) => b.leftLines.map((l) => l.content))).toContain('only');
  });

  it('gives every block a distinct id', () => {
    const blocks = computeDiff('a\nb\nc\nd', 'a\nX\nc\nY');
    const ids = blocks.map((b) => b.id);
    expect(new Set(ids).size).toBe(ids.length);
  });
});
