import { type FC } from 'react';
import { CollapsibleSection, FormField } from '../form';
import type { CDPConfig } from '../../api/types';
import type { ProtocolSectionProps } from './types';
import { inputClassName } from './types';

export const CDPSection: FC<ProtocolSectionProps> = ({
  device,
  isExpanded,
  onToggle,
  onUpdate,
}) => {
  const updateCDP = (config: CDPConfig | undefined) => {
    onUpdate('cdp', config);
  };

  return (
    <CollapsibleSection
      title="CDP (Cisco Discovery Protocol)"
      isExpanded={isExpanded}
      onToggle={onToggle}
      enabled={device.cdp?.enabled ?? false}
      onEnableChange={(enabled) => {
        updateCDP(enabled ? { enabled: true } as CDPConfig : undefined);
      }}
    >
      {device.cdp?.enabled && (
        <div className="grid gap-4 md:grid-cols-2">
          <FormField label="Platform" helpText="Hardware platform">
            <input
              type="text"
              value={device.cdp.platform || ''}
              onChange={(e) =>
                updateCDP({ ...device.cdp!, platform: e.target.value })
              }
              placeholder="cisco WS-C3750-48P"
              className={inputClassName}
            />
          </FormField>

          <FormField label="Port ID" helpText="Local port identifier">
            <input
              type="text"
              value={device.cdp.port_id || ''}
              onChange={(e) =>
                updateCDP({ ...device.cdp!, port_id: e.target.value })
              }
              placeholder="GigabitEthernet0/1"
              className={inputClassName}
            />
          </FormField>

          <FormField label="Software Version" helpText="IOS/NX-OS version">
            <input
              type="text"
              value={device.cdp.software_version || ''}
              onChange={(e) =>
                updateCDP({ ...device.cdp!, software_version: e.target.value })
              }
              placeholder="Cisco IOS Software, Version 15.1(4)M"
              className={inputClassName}
            />
          </FormField>

          <FormField label="Holdtime (seconds)" helpText="How long to hold neighbor information">
            <input
              type="number"
              value={device.cdp.holdtime || 180}
              onChange={(e) =>
                updateCDP({ ...device.cdp!, holdtime: parseInt(e.target.value) })
              }
              min={10}
              max={255}
              className={inputClassName}
            />
          </FormField>
        </div>
      )}
    </CollapsibleSection>
  );
};
