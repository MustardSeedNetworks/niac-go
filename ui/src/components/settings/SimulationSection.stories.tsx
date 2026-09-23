import type { Meta, StoryObj } from '@storybook/react-vite';
import { expect, userEvent, within } from 'storybook/test';
import { useUIStore } from '../../stores/ui-store';
import { LOADED_ROUTES, pageMeta, settled } from '../../test/storybook/pageStory';
import { SimulationSection } from './SimulationSection';

const meta: Meta<typeof SimulationSection> = {
  ...pageMeta('SimulationSection', SimulationSection),
  title: 'Settings/SimulationSection',
  parameters: { api: LOADED_ROUTES },
  beforeEach: () => {
    useUIStore.getState().resetSimulationSettings();
  },
};

export default meta;
type Story = StoryObj<typeof SimulationSection>;

export const Templates: Story = { play: settled() };

export const SavedConfigurations: Story = {
  play: async ({ canvasElement }) => {
    await settled()();
    const canvas = within(canvasElement);
    await userEvent.click(canvas.getByRole('tab', { name: 'My Configs' }));
    await expect(canvas.getByRole('tabpanel', { name: 'My Configs' })).toBeVisible();
  },
};

export const Upload: Story = {
  play: async ({ canvasElement }) => {
    await settled()();
    const canvas = within(canvasElement);
    await userEvent.click(canvas.getByRole('tab', { name: 'Upload' }));
    await expect(canvas.getByRole('tabpanel', { name: 'Upload' })).toBeVisible();
  },
};
