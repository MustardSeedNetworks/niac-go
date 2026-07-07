import type { FC } from 'react';
import type { SNMPAgent } from '../../api/types';
import { CollapsibleSection, FormField } from '../form';
import { SynthesizeWalkControl } from './SynthesizeWalkControl';
import type { SNMPSectionProps } from './types';
import { inputClassName, selectClassName } from './types';

export const SnmpSection: FC<SNMPSectionProps> = ({
  device,
  isExpanded,
  onToggle,
  onUpdate,
  walkFiles,
  isNewDevice,
}) => {
  const getSnmpConfig = (): SNMPAgent => ({
    community: 'public',
    sysName: device.hostname,
    ...(device.snmpAgent ?? {}),
  });

  const updateSnmp = (config: SNMPAgent | undefined) => {
    onUpdate('snmpAgent', config);
  };

  // Walks are addressed by library-relative name (e.g. "cisco/cisco-c3900-01.walk")
  // and resolved server-side against ~/.niac/library/walks/. A config may still
  // reference a walk that has since been renamed or removed (the corpus moved to
  // IP-free canonical names), so keep the current value selectable rather than
  // letting it silently vanish from the picker.
  const currentWalk = device.snmpAgent?.walkFile ?? '';
  const currentWalkMissing =
    currentWalk !== '' && !(walkFiles ?? []).some((file) => file.name === currentWalk);

  return (
    <CollapsibleSection
      id="snmp-section"
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
            helpText="SNMP walk file from the library (Library › Walks)"
            className="md:col-span-2"
          >
            <select
              value={currentWalk}
              onChange={(e) => updateSnmp({ ...getSnmpConfig(), walkFile: e.target.value })}
              className={selectClassName}
            >
              <option value="">Select a walk file...</option>
              {currentWalkMissing && (
                <option value={currentWalk}>{currentWalk} (not in library)</option>
              )}
              {walkFiles?.map((file) => (
                <option key={file.name} value={file.name}>
                  {file.name}
                </option>
              ))}
            </select>
          </FormField>

          <div className="md:col-span-2">
            <SynthesizeWalkControl
              hostname={device.hostname}
              disabled={isNewDevice}
              onSynthesized={(walkPath) => updateSnmp({ ...getSnmpConfig(), walkFile: walkPath })}
            />
          </div>
        </div>
      )}
    </CollapsibleSection>
  );
};
