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
  it('gates exactly the engines the product targets', () => {
    expect(projectNames(criticalConfig)).toEqual(['chromium', 'webkit']);
  });
});
