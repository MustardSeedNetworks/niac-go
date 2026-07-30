import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import type { ScenarioDraft } from '../../api/library-client';
import '../../i18n';
import { DevicesStep } from './DevicesStep';

vi.mock('./DraftTopologyComposer', () => ({
  DraftTopologyComposer: () => <div data-testid="topology-composer" />,
}));

vi.mock('./DraftBehaviorComposer', () => ({
  DraftBehaviorComposer: () => <div data-testid="behavior-composer" />,
}));

vi.mock('../config/YamlEditor', () => ({
  YamlEditor: () => <div data-testid="yaml-editor" />,
}));

const draft: ScenarioDraft = {
  name: 'campus-draft',
  content: 'devices: []\n',
  format: 'yaml',
  revision: 'revision-1',
  modifiedAt: '2026-07-29T12:00:00Z',
  sizeBytes: 12,
};

describe('DevicesStep', () => {
  it('keeps config-backed segmented drafts in YAML mode', () => {
    const referencedDraft = {
      ...draft,
      content: 'segments:\n  - tag: 200\n    config: campus.yaml\n',
    };
    render(
      <DevicesStep
        draftName={referencedDraft.name}
        draft={referencedDraft}
        content={referencedDraft.content}
        dirty={false}
        saving={false}
        onChange={vi.fn()}
        onSave={vi.fn()}
        onDraftUpdate={vi.fn()}
        onBusyChange={vi.fn()}
      />,
    );

    expect(screen.getByTestId('yaml-editor')).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: 'Visual topology' })).toBeDisabled();
    expect(screen.getByRole('tab', { name: 'Behaviors' })).toBeDisabled();
    expect(screen.getByText(/must be edited in YAML/)).toBeVisible();
  });

  it('defaults to visual editing and saves dirty YAML before returning to it', async () => {
    const user = userEvent.setup();
    const onSave = vi.fn().mockResolvedValue(true);
    render(
      <DevicesStep
        draftName={draft.name}
        draft={draft}
        content={draft.content}
        dirty={true}
        saving={false}
        onChange={vi.fn()}
        onSave={onSave}
        onDraftUpdate={vi.fn()}
        onBusyChange={vi.fn()}
      />,
    );

    expect(screen.getByTestId('topology-composer')).toBeInTheDocument();
    await user.click(screen.getByRole('tab', { name: 'YAML' }));
    expect(screen.getByTestId('yaml-editor')).toBeInTheDocument();
    expect(onSave).not.toHaveBeenCalled();

    await user.click(screen.getByRole('tab', { name: 'Behaviors' }));
    expect(onSave).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId('behavior-composer')).toBeInTheDocument();
    await user.click(screen.getByRole('tab', { name: 'YAML' }));
    expect(screen.getByTestId('yaml-editor')).toBeInTheDocument();

    await user.click(screen.getByRole('tab', { name: 'Visual topology' }));
    expect(onSave).toHaveBeenCalledTimes(2);
    expect(screen.getByTestId('topology-composer')).toBeInTheDocument();
  });
});
