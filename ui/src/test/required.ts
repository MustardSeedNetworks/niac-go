/**
 * required reads a value the test's own setup guarantees is there.
 *
 * Tests are full of `calls[0]`, `instances[0]` and `result[2]` — reads that
 * are safe because the lines above put something there. Under
 * noUncheckedIndexedAccess the compiler cannot see that, and the two usual
 * answers are both bad: a non-null assertion silences the check, and an
 * inline `if (!x) throw` at every site buries the assertion being made.
 *
 * This keeps the check and improves the failure. When the setup did not
 * produce what the test assumed, the error names the thing that was missing
 * rather than surfacing later as "cannot read properties of undefined".
 */
export function required<T>(value: T | undefined | null, what = 'value'): T {
  if (value === undefined || value === null) {
    throw new Error(`expected ${what} to be present, got ${String(value)}`);
  }
  return value;
}
