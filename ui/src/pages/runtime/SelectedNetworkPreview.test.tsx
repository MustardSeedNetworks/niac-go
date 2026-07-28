import { render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import '../../i18n';
import { SelectedNetworkPreview } from './SelectedNetworkPreview';

const fetchTemplateContent = vi.fn();
const fetchLibraryNetworkContent = vi.fn();

vi.mock('../../api/client', () => ({
  fetchTemplateContent: (name: string) => fetchTemplateContent(name),
}));
vi.mock('../../api/library-client', () => ({
  fetchLibraryNetworkContent: (name: string) => fetchLibraryNetworkContent(name),
}));

describe('SelectedNetworkPreview', () => {
  beforeEach(() => {
    fetchTemplateContent.mockReset();
    fetchLibraryNetworkContent.mockReset();
  });

  it('shows the empty-devices message for a valid config with no devices', async () => {
    fetchTemplateContent.mockResolvedValue({ content: 'devices: []\n' });

    render(<SelectedNetworkPreview source="template" name="empty.yaml" />);

    expect(
      await screen.findByText('Picked config has no devices — nothing will run.'),
    ).toBeInTheDocument();
  });

  it('surfaces a distinct line-numbered parse-error message for malformed YAML instead of the empty-devices message', async () => {
    fetchTemplateContent.mockResolvedValue({ content: 'devices: [\n  - broken: [unterminated' });

    render(<SelectedNetworkPreview source="template" name="broken.yaml" />);

    await waitFor(() => expect(screen.getByRole('alert')).toBeInTheDocument());
    // yaml's YAMLParseError reports the unterminated flow sequence on line 2
    // (1-based) — assert the structured "at line N" wording rather than the
    // generic fallback, proving the line was actually extracted.
    expect(screen.getByRole('alert')).toHaveTextContent("Couldn't parse config at line 2:");
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

  it('shows configured draft protocols without treating disabled services as active', async () => {
    render(
      <SelectedNetworkPreview
        source="upload"
        name="draft"
        content={`devices:
  - name: switch-1
    type: switch
    lldp:
      enabled: true
    http:
      enabled: false
    cdp:
      advertise_interval: 30
    dhcp:
      pool_start: 192.0.2.10
    reflector:
      latency_ms: 2
    netbios:
      enabled: true
`}
        view="protocols"
      />,
    );

    expect(await screen.findByText('switch-1')).toBeInTheDocument();
    expect(screen.getByText('LLDP')).toBeInTheDocument();
    expect(screen.getByText('DHCP')).toBeInTheDocument();
    expect(screen.getByText('Reflector')).toBeInTheDocument();
    expect(screen.getByText('NetBIOS')).toBeInTheDocument();
    expect(screen.queryByText('HTTP')).not.toBeInTheDocument();
    expect(screen.queryByText('CDP')).not.toBeInTheDocument();
  });

  it('includes devices nested under segments in the draft protocol view', async () => {
    render(
      <SelectedNetworkPreview
        source="upload"
        name="segmented-draft"
        content={`segments:
  - vlan: 200
    devices:
      - name: access-1
        type: switch
        lldp:
          enabled: true
`}
        view="protocols"
      />,
    );

    expect(await screen.findByText('access-1')).toBeInTheDocument();
    expect(screen.getByText('LLDP')).toBeInTheDocument();
  });
});
