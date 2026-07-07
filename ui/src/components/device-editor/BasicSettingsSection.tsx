import type { FC } from 'react';
import type { FieldErrors } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import type { Device, DeviceType } from '../../api/types';
import { deviceTypeOptions as deviceTypes } from '../../constants/device-types';
import { CollapsibleSection } from '../form/CollapsibleSection';
import { FormField } from '../form/FormField';
import { inputClassName, monoInputClassName, selectClassName } from './types';

export interface BasicSettingsSectionProps {
  device: Device;
  isExpanded: boolean;
  onToggle: () => void;
  onUpdate: <K extends keyof Device>(field: K, value: Device[K]) => void;
  /** Inline validation errors from the editor's failed-save valibot check. */
  errors?: FieldErrors<Device>;
}

export const BasicSettingsSection: FC<BasicSettingsSectionProps> = ({
  device,
  isExpanded,
  onToggle,
  onUpdate,
  errors,
}) => {
  const { t } = useTranslation('devices');
  return (
    <CollapsibleSection
      title={t('editor.sections.basic.title')}
      isExpanded={isExpanded}
      onToggle={onToggle}
      required={true}
    >
      <div className="grid gap-comfortable md:grid-cols-2">
        <FormField
          label={t('editor.sections.basic.hostnameLabel')}
          required={true}
          helpText={t('editor.sections.basic.hostnameHelp')}
          htmlFor="device-hostname"
          error={errors?.hostname?.message}
        >
          <input
            id="device-hostname"
            type="text"
            value={device.hostname}
            onChange={(e) => onUpdate('hostname', e.target.value)}
            placeholder={t('editor.sections.basic.hostnamePlaceholder')}
            className={inputClassName}
          />
        </FormField>

        <FormField
          label={t('editor.sections.basic.macLabel')}
          required={true}
          helpText={t('editor.sections.basic.macHelp')}
          htmlFor="device-mac"
          error={errors?.mac?.message}
        >
          <input
            id="device-mac"
            type="text"
            value={device.mac}
            onChange={(e) => onUpdate('mac', e.target.value.toUpperCase())}
            placeholder={t('editor.sections.basic.macPlaceholder')}
            className={monoInputClassName}
          />
        </FormField>

        <FormField
          label={t('editor.sections.basic.typeLabel')}
          helpText={t('editor.sections.basic.typeHelp')}
          htmlFor="device-type"
        >
          <select
            id="device-type"
            value={device.type || 'unknown'}
            onChange={(e) => onUpdate('type', e.target.value as DeviceType)}
            className={selectClassName}
          >
            {deviceTypes.map((type) => (
              <option key={type.value} value={type.value}>
                {type.label}
              </option>
            ))}
          </select>
        </FormField>

        <FormField
          label={t('editor.sections.basic.ipLabel')}
          helpText={t('editor.sections.basic.ipHelp')}
          htmlFor="device-primary-ip"
          error={errors?.ip?.message}
        >
          <input
            id="device-primary-ip"
            type="text"
            value={device.ip || ''}
            onChange={(e) => onUpdate('ip', e.target.value)}
            placeholder={t('editor.sections.basic.ipPlaceholder')}
            className={monoInputClassName}
          />
        </FormField>
      </div>
    </CollapsibleSection>
  );
};
