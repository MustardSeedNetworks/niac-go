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

  it('dismisses a hover-only explanation before outer Escape handlers without moving focus', async () => {
    const user = userEvent.setup();
    render(
      <>
        <button type="button">Other control</button>
        <Tooltip text={REASON}>
          <button type="button">Start</button>
        </Tooltip>
      </>,
    );
    const other = screen.getByRole('button', { name: 'Other control' });
    const outerEscape = vi.fn();
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') outerEscape();
    };
    document.addEventListener('keydown', onKeyDown);
    try {
      await user.tab();
      await user.hover(screen.getByRole('button', { name: 'Start' }));
      expect(other).toHaveFocus();
      expect(bubble()).toBeVisible();
      await user.keyboard('{Escape}');
      expect(bubble()).not.toBeVisible();
      expect(other).toHaveFocus();
      expect(outerEscape).not.toHaveBeenCalled();
      await user.keyboard('{Escape}');
      expect(outerEscape).toHaveBeenCalledTimes(1);
    } finally {
      document.removeEventListener('keydown', onKeyDown);
    }
  });

  it.each(['pointer', 'Enter', 'Space'])(
    'dismisses after %s activates an enabled trigger without moving focus',
    async (activation) => {
      const user = userEvent.setup();
      const action = vi.fn((event: React.MouseEvent) => event.stopPropagation());
      render(
        <Tooltip text="Open help">
          <button type="button" onClick={action}>
            Help
          </button>
        </Tooltip>,
      );
      const trigger = screen.getByRole('button', { name: 'Help' });
      await user.hover(trigger);
      expect(bubble()).toBeVisible();
      if (activation === 'pointer') await user.click(trigger);
      else {
        await user.tab();
        await user.keyboard(activation === 'Enter' ? '{Enter}' : ' ');
      }
      expect(action).toHaveBeenCalledTimes(1);
      expect(trigger).toHaveFocus();
      expect(bubble()).not.toBeVisible();
    },
  );

  it.each([
    { side: 'top' as const, top: 0, bottom: 40, expectedTop: 40 },
    { side: 'bottom' as const, top: 740, bottom: 768, expectedTop: 710 },
  ])(
    'keeps a $side tooltip off its trigger at the viewport edge',
    async ({ side, top, bottom, expectedTop }) => {
      const user = userEvent.setup();
      render(
        <Tooltip text={REASON} side={side}>
          <button type="button">Start</button>
        </Tooltip>,
      );
      const trigger = screen.getByRole('button', { name: 'Start' });
      vi.spyOn(trigger, 'getBoundingClientRect').mockReturnValue({
        x: 100,
        y: top,
        top,
        bottom,
        left: 100,
        right: 200,
        width: 100,
        height: bottom - top,
        toJSON: () => ({}),
      });
      vi.spyOn(bubble(), 'getBoundingClientRect').mockReturnValue({
        x: 0,
        y: 0,
        top: 0,
        bottom: 30,
        left: 0,
        right: 200,
        width: 200,
        height: 30,
        toJSON: () => ({}),
      });
      await user.tab();
      expect(bubble()).toHaveStyle({ top: `${expectedTop}px` });
    },
  );

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
    expect(bubble()).toBeVisible();
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

  it('preserves focus when a permission explanation disappears', async () => {
    const user = userEvent.setup();
    const { rerender } = renderTooltip();
    await user.tab();
    expect(screen.getByRole('button', { name: 'Start' })).toHaveFocus();
    rerender(
      <Tooltip>
        <button type="button">Start</button>
      </Tooltip>,
    );
    expect(screen.getByRole('button', { name: 'Start' })).toHaveFocus();
    expect(screen.getByRole('button', { name: 'Start' })).not.toHaveAccessibleDescription();
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
