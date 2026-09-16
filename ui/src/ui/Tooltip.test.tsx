import { fireEvent, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { expectNoAxeViolations } from '../test/a11y';
import { Tooltip } from './Tooltip';

const REASON = 'Your token does not allow this action.';
const bubble = () => screen.getByRole('tooltip', { hidden: true });

function renderTooltip() {
  return render(
    <Tooltip text={REASON}>
      <button type="button">Start</button>
    </Tooltip>,
  );
}

describe('Tooltip', () => {
  it('opens by Tab, preserves the name, and closes on Escape without moving focus', async () => {
    const user = userEvent.setup();
    renderTooltip();
    const trigger = screen.getByRole('button', { name: 'Start' });
    expect(trigger).toHaveAccessibleDescription(REASON);
    expect(bubble()).not.toBeVisible();
    await user.tab();
    expect(trigger).toHaveFocus();
    expect(trigger).toHaveAccessibleName('Start');
    expect(bubble()).toBeVisible();
    await user.keyboard('{Escape}');
    expect(bubble()).not.toBeVisible();
    expect(trigger).toHaveFocus();
  });

  it('stays open while the trigger is focused after the pointer leaves', async () => {
    const user = userEvent.setup();
    renderTooltip();
    const trigger = screen.getByRole('button', { name: 'Start' });
    await user.hover(trigger);
    expect(bubble()).toBeVisible();
    await user.tab();
    await user.unhover(trigger);
    expect(bubble()).toBeVisible();
    await user.tab();
    expect(bubble()).not.toBeVisible();
  });

  it('keeps unavailable actions focusable but rejects clicks, Enter, Space and form submission', async () => {
    const user = userEvent.setup();
    const action = vi.fn();
    const submit = vi.fn((event: React.FormEvent) => event.preventDefault());
    render(
      <form onSubmit={submit}>
        <Tooltip text={REASON}>
          <button type="submit" disabled onClick={action}>
            Delete
          </button>
        </Tooltip>
      </form>,
    );
    const trigger = screen.getByRole('button', { name: 'Delete' });
    await user.tab();
    expect(trigger).toHaveFocus();
    expect(trigger).toHaveAttribute('aria-disabled', 'true');
    expect(bubble()).toBeVisible();
    await user.keyboard('{Enter} ');
    await user.click(trigger);
    fireEvent.click(trigger);
    expect(action).not.toHaveBeenCalled();
    expect(submit).not.toHaveBeenCalled();
  });

  it('describes a nested input without changing its label when the bubble opens', async () => {
    const user = userEvent.setup();
    render(
      <Tooltip text="Choose a saved file">
        {(description) => (
          <label>
            Config
            <input {...description} />
          </label>
        )}
      </Tooltip>,
    );
    await user.tab();
    const input = screen.getByRole('textbox', { name: 'Config' });
    expect(input).toHaveFocus();
    expect(input).toHaveAccessibleDescription('Choose a saved file');
    expect(input).toHaveAccessibleName('Config');
    expect(bubble()).toBeVisible();
  });

  it('keeps a description the caller already set', () => {
    render(
      <>
        <span id="own-note">Existing note.</span>
        <Tooltip text={REASON}>
          <button type="button" aria-describedby="own-note">
            Start
          </button>
        </Tooltip>
      </>,
    );
    expect(screen.getByRole('button', { name: 'Start' })).toHaveAccessibleDescription(
      `Existing note. ${REASON}`,
    );
  });

  it('renders children unchanged when there is no text', () => {
    render(
      <Tooltip>
        <button type="button" disabled>
          Start
        </button>
      </Tooltip>,
    );
    expect(screen.queryByRole('tooltip', { hidden: true })).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Start' })).toBeDisabled();
  });

  it('has no axe violations with the explanation visible', async () => {
    const user = userEvent.setup();
    const { container } = renderTooltip();
    await user.tab();
    await expectNoAxeViolations(container);
  });
});
