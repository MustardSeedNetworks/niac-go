import type { Meta, StoryObj } from '@storybook/react';
import { Button } from './Button';
import { Card, CardContent, CardFooter, CardHeader, CardRow, CardValue, StatCard } from './Card';

const meta: Meta<typeof Card> = {
  title: 'UI/Card',
  component: Card,
  tags: ['autodocs'],
  decorators: [
    (Story) => (
      <div className="p-8 bg-gray-950 max-w-md">
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof Card>;

export const Default: Story = {
  render: () => (
    <Card>
      <CardContent>
        <p className="text-white">Basic card content</p>
      </CardContent>
    </Card>
  ),
};

export const WithHeader: Story = {
  render: () => (
    <Card>
      <CardHeader
        actions={
          <Button size="xs" variant="ghost">
            Edit
          </Button>
        }
      >
        <h3 className="text-white font-semibold">Device Status</h3>
      </CardHeader>
      <CardContent className="space-y-2">
        <CardRow label="IP Address" value="192.168.1.1" mono />
        <CardRow label="Status" value="Online" status="success" />
        <CardRow label="Uptime" value="24h 30m" />
      </CardContent>
    </Card>
  ),
};

export const WithFooter: Story = {
  render: () => (
    <Card>
      <CardContent>
        <p className="text-white">Card with footer actions</p>
      </CardContent>
      <CardFooter>
        <div className="flex gap-2">
          <Button size="sm" variant="outline">
            Cancel
          </Button>
          <Button size="sm" tone="violet">
            Save
          </Button>
        </div>
      </CardFooter>
    </Card>
  ),
};

export const Variants: Story = {
  render: () => (
    <div className="space-y-4">
      <Card variant="default">
        <CardContent>
          <p className="text-white">Default</p>
        </CardContent>
      </Card>
      <Card variant="elevated">
        <CardContent>
          <p className="text-white">Elevated</p>
        </CardContent>
      </Card>
      <Card variant="outlined">
        <CardContent>
          <p className="text-white">Outlined</p>
        </CardContent>
      </Card>
      <Card variant="ghost">
        <CardContent>
          <p className="text-white">Ghost</p>
        </CardContent>
      </Card>
    </div>
  ),
};

export const CardValues: Story = {
  render: () => (
    <Card>
      <CardContent className="space-y-3">
        <CardValue label="Packets Sent" value={1234} size="lg" />
        <CardValue label="Errors" value={0} status="success" />
        <CardValue label="Latency" value="12.5" unit="ms" mono />
      </CardContent>
    </Card>
  ),
};

export const Stat: Story = {
  render: () => (
    <div className="grid grid-cols-2 gap-4">
      <StatCard label="Active Devices" value={42} trend={{ value: 12, label: 'vs last week' }} />
      <StatCard label="Errors" value={3} trend={{ value: -5 }} />
    </div>
  ),
};
