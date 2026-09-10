import { Trash2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import {
  behaviorFaultMaximum,
  type DraftBehaviorFault,
  deviceBehaviorFaultTypes,
  interfaceBehaviorFaultTypes,
  isDeviceBehaviorFaultType,
  isInterfaceBehaviorFaultType,
  isResourceBehaviorFaultType,
} from '../../api/behavior-fault-types';
import { iconSizes } from '../../constants/sizes';
import { Button } from '../../ui/Button';
import { Input, Select } from '../../ui/Input';
import { SmallText } from '../../ui/Typography';

interface Option {
  value: string;
  label: string;
}
interface BehaviorFaultActionProps {
  dhcpDevices: string[];
  action: DraftBehaviorFault;
  deviceOptions: Option[];
  interfaceOptions: (device: string) => Option[];
  onChange: (action: DraftBehaviorFault) => void;
  onRemove: () => void;
}

export function BehaviorFaultAction({
  dhcpDevices,
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
        options={
          action.type === 'duplicate_dhcp_offer'
            ? [
                { value: '', label: t('newSimWizard.behaviors.selectDhcpServer') },
                ...deviceOptions.filter((device) => dhcpDevices.includes(device.value)),
              ]
            : deviceOptions
        }
        onChange={(device) =>
          onChange(
            action.type === 'duplicate_dhcp_offer'
              ? { device, type: action.type, address: action.address }
              : action.type === 'duplicate_ip'
                ? {
                    device,
                    type: action.type,
                    address: action.address,
                    interface: interfaceOptions(device)[0]?.value ?? '',
                  }
                : isDeviceBehaviorFaultType(action.type)
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
            if (action.type === 'duplicate_ip') onChange({ ...action, interface: interfaceName });
            else if (
              action.type !== 'duplicate_dhcp_offer' &&
              isInterfaceBehaviorFaultType(action.type)
            )
              onChange({
                device: action.device,
                type: action.type,
                value: action.value,
                interface: interfaceName,
              });
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
          if (type === 'duplicate_dhcp_offer')
            onChange({
              device: dhcpDevices.includes(action.device) ? action.device : (dhcpDevices[0] ?? ''),
              type,
              address: '',
            });
          else if (type === 'duplicate_ip')
            onChange({
              device: action.device,
              type,
              interface: action.interface ?? '',
              address: '',
            });
          else if (isDeviceBehaviorFaultType(type) && type !== 'duplicate_dhcp_offer')
            onChange({
              device: action.device,
              type,
              value: type === 'captive_portal' ? 1 : (action.value ?? 1),
            });
          else if (isInterfaceBehaviorFaultType(type) && type !== 'duplicate_ip')
            onChange({
              device: action.device,
              type,
              interface: action.interface ?? '',
              value: type === 'link_down' ? 1 : (action.value ?? 1),
            });
        }}
      />
      {action.type === 'duplicate_dhcp_offer' || action.type === 'duplicate_ip' ? (
        <Input
          label={t('newSimWizard.behaviors.conflictAddress')}
          value={action.address}
          onChange={(event) => onChange({ ...action, address: event.target.value })}
        />
      ) : action.type === 'link_down' ? (
        <SmallText>{t('newSimWizard.behaviors.linkDownEffect')}</SmallText>
      ) : action.type === 'captive_portal' ? (
        <SmallText>{t('newSimWizard.behaviors.captivePortalEffect')}</SmallText>
      ) : (
        <Input
          label={
            action.type === 'latency'
              ? t('newSimWizard.behaviors.latencyMs')
              : isResourceBehaviorFaultType(action.type)
                ? t('newSimWizard.behaviors.resourcePercent')
                : t('newSimWizard.behaviors.faultRate')
          }
          type="number"
          min={1}
          max={behaviorFaultMaximum(action.type)}
          step={1}
          value={action.value}
          onChange={(event) => onChange({ ...action, value: Number(event.target.value) })}
        />
      )}
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
