import type { Meta, StoryObj } from '@storybook/react';
import { Mail, Search } from 'lucide-react';
import { Input } from './Input';

const meta: Meta<typeof Input> = {
  title: 'UI/Input',
  component: Input,
  tags: ['autodocs'],
  decorators: [
    (Story) => (
      <div className="p-8 bg-gray-950 max-w-sm">
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof Input>;

export const Default: Story = {
  args: {
    placeholder: 'Enter text...',
  },
};

export const WithLabel: Story = {
  args: {
    label: 'Hostname',
    placeholder: 'e.g. switch-01',
  },
};

export const WithError: Story = {
  args: {
    label: 'IP Address',
    value: 'not-an-ip',
    error: 'Please enter a valid IP address',
  },
};

export const WithHint: Story = {
  args: {
    label: 'MAC Address',
    placeholder: 'AA:BB:CC:DD:EE:FF',
    hint: 'Use colon-separated hex format',
  },
};

export const WithLeftIcon: Story = {
  args: {
    placeholder: 'Search devices...',
    leftIcon: <Search className="h-4 w-4 text-gray-500" />,
  },
};

export const WithRightIcon: Story = {
  args: {
    label: 'Email',
    placeholder: 'admin@example.com',
    rightIcon: <Mail className="h-4 w-4 text-gray-500" />,
  },
};

export const Disabled: Story = {
  args: {
    label: 'Read Only',
    value: '192.168.1.1',
    disabled: true,
  },
};
