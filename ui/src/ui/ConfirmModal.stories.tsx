/**
 * ConfirmModal primitive stories (Wave 5 / #636).
 *
 * Destructive/neutral/info variants plus a custom-icon and a
 * rich-message example.
 */
import type { Meta, StoryObj } from '@storybook/react-vite';
import { AlertTriangle, Trash2 } from 'lucide-react';
import { ConfirmModal } from './ConfirmModal';

const meta: Meta<typeof ConfirmModal> = {
  title: 'UI/ConfirmModal',
  component: ConfirmModal,
  parameters: { layout: 'fullscreen' },
  argTypes: {
    isOpen: { control: 'boolean' },
    confirmTone: { control: 'select', options: ['red', 'violet', 'blue', 'green'] },
  },
};
export default meta;

type Story = StoryObj<typeof ConfirmModal>;

const noop = () => undefined;

export const DestructiveDelete: Story = {
  args: {
    isOpen: true,
    title: 'Delete device?',
    message: 'This permanently removes the device and its simulation history.',
    confirmLabel: 'Delete',
    cancelLabel: 'Cancel',
    confirmTone: 'red',
    icon: <Trash2 className="w-6 h-6" />,
    onConfirm: noop,
    onCancel: noop,
  },
};

export const Neutral: Story = {
  args: {
    isOpen: true,
    title: 'Restart simulator?',
    message: 'In-flight protocol sessions will be interrupted.',
    confirmLabel: 'Restart',
    confirmTone: 'violet',
    onConfirm: noop,
    onCancel: noop,
  },
};

export const Info: Story = {
  args: {
    isOpen: true,
    title: 'Apply changes?',
    message: 'The configuration takes effect immediately and survives restart.',
    confirmLabel: 'Apply',
    confirmTone: 'blue',
    onConfirm: noop,
    onCancel: noop,
  },
};

export const WithCustomIcon: Story = {
  args: {
    isOpen: true,
    title: 'Switch protocol set?',
    message: 'Active protocol bindings will be dropped and re-initialized.',
    confirmLabel: 'Switch',
    confirmTone: 'violet',
    icon: <AlertTriangle className="w-6 h-6" />,
    onConfirm: noop,
    onCancel: noop,
  },
};

export const RichMessage: Story = {
  args: {
    isOpen: true,
    title: 'Bulk delete devices?',
    message: (
      <div className="space-y-2">
        <p>You're about to delete 18 devices:</p>
        <ul className="list-disc list-inside text-sm text-text-muted">
          <li>15 routers</li>
          <li>3 switches</li>
        </ul>
        <p className="text-status-error text-sm font-medium">This cannot be undone.</p>
      </div>
    ),
    confirmLabel: 'Delete all',
    confirmTone: 'red',
    onConfirm: noop,
    onCancel: noop,
  },
};
