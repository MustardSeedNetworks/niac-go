import { describe, expect, it } from 'vitest';
import authConfig from '../../playwright.auth.config';
import criticalConfig from '../../playwright.config';

function projectNames(config: {
  projects?: Array<{ name?: string }> | readonly { name?: string }[];
}): string[] {
  return config.projects?.map((project) => project.name ?? '') ?? [];
}

describe('browser support matrix', () => {
  // Exact equality, not a subset check: this guard exists to make matrix drift
  // loud in both directions. Adding an engine is a CI-time decision, and
  // quietly dropping one would otherwise look like a passing suite.
  // firefox joined the list in #1637. docs/WEBUI.md had listed it under
  // "Engine CI" and again under Compatibility as the independent-engine
  // coverage, while the config ran Blink and WebKit only -- so the third engine
  // the table promised twice was never exercised. This guard is what makes that
  // addition a recorded decision rather than a silent one.
  //
  // The three small-screen projects joined in #1320. Every project had been a
  // `Desktop *` preset, so no phone or tablet layout was exercised on any run
  // and a mobile-only regression shipped green.
  it('gates exactly the engines and form factors the product targets', () => {
    expect(projectNames(criticalConfig)).toEqual([
      'chromium',
      'webkit',
      'firefox',
      'tablet-safari',
      'mobile-chrome',
      'mobile-safari',
    ]);
  });

  // playwright.auth.config.ts spreads the base config, so it inherits this
  // matrix — and a project's own testMatch beats the config-level one, which
  // silently dragged the small-screen smoke file into the auth suite where it
  // ran against a token-gated daemon and failed nine times. The auth suite is a
  // three-engine check; nothing about it is form-factor dependent.
  it('keeps the auth suite on the desktop engines only', () => {
    expect(projectNames(authConfig)).toEqual(['chromium', 'webkit', 'firefox']);
  });

  // The desktop/small-screen split is the cost control that makes the added
  // projects affordable: the full suite stays on desktop and the small screens
  // run only the smoke subset. A project that silently lost its filter would
  // run all 26 spec files on three more devices.
  it('keeps the full suite on desktop and the smoke subset on small screens', () => {
    const projects = (criticalConfig.projects ?? []) as Array<{
      name?: string;
      testMatch?: unknown;
      testIgnore?: unknown;
    }>;

    for (const project of projects) {
      const isDesktop = ['chromium', 'webkit', 'firefox'].includes(project.name ?? '');
      if (isDesktop) {
        expect(project.testIgnore, `${project.name} must ignore *.mobile.spec.ts`).toBeDefined();
        expect(
          project.testMatch,
          `${project.name} must not be filtered to one file`,
        ).toBeUndefined();
      } else {
        expect(project.testMatch, `${project.name} must match only *.mobile.spec.ts`).toBeDefined();
      }
    }
  });
});
