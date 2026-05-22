/**
 * BaseCard primitive stories (Wave 5 / niac-W5-2b).
 *
 * Exercises the loading/error/empty/data render branches. BaseCard
 * is a generic wrapper that derives status from data + a getStatus
 * function and renders via a children render-prop.
 */
import type { Meta, StoryObj } from '@storybook/react-vite';
import { Database } from 'lucide-react';
import { BaseCard } from './BaseCard';

interface Sample {
  ip: string;
  uptime: string;
  packets: number;
}

const meta: Meta<typeof BaseCard<Sample>> = {
  title: 'UI/BaseCard',
  component: BaseCard<Sample>,
  parameters: { layout: 'centered' },
};
export default meta;

type Story = StoryObj<typeof BaseCard<Sample>>;

const sample: Sample = { ip: '192.168.1.42', uptime: '4h 22m', packets: 18234 };

const renderRow = (d: Sample) => (
  <dl className="grid grid-cols-2 gap-1 text-sm">
    <dt className="text-text-muted">IP</dt>
    <dd>{d.ip}</dd>
    <dt className="text-text-muted">Uptime</dt>
    <dd>{d.uptime}</dd>
    <dt className="text-text-muted">Packets</dt>
    <dd>{d.packets.toLocaleString()}</dd>
  </dl>
);

export const WithData: Story = {
  args: {
    title: 'Device',
    subtitle: 'eth0',
    icon: <Database className="w-4 h-4" />,
    data: sample,
    getStatus: () => 'success',
    children: renderRow,
  },
};

export const Loading: Story = {
  args: {
    title: 'Device',
    icon: <Database className="w-4 h-4" />,
    data: null,
    loading: true,
    getStatus: () => 'unknown',
    children: renderRow,
  },
};

export const Empty: Story = {
  args: {
    title: 'Device',
    icon: <Database className="w-4 h-4" />,
    data: null,
    emptyMessage: 'No devices configured',
    getStatus: () => 'unknown',
    children: renderRow,
  },
};

export const ErrorState: Story = {
  name: 'Error',
  args: {
    title: 'Device',
    icon: <Database className="w-4 h-4" />,
    data: null,
    error: 'Failed to fetch device state (connection refused).',
    getStatus: () => 'error',
    children: renderRow,
  },
};
