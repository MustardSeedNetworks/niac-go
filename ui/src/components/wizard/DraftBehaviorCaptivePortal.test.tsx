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
  name: 'portal',
  revision: 'r1',
  format: 'yaml',
  modifiedAt: '',
  sizeBytes: 0,
  content: 'devices:\n  - name: gateway-1\n    type: router\n    mac: 02:00:00:00:00:10\n',
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

async function expectPortalSaved() {
  expect(screen.queryByLabelText('Interface')).not.toBeInTheDocument();
  expect(screen.queryByLabelText('Rate (%)')).not.toBeInTheDocument();
  expect(screen.queryByLabelText('Utilization (%)')).not.toBeInTheDocument();
  expect(screen.queryByLabelText('Latency (milliseconds)')).not.toBeInTheDocument();
  expect(
    screen.getByText('Redirects HTTP requests to the simulated captive portal.'),
  ).toBeVisible();
  await userEvent.click(screen.getByTestId('save-behaviors'));
  await waitFor(() => expect(replace).toHaveBeenCalledOnce());
  const timelines: DraftBehaviorTimeline[] = replace.mock.calls[0]?.[2];
  expect(timelines[0]?.phases[0]?.faults).toEqual([
    { device: 'gateway-1', type: 'captive_portal', value: 1 },
  ]);
}

it('arms a device-scoped captive portal without retaining the previous rate', async () => {
  renderDraft();
  await userEvent.click(screen.getByTestId('add-timeline'));
  await userEvent.click(screen.getByTestId('add-device-fault'));
  await userEvent.clear(screen.getByLabelText('Latency (milliseconds)'));
  await userEvent.type(screen.getByLabelText('Latency (milliseconds)'), '500');
  await userEvent.selectOptions(screen.getByLabelText('Fault'), 'captive_portal');
  await expectPortalSaved();
});

it('preserves an imported captive portal when saving the timeline', async () => {
  renderDraft(`${draft.content}
behavior_timelines:
  - name: Guest access
    repeat_count: 1
    phases:
      - name: Sign in
        duration_ms: 1000
        faults:
          - device: gateway-1
            type: captive_portal
            value: 1
`);
  await expectPortalSaved();
});
