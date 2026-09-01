import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import type { Device, DeviceInterface } from '../../api/types';
import { Button } from '../../ui/Button';
import { CollapsibleSection } from '../form/CollapsibleSection';
import { FormField } from '../form/FormField';
import { inputClassName, monoInputClassName, selectClassName } from './types';

export interface InterfacesSectionProps {
  device: Device;
  isExpanded: boolean;
  onToggle: () => void;
  onUpdate: <K extends keyof Device>(field: K, value: Device[K]) => void;
}

const emptyInterface = (): DeviceInterface => ({
  name: '',
  speed: 1000,
  duplex: 'full',
  adminStatus: 'up',
  operStatus: 'up',
});

export const InterfacesSection: FC<InterfacesSectionProps> = ({
  device,
  isExpanded,
  onToggle,
  onUpdate,
}) => {
  const { t } = useTranslation('devices');
  const { t: tCommon } = useTranslation('common');
  const interfaces = device.interfaceDetails ?? [];

  const updateInterface = <K extends keyof DeviceInterface>(
    index: number,
    field: K,
    value: DeviceInterface[K],
  ) => {
    const next = interfaces.map((iface, ifaceIndex) =>
      ifaceIndex === index ? { ...iface, [field]: value } : iface,
    );
    onUpdate('interfaceDetails', next);
  };

  const addInterface = () => {
    onUpdate('interfaceDetails', [...interfaces, emptyInterface()]);
  };

  const removeInterface = (index: number) => {
    onUpdate(
      'interfaceDetails',
      interfaces.filter((_, ifaceIndex) => ifaceIndex !== index),
    );
  };

  return (
    <CollapsibleSection
      title={t('editor.interfaces')}
      isExpanded={isExpanded}
      onToggle={onToggle}
      required={false}
    >
      <div className="stack-lg">
        {interfaces.map((iface, index) => (
          <div key={`${iface.name || 'interface'}-${index}`} className="device-editor-row-panel">
            <div className="device-editor-field-grid">
              <FormField
                label={t('editor.sections.interfaces.nameLabel')}
                required={true}
                htmlFor={`interface-${index}-name`}
              >
                <input
                  id={`interface-${index}-name`}
                  type="text"
                  aria-label={t('editor.sections.interfaces.nameAria', { index: index + 1 })}
                  value={iface.name}
                  onChange={(event) => updateInterface(index, 'name', event.target.value)}
                  placeholder={t('editor.sections.interfaces.namePlaceholder')}
                  className={monoInputClassName}
                />
              </FormField>

              <FormField
                label={t('editor.sections.interfaces.speedLabel')}
                htmlFor={`interface-${index}-speed`}
              >
                <input
                  id={`interface-${index}-speed`}
                  type="number"
                  min="0"
                  step="1"
                  aria-label={t('editor.sections.interfaces.speedAria', { index: index + 1 })}
                  value={iface.speed ?? 0}
                  onChange={(event) =>
                    updateInterface(index, 'speed', Number.parseInt(event.target.value, 10) || 0)
                  }
                  className={inputClassName}
                />
              </FormField>

              <FormField
                label={t('editor.sections.interfaces.duplexLabel')}
                htmlFor={`interface-${index}-duplex`}
              >
                <select
                  id={`interface-${index}-duplex`}
                  aria-label={t('editor.sections.interfaces.duplexAria', { index: index + 1 })}
                  value={iface.duplex ?? ''}
                  onChange={(event) =>
                    updateInterface(
                      index,
                      'duplex',
                      event.target.value as DeviceInterface['duplex'],
                    )
                  }
                  className={selectClassName}
                >
                  <option value="">{t('editor.sections.interfaces.unsetOption')}</option>
                  <option value="full">{t('editor.sections.interfaces.fullOption')}</option>
                  <option value="half">{t('editor.sections.interfaces.halfOption')}</option>
                  <option value="auto">{t('editor.sections.interfaces.autoOption')}</option>
                </select>
              </FormField>

              <FormField
                label={t('editor.sections.interfaces.adminStatusLabel')}
                htmlFor={`interface-${index}-admin-status`}
              >
                <select
                  id={`interface-${index}-admin-status`}
                  aria-label={t('editor.sections.interfaces.adminStatusAria', { index: index + 1 })}
                  value={iface.adminStatus ?? ''}
                  onChange={(event) =>
                    updateInterface(
                      index,
                      'adminStatus',
                      event.target.value as DeviceInterface['adminStatus'],
                    )
                  }
                  className={selectClassName}
                >
                  <option value="">{t('editor.sections.interfaces.unsetOption')}</option>
                  <option value="up">{t('editor.sections.interfaces.upOption')}</option>
                  <option value="down">{t('editor.sections.interfaces.downOption')}</option>
                </select>
              </FormField>

              <FormField
                label={t('editor.sections.interfaces.operStatusLabel')}
                htmlFor={`interface-${index}-oper-status`}
              >
                <select
                  id={`interface-${index}-oper-status`}
                  aria-label={t('editor.sections.interfaces.operStatusAria', { index: index + 1 })}
                  value={iface.operStatus ?? ''}
                  onChange={(event) =>
                    updateInterface(
                      index,
                      'operStatus',
                      event.target.value as DeviceInterface['operStatus'],
                    )
                  }
                  className={selectClassName}
                >
                  <option value="">{t('editor.sections.interfaces.unsetOption')}</option>
                  <option value="up">{t('editor.sections.interfaces.upOption')}</option>
                  <option value="down">{t('editor.sections.interfaces.downOption')}</option>
                  <option value="testing">{t('editor.sections.interfaces.testingOption')}</option>
                </select>
              </FormField>

              <FormField
                label={tCommon('labels.description')}
                htmlFor={`interface-${index}-description`}
              >
                <input
                  id={`interface-${index}-description`}
                  type="text"
                  aria-label={t('editor.sections.interfaces.descriptionAria', { index: index + 1 })}
                  value={iface.description ?? ''}
                  onChange={(event) => updateInterface(index, 'description', event.target.value)}
                  placeholder={t('editor.sections.interfaces.descriptionPlaceholder')}
                  className={inputClassName}
                />
              </FormField>
            </div>

            <div className="device-editor-row-actions">
              <Button
                type="button"
                onClick={() => removeInterface(index)}
                variant="outline"
                tone="red"
                size="sm"
              >
                {tCommon('buttons.remove')}
              </Button>
            </div>
          </div>
        ))}

        <Button
          type="button"
          data-testid="add-interface"
          onClick={addInterface}
          variant="outline"
          tone="gray"
          size="sm"
        >
          {t('editor.sections.interfaces.addButton')}
        </Button>
      </div>
    </CollapsibleSection>
  );
};
