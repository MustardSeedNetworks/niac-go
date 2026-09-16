import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import { expect, fn, userEvent, within } from 'storybook/test';
import { Tooltip } from './Tooltip';

const meta: Meta<typeof Tooltip> = {
  title: 'UI/Tooltip',
  component: Tooltip,
  parameters: { layout: 'centered' },
  argTypes: {
    side: { control: 'select', options: ['top', 'bottom', 'left', 'right'] },
  },
};
export default meta;

type Story = StoryObj<typeof Tooltip>;

const trigger = (
  <button
    type="button"
    className="inline-block px-3 py-2 rounded border border-border-muted bg-bg-muted/20 text-text-secondary"
  >
    Show details
  </button>
);

export const Top: Story = {
  args: { side: 'top', text: 'Tooltip on top', children: trigger },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const button = canvas.getByRole('button', { name: 'Show details' });
    const tooltip = within(document.body).getByRole('tooltip', { hidden: true });
    await expect(tooltip).not.toBeVisible();
    const originalFocus = document.activeElement;
    await userEvent.hover(button);
    await expect(tooltip).toBeVisible();
    await userEvent.keyboard('{Escape}');
    await expect(tooltip).not.toBeVisible();
    await expect(document.activeElement).toBe(originalFocus);
    await userEvent.unhover(button);
    await expect(tooltip).not.toBeVisible();
    await userEvent.tab();
    await expect(button).toHaveFocus();
    await expect(tooltip).toBeVisible();
    await expect(button).toHaveAccessibleDescription('Tooltip on top');
    await userEvent.keyboard('{Escape}');
    await expect(tooltip).not.toBeVisible();
    await expect(button).toHaveFocus();
  },
};
export const Bottom: Story = {
  args: { side: 'bottom', text: 'Tooltip on bottom', children: trigger },
};
export const Left: Story = {
  args: { side: 'left', text: 'Tooltip on left', children: trigger },
};
export const Right: Story = {
  args: { side: 'right', text: 'Tooltip on right', children: trigger },
};

export const RichContent: Story = {
  args: {
    side: 'top',
    text: (
      <div className="space-y-1">
        <p className="font-medium">SNMPv3 user</p>
        <p className="text-xs text-text-muted">authPriv with SHA-256/AES-128</p>
      </div>
    ),
    children: trigger,
  },
};

export const NoText: Story = {
  args: { children: trigger },
  parameters: {
    docs: {
      description: { story: 'When `text` is omitted, the wrapper renders children unchanged.' },
    },
  },
};

const activate = fn();
const submit = fn((event: React.FormEvent) => event.preventDefault());

export const UnavailableAction: Story = {
  args: {
    text: 'An admin token is required.',
    children: (
      <button type="submit" disabled onClick={activate}>
        Delete
      </button>
    ),
  },
  render: (args) => (
    <form onSubmit={submit}>
      <Tooltip {...args} />
    </form>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const button = canvas.getByRole('button', { name: 'Delete' });
    await userEvent.tab();
    await expect(button).toHaveFocus();
    await expect(button).toHaveAttribute('aria-disabled', 'true');
    await expect(button).toHaveAccessibleDescription('An admin token is required.');
    await expect(within(document.body).getByRole('tooltip', { hidden: true })).toBeVisible();
    await userEvent.keyboard('{Enter} ');
    await userEvent.click(button);
    await expect(activate).not.toHaveBeenCalled();
    await expect(submit).not.toHaveBeenCalled();
  },
};

export const ExplanationCleared: Story = {
  render: () => {
    const [text, setText] = useState<string | undefined>('Press Enter to acknowledge.');
    return (
      <Tooltip text={text}>
        <button type="button" onClick={() => setText(undefined)}>
          Acknowledge
        </button>
      </Tooltip>
    );
  },
  play: async ({ canvasElement }) => {
    const button = within(canvasElement).getByRole('button', { name: 'Acknowledge' });
    await userEvent.tab();
    await expect(button).toHaveFocus();
    await expect(button).toHaveAccessibleDescription('Press Enter to acknowledge.');
    await userEvent.keyboard('{Enter}');
    await expect(button).toHaveFocus();
    await expect(button).not.toHaveAccessibleDescription();
  },
};
