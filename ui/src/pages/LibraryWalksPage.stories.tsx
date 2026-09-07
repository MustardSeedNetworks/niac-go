/**
 * Stories for LibraryWalksPage (/library/walks).
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
  settled,
  withFailure,
} from '../test/storybook/pageStory';
import { LibraryWalksPage } from './LibraryFilesPage';

const meta: Meta<typeof LibraryWalksPage> = {
  ...pageMeta('LibraryWalksPage', LibraryWalksPage),
  parameters: { route: '/library/walks' },
};

export default meta;
type Story = StoryObj<typeof LibraryWalksPage>;

/** Fresh install: nothing running and nothing authored. */
export const Empty: Story = { parameters: { api: EMPTY_ROUTES }, play: settled() };

/** The scenario running, with data on every read. */
export const Loaded: Story = { parameters: { api: LOADED_ROUTES }, play: settled() };

/** The walk listing returns 500. */
export const Error: Story = {
  parameters: { api: withFailure('/api/v1/library/walks') },
  play: settled(),
};
