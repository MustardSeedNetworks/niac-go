import type { Meta, StoryObj } from '@storybook/react';
import { useState } from 'react';
import { Button } from './Button';
import { Modal } from './Modal';

const meta: Meta<typeof Modal> = {
  title: 'UI/Modal',
  component: Modal,
  tags: ['autodocs'],
  decorators: [
    (Story) => (
      <div className="p-8 bg-gray-950 min-h-[400px]">
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof Modal>;

export const Default: Story = {
  render: () => {
    const [open, setOpen] = useState(false);
    return (
      <>
        <Button onClick={() => setOpen(true)}>Open Modal</Button>
        <Modal isOpen={open} onClose={() => setOpen(false)} title="Confirm Action">
          <p className="text-gray-300 mb-4">Are you sure you want to proceed?</p>
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button tone="violet" onClick={() => setOpen(false)}>
              Confirm
            </Button>
          </div>
        </Modal>
      </>
    );
  },
};

export const Sizes: Story = {
  render: () => {
    const [size, setSize] = useState<'sm' | 'md' | 'lg' | 'xl' | 'full' | null>(null);
    return (
      <>
        <div className="flex gap-2">
          <Button size="sm" onClick={() => setSize('sm')}>
            Small
          </Button>
          <Button size="sm" onClick={() => setSize('md')}>
            Medium
          </Button>
          <Button size="sm" onClick={() => setSize('lg')}>
            Large
          </Button>
          <Button size="sm" onClick={() => setSize('xl')}>
            XL
          </Button>
          <Button size="sm" onClick={() => setSize('full')}>
            Full
          </Button>
        </div>
        <Modal
          isOpen={size !== null}
          onClose={() => setSize(null)}
          title={`${size} Modal`}
          size={size ?? 'md'}
        >
          <p className="text-gray-300">This modal uses size="{size}"</p>
        </Modal>
      </>
    );
  },
};

export const NoCloseButton: Story = {
  render: () => {
    const [open, setOpen] = useState(false);
    return (
      <>
        <Button onClick={() => setOpen(true)}>Open (no close button)</Button>
        <Modal
          isOpen={open}
          onClose={() => setOpen(false)}
          title="Important"
          showCloseButton={false}
        >
          <p className="text-gray-300 mb-4">You must click the button below to dismiss.</p>
          <Button tone="violet" onClick={() => setOpen(false)}>
            Got it
          </Button>
        </Modal>
      </>
    );
  },
};
