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
  //
  // `edge` joined for the scenario-authoring epic (#1151), whose Definition of
  // Complete names Chrome, Edge and Safari as first-class authoring browsers.
  // Only two of the three were covered. It runs the real installed Edge via
  // `channel: 'msedge'`, not Chromium — the Blink engine is already exercised
  // by the chromium project, so what this adds is Edge's own shell.
  it('gates exactly the engines and form factors the product targets', () => {
    expect(projectNames(criticalConfig)).toEqual([
      'chromium',
      'webkit',
      'firefox',
      'edge',
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

  // The split is the cost control that makes the added projects affordable: the
  // full suite runs on the three desktop engines, and every other project is
  // narrowed to a subset by testMatch. A project that silently lost its filter
  // would run all 26 spec files on four more browsers.
  it('runs the full suite on the desktop engines and a subset everywhere else', () => {
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
