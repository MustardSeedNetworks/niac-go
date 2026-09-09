import type { Meta, StoryObj } from '@storybook/react-vite';
import { expect } from 'storybook/test';
import { RuntimeRollup } from './RuntimeRollup';

const meta: Meta<typeof RuntimeRollup> = {
  title: 'Runtime/Idle status',
  component: RuntimeRollup,
  args: { simStatus: { running: false, deviceCount: 0, uptimeSeconds: 0 }, loading: false },
  decorators: [
    (Story) => (
      <main>
        <Story />
      </main>
    ),
  ],
  play: async ({ canvas }) => {
    await expect(canvas.getByTestId('status-rollup')).toHaveAttribute('data-state', 'idle');
    await expect(canvas.getByText('Idle')).toBeVisible();
    await expect(canvas.queryByText('All clear')).not.toBeInTheDocument();
  },
};

export default meta;
type Story = StoryObj<typeof RuntimeRollup>;

export const Light: Story = {};
export const Dark: Story = {
  decorators: [
    (Story) => (
      <div className="dark bg-bg-base p-6">
        <Story />
      </div>
    ),
  ],
};
