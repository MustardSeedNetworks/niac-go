import { render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { SelectedNetworkPreview } from './SelectedNetworkPreview';

const fetchTemplateContent = vi.fn();
const fetchUserConfigContent = vi.fn();

vi.mock('../../api/client', () => ({
  fetchTemplateContent: (name: string) => fetchTemplateContent(name),
  fetchUserConfigContent: (name: string) => fetchUserConfigContent(name),
}));

describe('SelectedNetworkPreview', () => {
  beforeEach(() => {
    fetchTemplateContent.mockReset();
    fetchUserConfigContent.mockReset();
  });

  it('shows the empty-devices message for a valid config with no devices', async () => {
    fetchTemplateContent.mockResolvedValue({ content: 'devices: []\n' });

    render(<SelectedNetworkPreview source="template" name="empty.yaml" />);

    expect(
      await screen.findByText('Picked config has no devices — nothing will run.'),
    ).toBeInTheDocument();
  });

  it('surfaces a distinct parse-error message for malformed YAML instead of the empty-devices message', async () => {
    fetchTemplateContent.mockResolvedValue({ content: 'devices: [\n  - broken: [unterminated' });

    render(<SelectedNetworkPreview source="template" name="broken.yaml" />);

    await waitFor(() => expect(screen.getByRole('alert')).toBeInTheDocument());
    expect(screen.getByRole('alert')).toHaveTextContent("Couldn't parse config:");
    expect(
      screen.queryByText('Picked config has no devices — nothing will run.'),
    ).not.toBeInTheDocument();
  });

  it('lists devices from a valid config', async () => {
    fetchTemplateContent.mockResolvedValue({
      content: 'devices:\n  - name: router1\n    type: router\n    ip: 10.0.0.1\n',
    });

    render(<SelectedNetworkPreview source="template" name="valid.yaml" />);

    expect(await screen.findByText('router1')).toBeInTheDocument();
  });
});
