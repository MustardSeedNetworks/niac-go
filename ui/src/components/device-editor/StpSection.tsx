import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import type { STPConfig } from '../../api/types';
import { CollapsibleSection, FormField } from '../form';
import type { ProtocolSectionProps } from './types';
import { inputClassName } from './types';

export const StpSection: FC<ProtocolSectionProps> = ({
  device,
  isExpanded,
  onToggle,
  onUpdate,
}) => {
  const { t } = useTranslation('devices');
  const getStpConfig = (): STPConfig => ({
    enabled: true,
    bridgePriority: 32768,
    ...(device.stp ?? {}),
  });

  const updateStp = (config: STPConfig | undefined) => {
    onUpdate('stp', config);
  };

  return (
    <CollapsibleSection
      title={t('editor.sections.stp.title')}
      isExpanded={isExpanded}
      onToggle={onToggle}
      enabled={device.stp?.enabled ?? false}
      onEnableChange={(enabled) => {
        updateStp(enabled ? getStpConfig() : undefined);
      }}
    >
      {device.stp?.enabled && (
        <div className="grid gap-comfortable md:grid-cols-2">
          <FormField
            label={t('editor.sections.stp.bridgePriorityLabel')}
            helpText={t('editor.sections.stp.bridgePriorityHelp')}
          >
            <input
              type="number"
              value={device.stp.bridgePriority ?? 32768}
              onChange={(e) =>
                updateStp({
                  ...getStpConfig(),
                  bridgePriority: Number.parseInt(e.target.value, 10),
                })
              }
              min={0}
              max={61440}
              step={4096}
              className={inputClassName}
            />
          </FormField>
        </div>
      )}
    </CollapsibleSection>
  );
};
