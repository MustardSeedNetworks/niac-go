import type { FC } from 'react';
import type { FieldErrors } from 'react-hook-form';
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
  return (
    <CollapsibleSection
      title="Basic Settings"
      isExpanded={isExpanded}
      onToggle={onToggle}
      required={true}
    >
      <div className="grid gap-comfortable md:grid-cols-2">
        <FormField
          label="Hostname"
          required={true}
          helpText="Unique identifier for the device"
          htmlFor="device-hostname"
          error={errors?.hostname?.message}
        >
          <input
            id="device-hostname"
            type="text"
            value={device.hostname}
            onChange={(e) => onUpdate('hostname', e.target.value)}
            placeholder="e.g., core-switch-01"
            className={inputClassName}
          />
        </FormField>

        <FormField
          label="MAC Address"
          required={true}
          helpText="Hardware address in format XX:XX:XX:XX:XX:XX"
          htmlFor="device-mac"
          error={errors?.mac?.message}
        >
          <input
            id="device-mac"
            type="text"
            value={device.mac}
            onChange={(e) => onUpdate('mac', e.target.value.toUpperCase())}
            placeholder="e.g., 00:1A:2B:3C:4D:5E"
            className={monoInputClassName}
          />
        </FormField>

        <FormField label="Device Type" helpText="Category of network device" htmlFor="device-type">
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
          label="Primary IP Address"
          helpText="Main management IP address"
          htmlFor="device-primary-ip"
          error={errors?.ip?.message}
        >
          <input
            id="device-primary-ip"
            type="text"
            value={device.ip || ''}
            onChange={(e) => onUpdate('ip', e.target.value)}
            placeholder="e.g., 192.168.1.1"
            className={monoInputClassName}
          />
        </FormField>
      </div>
    </CollapsibleSection>
  );
};
