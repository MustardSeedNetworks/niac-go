/**
 * Card primitive stories (Wave 5 / niac-W5-2b).
 *
 * Covers the 4 variants (default/elevated/outlined/ghost), the
 * StatusCard composite, and the CardContent inner wrapper.
 */
import type { Meta, StoryObj } from '@storybook/react-vite';
import { Card, CardContent, StatusCard } from './Card';

const meta: Meta<typeof Card> = {
  title: 'UI/Card',
  component: Card,
  parameters: { layout: 'centered' },
  argTypes: {
    variant: { control: 'select', options: ['default', 'elevated', 'outlined', 'ghost'] },
    hover: { control: 'boolean' },
  },
};
export default meta;

type Story = StoryObj<typeof Card>;

const SampleBody = (
  <CardContent>
    <p className="text-text-primary">Card body content.</p>
    <p className="text-text-muted text-sm mt-1">Secondary detail line.</p>
  </CardContent>
);

export const Default: Story = { args: { variant: 'default', children: SampleBody } };
export const Elevated: Story = { args: { variant: 'elevated', children: SampleBody } };
export const Outlined: Story = { args: { variant: 'outlined', children: SampleBody } };
export const Ghost: Story = { args: { variant: 'ghost', children: SampleBody } };
export const Hoverable: Story = { args: { variant: 'default', hover: true, children: SampleBody } };

export const StatusSuccess: Story = {
  name: 'StatusCard / success',
  render: () => (
    <StatusCard title="Daemon" subtitle="Running for 4h 22m" status="success">
      <p className="text-text-muted text-sm">All protocol bindings nominal.</p>
    </StatusCard>
  ),
};

export const StatusError: Story = {
  name: 'StatusCard / error',
  render: () => (
    <StatusCard title="Daemon" subtitle="Last restart 12s ago" status="error">
      <p className="text-text-muted text-sm">DHCP module failed to bind to eth0.</p>
    </StatusCard>
  ),
};
