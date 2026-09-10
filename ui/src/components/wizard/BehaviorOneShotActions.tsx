import { Trash2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import {
  type DraftBehaviorAction,
  isBehaviorActionType,
  validBehaviorActions,
} from '../../api/behavior-timeline-types';
import { iconSizes } from '../../constants/sizes';
import { Button } from '../../ui/Button';
import { Select } from '../../ui/Input';
import { SmallText } from '../../ui/Typography';

interface BehaviorOneShotActionsProps {
  actions: DraftBehaviorAction[];
  deviceOptions: { value: string; label: string }[];
  firstDevice: string;
  onChange: (actions: DraftBehaviorAction[]) => void;
}

export function BehaviorOneShotActions({
  actions,
  deviceOptions,
  firstDevice,
  onChange,
}: BehaviorOneShotActionsProps) {
  const { t } = useTranslation('pages');
  const update = (index: number, action: DraftBehaviorAction) =>
    onChange(actions.map((current, item) => (item === index ? action : current)));
  return (
    <>
      {actions.map((action, index) => (
        <div
          key={`operation-${index}`}
          data-testid="behavior-device-action"
          className="grid gap-default md:grid-cols-[1fr_1fr_auto] md:items-end"
        >
          <Select
            label={t('newSimWizard.behaviors.device')}
            value={action.device}
            options={deviceOptions}
            onChange={(device) => update(index, { device, type: action.type })}
          />
          <Select
            label={t('newSimWizard.behaviors.operation')}
            value={action.type}
            options={[
              { value: 'reboot', label: t('newSimWizard.behaviors.operations.reboot') },
              {
                value: 'stp_topology_change',
                label: t('newSimWizard.behaviors.operations.stp_topology_change'),
              },
            ]}
            onChange={(type) => {
              if (isBehaviorActionType(type)) update(index, { device: action.device, type });
            }}
          />
          <Button
            variant="outline"
            tone="red"
            aria-label={t('newSimWizard.behaviors.removeDeviceAction')}
            data-testid="remove-device-action"
            onClick={() => onChange(actions.filter((_, item) => item !== index))}
          >
            <Trash2 className={iconSizes.md} />
          </Button>
        </div>
      ))}
      {actions.length > 0 && <SmallText>{t('newSimWizard.behaviors.operationEffect')}</SmallText>}
      {!validBehaviorActions(actions) && (
        <SmallText role="alert">{t('newSimWizard.behaviors.duplicateOperation')}</SmallText>
      )}
      <div>
        <Button
          variant="outline"
          disabled={!firstDevice}
          data-testid="add-device-action"
          onClick={() => onChange([...actions, { device: firstDevice, type: 'reboot' }])}
        >
          {t('newSimWizard.behaviors.addDeviceAction')}
        </Button>
      </div>
    </>
  );
}
