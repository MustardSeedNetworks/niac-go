/**
 * SidebarLayout primitive stories (Wave 5 / niac-W5-2c).
 *
 * SidebarLayout consumes React Router (NavLink) + a SidebarNavGroup
 * array. Wrap in MemoryRouter and pass a synthetic nav tree.
 */
import type { Meta, StoryObj } from '@storybook/react-vite';
import { Activity, Cog, Layers, Network, ScrollText, Sliders, Workflow } from 'lucide-react';
import { MemoryRouter } from 'react-router-dom';
import { SidebarLayout, type SidebarNavGroup } from './Sidebar';

const sampleGroups: SidebarNavGroup[] = [
  {
    label: 'Network',
    items: [
      { path: '/devices', label: 'Devices', icon: Network },
      { path: '/topology', label: 'Topology', icon: Workflow },
    ],
  },
  {
    label: 'Protocols',
    items: [
      { path: '/protocols/dhcp', label: 'DHCP', icon: Layers },
      { path: '/protocols/dns', label: 'DNS', icon: Layers, badge: 'new' },
      { path: '/protocols/snmp', label: 'SNMP', icon: Layers },
    ],
  },
  {
    label: 'Operations',
    items: [
      { path: '/simulation', label: 'Simulation', icon: Activity },
      { path: '/logs', label: 'Logs', icon: ScrollText },
      { path: '/settings', label: 'Settings', icon: Cog },
      { path: '/advanced', label: 'Advanced', icon: Sliders },
    ],
  },
];

const meta: Meta<typeof SidebarLayout> = {
  title: 'UI/SidebarLayout',
  component: SidebarLayout,
  parameters: { layout: 'fullscreen' },
  decorators: [
    (Story, ctx) => {
      const path = (ctx.parameters.path as string | undefined) ?? '/devices';
      return (
        <MemoryRouter initialEntries={[path]}>
          <Story />
        </MemoryRouter>
      );
    },
  ],
};
export default meta;

type Story = StoryObj<typeof SidebarLayout>;

const PageContent = ({ title }: { title: string }) => (
  <div className="p-8">
    <h1 className="text-2xl font-bold text-text-primary">{title}</h1>
    <p className="text-text-muted mt-2">Routed page content goes here.</p>
  </div>
);

export const Default: Story = {
  args: {
    groups: sampleGroups,
    version: 'v2.10.0',
    children: <PageContent title="Devices" />,
  },
};

export const WithBadge: Story = {
  args: {
    groups: sampleGroups,
    children: <PageContent title="DNS" />,
  },
  parameters: { path: '/protocols/dns' },
};

export const NoVersionFooter: Story = {
  args: {
    groups: sampleGroups,
    children: <PageContent title="Simulation" />,
  },
  parameters: { path: '/simulation' },
};
