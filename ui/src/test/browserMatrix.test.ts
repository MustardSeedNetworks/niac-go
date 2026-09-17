import { describe, expect, it } from 'vitest';
import authConfig from '../../playwright.auth.config';
import criticalConfig from '../../playwright.config';

/** Browsers `E2E_CONVENTIONS.md` names explicitly as not to be configured. */
const BANNED_BROWSERS = ['firefox', 'chrome', 'edge', 'msedge', 'tablet', 'mobile'] as const;

function projectNames(config: {
  projects?: Array<{ name?: string }> | readonly { name?: string }[];
}): string[] {
  return config.projects?.map((project) => project.name ?? '') ?? [];
}

describe('browser support matrix', () => {
  // Exact equality, not a subset check: this guard exists to make matrix drift
  // loud in both directions. Quietly dropping an engine would otherwise look
  // like a passing suite, and adding one has to be a recorded decision.
  //
  // The list is not this repo's to choose. `E2E_CONVENTIONS.md` is the single
  // source of truth for all four products: "Chromium AND WebKit run on every
  // PR … No other browsers. Delete Firefox, mobile-chrome, mobile-safari,
  // tablet, edge from playwright.config.ts projects arrays. If a future
  // customer commitment requires another browser, file an issue and amend this
  // doc first." Chromium covers Chrome and Edge — one engine.
  //
  // This test used to assert the opposite: it pinned an eight-project matrix
  // (firefox from #1637, the three small-screen projects from #1320, edge from
  // #1151) and so held niac out of policy while seed, stem and trellis all ran
  // two. Amending the doc first is the route to changing this list (#2246).
  it('gates exactly the two engines the fleet policy allows', () => {
    expect(projectNames(criticalConfig)).toEqual(['chromium', 'webkit']);
  });

  // Named separately from the equality check above so a failure says *why*
  // rather than just printing two arrays: a project named for a banned browser,
  // or a `channel` driving an installed vendor build, is the specific thing the
  // policy forbids.
  it('configures no banned browser, by name or by channel', () => {
    const projects = (criticalConfig.projects ?? []) as Array<{
      name?: string;
      use?: { channel?: string };
    }>;

    for (const project of projects) {
      const name = (project.name ?? '').toLowerCase();
      for (const banned of BANNED_BROWSERS) {
        expect(name, `project "${project.name}" names a browser the policy excludes`).not.toContain(
          banned,
        );
      }
      expect(
        project.use?.channel,
        `project "${project.name}" drives an installed browser channel`,
      ).toBeUndefined();
    }
  });

  it('retains failed attempts for diagnosis', () => {
    expect(criticalConfig.use?.trace).toBe('retain-on-failure');
  });

  // playwright.auth.config.ts derives its projects from this config rather than
  // restating them, so the policy reaches it without a second list to keep in
  // step. It must land on the same two engines.
  it('keeps the auth suite on the same two engines', () => {
    expect(projectNames(authConfig)).toEqual(['chromium', 'webkit']);
  });

  // Small-screen coverage did not leave with the phone and tablet projects: it
  // moved into the spec, which sets a real device preset with
  // `test.use({ ...devices['Pixel 7'] })`. The policy bans browser projects,
  // not device emulation, and a narrow window is not a phone — the user agent,
  // touch support and input modality are what decide whether a control is
  // reachable. Pixel 7 is a Chromium device, so the spec runs under chromium.
  //
  // What this asserts is the consequence for the matrix: no project may be
  // narrowed to a single file any more. A project carrying its own testMatch
  // is how the form-factor projects were kept affordable, and one reappearing
  // means the old matrix is creeping back.
  it('runs the full suite on both engines, with no per-file project', () => {
    const projects = (criticalConfig.projects ?? []) as Array<{
      name?: string;
      testMatch?: unknown;
    }>;

    expect(projects.length).toBeGreaterThan(0);
    for (const project of projects) {
      expect(project.testMatch, `${project.name} must not be filtered to one file`).toBeUndefined();
    }
  });
});
