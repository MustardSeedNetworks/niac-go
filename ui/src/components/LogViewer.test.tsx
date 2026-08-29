/**
 * LogViewer — the debug console's log list.
 *
 * The parts worth pinning are the search highlighting (a regex built from user
 * input, so a metacharacter in the query must not blow it up) and the
 * expand/copy row affordances, which are keyboard-reachable and were the class
 * of defect the UI sweep kept finding.
 */

import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { LogEntry } from '../api/debug-types';
import '../i18n';
import { LogViewer } from './LogViewer';

const log = (over: Partial<LogEntry> = {}): LogEntry => ({
  id: '1',
  timestamp: '2026-08-29T10:00:00.000Z',
  level: 'INFO',
  protocol: 'SNMP',
  message: 'agent responded',
  ...over,
});

function renderViewer(logs: LogEntry[], searchQuery = '', autoScroll = false) {
  return render(<LogViewer logs={logs} searchQuery={searchQuery} autoScroll={autoScroll} />);
}

afterEach(() => {
  vi.restoreAllMocks();
});

describe('empty state', () => {
  it('renders an empty state instead of a list when there are no logs', () => {
    renderViewer([]);

    expect(screen.queryByText('agent responded')).toBeNull();
    // The empty state carries its own hint text, so something is on screen.
    expect(screen.getAllByText(/log/i).length).toBeGreaterThan(0);
  });
});

describe('rendering entries', () => {
  it('renders one row per log with its level and protocol', () => {
    renderViewer([
      log({ id: '1', level: 'ERROR', protocol: 'DHCP', message: 'no offer' }),
      log({ id: '2', level: 'DEBUG', protocol: 'LLDP', message: 'frame sent' }),
    ]);

    expect(screen.getByText('no offer')).toBeDefined();
    expect(screen.getByText('frame sent')).toBeDefined();
    expect(screen.getByText('DHCP')).toBeDefined();
    expect(screen.getByText('LLDP')).toBeDefined();
  });

  it('renders every level, including one with no dedicated colour', () => {
    renderViewer([
      log({ id: '1', level: 'ERROR', message: 'e' }),
      log({ id: '2', level: 'WARN', message: 'w' }),
      log({ id: '3', level: 'INFO', message: 'i' }),
      log({ id: '4', level: 'DEBUG', message: 'd' }),
      log({ id: '5', level: 'TRACE' as LogEntry['level'], message: 'unknown-level' }),
    ]);

    // An unrecognised level must still render, falling back to the info style.
    expect(screen.getByText('unknown-level')).toBeDefined();
  });

  it('formats the timestamp as a 24-hour time', () => {
    renderViewer([log()]);

    expect(screen.getByText(/^\d{2}:\d{2}:\d{2}$/)).toBeDefined();
  });
});

describe('search highlighting', () => {
  it('marks the matching substring', () => {
    const { container } = renderViewer([log({ message: 'agent responded' })], 'agent');

    const marks = container.querySelectorAll('mark');
    expect(marks).toHaveLength(1);
    expect(marks[0]?.textContent).toBe('agent');
  });

  it('matches case-insensitively while preserving the original casing', () => {
    const { container } = renderViewer([log({ message: 'Agent responded' })], 'agent');

    expect(container.querySelector('mark')?.textContent).toBe('Agent');
  });

  it('marks every occurrence', () => {
    const { container } = renderViewer([log({ message: 'a b a b a' })], 'a');

    expect(container.querySelectorAll('mark')).toHaveLength(3);
  });

  it('does not mark anything for an empty or whitespace-only query', () => {
    const { container: blank } = renderViewer([log()], '');
    expect(blank.querySelector('mark')).toBeNull();

    const { container: spaces } = renderViewer([log()], '   ');
    expect(spaces.querySelector('mark')).toBeNull();
  });

  it('treats regex metacharacters in the query as literal text', () => {
    // An unescaped '(' would throw and the whole row would fall back to plain
    // text; worse, '.*' would match everything.
    const { container } = renderViewer([log({ message: 'value (x) here' })], '(x)');

    expect(container.querySelector('mark')?.textContent).toBe('(x)');
  });

  it('does not treat a dot as a wildcard', () => {
    const { container } = renderViewer([log({ message: 'ab cd' })], 'a.');

    expect(container.querySelector('mark')).toBeNull();
  });

  it('renders the row unmarked when nothing matches', () => {
    const { container } = renderViewer([log({ message: 'agent responded' })], 'zzz');

    expect(container.querySelectorAll('mark')).toHaveLength(0);
    expect(screen.getByText(/agent responded/)).toBeDefined();
  });
});

describe('row expansion', () => {
  it('exposes no expanded state for a log without details', () => {
    renderViewer([log()]);

    const row = screen.getAllByRole('button')[0];
    expect(row?.getAttribute('aria-expanded')).toBeNull();
  });

  it('expands and collapses a log that has details', () => {
    renderViewer([log({ details: { retries: 3 } })]);

    const row = screen.getAllByRole('button')[0] as HTMLElement;
    expect(row.getAttribute('aria-expanded')).toBe('false');

    fireEvent.click(row);
    expect(row.getAttribute('aria-expanded')).toBe('true');
    expect(screen.getByText(/retries/)).toBeDefined();

    fireEvent.click(row);
    expect(row.getAttribute('aria-expanded')).toBe('false');
  });

  it('expands from the keyboard with Enter and Space', () => {
    // Mouse-only affordances were the single most common defect in the UI
    // sweep, so the keyboard path is asserted explicitly.
    renderViewer([log({ details: { retries: 3 } })]);
    const row = screen.getAllByRole('button')[0] as HTMLElement;

    fireEvent.keyDown(row, { key: 'Enter' });
    expect(row.getAttribute('aria-expanded')).toBe('true');

    fireEvent.keyDown(row, { key: ' ' });
    expect(row.getAttribute('aria-expanded')).toBe('false');
  });

  it('ignores unrelated keys', () => {
    renderViewer([log({ details: { retries: 3 } })]);
    const row = screen.getAllByRole('button')[0] as HTMLElement;

    fireEvent.keyDown(row, { key: 'a' });
    expect(row.getAttribute('aria-expanded')).toBe('false');
  });

  it('does nothing on a row with no details', () => {
    renderViewer([log()]);
    const row = screen.getAllByRole('button')[0] as HTMLElement;

    fireEvent.keyDown(row, { key: 'Enter' });
    expect(row.getAttribute('aria-expanded')).toBeNull();
  });
});

describe('copy', () => {
  it('copies the whole entry to the clipboard', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal('navigator', { clipboard: { writeText } });

    renderViewer([log({ message: 'agent responded' })]);

    fireEvent.click(screen.getByLabelText(/copy/i));

    await waitFor(() => expect(writeText).toHaveBeenCalled());
    const copied = writeText.mock.calls[0]?.[0] as string;
    expect(copied).toContain('agent responded');
    expect(copied).toContain('INFO');
    expect(copied).toContain('SNMP');
  });

  it('includes the details block when the entry has one', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal('navigator', { clipboard: { writeText } });

    renderViewer([log({ details: { retries: 3 } })]);
    fireEvent.click(screen.getByLabelText(/copy/i));

    await waitFor(() => expect(writeText).toHaveBeenCalled());
    expect(writeText.mock.calls[0]?.[0]).toContain('retries');
  });

  it('copying does not expand the row', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal('navigator', { clipboard: { writeText } });

    renderViewer([log({ details: { retries: 3 } })]);
    const row = screen.getAllByRole('button')[0] as HTMLElement;

    fireEvent.click(screen.getByLabelText(/copy/i));

    await waitFor(() => expect(writeText).toHaveBeenCalled());
    expect(row.getAttribute('aria-expanded')).toBe('false');
  });
});

describe('auto-scroll', () => {
  it('scrolls to the end when auto-scroll is on', () => {
    const scrollIntoView = vi.fn();
    Element.prototype.scrollIntoView = scrollIntoView;

    renderViewer([log()], '', true);

    expect(scrollIntoView).toHaveBeenCalled();
  });

  it('does not scroll when auto-scroll is off', () => {
    const scrollIntoView = vi.fn();
    Element.prototype.scrollIntoView = scrollIntoView;

    renderViewer([log()], '', false);

    expect(scrollIntoView).not.toHaveBeenCalled();
  });
});
