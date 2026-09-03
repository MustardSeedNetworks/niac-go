import { fireEvent, render, screen } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import type { AuthoredValue } from './generated/authored-device.generated';
import { DEVICE_SECTIONS } from './generated/sections.generated';
import { SchemaSectionBody } from './SchemaFields';

/**
 * The generated manifest is only worth anything if a renderer can drive it, so
 * these exercise the five primitives against real sections rather than a
 * hand-made fixture: `icmpv6` nests an object inside an object inside a list,
 * `syslog` carries a scalar list and `routes` is a list of objects.
 */

const sectionNamed = (key: string) => {
  const section = DEVICE_SECTIONS.find((candidate) => candidate.key === key);
  if (!section) throw new Error(`no generated section for ${key}`);
  return section;
};

const renderSection = (key: string, initial: AuthoredValue = {}) => {
  const section = sectionNamed(key);
  const state: { current: AuthoredValue } = { current: initial };

  const Form = () => {
    const [value, setValue] = useState<AuthoredValue>(initial);
    state.current = value;
    return <SchemaSectionBody section={section} value={value} onChange={setValue} />;
  };

  render(<Form />);
  return state;
};

const block = (value: AuthoredValue): Record<string, AuthoredValue> =>
  value as Record<string, AuthoredValue>;

describe('SchemaSectionBody', () => {
  it('renders a closed vocabulary as a select carrying the schema enum', () => {
    renderSection('snmpv3');
    const users = sectionNamed('snmpv3').fields.find((f) => f.name === 'users');

    expect(users?.kind).toBe('objectList');
    expect(users?.fields?.find((f) => f.name === 'auth_protocol')?.options).toEqual([
      'none',
      'md5',
      'sha',
      'sha256',
      'sha512',
    ]);
  });

  it('writes a scalar edit back into the authored document', () => {
    const state = renderSection('ssh', { enabled: true });

    fireEvent.change(screen.getByLabelText('Username'), { target: { value: 'netops' } });

    expect(block(state.current).username).toBe('netops');
  });

  it('adds and edits an entry in a list of objects', () => {
    const state = renderSection('routes');

    fireEvent.click(screen.getByRole('button', { name: /add routes/i }));
    fireEvent.change(screen.getByLabelText('Destination'), { target: { value: '10.0.0.0/8' } });

    expect(state.current).toEqual([{ destination: '10.0.0.0/8' }]);
  });

  it('adds an entry to a list of scalars', () => {
    const state = renderSection('syslog', { enabled: true });

    fireEvent.click(screen.getByRole('button', { name: /add receivers/i }));
    fireEvent.change(screen.getByLabelText('Receivers 1'), { target: { value: '10.20.0.50' } });

    expect(block(state.current).receivers).toEqual(['10.20.0.50']);
  });

  it('edits a value nested two objects deep', () => {
    const state = renderSection('icmpv6', {
      router_advertisement: { lifetime: 1800 },
    });

    fireEvent.change(screen.getByLabelText('Cur hop limit'), { target: { value: '64' } });

    expect(block(state.current).router_advertisement).toEqual({
      lifetime: 1800,
      cur_hop_limit: 64,
    });
  });

  it('treats an emptied number input as unset rather than zero', () => {
    const state = renderSection('ttl', { ttl: 255 });

    fireEvent.change(screen.getByLabelText('TTL'), { target: { value: '' } });

    expect(block(state.current).ttl).toBeUndefined();
  });
});
