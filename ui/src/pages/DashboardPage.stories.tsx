/**
 * Stories for DashboardPage (/).
 *
 * U6: every route owes an Empty, a Loaded and an Error story, so the CI a11y
 * gate sees the screens operators use rather than only the atoms they are
 * built from. All three run through the shared page harness
 * (src/test/storybook/pageStory.tsx), which supplies the router, the
 * providers, the `<main>` landmark context and the fetch stub.
 */
import type { Meta, StoryObj } from '@storybook/react-vite';
import {
  EMPTY_ROUTES,
  LOADED_ROUTES,
  pageMeta,
  sessionResource,
  withFailure,
} from '../test/storybook/pageStory';
import { DashboardPage } from './DashboardPage';

const meta: Meta<typeof DashboardPage> = {
  ...pageMeta('DashboardPage', DashboardPage),
  parameters: { route: '/' },
};

export default meta;
type Story = StoryObj<typeof DashboardPage>;

/** Fresh install: nothing running and nothing authored. */
export const Empty: Story = { parameters: { api: EMPTY_ROUTES } };

/** The scenario running, with data on every read. */
export const Loaded: Story = { parameters: { api: LOADED_ROUTES } };

/** The rollup that drives every tile returns 500. */
export const Error: Story = { parameters: { api: withFailure(sessionResource('stats')) } };
