import { Trash2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import {
  behaviorFaultMaximum,
  type DraftBehaviorFault,
  deviceBehaviorFaultTypes,
  interfaceBehaviorFaultTypes,
  isDeviceBehaviorFaultType,
  isInterfaceBehaviorFaultType,
} from '../../api/behavior-fault-types';
import { iconSizes } from '../../constants/sizes';
import { Button } from '../../ui/Button';
import { Input, Select } from '../../ui/Input';

interface Option {
  value: string;
  label: string;
}
interface BehaviorFaultActionProps {
  action: DraftBehaviorFault;
  deviceOptions: Option[];
  interfaceOptions: (device: string) => Option[];
  onChange: (action: DraftBehaviorFault) => void;
  onRemove: () => void;
}

export function BehaviorFaultAction({
  action,
  deviceOptions,
  interfaceOptions,
  onChange,
  onRemove,
}: BehaviorFaultActionProps) {
  const { t } = useTranslation('pages');
  const deviceScoped = isDeviceBehaviorFaultType(action.type);
  const faultTypes = deviceScoped ? deviceBehaviorFaultTypes : interfaceBehaviorFaultTypes;
  return (
    <div className="grid gap-default md:grid-cols-[1fr_1fr_1fr_1fr_auto] md:items-end">
      <Select
        label={t('newSimWizard.behaviors.device')}
        value={action.device}
        options={deviceOptions}
        onChange={(device) =>
          onChange(
            isDeviceBehaviorFaultType(action.type)
              ? { device, type: action.type, value: action.value }
              : {
                  device,
                  type: action.type,
                  value: action.value,
                  interface: interfaceOptions(device)[0]?.value ?? '',
                },
          )
        }
      />
      {!deviceScoped && (
        <Select
          label={t('newSimWizard.behaviors.interface')}
          value={action.interface ?? ''}
          options={interfaceOptions(action.device)}
          onChange={(interfaceName) => {
            if (isInterfaceBehaviorFaultType(action.type))
              onChange({ ...action, type: action.type, interface: interfaceName });
          }}
        />
      )}
      <Select
        label={t('newSimWizard.behaviors.faultType')}
        value={action.type}
        options={faultTypes.map((type) => ({
          value: type,
          label: t(`newSimWizard.behaviors.faults.${type}`),
        }))}
        onChange={(type) => {
          if (isDeviceBehaviorFaultType(type))
            onChange({ device: action.device, type, value: action.value });
          else if (isInterfaceBehaviorFaultType(type))
            onChange({
              device: action.device,
              type,
              interface: action.interface ?? '',
              value: action.value,
            });
        }}
      />
      <Input
        label={
          action.type === 'latency'
            ? t('newSimWizard.behaviors.latencyMs')
            : t('newSimWizard.behaviors.faultRate')
        }
        type="number"
        min={1}
        max={behaviorFaultMaximum(action.type)}
        step={1}
        value={action.value}
        onChange={(event) => onChange({ ...action, value: Number(event.target.value) })}
      />
      <Button
        variant="outline"
        tone="red"
        aria-label={t('newSimWizard.behaviors.removeFault')}
        onClick={onRemove}
      >
        <Trash2 className={iconSizes.md} />
      </Button>
    </div>
  );
}
