/**
 * CommandPalette primitive stories (Wave 5 / niac-W5-2c).
 *
 * Needs React Router for navigation actions. Open state is held by
 * the story so the open/closed variants render as themselves.
 */
import type { Meta, StoryObj } from '@storybook/react-vite';
import { Cog, Layers, Network } from 'lucide-react';
import { useState } from 'react';
import { MemoryRouter } from 'react-router';
import { Button } from './Button';
import { CommandPalette } from './CommandPalette';
import type { SidebarNavGroup } from './Sidebar';

const groups: SidebarNavGroup[] = [
  {
    label: 'Network',
    items: [
      { path: '/devices', label: 'Devices', icon: Network },
      { path: '/topology', label: 'Topology', icon: Network },
    ],
  },
  {
    label: 'Protocols',
    items: [
      { path: '/protocols/dhcp', label: 'DHCP', icon: Layers },
      { path: '/protocols/dns', label: 'DNS', icon: Layers },
      { path: '/protocols/snmp', label: 'SNMP', icon: Layers },
    ],
  },
];

const meta: Meta<typeof CommandPalette> = {
  title: 'UI/CommandPalette',
  component: CommandPalette,
  parameters: { layout: 'fullscreen' },
  decorators: [
    (Story) => (
      <MemoryRouter initialEntries={['/']}>
        <Story />
      </MemoryRouter>
    ),
  ],
};
export default meta;

type Story = StoryObj<typeof CommandPalette>;

const noop = () => undefined;

export const Open: Story = {
  args: {
    open: true,
    onOpenChange: noop,
    groups,
    onOpenSettings: noop,
    onOpenHelp: noop,
    onToggleTheme: noop,
    isDark: true,
  },
};

export const Closed: Story = {
  args: {
    open: false,
    onOpenChange: noop,
    groups,
  },
};

export const WithExtraActions: Story = {
  args: {
    open: true,
    onOpenChange: noop,
    groups,
    extraActions: [
      {
        id: 'refresh-cache',
        label: 'Refresh device cache',
        hint: 'Re-fetches the device registry from the daemon',
        icon: Cog,
        perform: () => undefined,
      },
      {
        id: 'clear-logs',
        label: 'Clear local logs',
        hint: 'Removes session logs (does not touch the daemon log file)',
        icon: Cog,
        perform: () => undefined,
      },
    ],
  },
};

export const InteractiveTrigger: Story = {
  args: { open: false, onOpenChange: noop, groups },
  render: function interactiveRender(args) {
    const [open, setOpen] = useState(false);
    return (
      <div className="p-8">
        <Button onClick={() => setOpen(true)}>Open command palette (⌘K)</Button>
        <CommandPalette {...args} open={open} onOpenChange={setOpen} />
      </div>
    );
  },
};
