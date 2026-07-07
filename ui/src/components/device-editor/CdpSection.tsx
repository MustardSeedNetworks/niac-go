import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import type { CDPConfig } from '../../api/types';
import { CollapsibleSection, FormField } from '../form';
import type { ProtocolSectionProps } from './types';
import { inputClassName } from './types';

export const CdpSection: FC<ProtocolSectionProps> = ({
  device,
  isExpanded,
  onToggle,
  onUpdate,
}) => {
  const { t } = useTranslation('devices');
  const getCdpConfig = (): CDPConfig => ({
    enabled: true,
    ...(device.cdp ?? {}),
  });

  const updateCdp = (config: CDPConfig | undefined) => {
    onUpdate('cdp', config);
  };

  return (
    <CollapsibleSection
      title={t('editor.sections.cdp.title')}
      isExpanded={isExpanded}
      onToggle={onToggle}
      enabled={device.cdp?.enabled ?? false}
      onEnableChange={(enabled) => {
        updateCdp(enabled ? getCdpConfig() : undefined);
      }}
    >
      {device.cdp?.enabled && (
        <div className="grid gap-comfortable md:grid-cols-2">
          <FormField
            label={t('editor.sections.cdp.platformLabel')}
            helpText={t('editor.sections.cdp.platformHelp')}
          >
            <input
              type="text"
              value={device.cdp.platform || ''}
              onChange={(e) => updateCdp({ ...getCdpConfig(), platform: e.target.value })}
              placeholder={t('editor.sections.cdp.platformPlaceholder')}
              className={inputClassName}
            />
          </FormField>

          <FormField
            label={t('editor.sections.cdp.portIdLabel')}
            helpText={t('editor.sections.cdp.portIdHelp')}
          >
            <input
              type="text"
              value={device.cdp.portId || ''}
              onChange={(e) => updateCdp({ ...getCdpConfig(), portId: e.target.value })}
              placeholder={t('editor.sections.cdp.portIdPlaceholder')}
              className={inputClassName}
            />
          </FormField>

          <FormField
            label={t('editor.sections.cdp.softwareVersionLabel')}
            helpText={t('editor.sections.cdp.softwareVersionHelp')}
          >
            <input
              type="text"
              value={device.cdp.softwareVersion || ''}
              onChange={(e) =>
                updateCdp({
                  ...getCdpConfig(),
                  softwareVersion: e.target.value,
                })
              }
              placeholder={t('editor.sections.cdp.softwareVersionPlaceholder')}
              className={inputClassName}
            />
          </FormField>

          <FormField
            label={t('editor.sections.cdp.holdtimeLabel')}
            helpText={t('editor.sections.cdp.holdtimeHelp')}
          >
            <input
              type="number"
              value={device.cdp.holdtime || 180}
              onChange={(e) =>
                updateCdp({
                  ...getCdpConfig(),
                  holdtime: Number.parseInt(e.target.value, 10),
                })
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
