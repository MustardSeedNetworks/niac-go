/**
 * Modal primitive stories (Wave 5 / #636).
 *
 * Size matrix, showCloseButton/closeOnBackdropClick/closeOnEscape
 * flag combinations, with-title vs no-title shells, and the header/footer
 * regions the migrated dialogs (#1828 follow-up) render into — the a11y gate
 * only sees a layout that some story puts on screen.
 */
import type { Meta, StoryObj } from '@storybook/react-vite';
import { FileCode } from 'lucide-react';
import { useState } from 'react';
import { iconSizes } from '../constants/sizes';
import { Button } from './Button';
import { Modal } from './Modal';
import { SmallText } from './Typography';

const meta: Meta<typeof Modal> = {
  title: 'UI/Modal',
  component: Modal,
  parameters: { layout: 'fullscreen' },
  argTypes: {
    size: { control: 'select', options: ['sm', 'md', 'lg', 'xl', '3xl', 'full'] },
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

// A chrome-less dialog still needs a name; with no visible heading to point
// at, ariaLabel is the way to give it one. The story previously passed neither
// and rendered a dialog a screen reader announces as "dialog" and nothing else.
export const NoTitleNoCloseButton: Story = {
  args: {
    isOpen: true,
    showCloseButton: false,
    ariaLabel: 'Example dialog',
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

// The header/footer regions: a heading with a subtitle beside an icon is what
// `title` alone cannot express, and actions in the footer stay put while a long
// body scrolls. `labelledBy` points at the caller's own heading, which is what
// gives the dialog its accessible name.
export const HeaderAndFooterRegions: Story = {
  args: {
    isOpen: true,
    size: 'full',
    labelledBy: 'story-modal-title',
    onClose: () => undefined,
    header: (
      <div className="flex items-center gap-default">
        <div className="rounded-lg bg-brand-primary/20 pad-xs">
          <FileCode className={`${iconSizes.lg} text-brand-accent`} />
        </div>
        <div>
          <h2 id="story-modal-title" className="heading-3 text-text-primary">
            hospital-pack.yaml
          </h2>
          <SmallText className="text-text-muted">248 lines · 12 devices</SmallText>
        </div>
      </div>
    ),
    footer: (
      <>
        <Button variant="outline">Close</Button>
        <Button tone="violet">Use template</Button>
      </>
    ),
    children: (
      <div className="space-y-2">
        {Array.from({ length: 40 }, (_, line) => (
          <p key={line} className="font-mono text-xs text-text-secondary">
            line {line + 1}: a body long enough to scroll under a pinned footer
          </p>
        ))}
      </div>
    ),
  },
};
