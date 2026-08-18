/**
 * StatusBadge primitive stories (Wave 5 / niac-W5-2b).
 *
 * Status matrix × icon/dot variants × sizes.
 */
import type { Meta, StoryObj } from '@storybook/react-vite';
import { StatusBadge } from './StatusBadge';

const meta: Meta<typeof StatusBadge> = {
  title: 'UI/StatusBadge',
  component: StatusBadge,
  parameters: { layout: 'centered' },
  argTypes: {
    status: {
      control: 'select',
      options: ['success', 'warning', 'error', 'loading', 'unknown'],
    },
    variant: { control: 'select', options: ['icon', 'dot'] },
    size: { control: 'select', options: ['sm', 'md', 'lg'] },
  },
};
export default meta;

type Story = StoryObj<typeof StatusBadge>;

export const Success: Story = { args: { status: 'success', variant: 'icon' } };
export const Warning: Story = { args: { status: 'warning', variant: 'icon' } };
export const ErrorState: Story = {
  name: 'Error',
  args: { status: 'error', variant: 'icon' },
};
export const Loading: Story = { args: { status: 'loading', variant: 'icon' } };
export const Unknown: Story = { args: { status: 'unknown', variant: 'icon' } };

export const DotVariant: Story = { args: { status: 'success', variant: 'dot' } };

export const FullMatrix: Story = {
  render: () => (
    <div className="grid grid-cols-6 gap-3 items-center">
      {(['success', 'warning', 'error', 'loading', 'unknown'] as const).map((s) => (
        <StatusBadge key={s} status={s} variant="icon" />
      ))}
      {(['success', 'warning', 'error', 'loading', 'unknown'] as const).map((s) => (
        <StatusBadge key={`dot-${s}`} status={s} variant="dot" />
      ))}
    </div>
  ),
};
