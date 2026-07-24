import { describe, expect, it } from 'vitest';
import browserChannelAuthConfig from '../../playwright.channels.auth.config';
import browserChannelsConfig from '../../playwright.channels.config';
import criticalConfig from '../../playwright.config';

function projectNames(config: {
  projects?: Array<{ name?: string }> | readonly { name?: string }[];
}): string[] {
  return config.projects?.map((project) => project.name ?? '') ?? [];
}

describe('browser support matrix', () => {
  it('gates every critical browser engine', () => {
    expect(projectNames(criticalConfig)).toEqual(['chromium', 'webkit', 'firefox']);
  });

  it('tests the installed first-class Chromium browser channels', () => {
    expect(projectNames(browserChannelsConfig)).toEqual(['chrome', 'edge']);
    expect(projectNames(browserChannelAuthConfig)).toEqual(['chrome', 'edge']);
  });
});
