/**
 * DebugConsolePage — the live log console.
 *
 * Two things carry real risk here: mapToLogEntry, which normalises whatever the
 * SSE stream sends into a LogEntry (a wrong default silently mislabels every
 * log), and the filter stack, where level, protocol and search combine. Both are
 * driven through the page by feeding the stream, so the assertions are on what
 * an operator would actually see.
 */

import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import '../i18n';
import { DebugConsolePage } from './DebugConsolePage';

/** Captures the callbacks the page hands to useLogStream so tests can drive it. */
let streamHandlers: {
  onMessage?: (data: unknown) => void;
  onConnect?: () => void;
  onDisconnect?: () => void;
} = {};
let connected = true;
const reconnect = vi.fn();

vi.mock('../hooks/useEventSource', () => ({
  useLogStream: (options: typeof streamHandlers) => {
    streamHandlers = options;
    return { data: null, connected, error: null, close: vi.fn(), reconnect };
  },
}));

const addNotification = vi.fn();
vi.mock('../stores/ui-store', () => ({
  useUIStore: (selector: (s: { addNotification: typeof addNotification }) => unknown) =>
    selector({ addNotification }),
}));

/** Pushes one SSE payload into the page. */
function emit(payload: unknown): void {
  act(() => {
    streamHandlers.onMessage?.(payload);
  });
}

beforeEach(() => {
  streamHandlers = {};
  connected = true;
  reconnect.mockReset();
  addNotification.mockReset();
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe('mapping stream payloads', () => {
  it('renders a well-formed log entry', () => {
    render(<DebugConsolePage />);
    emit({ level: 'error', protocol: 'DHCP', message: 'no offer' });

    expect(screen.getByText('no offer')).toBeDefined();
    // 'DHCP' also appears as a filter <option>, so scope to the log list.
    expect(screen.getAllByText('DHCP').some((el) => el.tagName !== 'OPTION')).toBe(true);
  });

  it('upper-cases the level', () => {
    render(<DebugConsolePage />);
    emit({ level: 'warn', protocol: 'LLDP', message: 'held down' });

    expect(screen.getByText('held down')).toBeDefined();
  });

  it('falls back to INFO for an unrecognised level', () => {
    // A level the daemon invents must not become a blank badge.
    render(<DebugConsolePage />);
    emit({ level: 'catastrophe', protocol: 'SNMP', message: 'odd level' });

    expect(screen.getByText('odd level')).toBeDefined();
    expect(screen.getAllByText('INFO').some((el) => el.tagName !== 'OPTION')).toBe(true);
  });

  it('falls back to the source, then to SYSTEM, when protocol is absent', () => {
    render(<DebugConsolePage />);
    emit({ level: 'info', source: 'daemon', message: 'from source' });
    emit({ level: 'info', message: 'no protocol at all' });

    expect(screen.getByText('daemon')).toBeDefined();
    expect(screen.getAllByText('SYSTEM').some((el) => el.tagName !== 'OPTION')).toBe(true);
  });

  it('stringifies the payload when there is no message', () => {
    render(<DebugConsolePage />);
    emit({ level: 'info', protocol: 'SNMP', code: 42 });

    expect(screen.getByText(/"code": ?42/)).toBeDefined();
  });

  it('ignores a payload that is not an object', () => {
    render(<DebugConsolePage />);
    emit('just a string');
    emit(null);

    // Nothing was added, so the empty state is still showing.
    expect(screen.queryByText('just a string')).toBeNull();
  });
});

describe('filtering', () => {
  function seed(): void {
    emit({ level: 'error', protocol: 'DHCP', message: 'dhcp failure' });
    emit({ level: 'info', protocol: 'SNMP', message: 'snmp ok' });
    emit({ level: 'debug', protocol: 'SNMP', message: 'snmp detail' });
  }

  it('shows everything by default', () => {
    render(<DebugConsolePage />);
    seed();

    expect(screen.getByText('dhcp failure')).toBeDefined();
    expect(screen.getByText('snmp ok')).toBeDefined();
    expect(screen.getByText('snmp detail')).toBeDefined();
  });

  it('filters by level', () => {
    render(<DebugConsolePage />);
    seed();

    const levelSelect = screen.getAllByRole('combobox')[0] as HTMLSelectElement;
    fireEvent.change(levelSelect, { target: { value: 'ERROR' } });

    expect(screen.getByText('dhcp failure')).toBeDefined();
    expect(screen.queryByText('snmp ok')).toBeNull();
  });

  it('filters by protocol', () => {
    render(<DebugConsolePage />);
    seed();

    const protocolSelect = screen.getAllByRole('combobox')[1] as HTMLSelectElement;
    fireEvent.change(protocolSelect, { target: { value: 'SNMP' } });

    expect(screen.queryByText('dhcp failure')).toBeNull();
    expect(screen.getByText('snmp ok')).toBeDefined();
  });

  it('searches the message text', () => {
    render(<DebugConsolePage />);
    seed();

    const search = screen.getByLabelText('Search logs');
    fireEvent.change(search, { target: { value: 'failure' } });

    // The matching run is split into <mark> and <span> nodes by the
    // highlighter, so match the fragments rather than one whole string.
    expect(screen.getAllByText(/failure/).length).toBeGreaterThan(0);
    expect(screen.queryByText('snmp ok')).toBeNull();
  });

  it('searches the protocol as well as the message', () => {
    render(<DebugConsolePage />);
    seed();

    fireEvent.change(screen.getByLabelText('Search logs'), { target: { value: 'dhcp' } });

    expect(screen.queryByText('snmp ok')).toBeNull();
  });

  it('combines the level and protocol filters', () => {
    render(<DebugConsolePage />);
    seed();

    fireEvent.change(screen.getAllByRole('combobox')[1] as HTMLSelectElement, {
      target: { value: 'SNMP' },
    });
    fireEvent.change(screen.getAllByRole('combobox')[0] as HTMLSelectElement, {
      target: { value: 'DEBUG' },
    });

    expect(screen.getByText('snmp detail')).toBeDefined();
    expect(screen.queryByText('snmp ok')).toBeNull();
    expect(screen.queryByText('dhcp failure')).toBeNull();
  });
});

describe('clearing', () => {
  it('asks for confirmation and only then empties the buffer', async () => {
    render(<DebugConsolePage />);
    emit({ level: 'info', protocol: 'SNMP', message: 'keep me' });

    fireEvent.click(screen.getByRole('button', { name: /clear/i }));

    // Still there: the confirm step has not been answered yet.
    expect(screen.getByText('keep me')).toBeDefined();

    const confirm = screen.getAllByRole('button', { name: /clear/i }).at(-1) as HTMLElement;
    fireEvent.click(confirm);

    await waitFor(() => expect(screen.queryByText('keep me')).toBeNull());
  });
});

describe('pause', () => {
  it('stops accepting new logs while paused', () => {
    render(<DebugConsolePage />);
    emit({ level: 'info', protocol: 'SNMP', message: 'before pause' });

    fireEvent.click(screen.getByLabelText(/pause/i));
    emit({ level: 'info', protocol: 'SNMP', message: 'during pause' });

    expect(screen.getByText('before pause')).toBeDefined();
    expect(screen.queryByText('during pause')).toBeNull();
  });
});

describe('connection notifications', () => {
  it('does not warn about a disconnect before ever connecting', () => {
    render(<DebugConsolePage />);

    act(() => {
      streamHandlers.onDisconnect?.();
    });

    // The initial "still connecting" window is not a dropped connection.
    expect(addNotification).not.toHaveBeenCalled();
  });

  it('warns when an established connection drops', () => {
    render(<DebugConsolePage />);

    act(() => {
      streamHandlers.onConnect?.();
    });
    act(() => {
      streamHandlers.onDisconnect?.();
    });

    expect(addNotification).toHaveBeenCalled();
  });
});
