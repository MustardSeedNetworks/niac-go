/**
 * FinishStep.test.tsx
 *
 * The wizard's Save action must go through the existing config-save
 * endpoint (`PUT /api/v1/config`, wrapped by `updateConfig`) — same one
 * the Devices page's YAML editor uses. No new save endpoint.
 */
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { ConfigDocument } from '../../api/types';
import '../../i18n';
import { FinishStep } from './FinishStep';

const fetchConfig = vi.fn<() => Promise<ConfigDocument>>();
const updateConfig = vi.fn<(payload: { content: string }) => Promise<ConfigDocument>>();

vi.mock('../../api/client', () => ({
  fetchConfig: () => fetchConfig(),
  updateConfig: (payload: { content: string }) => updateConfig(payload),
}));

const doc: ConfigDocument = {
  path: '/tmp/config.yaml',
  filename: 'config.yaml',
  modifiedAt: new Date().toISOString(),
  sizeBytes: 42,
  deviceCount: 0,
  content: 'devices: []\n',
};

function renderStep() {
  return render(
    <MemoryRouter>
      <FinishStep />
    </MemoryRouter>,
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  fetchConfig.mockResolvedValue(doc);
  updateConfig.mockResolvedValue(doc);
});

describe('FinishStep', () => {
  it('calls the existing config-save endpoint on Save', async () => {
    const user = userEvent.setup();
    renderStep();

    await user.click(screen.getByTestId('wizard-save-button'));

    await waitFor(() => expect(updateConfig).toHaveBeenCalledTimes(1));
    expect(updateConfig).toHaveBeenCalledWith({ content: doc.content });
    expect(await screen.findByRole('status')).toHaveTextContent(/saved/i);
  });

  it('links to the existing Simulation page', () => {
    renderStep();
    expect(screen.getByTestId('wizard-goto-runtime').closest('a')).toHaveAttribute(
      'href',
      '/runtime',
    );
  });
});
