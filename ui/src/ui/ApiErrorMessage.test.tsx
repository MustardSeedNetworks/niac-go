import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { ApiErrorMessage } from './ApiErrorMessage';

/**
 * Guards #1499.
 *
 * The API enumerates its failures one per offending field — preflight
 * validation (#1461), walk capture (#1488) — and each inline surface kept only
 * `err.message`, so the diagnosis travelled the whole way over the wire and
 * died at the last step. On the released 0.94.66 the walk-capture response
 * carried "no response from the target: it did not answer, or the community or
 * SNMPv3 credentials are wrong" while the screen said only "SNMP walk capture
 * failed".
 */
describe('ApiErrorMessage', () => {
  it('lists every detail beneath the message', () => {
    render(
      <ApiErrorMessage
        message="Simulation preflight failed"
        details={[
          {
            field: 'devices[0].snmp_agent.community',
            issue: 'SNMPv1/v2c requires an explicit community',
          },
          {
            field: 'devices[1].snmp_agent.community',
            issue: 'SNMPv1/v2c requires an explicit community',
          },
        ]}
      />,
    );

    expect(screen.getByText('Simulation preflight failed')).toBeInTheDocument();
    expect(screen.getAllByRole('listitem')).toHaveLength(2);
    expect(screen.getByText(/devices\[0\]\.snmp_agent\.community/)).toBeInTheDocument();
  });

  it('renders a detail with no field as its bare issue', () => {
    render(
      <ApiErrorMessage
        message="SNMP walk capture failed"
        details={[{ issue: 'no response from the target' }]}
      />,
    );

    expect(screen.getByText('no response from the target')).toBeInTheDocument();
  });

  it('shows just the message when the server sent no details', () => {
    render(<ApiErrorMessage message="Something went wrong" />);

    expect(screen.getByText('Something went wrong')).toBeInTheDocument();
    expect(screen.queryByRole('listitem')).toBeNull();
  });

  it('is announced as an alert', () => {
    render(<ApiErrorMessage message="Failed" />);
    expect(screen.getByRole('alert')).toBeInTheDocument();
  });
});
