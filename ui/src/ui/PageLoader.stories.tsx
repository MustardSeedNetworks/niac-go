/**
 * PageLoader primitive stories (Wave 5 / niac-W5-2b).
 *
 * Full-page loading shell shown during code-split chunk fetch.
 */
import type { Meta, StoryObj } from '@storybook/react-vite';
import { PageLoader } from './PageLoader';

const meta: Meta<typeof PageLoader> = {
  title: 'UI/PageLoader',
  component: PageLoader,
  parameters: { layout: 'fullscreen' },
};
export default meta;

type Story = StoryObj<typeof PageLoader>;

export const Default: Story = {};
