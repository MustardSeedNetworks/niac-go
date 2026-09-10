import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, expect, it, vi } from 'vitest';
import type { DraftBehaviorTimeline, ScenarioDraft } from '../../api/library-client';
import '../../i18n';
import { DraftBehaviorComposer } from './DraftBehaviorComposer';

vi.mock('../../contexts/ScopeContext', () => ({
  useActionPermission: () => ({ disabled: false }),
}));
const replace = vi.hoisted(() => vi.fn());
vi.mock('../../api/library-client', async (original) => ({
  ...(await original<typeof import('../../api/library-client')>()),
  replaceScenarioDraftBehaviors: replace,
}));
const draft: ScenarioDraft = {
  name: 'faults',
  revision: 'r1',
  format: 'yaml',
  modifiedAt: '',
  sizeBytes: 0,
  content:
    'devices:\n  - name: resolver-1\n    type: server\n    mac: 02:00:00:00:00:10\n    ips: [192.0.2.10]\n',
};
beforeEach(() => {
  replace.mockReset();
  replace.mockResolvedValue(draft);
});

it.each(['dhcp_no_offer', 'dns_nxdomain', 'dns_timeout', 'latency'])(
  'saves %s without an interface and preserves the selected value',
  async (type) => {
    const user = userEvent.setup();
    render(<DraftBehaviorComposer draft={draft} onDraftUpdate={vi.fn()} onBusyChange={vi.fn()} />);
    await user.click(screen.getByTestId('add-timeline'));
    await user.click(screen.getByTestId('add-device-fault'));
    await user.selectOptions(screen.getByLabelText('Fault'), type);
    expect(screen.queryByLabelText('Interface')).not.toBeInTheDocument();
    const value = type === 'latency' ? 60000 : 100;
    const input = screen.getByLabelText(type === 'latency' ? 'Latency (milliseconds)' : 'Rate (%)');
    fireEvent.change(input, { target: { value: String(value + 1) } });
    expect(screen.getByTestId('save-behaviors')).toBeDisabled();
    fireEvent.change(input, { target: { value: '1.5' } });
    expect(screen.getByTestId('save-behaviors')).toBeDisabled();
    fireEvent.change(input, { target: { value: String(value) } });
    await user.click(screen.getByTestId('save-behaviors'));
    await waitFor(() => expect(replace).toHaveBeenCalledOnce());
    const timelines: DraftBehaviorTimeline[] = replace.mock.calls[0]?.[2];
    expect(timelines[0]?.phases[0]?.faults).toEqual([{ device: 'resolver-1', type, value }]);
  },
);

it('keeps imported device faults when saving a loaded timeline', async () => {
  const content = `${draft.content}
behavior_timelines:
  - name: Outage
    repeat_count: 1
    phases:
      - name: Delay
        duration_ms: 1000
        faults:
          - device: resolver-1
            type: latency
            value: 60000
`;
  render(
    <DraftBehaviorComposer
      draft={{ ...draft, content }}
      onDraftUpdate={vi.fn()}
      onBusyChange={vi.fn()}
    />,
  );
  expect(screen.getByLabelText('Latency (milliseconds)')).toHaveValue(60000);
  await userEvent.click(screen.getByTestId('save-behaviors'));
  await waitFor(() => expect(replace).toHaveBeenCalledOnce());
  const timelines: DraftBehaviorTimeline[] = replace.mock.calls[0]?.[2];
  expect(timelines[0]?.phases[0]?.faults).toEqual([
    { device: 'resolver-1', type: 'latency', value: 60000 },
  ]);
});

it('saves link down as an interface outcome without a percentage control', async () => {
  const user = userEvent.setup();
  render(
    <DraftBehaviorComposer
      draft={{ ...draft, content: `${draft.content}    interfaces: [{name: eth0}]\n` }}
      onDraftUpdate={vi.fn()}
      onBusyChange={vi.fn()}
    />,
  );
  await user.click(screen.getByTestId('add-timeline'));
  await user.click(screen.getByRole('button', { name: /^Add fault$/ }));
  await user.selectOptions(screen.getByLabelText('Fault'), 'link_down');
  expect(screen.queryByLabelText('Rate (%)')).not.toBeInTheDocument();
  expect(screen.getByText('Forces the selected interface link down.')).toBeVisible();
  await user.click(screen.getByTestId('save-behaviors'));
  await waitFor(() => expect(replace).toHaveBeenCalledOnce());
  const timelines: DraftBehaviorTimeline[] = replace.mock.calls[0]?.[2];
  expect(timelines[0]?.phases[0]?.faults).toEqual([
    { device: 'resolver-1', interface: 'eth0', type: 'link_down', value: 1 },
  ]);
});

it('keeps imported link-down values without converting the outcome to a rate', async () => {
  const content = `${draft.content}    interfaces: [{name: eth0}]
behavior_timelines:
  - name: Link outage
    repeat_count: 1
    phases:
      - name: Down
        duration_ms: 1000
        faults:
          - device: resolver-1
            interface: eth0
            type: link_down
            value: 100
`;
  render(
    <DraftBehaviorComposer
      draft={{ ...draft, content }}
      onDraftUpdate={vi.fn()}
      onBusyChange={vi.fn()}
    />,
  );
  expect(screen.queryByLabelText('Rate (%)')).not.toBeInTheDocument();
  expect(screen.getByText('Forces the selected interface link down.')).toBeVisible();
  await userEvent.click(screen.getByTestId('save-behaviors'));
  await waitFor(() => expect(replace).toHaveBeenCalledOnce());
  const timelines: DraftBehaviorTimeline[] = replace.mock.calls[0]?.[2];
  expect(timelines[0]?.phases[0]?.faults).toEqual([
    { device: 'resolver-1', interface: 'eth0', type: 'link_down', value: 100 },
  ]);
});
