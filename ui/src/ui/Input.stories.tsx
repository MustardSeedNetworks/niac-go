/**
 * Input primitive stories (Wave 5 / #636).
 *
 * label/hint/error decorations, leftIcon/rightIcon slots, password
 * and disabled states.
 */
import type { Meta, StoryObj } from '@storybook/react-vite';
import { Lock, Mail } from 'lucide-react';
import { Input } from './Input';

const meta: Meta<typeof Input> = {
  title: 'UI/Input',
  component: Input,
  parameters: { layout: 'centered' },
  argTypes: {
    label: { control: 'text' },
    placeholder: { control: 'text' },
    hint: { control: 'text' },
    error: { control: 'text' },
    disabled: { control: 'boolean' },
  },
};
export default meta;

type Story = StoryObj<typeof Input>;

export const Default: Story = { args: { label: 'Username', placeholder: 'admin' } };

export const WithHint: Story = {
  args: { label: 'API token', placeholder: 'nia_…', hint: 'Found in Settings → API' },
};

export const WithError: Story = {
  args: { label: 'Email', placeholder: 'you@example.com', error: 'Email is required' },
};

export const WithLeftIcon: Story = {
  args: { label: 'Email', placeholder: 'you@example.com', leftIcon: <Mail className="w-4 h-4" /> },
};

export const Password: Story = {
  args: {
    label: 'Password',
    type: 'password',
    placeholder: '••••••••',
    leftIcon: <Lock className="w-4 h-4" />,
  },
};

export const Disabled: Story = {
  args: { label: 'Locked field', value: 'cannot edit', disabled: true },
};
