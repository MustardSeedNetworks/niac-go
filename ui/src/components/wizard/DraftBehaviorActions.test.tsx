import { render, screen, waitFor, within } from '@testing-library/react';
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
  name: 'actions',
  revision: 'r1',
  format: 'yaml',
  modifiedAt: '',
  sizeBytes: 0,
  content:
    'devices:\n  - name: switch-1\n    type: switch\n    mac: 02:00:00:00:00:10\n  - name: switch-2\n    type: switch\n    mac: 02:00:00:00:00:11\n',
};
beforeEach(() => {
  replace.mockReset();
  replace.mockResolvedValue(draft);
});

it.each(['reboot', 'stp_topology_change'])(
  'authors %s without an interface or value',
  async (type) => {
    const user = userEvent.setup();
    render(<DraftBehaviorComposer draft={draft} onDraftUpdate={vi.fn()} onBusyChange={vi.fn()} />);
    await user.click(screen.getByTestId('add-timeline'));
    await user.click(screen.getByTestId('add-device-action'));
    const row = within(screen.getByTestId('behavior-device-action'));
    await user.selectOptions(row.getByLabelText('Operation'), type);
    await user.selectOptions(row.getByLabelText('Device'), 'switch-2');
    expect(row.queryByRole('spinbutton')).not.toBeInTheDocument();
    expect(row.queryByLabelText('Interface')).not.toBeInTheDocument();
    expect(row.queryByRole('checkbox')).not.toBeInTheDocument();
    await user.click(screen.getByTestId('save-behaviors'));
    await waitFor(() => expect(replace).toHaveBeenCalledOnce());
    const timelines: DraftBehaviorTimeline[] = replace.mock.calls[0]?.[2];
    expect(timelines[0]?.phases[0]).toMatchObject({
      actions: [{ device: 'switch-2', type }],
      faults: [],
      traffic: [],
      reset: true,
    });
  },
);

it('blocks duplicate operations and permits a different kind or device', async () => {
  const user = userEvent.setup();
  render(<DraftBehaviorComposer draft={draft} onDraftUpdate={vi.fn()} onBusyChange={vi.fn()} />);
  await user.click(screen.getByTestId('add-timeline'));
  await user.click(screen.getByTestId('add-device-action'));
  await user.click(screen.getByTestId('add-device-action'));
  expect(screen.getByTestId('save-behaviors')).toBeDisabled();
  const secondRow = screen.getAllByTestId('behavior-device-action')[1];
  if (!secondRow) throw new Error('Missing second operation');
  const second = within(secondRow);
  await user.selectOptions(second.getByLabelText('Operation'), 'stp_topology_change');
  expect(screen.getByTestId('save-behaviors')).toBeEnabled();
  await user.selectOptions(second.getByLabelText('Operation'), 'reboot');
  await user.selectOptions(second.getByLabelText('Device'), 'switch-2');
  expect(screen.getByTestId('save-behaviors')).toBeEnabled();
  await user.click(second.getByTestId('remove-device-action'));
  expect(screen.getAllByTestId('behavior-device-action')).toHaveLength(1);
});

it('keeps imported action order when saving an action-only phase', async () => {
  const content = `${draft.content}\nbehavior_timelines:\n  - name: Maintenance\n    repeat_count: 2\n    phases:\n      - name: Changes\n        duration_ms: 1000\n        reset: true\n        actions:\n          - {device: switch-2, type: stp_topology_change}\n          - {device: switch-1, type: reboot}\n`;
  render(
    <DraftBehaviorComposer
      draft={{ ...draft, content }}
      onDraftUpdate={vi.fn()}
      onBusyChange={vi.fn()}
    />,
  );
  expect(screen.getAllByTestId('behavior-device-action')).toHaveLength(2);
  await userEvent.click(screen.getByTestId('save-behaviors'));
  await waitFor(() => expect(replace).toHaveBeenCalledOnce());
  const timelines: DraftBehaviorTimeline[] = replace.mock.calls[0]?.[2];
  expect(timelines[0]).toMatchObject({
    repeatCount: 2,
    phases: [
      {
        reset: true,
        actions: [
          { device: 'switch-2', type: 'stp_topology_change' },
          { device: 'switch-1', type: 'reboot' },
        ],
      },
    ],
  });
});
