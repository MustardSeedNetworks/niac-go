import { describe, expect, it } from 'vitest';
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
  it('gates exactly the engines the product targets', () => {
    expect(projectNames(criticalConfig)).toEqual(['chromium', 'webkit', 'firefox']);
  });
});
