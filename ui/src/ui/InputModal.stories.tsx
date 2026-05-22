/**
 * InputModal primitive stories (Wave 5 / niac-W5-2b).
 *
 * Single-field prompt modal — covers the tone matrix + defaultValue.
 */
import type { Meta, StoryObj } from '@storybook/react-vite';
import { InputModal } from './InputModal';

const meta: Meta<typeof InputModal> = {
  title: 'UI/InputModal',
  component: InputModal,
  parameters: { layout: 'fullscreen' },
  argTypes: {
    isOpen: { control: 'boolean' },
    submitTone: { control: 'select', options: ['violet', 'blue', 'green', 'red'] },
  },
};
export default meta;

type Story = StoryObj<typeof InputModal>;

const noop = () => undefined;
const onSubmit = (_value: string) => undefined;

export const Default: Story = {
  args: {
    isOpen: true,
    title: 'Rename device',
    message: 'Enter a new name for this device.',
    placeholder: 'router-01',
    submitLabel: 'Rename',
    submitTone: 'violet',
    onSubmit,
    onCancel: noop,
  },
};

export const WithDefaultValue: Story = {
  args: {
    isOpen: true,
    title: 'Edit hostname',
    message: 'This hostname is announced in DHCP, LLDP, and CDP.',
    defaultValue: 'router-01',
    submitLabel: 'Save',
    submitTone: 'blue',
    onSubmit,
    onCancel: noop,
  },
};

export const Destructive: Story = {
  args: {
    isOpen: true,
    title: 'Type DELETE to confirm',
    message: 'This action cannot be undone.',
    placeholder: 'DELETE',
    submitLabel: 'Delete',
    submitTone: 'red',
    onSubmit,
    onCancel: noop,
  },
};
