import type { FC } from 'react';
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
      title="Interfaces"
      isExpanded={isExpanded}
      onToggle={onToggle}
      required={false}
    >
      <div className="stack-lg">
        {interfaces.map((iface, index) => (
          <div key={`${iface.name || 'interface'}-${index}`} className="device-editor-row-panel">
            <div className="device-editor-field-grid">
              <FormField label="Name" required={true} htmlFor={`interface-${index}-name`}>
                <input
                  id={`interface-${index}-name`}
                  type="text"
                  aria-label={`Interface ${index + 1} name`}
                  value={iface.name}
                  onChange={(event) => updateInterface(index, 'name', event.target.value)}
                  placeholder="Ethernet1/1"
                  className={monoInputClassName}
                />
              </FormField>

              <FormField label="Speed (Mbps)" htmlFor={`interface-${index}-speed`}>
                <input
                  id={`interface-${index}-speed`}
                  type="number"
                  min="0"
                  step="1"
                  aria-label={`Interface ${index + 1} speed Mbps`}
                  value={iface.speed ?? 0}
                  onChange={(event) =>
                    updateInterface(index, 'speed', Number.parseInt(event.target.value, 10) || 0)
                  }
                  className={inputClassName}
                />
              </FormField>

              <FormField label="Duplex" htmlFor={`interface-${index}-duplex`}>
                <select
                  id={`interface-${index}-duplex`}
                  aria-label={`Interface ${index + 1} duplex`}
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
                  <option value="">Unset</option>
                  <option value="full">Full</option>
                  <option value="half">Half</option>
                  <option value="auto">Auto</option>
                </select>
              </FormField>

              <FormField label="Admin Status" htmlFor={`interface-${index}-admin-status`}>
                <select
                  id={`interface-${index}-admin-status`}
                  aria-label={`Interface ${index + 1} admin status`}
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
                  <option value="">Unset</option>
                  <option value="up">Up</option>
                  <option value="down">Down</option>
                </select>
              </FormField>

              <FormField label="Oper Status" htmlFor={`interface-${index}-oper-status`}>
                <select
                  id={`interface-${index}-oper-status`}
                  aria-label={`Interface ${index + 1} oper status`}
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
                  <option value="">Unset</option>
                  <option value="up">Up</option>
                  <option value="down">Down</option>
                  <option value="testing">Testing</option>
                </select>
              </FormField>

              <FormField label="Description" htmlFor={`interface-${index}-description`}>
                <input
                  id={`interface-${index}-description`}
                  type="text"
                  aria-label={`Interface ${index + 1} description`}
                  value={iface.description ?? ''}
                  onChange={(event) => updateInterface(index, 'description', event.target.value)}
                  placeholder="uplink"
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
                Remove
              </Button>
            </div>
          </div>
        ))}

        <Button type="button" onClick={addInterface} variant="outline" tone="gray" size="sm">
          Add Interface
        </Button>
      </div>
    </CollapsibleSection>
  );
};
