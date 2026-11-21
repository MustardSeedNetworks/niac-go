import type { Meta, StoryObj } from '@storybook/react-vite';
import { useEffect } from 'react';
import { useToast } from '../hooks/useToast';
import { ToastProvider } from './Toast';
import type { ToastType } from './toastContext';

function SeededToasts({ kinds }: { kinds: ToastType[] }) {
  const { addToast } = useToast();
  useEffect(() => {
    for (const type of kinds) {
      addToast({
        type,
        title: `${type.charAt(0).toUpperCase()}${type.slice(1)} notification`,
        message: `This is a ${type}-level toast example.`,
        duration: 0,
      });
    }
  }, [kinds, addToast]);
  return null;
}

const meta: Meta<typeof ToastProvider> = {
  title: 'UI/Toast',
  component: ToastProvider,
  parameters: { layout: 'fullscreen' },
};
export default meta;

type Story = StoryObj<typeof ToastProvider>;

export const AllTypes: Story = {
  render: () => (
    <ToastProvider>
      <SeededToasts kinds={['success', 'info', 'warning', 'error']} />
    </ToastProvider>
  ),
};

export const ErrorOnly: Story = {
  render: () => (
    <ToastProvider>
      <SeededToasts kinds={['error']} />
    </ToastProvider>
  ),
};
