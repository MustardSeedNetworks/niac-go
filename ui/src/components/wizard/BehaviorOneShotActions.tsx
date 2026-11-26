import { Trash2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import {
  type DraftBehaviorAction,
  isBehaviorActionType,
  validBehaviorActions,
} from '../../api/behavior-timeline-types';
import { iconSizes } from '../../constants/sizes';
import { Button } from '../../shared/Button';
import { FormField } from '../../shared/FormField';
import { Select } from '../../shared/Select';
import { SmallText } from '../../ui/Typography';
import type { DraftTopologyModel } from './draft-topology';

interface BehaviorOneShotActionsProps {
  actions: DraftBehaviorAction[];
  deviceOptions: { value: string; label: string }[];
  deviceActions: DraftTopologyModel['deviceActions'];
  onChange: (actions: DraftBehaviorAction[]) => void;
}

export function BehaviorOneShotActions({
  actions,
  deviceOptions,
  deviceActions,
  onChange,
}: BehaviorOneShotActionsProps) {
  const { t } = useTranslation('pages');
  const eligibleOptions = (type: DraftBehaviorAction['type']) =>
    deviceOptions.filter((option) => deviceActions[option.value]?.includes(type));
  const firstDevice = eligibleOptions('reboot')[0]?.value ?? '';
  const supports = (action: DraftBehaviorAction) =>
    deviceActions[action.device]?.includes(action.type);
  const optionsFor = (action: DraftBehaviorAction) =>
    supports(action)
      ? eligibleOptions(action.type)
      : [
          { value: action.device, label: action.device, disabled: true },
          ...eligibleOptions(action.type),
        ];
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
          <FormField label={t('newSimWizard.behaviors.device')}>
            {(control) => (
              <Select
                {...control}
                value={action.device}
                options={optionsFor(action)}
                onValueChange={(device) => update(index, { device, type: action.type })}
              />
            )}
          </FormField>
          <FormField label={t('newSimWizard.behaviors.operation')}>
            {(control) => (
              <Select
                {...control}
                value={action.type}
                options={[
                  { value: 'reboot', label: t('newSimWizard.behaviors.operations.reboot') },
                  {
                    value: 'stp_topology_change',
                    label: t('newSimWizard.behaviors.operations.stp_topology_change'),
                  },
                ].map((option) => ({
                  ...option,
                  disabled:
                    isBehaviorActionType(option.value) &&
                    eligibleOptions(option.value).length === 0,
                }))}
                onValueChange={(type) => {
                  if (isBehaviorActionType(type))
                    update(index, {
                      device: deviceActions[action.device]?.includes(type)
                        ? action.device
                        : (eligibleOptions(type)[0]?.value ?? ''),
                      type,
                    });
                }}
              />
            )}
          </FormField>
          <Button
            variant="destructive"
            aria-label={t('newSimWizard.behaviors.removeDeviceAction')}
            data-testid="remove-device-action"
            onClick={() => onChange(actions.filter((_, item) => item !== index))}
          >
            <Trash2 className={iconSizes.md} />
          </Button>
        </div>
      ))}
      {actions.length > 0 && <SmallText>{t('newSimWizard.behaviors.operationEffect')}</SmallText>}
      {(!firstDevice || actions.some((action) => !supports(action))) && (
        <SmallText role="alert">{t('newSimWizard.behaviors.ineligibleOperation')}</SmallText>
      )}
      {!validBehaviorActions(actions) && (
        <SmallText role="alert">{t('newSimWizard.behaviors.duplicateOperation')}</SmallText>
      )}
      <div>
        <Button
          variant="secondary"
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
