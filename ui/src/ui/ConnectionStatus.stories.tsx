import type { Meta, StoryObj } from '@storybook/react-vite';
import { ConnectionStatus } from './ConnectionStatus';

const meta: Meta<typeof ConnectionStatus> = {
  title: 'UI/ConnectionStatus',
  component: ConnectionStatus,
  parameters: { layout: 'centered' },
  args: { status: 'connected' },
};
export default meta;

type Story = StoryObj<typeof ConnectionStatus>;

export const Connected: Story = {};
export const Disconnected: Story = { args: { status: 'disconnected' } };
export const Checking: Story = { args: { status: 'checking' } };
export const Compact: Story = { args: { compact: true } };
