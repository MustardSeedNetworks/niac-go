/**
 * ErrorBoundary fallback stories (#2187).
 *
 * The fallbacks only render after a child throws, so nothing in the running
 * app ever put them in front of the a11y addon — which is how both panels
 * shipped painting their text the fill colour underneath it. Each panel is
 * rendered on the real page ground in both modes.
 *
 * `assertReadable` is here because axe is not enough on its own: it reported
 * no colour-contrast violation for the page panel when its text and its
 * background were the SAME token (1.00:1), the worst case of the defect.
 */
import type { Decorator, Meta, StoryObj } from '@storybook/react-vite';
import { useEffect } from 'react';
import { expect } from 'storybook/test';
import { ErrorBoundary, PageErrorBoundary } from './ErrorBoundary';

function Throws(): never {
  // Long enough that a capped or clipping pane would scroll, which is the
  // other half of the defect: a scroll box no keyboard can reach.
  throw new Error(
    `Simulation engine stopped responding while replaying walk 'hospital-2b' on interface eth0 after 41 of 248 devices had been brought up`,
  );
}

type Rgb = readonly [number, number, number];

function channel(v: number): number {
  const c = v / 255;
  return c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
}

function luminance([r, g, b]: Rgb): number {
  return 0.2126 * channel(r) + 0.7152 * channel(g) + 0.0722 * channel(b);
}

function parse(color: string): Rgb | null {
  const m = /rgba?\(([^)]+)\)/.exec(color);
  if (!m?.[1]) return null;
  const parts = m[1]
    .split(/[\s,/]+/)
    .filter(Boolean)
    .map(Number);
  const [r, g, b, a] = parts;
  if (r === undefined || g === undefined || b === undefined) return null;
  // A transparent ground tells us nothing; the walk continues to the parent.
  if (a === 0) return null;
  return [r, g, b];
}

/** The nearest ancestor that actually paints a background. */
function groundOf(el: Element): Rgb {
  for (let node: Element | null = el; node; node = node.parentElement) {
    const rgb = parse(getComputedStyle(node).backgroundColor);
    if (rgb) return rgb;
  }
  return [255, 255, 255];
}

function ratio(a: Rgb, b: Rgb): number {
  const la = luminance(a);
  const lb = luminance(b);
  return (Math.max(la, lb) + 0.05) / (Math.min(la, lb) + 0.05);
}

/** Every text-bearing element in the panel clears WCAG AA (1.4.3). */
async function assertReadable(): Promise<void> {
  const panel = document.querySelector('[data-testid="error-boundary-fallback"], .rounded-lg');
  expect(panel, 'the fallback rendered').not.toBeNull();
  const scope = panel as Element;
  const texts = [scope, ...scope.querySelectorAll('*')].filter((el) =>
    [...el.childNodes].some((n) => n.nodeType === Node.TEXT_NODE && n.textContent?.trim()),
  );
  expect(texts.length, 'the panel carries copy').toBeGreaterThan(0);
  for (const el of texts) {
    const fg = parse(getComputedStyle(el).color);
    const measured = fg ? ratio(fg, groundOf(el)) : 0;
    const label = `${el.tagName.toLowerCase()} — "${el.textContent?.trim().slice(0, 40)}"`;
    expect(Math.round(measured * 100) / 100, label).toBeGreaterThanOrEqual(4.5);
  }
}

/** Paints the story on the page ground so the walk resolves a real background. */
const onPageGround =
  (isDark: boolean): Decorator =>
  (Story) => {
    const Themed = (): React.ReactElement => {
      useEffect(() => {
        const root = document.documentElement;
        const had = root.classList.contains('dark');
        root.classList.toggle('dark', isDark);
        return () => {
          root.classList.toggle('dark', had);
        };
      }, []);
      return (
        <div className="bg-bg-base pad-xl">
          <Story />
        </div>
      );
    };
    return <Themed />;
  };

const meta: Meta<typeof ErrorBoundary> = {
  title: 'Components/ErrorBoundary',
  component: ErrorBoundary,
};
export default meta;

type Story = StoryObj<typeof ErrorBoundary>;

export const FallbackLight: Story = {
  name: 'Fallback (light)',
  decorators: [onPageGround(false)],
  play: assertReadable,
  render: () => (
    <ErrorBoundary>
      <Throws />
    </ErrorBoundary>
  ),
};

export const FallbackDark: Story = {
  name: 'Fallback (dark)',
  decorators: [onPageGround(true)],
  play: assertReadable,
  render: () => (
    <ErrorBoundary>
      <Throws />
    </ErrorBoundary>
  ),
};

export const PageFallbackLight: Story = {
  name: 'Page fallback (light)',
  decorators: [onPageGround(false)],
  play: assertReadable,
  render: () => (
    <PageErrorBoundary>
      <Throws />
    </PageErrorBoundary>
  ),
};

export const PageFallbackDark: Story = {
  name: 'Page fallback (dark)',
  decorators: [onPageGround(true)],
  play: assertReadable,
  render: () => (
    <PageErrorBoundary>
      <Throws />
    </PageErrorBoundary>
  ),
};
