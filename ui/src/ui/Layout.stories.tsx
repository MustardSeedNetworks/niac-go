/**
 * Layout primitive stories (Wave 5 / niac-W5-2b).
 *
 * Covers PageShell, PageHeader, and StatusIndicator. PrimaryNav is
 * deferred — it needs a React Router context the storybook shell
 * doesn't provide by default.
 */
import type { Meta, StoryObj } from '@storybook/react-vite';
import { Database, Plus } from 'lucide-react';
import { Button } from './Button';
import { PageHeader, PageShell, StatusIndicator } from './Layout';

const meta: Meta = {
  title: 'UI/Layout',
  parameters: { layout: 'fullscreen' },
};
export default meta;

type Story = StoryObj;

export const PageShellExample: Story = {
  name: 'PageShell',
  render: () => (
    <PageShell>
      <div className="p-6 rounded-md border border-border-muted bg-bg-elevated text-text-primary">
        PageShell wraps the routed page with a gradient background and a constrained-width inner
        container.
      </div>
    </PageShell>
  ),
};

export const PageHeaderExample: Story = {
  name: 'PageHeader',
  render: () => (
    <PageShell>
      <PageHeader
        title="Devices"
        description="Simulated hosts visible on the data plane."
        icon={Database}
        actions={<Button leftIcon={<Plus className="w-4 h-4" />}>Add device</Button>}
      />
    </PageShell>
  ),
};

export const StatusIndicatorMatrix: Story = {
  name: 'StatusIndicator / all states',
  render: () => (
    <div className="p-8 flex flex-col gap-3">
      <StatusIndicator status="online" label="Online" />
      <StatusIndicator status="warning" label="Warning" pulse />
      <StatusIndicator status="error" label="Error" />
      <StatusIndicator status="offline" label="Offline" />
      <StatusIndicator status="pending" label="Pending" pulse />
    </div>
  ),
};
