import type { Meta, StoryObj } from '@storybook/react';
import { Alert } from './Alert';

const meta: Meta<typeof Alert> = {
  title: 'UI/Alert',
  component: Alert,
  tags: ['autodocs'],
  argTypes: {
    status: {
      control: 'select',
      options: ['success', 'error', 'warning', 'info'],
    },
  },
  decorators: [
    (Story) => (
      <div className="p-8 bg-gray-950 max-w-md">
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof Alert>;

export const Success: Story = {
  args: {
    status: 'success',
    children: 'Device configuration saved successfully.',
  },
};

export const ErrorStatus: Story = {
  args: {
    status: 'error',
    children: 'Failed to connect to device. Check network settings.',
  },
};

export const Warning: Story = {
  args: {
    status: 'warning',
    children: 'Configuration has unsaved changes.',
  },
};

export const Info: Story = {
  args: {
    status: 'info',
    children: 'Simulation is running in background.',
  },
};

export const Dismissible: Story = {
  args: {
    status: 'info',
    children: 'Click the X to dismiss this alert.',
    onDismiss: () => undefined,
  },
};

export const AllStatuses: Story = {
  render: () => (
    <div className="space-y-3">
      <Alert status="success">Operation completed successfully.</Alert>
      <Alert status="error">An error occurred during the operation.</Alert>
      <Alert status="warning">This action cannot be undone.</Alert>
      <Alert status="info">New update available.</Alert>
    </div>
  ),
};
