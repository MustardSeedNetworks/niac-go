import { type FC } from 'react';
import { CollapsibleSection, FormField } from '../form';
import type { LLDPConfig } from '../../api/types';
import type { ProtocolSectionProps } from './types';
import { inputClassName } from './types';

export const LLDPSection: FC<ProtocolSectionProps> = ({
  device,
  isExpanded,
  onToggle,
  onUpdate,
}) => {
  const updateLLDP = (config: LLDPConfig | undefined) => {
    onUpdate('lldp', config);
  };

  return (
    <CollapsibleSection
      title="LLDP (Link Layer Discovery Protocol)"
      isExpanded={isExpanded}
      onToggle={onToggle}
      enabled={device.lldp?.enabled ?? false}
      onEnableChange={(enabled) => {
        updateLLDP(enabled ? { enabled: true } as LLDPConfig : undefined);
      }}
    >
      {device.lldp?.enabled && (
        <div className="grid gap-4 md:grid-cols-2">
          <FormField label="System Description" helpText="LLDP system description">
            <input
              type="text"
              value={device.lldp.system_description || ''}
              onChange={(e) =>
                updateLLDP({ ...device.lldp!, system_description: e.target.value })
              }
              placeholder="Cisco IOS Software..."
              className={inputClassName}
            />
          </FormField>

          <FormField label="Port Description" helpText="LLDP port description">
            <input
              type="text"
              value={device.lldp.port_description || ''}
              onChange={(e) =>
                updateLLDP({ ...device.lldp!, port_description: e.target.value })
              }
              placeholder="GigabitEthernet0/1"
              className={inputClassName}
            />
          </FormField>

          <FormField label="Advertise Interval (seconds)" helpText="How often to send LLDP frames">
            <input
              type="number"
              value={device.lldp.advertise_interval || 30}
              onChange={(e) =>
                updateLLDP({ ...device.lldp!, advertise_interval: parseInt(e.target.value) })
              }
              min={5}
              max={32768}
              className={inputClassName}
            />
          </FormField>

          <FormField label="TTL (seconds)" helpText="Time-to-live for LLDP information">
            <input
              type="number"
              value={device.lldp.ttl || 120}
              onChange={(e) =>
                updateLLDP({ ...device.lldp!, ttl: parseInt(e.target.value) })
              }
              min={1}
              max={65535}
              className={inputClassName}
            />
          </FormField>
        </div>
      )}
    </CollapsibleSection>
  );
};
