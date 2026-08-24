import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import type { AddMib, MibType } from '../../api/service-protocol-types';
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
  const { t } = useTranslation('devices');
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

  const addMibs = device.snmpAgent?.addMibs ?? [];
  const updateAddMibs = (next: AddMib[]) => updateSnmp({ ...getSnmpConfig(), addMibs: next });
  const mibTypes: MibType[] = [
    'STRING',
    'INTEGER',
    'Counter32',
    'Counter64',
    'Gauge32',
    'TimeTicks',
    'OID',
    'IpAddress',
    'Hex-STRING',
  ];

  return (
    <CollapsibleSection
      id="snmp-section"
      title={t('editor.sections.snmp.title')}
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
          <FormField
            label={t('editor.sections.snmp.communityLabel')}
            helpText={t('editor.sections.snmp.communityHelp')}
          >
            <input
              type="text"
              value={device.snmpAgent.community || ''}
              onChange={(e) => updateSnmp({ ...getSnmpConfig(), community: e.target.value })}
              placeholder="public"
              className={inputClassName}
            />
          </FormField>

          <FormField
            label={t('editor.sections.snmp.sysNameLabel')}
            helpText={t('editor.sections.snmp.sysNameHelp')}
          >
            <input
              type="text"
              value={device.snmpAgent.sysName || ''}
              onChange={(e) => updateSnmp({ ...getSnmpConfig(), sysName: e.target.value })}
              placeholder={t('editor.sections.snmp.sysNamePlaceholder')}
              className={inputClassName}
            />
          </FormField>

          <FormField
            label={t('editor.sections.snmp.walkFileLabel')}
            helpText={t('editor.sections.snmp.walkFileHelp')}
            className="md:col-span-2"
          >
            <select
              value={currentWalk}
              onChange={(e) => updateSnmp({ ...getSnmpConfig(), walkFile: e.target.value })}
              className={selectClassName}
            >
              <option value="">{t('editor.sections.snmp.walkFileSelectPrompt')}</option>
              {currentWalkMissing && (
                <option value={currentWalk}>
                  {currentWalk} {t('editor.sections.snmp.walkFileMissingSuffix')}
                </option>
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

          <div className="md:col-span-2 space-y-2">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium text-text-primary">MIB overrides</span>
              <button
                type="button"
                className="rounded bg-brand-primary px-2 py-1 text-xs text-knob"
                onClick={() => updateAddMibs([...addMibs, { oid: '', type: 'STRING', value: '' }])}
              >
                {t('editor.addOid')}
              </button>
            </div>
            {addMibs.map((mib, index) => (
              <div
                className="grid gap-2 md:grid-cols-[2fr_1fr_2fr_auto]"
                key={`${index}-${mib.oid}`}
              >
                <input
                  aria-label="MIB OID"
                  className={inputClassName}
                  placeholder="1.3.6.1.4.1..."
                  value={mib.oid}
                  onChange={(e) =>
                    updateAddMibs(
                      addMibs.map((item, i) =>
                        i === index ? { ...item, oid: e.target.value } : item,
                      ),
                    )
                  }
                />
                <select
                  aria-label="MIB type"
                  className={selectClassName}
                  value={mib.type}
                  onChange={(e) =>
                    updateAddMibs(
                      addMibs.map((item, i) =>
                        i === index ? { ...item, type: e.target.value as MibType } : item,
                      ),
                    )
                  }
                >
                  {mibTypes.map((type) => (
                    <option key={type} value={type}>
                      {type}
                    </option>
                  ))}
                </select>
                <input
                  aria-label="MIB value"
                  className={inputClassName}
                  value={mib.value}
                  onChange={(e) =>
                    updateAddMibs(
                      addMibs.map((item, i) =>
                        i === index ? { ...item, value: e.target.value } : item,
                      ),
                    )
                  }
                />
                <button
                  type="button"
                  aria-label="Remove MIB override"
                  className="rounded border border-status-error px-2 text-status-error"
                  onClick={() => updateAddMibs(addMibs.filter((_, i) => i !== index))}
                >
                  Remove
                </button>
              </div>
            ))}
          </div>
        </div>
      )}
    </CollapsibleSection>
  );
};
