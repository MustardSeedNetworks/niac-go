import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { useState } from 'react';
import { describe, expect, it, vi } from 'vitest';
import '../../i18n';
import { DEVICE_SECTIONS } from '../device-editor/generated/sections.generated';
import { DeviceProtocolsEditor } from './DeviceProtocolsEditor';

const config = `# clinic branch office
devices:
  - name: clinic-rtr-01
    type: router
    mac: "00:1A:2B:20:00:20"
    snmp_agent:
      enabled: true
      community: msn_public
      sysname: clinic-rtr-01

  - name: clinic-srv-01
    type: server
    mac: "00:1A:2B:20:00:21"
`;

const Harness = ({ initial = config }: { initial?: string }) => {
  const [content, setContent] = useState(initial);
  return (
    <>
      <DeviceProtocolsEditor
        content={content}
        onChange={setContent}
        devices={['clinic-rtr-01', 'clinic-srv-01']}
      />
      <pre data-testid="content">{content}</pre>
    </>
  );
};

describe('DeviceProtocolsEditor', () => {
  it('offers every generated section for every device', () => {
    render(
      <DeviceProtocolsEditor
        content={config}
        onChange={vi.fn()}
        devices={['clinic-rtr-01', 'clinic-srv-01']}
      />,
    );

    // Parity is the point: the wizard renders the same DEVICE_SECTIONS
    // manifest the device editor does, so a field one can set the other can.
    for (const section of DEVICE_SECTIONS) {
      expect(screen.getAllByText(section.title).length).toBeGreaterThanOrEqual(2);
    }
  });

  it('writes an edited field back into that device only', async () => {
    const user = userEvent.setup();
    render(<Harness />);

    const [firstDevice] = screen.getAllByText('Snmp agent');
    await user.click(firstDevice as HTMLElement);
    const sysname = screen.getByLabelText('Sysname');
    await user.clear(sysname);
    await user.type(sysname, 'renamed-rtr');

    const content = screen.getByTestId('content').textContent ?? '';
    expect(content).toContain('sysname: renamed-rtr');
    // The other device and the file's comment are untouched.
    expect(content).toContain('# clinic branch office');
    expect(content).toContain('mac: "00:1A:2B:20:00:21"');
  });

  it('renders nothing to author when the draft has no devices', () => {
    render(<DeviceProtocolsEditor content="devices: []\n" onChange={vi.fn()} devices={[]} />);

    expect(screen.queryByTestId('wizard-protocols-editor')).not.toBeInTheDocument();
  });

  it('skips a device the config does not actually contain', () => {
    render(
      <DeviceProtocolsEditor content={config} onChange={vi.fn()} devices={['no-such-device']} />,
    );

    const editor = screen.getByTestId('wizard-protocols-editor');
    expect(within(editor).queryByText('Snmp agent')).not.toBeInTheDocument();
  });
});
