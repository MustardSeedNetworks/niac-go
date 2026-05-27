import type { FC } from 'react';
import type { FileEntry, SNMPAgent } from '../../api/types';
import { CollapsibleSection, FormField } from '../form';
import type { SNMPSectionProps } from './types';
import { inputClassName, selectClassName } from './types';

export const SnmpSection: FC<SNMPSectionProps> = ({
  device,
  isExpanded,
  onToggle,
  onUpdate,
  walkFiles,
}) => {
  const getSnmpConfig = (): SNMPAgent => ({
    community: 'public',
    sysName: device.hostname,
    ...(device.snmpAgent ?? {}),
  });

  const updateSnmp = (config: SNMPAgent | undefined) => {
    onUpdate('snmpAgent', config);
  };

  return (
    <CollapsibleSection
      title="SNMP Agent"
      isExpanded={isExpanded}
      onToggle={onToggle}
      enabled={!!device.snmpAgent}
      onEnableChange={(enabled) => {
        if (enabled) {
          updateSnmp(getSnmpConfig());
        } else {
          updateSnmp(undefined);
        }
      }}
    >
      {device.snmpAgent && (
        <div className="grid gap-comfortable md:grid-cols-2">
          <FormField label="Community String" helpText="SNMP v1/v2c community string">
            <input
              type="text"
              value={device.snmpAgent.community || ''}
              onChange={(e) => updateSnmp({ ...getSnmpConfig(), community: e.target.value })}
              placeholder="public"
              className={inputClassName}
            />
          </FormField>

          <FormField label="System Name" helpText="SNMP sysName value">
            <input
              type="text"
              value={device.snmpAgent.sysName || ''}
              onChange={(e) => updateSnmp({ ...getSnmpConfig(), sysName: e.target.value })}
              placeholder="Device hostname"
              className={inputClassName}
            />
          </FormField>

          <FormField
            label="Walk File"
            helpText="Path to SNMP walk file for emulation"
            className="md:col-span-2"
          >
            <select
              value={device.snmpAgent.walkFile || ''}
              onChange={(e) => updateSnmp({ ...getSnmpConfig(), walkFile: e.target.value })}
              className={selectClassName}
            >
              <option value="">Select a walk file...</option>
              {walkFiles?.map((file: FileEntry) => (
                <option key={file.path} value={file.path}>
                  {file.name}
                </option>
              ))}
            </select>
          </FormField>
        </div>
      )}
    </CollapsibleSection>
  );
};
