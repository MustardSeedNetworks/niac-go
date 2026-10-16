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

/**
 * axe-core 4.14 enables `label-content-name-mismatch` by default. The check
 * reads on-screen text with `visibleVirtual`, and that walk calls
 * `canvas.getContext('2d')` to decide whether a string is an icon ligature.
 * jsdom's `getContext('2d')` returns null, so the check throws
 * `Cannot read properties of null (reading 'canvas')`. axe does not fail the
 * run: it records the rule as incomplete, which is the "could not evaluate
 * rule label-content-name-mismatch" result. The words were never compared.
 *
 * This context reports a width proportional to the string and a non-empty
 * pixel, which is the signal axe treats as ordinary text rather than a
 * ligature. Icons here are SVG, not ligature fonts, so the stub does not
 * hide a real mismatch — a visible label missing from the accessible name
 * still comes back as a violation. Storybook runs the same rule on
 * Chromium's real canvas.
 */
type TextCanvasContext = ((
  this: HTMLCanvasElement,
  contextId: string,
  options?: unknown,
) => RenderingContext | null) & { niacTextCanvas?: boolean };

function installTextMeasurementCanvas(): void {
  const proto = HTMLCanvasElement.prototype;
  const current = proto.getContext as TextCanvasContext;
  if (current.niacTextCanvas) return;

  const getContext: TextCanvasContext = function (contextId, options) {
    if (contextId !== '2d') {
      return current.call(this, contextId, options);
    }
    return {
      canvas: this,
      font: '',
      textAlign: 'left',
      textBaseline: 'top',
      fillText() {},
      clearRect() {},
      measureText(text: string) {
        return { width: Math.max(text.length, 1) * 16 };
      },
      getImageData(_x: number, _y: number, w: number, h: number) {
        const pixels = Math.max(1, Math.floor(w) * Math.floor(h));
        const data = new Uint8ClampedArray(pixels * 4);
        data[0] = 1;
        return { data };
      },
    } as unknown as CanvasRenderingContext2D;
  };
  getContext.niacTextCanvas = true;
  proto.getContext = getContext as typeof proto.getContext;
}

installTextMeasurementCanvas();

/** Default rule set focused on the violations our app cares about. */
const DEFAULT_RULES: RunOptions = {
  rules: {
    // 'region' fires on isolated component renders that lack a landmark. Off
    // for component-scoped tests; re-enable in screen-level harness tests.
    region: { enabled: false },
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
  const unevaluable = results.incomplete.find(
    (result) => result.id === 'label-content-name-mismatch' && result.error,
  );
  if (unevaluable?.error) {
    throw new Error(
      `axe could not evaluate rule label-content-name-mismatch: ${unevaluable.error.message}`,
    );
  }
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
