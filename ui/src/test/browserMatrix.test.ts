import { describe, expect, it } from 'vitest';
import authConfig from '../../playwright.auth.config';
import criticalConfig from '../../playwright.config';
import queryConfig from '../../playwright.query.config';
import unsavedConfig from '../../playwright.unsaved.config';

type Project = { name?: string; testMatch?: unknown; testIgnore?: unknown };

function projectNames(config: { projects?: readonly Project[] }): string[] {
  return config.projects?.map((project) => project.name ?? '') ?? [];
}

/**
 * The browser matrix is fleet policy, not a per-repo choice:
 * msn-docs-internal/05-Engineering/E2E_CONVENTIONS.md ("Browser coverage")
 * runs Chromium and WebKit on every PR and nothing else. Chromium stands in
 * for Chrome and Edge (both Blink); WebKit stands in for Safari. Firefox,
 * vendor channels and device presets need an issue and a doc amendment first.
 *
 * niac drifted to eight projects (#1637 Firefox, #1320 three device presets,
 * #1151 installed Edge, #1968 installed Chrome), each citing docs/WEBUI.md
 * rather than the fleet doc, and the two were never reconciled. #2247 brought
 * the matrix back. Exact equality, so drift is loud in both directions.
 */
describe('browser support matrix', () => {
  it('gates exactly the two fleet-policy engines', () => {
    expect(projectNames(criticalConfig)).toEqual(['chromium', 'webkit']);
    expect(criticalConfig.use?.trace).toBe('retain-on-failure');
  });

  // three-way-authoring drives a started simulation and the daemon serves one
  // at a time, so a second engine starting it concurrently answers 409. It
  // runs on chromium only; nothing else is filtered, so a narrow-viewport spec
  // runs on both engines like any other.
  it('runs the full suite on both engines, except the single-session journey', () => {
    const projects = criticalConfig.projects ?? [];
    for (const project of projects) {
      expect(project.testMatch, `${project.name} must not be filtered to a subset`).toBeUndefined();
    }
    const webkit = projects.find((project) => project.name === 'webkit');
    expect(webkit?.testIgnore).toEqual([/three-way-authoring\.spec\.ts/]);
    const chromium = projects.find((project) => project.name === 'chromium');
    expect(chromium?.testIgnore).toBeUndefined();
  });

  // The derived configs spread the base and must not restate a matrix of
  // their own -- that is how the unsaved suite grew a five-engine list.
  it('keeps the derived suites on the same two engines', () => {
    expect(projectNames(authConfig)).toEqual(['chromium', 'webkit']);
    expect(projectNames(queryConfig)).toEqual(['chromium', 'webkit']);
    expect(projectNames(unsavedConfig)).toEqual(['chromium', 'webkit']);
  });
});
