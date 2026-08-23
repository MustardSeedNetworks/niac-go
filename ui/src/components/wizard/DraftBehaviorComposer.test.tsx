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

/**
 * Guards #1491.
 *
 * The gate asked whether the *first* device has an interface, not whether any
 * device does. The Start-empty seed puts `new-device` — a host with no
 * interfaces — at position 0, so on CT304 the step stayed disabled after two
 * switches with four interfaces each had been added and one of those interfaces
 * was already carrying a link the wizard itself created.
 *
 * The same two values seed the default phase's traffic entry, so the timeline
 * must also target the device that actually has the interface.
 */
const draftWithInterfacelessFirstDevice: ScenarioDraft = {
  ...draft,
  content: `
devices:
  - name: new-device
    type: host
    mac: "02:00:00:00:00:01"
  - name: UITEST-SW-A
    type: switch
    interfaces:
      - name: Ethernet1/1
        type: ethernet
        speed: 1000000000
`,
};

describe('DraftBehaviorComposer with an interfaceless first device', () => {
  beforeEach(() => replaceScenarioDraftBehaviors.mockReset());

  it('enables the step using a later device that has interfaces', async () => {
    render(
      <DraftBehaviorComposer
        draft={draftWithInterfacelessFirstDevice}
        onDraftUpdate={vi.fn()}
        onBusyChange={vi.fn()}
      />,
    );

    expect(screen.getByRole('button', { name: 'Add timeline' })).toBeEnabled();
  });

  it('seeds the default phase with the device that has the interface', async () => {
    const user = userEvent.setup();
    replaceScenarioDraftBehaviors.mockResolvedValue({
      ...draftWithInterfacelessFirstDevice,
      revision: 'revision-2',
    });

    render(
      <DraftBehaviorComposer
        draft={draftWithInterfacelessFirstDevice}
        onDraftUpdate={vi.fn()}
        onBusyChange={vi.fn()}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Add timeline' }));
    await user.click(screen.getByTestId('save-behaviors'));

    await waitFor(() => expect(replaceScenarioDraftBehaviors).toHaveBeenCalledTimes(1));
    const timelines = replaceScenarioDraftBehaviors.mock.calls[0]?.[2];
    expect(timelines?.[0]?.phases[0]?.traffic).toEqual([
      { device: 'UITEST-SW-A', interface: 'Ethernet1/1', utilization: 75 },
    ]);
  });

  it('stays disabled when no device has an interface', () => {
    render(
      <DraftBehaviorComposer
        draft={{ ...draft, content: 'devices:\n  - name: new-device\n    type: host\n' }}
        onDraftUpdate={vi.fn()}
        onBusyChange={vi.fn()}
      />,
    );

    expect(screen.getByRole('button', { name: 'Add timeline' })).toBeDisabled();
  });
});
