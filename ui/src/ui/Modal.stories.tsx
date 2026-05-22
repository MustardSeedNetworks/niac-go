/**
 * Modal primitive stories (Wave 5 / #636).
 *
 * Size matrix, showCloseButton/closeOnBackdropClick/closeOnEscape
 * flag combinations, with-title vs no-title shells.
 */
import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import { Button } from './Button';
import { Modal } from './Modal';

const meta: Meta<typeof Modal> = {
  title: 'UI/Modal',
  component: Modal,
  parameters: { layout: 'fullscreen' },
  argTypes: {
    size: { control: 'select', options: ['sm', 'md', 'lg', 'xl', 'full'] },
    isOpen: { control: 'boolean' },
    showCloseButton: { control: 'boolean' },
    closeOnBackdropClick: { control: 'boolean' },
    closeOnEscape: { control: 'boolean' },
  },
};
export default meta;

type Story = StoryObj<typeof Modal>;

const SampleBody = () => (
  <div className="space-y-2">
    <p>Modal content goes here.</p>
    <p className="text-text-muted text-sm">A second paragraph to show vertical rhythm.</p>
  </div>
);

export const Default: Story = {
  args: {
    isOpen: true,
    title: 'Default modal',
    onClose: () => undefined,
    children: <SampleBody />,
  },
};

export const NoTitleNoCloseButton: Story = {
  args: {
    isOpen: true,
    showCloseButton: false,
    onClose: () => undefined,
    children: <SampleBody />,
  },
};

export const SmallSize: Story = {
  args: {
    isOpen: true,
    size: 'sm',
    title: 'Small',
    onClose: () => undefined,
    children: <SampleBody />,
  },
};
export const LargeSize: Story = {
  args: {
    isOpen: true,
    size: 'lg',
    title: 'Large',
    onClose: () => undefined,
    children: <SampleBody />,
  },
};
export const FullSize: Story = {
  args: {
    isOpen: true,
    size: 'full',
    title: 'Full width',
    onClose: () => undefined,
    children: <SampleBody />,
  },
};

export const InteractiveToggle: Story = {
  args: {
    isOpen: false,
    title: 'Trigger demo',
    onClose: () => undefined,
    children: <SampleBody />,
  },
  render: function interactiveRender(args) {
    const [open, setOpen] = useState(false);
    return (
      <div className="p-8">
        <Button onClick={() => setOpen(true)}>Open modal</Button>
        <Modal {...args} isOpen={open} onClose={() => setOpen(false)}>
          <SampleBody />
        </Modal>
      </div>
    );
  },
};
