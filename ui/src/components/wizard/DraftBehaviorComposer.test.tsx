import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { ScenarioDraft } from '../../api/library-client';
import '../../i18n';
import { DraftBehaviorComposer } from './DraftBehaviorComposer';

const replaceScenarioDraftBehaviors = vi.hoisted(() => vi.fn());

vi.mock('../../api/library-client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../api/library-client')>();
  return { ...actual, replaceScenarioDraftBehaviors };
});

const draft: ScenarioDraft = {
  name: 'campus-draft',
  content: `
devices:
  - name: access-1
    type: switch
    interfaces:
      - name: Gi1/0/1
        type: ethernet
        speed: 1000000000
`,
  format: 'yaml',
  revision: 'revision-1',
  modifiedAt: '2026-07-29T12:00:00Z',
  sizeBytes: 150,
};

describe('DraftBehaviorComposer', () => {
  beforeEach(() => replaceScenarioDraftBehaviors.mockReset());

  it('builds and saves a deterministic traffic timeline', async () => {
    const user = userEvent.setup();
    const updated = { ...draft, revision: 'revision-2' };
    const onDraftUpdate = vi.fn();
    const onBusyChange = vi.fn();
    replaceScenarioDraftBehaviors.mockResolvedValue(updated);

    render(
      <DraftBehaviorComposer
        draft={draft}
        onDraftUpdate={onDraftUpdate}
        onBusyChange={onBusyChange}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Add timeline' }));
    await user.click(screen.getByTestId('save-behaviors'));

    await waitFor(() => expect(replaceScenarioDraftBehaviors).toHaveBeenCalledTimes(1));
    expect(replaceScenarioDraftBehaviors).toHaveBeenCalledWith('campus-draft', 'revision-1', [
      {
        name: 'Business day',
        startOffsetMs: 0,
        repeatCount: 1,
        phases: [
          {
            name: 'Busy period',
            startOffsetMs: 0,
            durationMs: 30000,
            reset: true,
            traffic: [{ device: 'access-1', interface: 'Gi1/0/1', utilization: 75 }],
            faults: [],
          },
        ],
      },
    ]);
    expect(onBusyChange.mock.calls).toEqual([[true], [false]]);
    expect(onDraftUpdate).toHaveBeenCalledWith(updated);
  });
});
