/**
 * Alert primitive stories (Wave 5 / #636).
 *
 * All 4 statuses, dismissible variant, rich-content body.
 */
import type { Meta, StoryObj } from '@storybook/react-vite';
import { Alert } from './Alert';

const meta: Meta<typeof Alert> = {
  title: 'UI/Alert',
  component: Alert,
  parameters: { layout: 'centered' },
  argTypes: {
    status: { control: 'select', options: ['success', 'error', 'warning', 'info'] },
  },
};
export default meta;

type Story = StoryObj<typeof Alert>;

export const Success: Story = {
  args: { status: 'success', children: 'Configuration saved successfully.' },
};
export const ErrorAlert: Story = {
  name: 'Error',
  args: { status: 'error', children: 'Failed to connect to the simulator daemon.' },
};
export const Warning: Story = {
  args: { status: 'warning', children: 'The selected interface is currently down.' },
};
export const Info: Story = {
  args: { status: 'info', children: 'Changes take effect after the next simulation cycle.' },
};

export const Dismissible: Story = {
  args: {
    status: 'info',
    children: 'You can dismiss this alert.',
    onDismiss: () => undefined,
  },
};

export const RichContent: Story = {
  args: {
    status: 'warning',
    children: (
      <div className="space-y-1">
        <p className="font-medium">License renewal due in 7 days</p>
        <p className="text-sm">
          Renew via Settings → License or contact{' '}
          <a className="underline" href="mailto:support@mustardseed.io">
            support@mustardseed.io
          </a>
          .
        </p>
      </div>
    ),
    onDismiss: () => undefined,
  },
};
