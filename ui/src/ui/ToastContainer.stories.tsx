/**
 * ToastContainer primitive stories (Wave 5 / niac-W5-2c).
 *
 * ToastContainer is driven entirely by the Zustand ui-store. The
 * decorator seeds the store with a synthetic notification per story
 * so each variant renders deterministically. duration=-1 keeps the
 * toast pinned (no auto-dismiss) so a screenshot can be captured.
 */
import type { Meta, StoryObj } from '@storybook/react-vite';
import { useEffect } from 'react';
import { useUIStore } from '../stores/ui-store';
import { ToastContainer } from './ToastContainer';

type ToastType = 'success' | 'error' | 'warning' | 'info';

function SeededToasts({ kinds }: { kinds: ToastType[] }) {
  useEffect(() => {
    // Reset any leftover toasts from a previous story
    useUIStore.setState({ notifications: [] });
    for (const type of kinds) {
      useUIStore.getState().addNotification({
        type,
        title: `${type.charAt(0).toUpperCase()}${type.slice(1)} notification`,
        message: `This is a ${type}-level toast example.`,
        duration: -1,
      });
    }
    return () => {
      useUIStore.setState({ notifications: [] });
    };
  }, [kinds]);
  return <ToastContainer />;
}

const meta: Meta<typeof ToastContainer> = {
  title: 'UI/ToastContainer',
  component: ToastContainer,
  parameters: { layout: 'fullscreen' },
};
export default meta;

type Story = StoryObj<typeof ToastContainer>;

export const Success: Story = { render: () => <SeededToasts kinds={['success']} /> };
export const ErrorToast: Story = {
  name: 'Error',
  render: () => <SeededToasts kinds={['error']} />,
};
export const Warning: Story = { render: () => <SeededToasts kinds={['warning']} /> };
export const Info: Story = { render: () => <SeededToasts kinds={['info']} /> };

export const Stacked: Story = {
  render: () => <SeededToasts kinds={['success', 'warning', 'error', 'info']} />,
};
