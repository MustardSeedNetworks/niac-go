/**
 * a11y.ts — accessibility test helper.
 *
 * Runs axe-core against rendered output and fails the test on any violation.
 * This is the "axe gate" referenced in the PR-A tooltip/a11y plan: pair every
 * component or screen with an a11y test that calls expectNoAxeViolations() on
 * the rendered container. Catches button-name, image-alt, link-name,
 * aria-allowed-attr, label, and other WCAG rules that no static lint detects.
 *
 * Usage:
 *   const { container } = render(<MyComponent />);
 *   await expectNoAxeViolations(container);
 *
 * The harness is intentionally minimal (raw axe-core, no jest-axe/vitest-axe
 * wrapper) so it ages well with axe-core upgrades and matches the seed/stem/
 * niac "each repo owns its own copy" convention — the file shape is identical
 * across repos.
 */
import axe, { type AxeResults, type Result, type RunOptions } from 'axe-core';

/** Default rule set focused on the violations our app cares about. */
const DEFAULT_RULES: RunOptions = {
  rules: {
    // 'region' fires on isolated component renders that lack a landmark. Off
    // for component-scoped tests; re-enable in screen-level harness tests.
    region: { enabled: false },
    // label-content-name-mismatch cannot run here. axe-core 4.14 enables it
    // by default (WCAG 2.5.3 Label in Name). The check compares visible words
    // to the accessible name, and to ignore icon ligatures it paints text
    // onto a canvas. jsdom's getContext() returns null, so the check throws
    // and axe reports the rule as `incomplete` ("encountered an error") on
    // every control that has visible text — including ones that really do
    // mismatch. Allowing that incomplete would hide the violation.
    //
    // Disabled because this environment cannot evaluate it. Storybook in
    // Chromium and the e2e axe run still enforce the rule. Fix mismatches
    // in the component; do not turn this back on under jsdom.
    'label-content-name-mismatch': { enabled: false },
  },
};

export async function runAxe(
  container: Element,
  options: RunOptions = DEFAULT_RULES,
): Promise<AxeResults> {
  return axe.run(container, options);
}

export async function expectNoAxeViolations(
  container: Element,
  options?: RunOptions,
): Promise<void> {
  const results = await runAxe(container, options);
  if (results.violations.length === 0) return;
  throw new Error(formatViolations(results.violations));
}

function formatViolations(violations: Result[]): string {
  const lines = [`axe found ${violations.length} accessibility violation(s):`];
  for (const v of violations) {
    lines.push(`  [${v.id}] ${v.help}  (${v.nodes.length} node${v.nodes.length === 1 ? '' : 's'})`);
    for (const node of v.nodes.slice(0, 3)) {
      lines.push(`    ${node.html.slice(0, 200)}`);
    }
    lines.push(`    help: ${v.helpUrl}`);
  }
  return lines.join('\n');
}
