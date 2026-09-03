import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import { CollapsibleSection } from '../form/CollapsibleSection';
import { FormField } from '../form/FormField';
import type { AuthoredDevice, AuthoredValue } from './generated/authored-device.generated';
import { DEVICE_TYPES } from './generated/sections.generated';
import { inputClassName, monoInputClassName, selectClassName } from './types';
import type { DeviceFieldErrors } from './useDeviceEditor';

export interface BasicSettingsSectionProps {
  device: AuthoredDevice;
  isNewDevice: boolean;
  isExpanded: boolean;
  onToggle: () => void;
  onUpdate: (key: keyof AuthoredDevice, value: AuthoredValue) => void;
  errors?: DeviceFieldErrors;
}

/**
 * Device identity: the fields that are more than a value.
 *
 * `name` is the route key, `type` orders the rest of the form, and MAC-vs-vendor
 * is one choice rather than two fields — the daemon rejects a device carrying
 * both (ErrDeviceMACSourceConflict) and one carrying neither, so the form makes
 * the illegal states unrepresentable instead of reporting them after a save.
 * Everything else on a device is generated from the schema.
 */
export const BasicSettingsSection: FC<BasicSettingsSectionProps> = ({
  device,
  isNewDevice,
  isExpanded,
  onToggle,
  onUpdate,
  errors,
}) => {
  const { t } = useTranslation('devices');
  const identity = device.vendor === undefined ? 'mac' : 'vendor';

  const selectIdentity = (next: 'mac' | 'vendor') => {
    if (next === identity) {
      return;
    }
    if (next === 'mac') {
      onUpdate('vendor', undefined);
      onUpdate('mac_suffix', undefined);
      return;
    }
    onUpdate('mac', undefined);
    onUpdate('vendor', '');
  };

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
          // A rename is not an edit: the daemon takes the device's name from
          // the URL, so a changed `name` on an existing device would be
          // accepted and ignored. Clone carries a new name.
          helpText={
            isNewDevice
              ? t('editor.sections.basic.hostnameHelp')
              : t('editor.sections.basic.hostnameFixedHelp')
          }
          htmlFor="device-hostname"
          error={errors?.name}
        >
          <input
            id="device-hostname"
            type="text"
            value={device.name ?? ''}
            readOnly={!isNewDevice}
            onChange={(e) => onUpdate('name', e.target.value)}
            placeholder={t('editor.sections.basic.hostnamePlaceholder')}
            className={inputClassName}
          />
        </FormField>

        <FormField
          label={t('editor.sections.basic.typeLabel')}
          helpText={t('editor.sections.basic.typeHelp')}
          htmlFor="device-type"
        >
          <select
            id="device-type"
            value={device.type ?? ''}
            onChange={(e) => onUpdate('type', e.target.value === '' ? undefined : e.target.value)}
            className={selectClassName}
          >
            <option value="">{t('editor.fields.unset')}</option>
            {DEVICE_TYPES.map((type) => (
              <option key={type} value={type}>
                {type}
              </option>
            ))}
          </select>
        </FormField>

        <fieldset className="md:col-span-2 stack">
          <legend className="label">{t('editor.sections.basic.identityLabel')}</legend>
          <div className="flex gap-comfortable">
            <label htmlFor="device-identity-mac" className="flex items-center gap-compact text-sm">
              <input
                id="device-identity-mac"
                type="radio"
                name="device-identity"
                checked={identity === 'mac'}
                onChange={() => selectIdentity('mac')}
              />
              {t('editor.sections.basic.identityMac')}
            </label>
            <label
              htmlFor="device-identity-vendor"
              className="flex items-center gap-compact text-sm"
            >
              <input
                id="device-identity-vendor"
                type="radio"
                name="device-identity"
                checked={identity === 'vendor'}
                onChange={() => selectIdentity('vendor')}
              />
              {t('editor.sections.basic.identityVendor')}
            </label>
          </div>
        </fieldset>

        {identity === 'mac' ? (
          <FormField
            label={t('editor.sections.basic.macLabel')}
            required={true}
            helpText={t('editor.sections.basic.macHelp')}
            htmlFor="device-mac"
            error={errors?.mac}
          >
            <input
              id="device-mac"
              type="text"
              value={device.mac ?? ''}
              onChange={(e) => onUpdate('mac', e.target.value.toUpperCase())}
              placeholder={t('editor.sections.basic.macPlaceholder')}
              className={monoInputClassName}
            />
          </FormField>
        ) : (
          <>
            <FormField
              label={t('editor.sections.basic.vendorLabel')}
              required={true}
              helpText={t('editor.sections.basic.vendorHelp')}
              htmlFor="device-vendor"
              error={errors?.vendor ?? errors?.mac}
            >
              <input
                id="device-vendor"
                type="text"
                value={device.vendor ?? ''}
                onChange={(e) => onUpdate('vendor', e.target.value)}
                placeholder={t('editor.sections.basic.vendorPlaceholder')}
                className={inputClassName}
              />
            </FormField>

            <FormField
              // Without a distinct suffix every device of one vendor is
              // allocated the same address, so this is part of the identity
              // choice rather than an advanced option.
              label={t('editor.sections.basic.macSuffixLabel')}
              helpText={t('editor.sections.basic.macSuffixHelp')}
              htmlFor="device-mac-suffix"
            >
              <input
                id="device-mac-suffix"
                type="number"
                min={0}
                value={device.mac_suffix ?? ''}
                onChange={(e) =>
                  onUpdate('mac_suffix', e.target.value === '' ? undefined : e.target.valueAsNumber)
                }
                className={monoInputClassName}
              />
            </FormField>
          </>
        )}
      </div>
    </CollapsibleSection>
  );
};
