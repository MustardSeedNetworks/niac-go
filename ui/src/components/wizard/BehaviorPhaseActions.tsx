import { Trash2 } from 'lucide-react';
import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import type {
  DraftBehaviorFault,
  DraftBehaviorPhase,
  DraftBehaviorTraffic,
} from '../../api/library-client';
import { iconSizes } from '../../constants/sizes';
import { Button } from '../../ui/Button';
import { Input, Select } from '../../ui/Input';

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

const faultTypes: DraftBehaviorFault['type'][] = [
  'fcs_errors',
  'packet_discards',
  'interface_errors',
  'high_utilization',
];

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
  const updateFault = (index: number, update: (item: DraftBehaviorFault) => DraftBehaviorFault) =>
    onChange({
      ...phase,
      faults: phase.faults.map((item, itemIndex) => (itemIndex === index ? update(item) : item)),
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
        <div
          key={`fault-${actionIndex}`}
          className="grid gap-default md:grid-cols-[1fr_1fr_1fr_1fr_auto] md:items-end"
        >
          <Select
            label={t('newSimWizard.behaviors.device')}
            value={action.device}
            options={deviceOptions}
            onChange={(device) =>
              updateFault(actionIndex, (item) => ({
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
              updateFault(actionIndex, (item) => ({ ...item, interface: interfaceName }))
            }
          />
          <Select
            label={t('newSimWizard.behaviors.faultType')}
            value={action.type}
            options={faultTypes.map((type) => ({
              value: type,
              label: t(`newSimWizard.behaviors.faults.${type}`),
            }))}
            onChange={(type) =>
              updateFault(actionIndex, (item) => ({
                ...item,
                type: type as DraftBehaviorFault['type'],
              }))
            }
          />
          <Input
            label={t('newSimWizard.behaviors.faultRate')}
            type="number"
            min={1}
            max={100}
            value={action.value}
            onChange={(event) =>
              updateFault(actionIndex, (item) => ({
                ...item,
                value: Number(event.target.value),
              }))
            }
          />
          <Button
            variant="outline"
            tone="red"
            aria-label={t('newSimWizard.behaviors.removeFault')}
            onClick={() =>
              onChange({
                ...phase,
                faults: phase.faults.filter((_, item) => item !== actionIndex),
              })
            }
          >
            <Trash2 className={iconSizes.md} />
          </Button>
        </div>
      ))}

      <div className="flex flex-wrap gap-tight">
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
        >
          {t('newSimWizard.behaviors.addFault')}
        </Button>
      </div>
    </>
  );
};
