/**
 * Tooltip primitive stories (Wave 5 / #636).
 *
 * All four sides (top/bottom/left/right), text and rich-content
 * bubbles.
 */
import type { Meta, StoryObj } from '@storybook/react-vite';
import { Tooltip } from './Tooltip';

const meta: Meta<typeof Tooltip> = {
  title: 'UI/Tooltip',
  component: Tooltip,
  parameters: { layout: 'centered' },
  argTypes: {
    side: { control: 'select', options: ['top', 'bottom', 'left', 'right'] },
  },
};
export default meta;

type Story = StoryObj<typeof Tooltip>;

const Trigger = () => (
  <span className="inline-block px-3 py-2 rounded border border-border-muted bg-bg-muted/20 text-text-secondary">
    hover me
  </span>
);

export const Top: Story = {
  args: { side: 'top', text: 'Tooltip on top', children: <Trigger /> },
};
export const Bottom: Story = {
  args: { side: 'bottom', text: 'Tooltip on bottom', children: <Trigger /> },
};
export const Left: Story = {
  args: { side: 'left', text: 'Tooltip on left', children: <Trigger /> },
};
export const Right: Story = {
  args: { side: 'right', text: 'Tooltip on right', children: <Trigger /> },
};

export const RichContent: Story = {
  args: {
    side: 'top',
    text: (
      <div className="space-y-1">
        <p className="font-medium">SNMPv3 user</p>
        <p className="text-xs text-text-muted">authPriv with SHA-256/AES-128</p>
      </div>
    ),
    children: <Trigger />,
  },
};

export const NoText: Story = {
  args: { children: <Trigger /> },
  parameters: {
    docs: {
      description: { story: 'When `text` is omitted, the wrapper renders children unchanged.' },
    },
  },
};
