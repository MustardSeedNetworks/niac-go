import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import type { LLDPConfig } from '../../api/types';
import { CollapsibleSection, FormField } from '../form';
import type { ProtocolSectionProps } from './types';
import { inputClassName } from './types';

export const LldpSection: FC<ProtocolSectionProps> = ({
  device,
  isExpanded,
  onToggle,
  onUpdate,
}) => {
  const { t } = useTranslation('devices');
  const getLldpConfig = (): LLDPConfig => ({
    enabled: true,
    ...(device.lldp ?? {}),
  });

  const updateLldp = (config: LLDPConfig | undefined) => {
    onUpdate('lldp', config);
  };

  return (
    <CollapsibleSection
      title={t('editor.sections.lldp.title')}
      isExpanded={isExpanded}
      onToggle={onToggle}
      enabled={device.lldp?.enabled ?? false}
      onEnableChange={(enabled) => {
        updateLldp(enabled ? getLldpConfig() : undefined);
      }}
    >
      {device.lldp?.enabled && (
        <div className="grid gap-comfortable md:grid-cols-2">
          <FormField
            label={t('editor.sections.lldp.systemDescriptionLabel')}
            helpText={t('editor.sections.lldp.systemDescriptionHelp')}
          >
            <input
              type="text"
              value={device.lldp.systemDescription || ''}
              onChange={(e) =>
                updateLldp({
                  ...getLldpConfig(),
                  systemDescription: e.target.value,
                })
              }
              placeholder={t('editor.sections.lldp.systemDescriptionPlaceholder')}
              className={inputClassName}
            />
          </FormField>

          <FormField
            label={t('editor.sections.lldp.portDescriptionLabel')}
            helpText={t('editor.sections.lldp.portDescriptionHelp')}
          >
            <input
              type="text"
              value={device.lldp.portDescription || ''}
              onChange={(e) =>
                updateLldp({
                  ...getLldpConfig(),
                  portDescription: e.target.value,
                })
              }
              placeholder={t('editor.sections.lldp.portDescriptionPlaceholder')}
              className={inputClassName}
            />
          </FormField>

          <FormField
            label={t('editor.sections.lldp.advertiseIntervalLabel')}
            helpText={t('editor.sections.lldp.advertiseIntervalHelp')}
          >
            <input
              type="number"
              value={device.lldp.advertiseInterval || 30}
              onChange={(e) =>
                updateLldp({
                  ...getLldpConfig(),
                  advertiseInterval: Number.parseInt(e.target.value, 10),
                })
              }
              min={5}
              max={32768}
              className={inputClassName}
            />
          </FormField>

          <FormField
            label={t('editor.sections.lldp.ttlLabel')}
            helpText={t('editor.sections.lldp.ttlHelp')}
          >
            <input
              type="number"
              value={device.lldp.ttl || 120}
              onChange={(e) =>
                updateLldp({
                  ...getLldpConfig(),
                  ttl: Number.parseInt(e.target.value, 10),
                })
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
