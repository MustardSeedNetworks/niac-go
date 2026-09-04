import { type FC, useMemo } from 'react';
import { DeviceProtocolsEditor } from './DeviceProtocolsEditor';
import { FleetDefaultsEditor } from './FleetDefaultsEditor';
import { parseNetworkModel } from './network-addressing';

interface ProtocolsStepProps {
  content: string;
  onChange: (content: string) => void;
}

/**
 * Step 4 authors protocols: the fleet-wide discovery defaults and the capture
 * playback, then every device's own blocks.
 *
 * It was a read-only summary. Phase 1b's parity decision is that the wizard
 * can author everything the daemon can run, so it edits the same generated
 * sections the device editor renders instead of describing them.
 */
export const ProtocolsStep: FC<ProtocolsStepProps> = ({ content, onChange }) => {
  const devices = useMemo(
    () => parseNetworkModel(content).devices.map((entry) => entry.device),
    [content],
  );

  return (
    <div className="stack">
      <FleetDefaultsEditor content={content} onChange={onChange} />
      <DeviceProtocolsEditor content={content} onChange={onChange} devices={devices} />
    </div>
  );
};
