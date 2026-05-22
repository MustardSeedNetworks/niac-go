/**
 * Skeleton primitive stories (Wave 5 / niac-W5-2b).
 *
 * All variants (text/circular/rectangular), sizing, multi-line text.
 */
import type { Meta, StoryObj } from '@storybook/react-vite';
import { Skeleton } from './Skeleton';

const meta: Meta<typeof Skeleton> = {
  title: 'UI/Skeleton',
  component: Skeleton,
  parameters: { layout: 'centered' },
  argTypes: {
    variant: { control: 'select', options: ['text', 'circular', 'rectangular'] },
  },
};
export default meta;

type Story = StoryObj<typeof Skeleton>;

export const Text: Story = { args: { variant: 'text', width: 220 } };
export const TextMultiLine: Story = { args: { variant: 'text', lines: 4, width: 280 } };
export const Circular: Story = { args: { variant: 'circular', width: 48, height: 48 } };
export const Rectangular: Story = { args: { variant: 'rectangular', width: 280, height: 120 } };

export const CardSkeleton: Story = {
  name: 'Composed: card placeholder',
  render: () => (
    <div className="w-72 p-4 border border-border-muted rounded-md space-y-3">
      <div className="flex items-center gap-3">
        <Skeleton variant="circular" width={40} height={40} />
        <div className="flex-1 space-y-2">
          <Skeleton variant="text" width="60%" />
          <Skeleton variant="text" width="40%" />
        </div>
      </div>
      <Skeleton variant="rectangular" height={80} />
      <Skeleton variant="text" lines={3} />
    </div>
  ),
};
