import { render, screen, waitFor } from '@testing-library/react';
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
  name: 'address',
  revision: 'r1',
  format: 'yaml',
  modifiedAt: '',
  sizeBytes: 0,
  content: `devices:
  - name: server
    mac: 02:00:00:00:00:01
    ips: [192.0.2.1]
    dhcp: {pool_start: 192.0.2.100, pool_end: 192.0.2.110}
  - name: peer
    mac: 02:00:00:00:00:02
    ips: [192.0.2.20]
`,
};
beforeEach(() => {
  replace.mockReset();
  replace.mockResolvedValue(draft);
});

function renderDraft(content = draft.content) {
  render(
    <DraftBehaviorComposer
      draft={{ ...draft, content }}
      onDraftUpdate={vi.fn()}
      onBusyChange={vi.fn()}
    />,
  );
}

async function expectAddressSaved() {
  expect(screen.queryByLabelText('Interface')).not.toBeInTheDocument();
  expect(screen.queryByLabelText('Latency (milliseconds)')).not.toBeInTheDocument();
  await userEvent.click(screen.getByTestId('save-behaviors'));
  await waitFor(() => expect(replace).toHaveBeenCalledOnce());
  const timelines: DraftBehaviorTimeline[] = replace.mock.calls[0]?.[2];
  expect(timelines[0]?.phases[0]?.faults).toEqual([
    { device: 'server', type: 'duplicate_dhcp_offer', address: '192.0.2.20' },
  ]);
}

it('authors an address fault without retaining the numeric payload', async () => {
  renderDraft();
  await userEvent.click(screen.getByTestId('add-timeline'));
  await userEvent.click(screen.getByTestId('add-device-fault'));
  await userEvent.selectOptions(screen.getByLabelText('Fault'), 'duplicate_dhcp_offer');
  expect(screen.getByTestId('save-behaviors')).toBeDisabled();
  await userEvent.type(screen.getByLabelText('Conflict IPv4 address'), '192.0.2.20');
  await expectAddressSaved();
});

it('preserves an imported address payload when saving', async () => {
  renderDraft(`${draft.content}
behavior_timelines:
  - name: Conflict
    repeat_count: 1
    phases:
      - name: Offer
        duration_ms: 1000
        faults:
          - device: server
            type: duplicate_dhcp_offer
            address: 192.0.2.20
`);
  expect(screen.getByLabelText('Conflict IPv4 address')).toHaveValue('192.0.2.20');
  await expectAddressSaved();
});

it('keeps an imported non-DHCP target invalid instead of silently saving it', async () => {
  renderDraft(`${draft.content}
behavior_timelines:
  - name: Conflict
    repeat_count: 1
    phases:
      - name: Offer
        duration_ms: 1000
        faults:
          - device: peer
            type: duplicate_dhcp_offer
            address: 192.0.2.1
`);
  expect(screen.getByTestId('save-behaviors')).toBeDisabled();
});

it('blocks a numeric fault imported with an extra address instead of dropping the invalid payload', () => {
  renderDraft(`${draft.content}
behavior_timelines:
  - name: Conflict
    repeat_count: 1
    phases:
      - name: Offer
        duration_ms: 1000
        faults:
          - device: server
            type: latency
            value: 50
            address: 192.0.2.20
`);
  expect(screen.getByLabelText('Fault')).toHaveValue('latency');
  expect(screen.getByTestId('save-behaviors')).toBeDisabled();
});
