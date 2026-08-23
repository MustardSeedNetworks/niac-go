import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import '../i18n';
import { FilterBar } from './FilterBar';

/**
 * Guards #1481.
 *
 * The quick protocol chips appended unconditionally and joined with `&&`, so
 * they could not be turned off, could be added twice, and combining two of them
 * built a filter that can never match — a frame is never both TCP and UDP.
 * Measured on CT304 against the live buffer: `(cdp || lldp)` matched 100/100
 * while `cdp && lldp` matched 0/100, and the chips built the second shape.
 */
function Harness({ initial = '' }: { initial?: string }) {
  const [value, setValue] = useState(initial);
  return (
    <>
      <FilterBar value={value} onChange={setValue} />
      <output data-testid="value">{value}</output>
    </>
  );
}

const chip = (name: string) => screen.getByRole('button', { name: new RegExp(`^${name}$`, 'i') });
const current = () => screen.getByTestId('value').textContent;

describe('FilterBar quick protocol chips', () => {
  it('combines two protocols with OR, not AND', async () => {
    const user = userEvent.setup();
    render(<Harness />);

    await user.click(chip('tcp'));
    expect(current()).toBe('tcp');

    await user.click(chip('udp'));
    expect(current()).toBe('(tcp || udp)');
  });

  it('toggles a chip off instead of adding it twice', async () => {
    const user = userEvent.setup();
    render(<Harness />);

    await user.click(chip('tcp'));
    await user.click(chip('tcp'));
    expect(current()).toBe('');
  });

  it('removes just one protocol from a group', async () => {
    const user = userEvent.setup();
    render(<Harness />);

    await user.click(chip('tcp'));
    await user.click(chip('udp'));
    await user.click(chip('tcp'));
    expect(current()).toBe('udp');
  });

  it('keeps a typed expression and ANDs the protocol group onto it', async () => {
    const user = userEvent.setup();
    render(<Harness initial="ip.src == 10.0.0.1" />);

    await user.click(chip('tcp'));
    expect(current()).toBe('ip.src == 10.0.0.1 && tcp');

    await user.click(chip('udp'));
    expect(current()).toBe('ip.src == 10.0.0.1 && (tcp || udp)');
  });

  it('marks applied chips with aria-pressed', async () => {
    const user = userEvent.setup();
    render(<Harness />);

    expect(chip('tcp')).toHaveAttribute('aria-pressed', 'false');
    await user.click(chip('tcp'));
    expect(chip('tcp')).toHaveAttribute('aria-pressed', 'true');
    expect(chip('udp')).toHaveAttribute('aria-pressed', 'false');
  });
});
