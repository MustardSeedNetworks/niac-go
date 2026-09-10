import { Trash2 } from 'lucide-react';
import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import type { DraftBehaviorPhase, DraftBehaviorTraffic } from '../../api/library-client';
import { iconSizes } from '../../constants/sizes';
import { Button } from '../../ui/Button';
import { Input, Select } from '../../ui/Input';
import { BehaviorFaultAction } from './BehaviorFaultAction';

interface Option {
  value: string;
  label: string;
}

interface BehaviorPhaseActionsProps {
  phase: DraftBehaviorPhase;
  deviceOptions: Option[];
  interfaceOptions: (device: string) => Option[];
  firstDevice: string;
  firstInterface: string;
  onChange: (phase: DraftBehaviorPhase) => void;
}

export const BehaviorPhaseActions: FC<BehaviorPhaseActionsProps> = ({
  phase,
  deviceOptions,
  interfaceOptions,
  firstDevice,
  firstInterface,
  onChange,
}) => {
  const { t } = useTranslation('pages');
  const updateTraffic = (
    index: number,
    update: (item: DraftBehaviorTraffic) => DraftBehaviorTraffic,
  ) =>
    onChange({
      ...phase,
      traffic: phase.traffic.map((item, itemIndex) => (itemIndex === index ? update(item) : item)),
    });

  return (
    <>
      {phase.traffic.map((action, actionIndex) => (
        <div
          key={`traffic-${actionIndex}`}
          className="grid gap-default md:grid-cols-[1fr_1fr_1fr_auto] md:items-end"
        >
          <Select
            label={t('newSimWizard.behaviors.device')}
            value={action.device}
            options={deviceOptions}
            onChange={(device) =>
              updateTraffic(actionIndex, (item) => ({
                ...item,
                device,
                interface: interfaceOptions(device)[0]?.value ?? '',
              }))
            }
          />
          <Select
            label={t('newSimWizard.behaviors.interface')}
            value={action.interface}
            options={interfaceOptions(action.device)}
            onChange={(interfaceName) =>
              updateTraffic(actionIndex, (item) => ({ ...item, interface: interfaceName }))
            }
          />
          <Input
            label={t('newSimWizard.behaviors.utilization')}
            type="number"
            min={1}
            max={100}
            value={action.utilization}
            onChange={(event) =>
              updateTraffic(actionIndex, (item) => ({
                ...item,
                utilization: Number(event.target.value),
              }))
            }
          />
          <Button
            variant="outline"
            tone="red"
            aria-label={t('newSimWizard.behaviors.removeTraffic')}
            onClick={() =>
              onChange({
                ...phase,
                traffic: phase.traffic.filter((_, item) => item !== actionIndex),
              })
            }
          >
            <Trash2 className={iconSizes.md} />
          </Button>
        </div>
      ))}

      {phase.faults.map((action, actionIndex) => (
        <BehaviorFaultAction
          key={`fault-${actionIndex}`}
          action={action}
          deviceOptions={deviceOptions}
          interfaceOptions={interfaceOptions}
          onChange={(updated) =>
            onChange({
              ...phase,
              faults: phase.faults.map((item, index) => (index === actionIndex ? updated : item)),
            })
          }
          onRemove={() =>
            onChange({ ...phase, faults: phase.faults.filter((_, index) => index !== actionIndex) })
          }
        />
      ))}

      <div className="flex flex-wrap gap-tight">
        <Button
          size="sm"
          variant="outline"
          disabled={!firstDevice}
          data-testid="add-device-fault"
          onClick={() =>
            onChange({
              ...phase,
              faults: [...phase.faults, { device: firstDevice, type: 'latency', value: 100 }],
            })
          }
        >
          {t('newSimWizard.behaviors.addDeviceFault')}
        </Button>
        <Button
          size="sm"
          variant="outline"
          disabled={!firstDevice || !firstInterface}
          onClick={() =>
            onChange({
              ...phase,
              traffic: [
                ...phase.traffic,
                { device: firstDevice, interface: firstInterface, utilization: 75 },
              ],
            })
          }
        >
          {t('newSimWizard.behaviors.addTraffic')}
        </Button>
        <Button
          size="sm"
          variant="outline"
          disabled={!firstDevice || !firstInterface}
          onClick={() =>
            onChange({
              ...phase,
              faults: [
                ...phase.faults,
                {
                  device: firstDevice,
                  interface: firstInterface,
                  type: 'packet_discards',
                  value: 5,
                },
              ],
            })
          }
          data-testid="add-interface-fault"
        >
          {t('newSimWizard.behaviors.addFault')}
        </Button>
      </div>
    </>
  );
};
