import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '../../ui/Button';
import { SmallText } from '../../ui/Typography';
import { FormField } from '../form/FormField';
import type { AuthoredAttachment, DeviceAddressing } from './network-addressing';

interface AttachmentPoolEditorProps {
  index: number;
  attachment: AuthoredAttachment;
  devices: DeviceAddressing[];
  onChange: (patch: Partial<AuthoredAttachment>) => void;
  inputClassName: string;
}

/**
 * Authors the pool of free ports an attachment offers, and the clients pinned
 * to them.
 *
 * The VLAN each port carries is shown and never edited here: it is the port's
 * own, decided by the device, and it is a different namespace from the VLAN the
 * NIAC host is cabled on. Offering to change it would invite the two to be
 * confused.
 */
export const AttachmentPoolEditor: FC<AttachmentPoolEditorProps> = ({
  index,
  attachment,
  devices,
  onChange,
  inputClassName,
}) => {
  const { t } = useTranslation('pages');
  const pool = attachment.at ?? { device: '', ports: [] };
  const device = devices.find((candidate) => candidate.device === pool.device);
  const free = (device?.ports ?? []).filter((port) => !port.occupied && port.vlan !== null);

  const setDevice = (name: string) => onChange({ at: { device: name, ports: [] }, pins: [] });

  const togglePort = (name: string) => {
    const ports = pool.ports.includes(name)
      ? pool.ports.filter((port) => port !== name)
      : [...pool.ports, name];
    onChange({
      at: { ...pool, ports },
      pins: (attachment.pins ?? []).filter((pin) => ports.includes(pin.interface)),
    });
  };

  return (
    <div className="stack md:col-span-2">
      <FormField
        label={t('newSimWizard.networks.attachmentDevice')}
        htmlFor={`attachment-device-${index}`}
      >
        <select
          id={`attachment-device-${index}`}
          data-testid={`attachment-device-${index}`}
          className={inputClassName}
          value={pool.device}
          onChange={(event) => setDevice(event.target.value)}
        >
          <option value="">{t('newSimWizard.networks.attachmentDeviceEmpty')}</option>
          {devices.map((candidate) => (
            <option key={candidate.device} value={candidate.device}>
              {candidate.device}
            </option>
          ))}
        </select>
      </FormField>

      {pool.device !== '' && free.length === 0 && (
        <SmallText className="text-status-warning" data-testid={`attachment-no-free-${index}`}>
          {t('newSimWizard.networks.attachmentNoFreePorts')}
        </SmallText>
      )}

      {free.length > 0 && (
        <div className="stack-xs">
          <SmallText className="text-text-muted">
            {t('newSimWizard.networks.attachmentPortsHelp')}
          </SmallText>
          {free.map((port) => (
            <label
              key={port.name}
              data-testid={`attachment-port-row-${index}-${port.name}`}
              className="flex items-center gap-default text-sm text-text-primary"
            >
              <input
                type="checkbox"
                data-testid={`attachment-port-${index}-${port.name}`}
                checked={pool.ports.includes(port.name)}
                onChange={() => togglePort(port.name)}
              />
              <span>{port.name}</span>
              <SmallText className="text-text-muted">
                {t('newSimWizard.networks.attachmentPortVlan', { vlan: port.vlan })}
              </SmallText>
            </label>
          ))}
        </div>
      )}

      {pool.ports.length > 0 && (
        <div className="stack-xs">
          <div className="flex items-center justify-between gap-default">
            <SmallText className="text-text-muted">
              {t('newSimWizard.networks.attachmentPinsHelp')}
            </SmallText>
            <Button
              variant="outline"
              data-testid={`attachment-pin-add-${index}`}
              onClick={() =>
                onChange({
                  pins: [
                    ...(attachment.pins ?? []),
                    { mac: '', device: pool.device, interface: pool.ports[0] ?? '' },
                  ],
                })
              }
            >
              {t('newSimWizard.networks.attachmentPinAdd')}
            </Button>
          </div>
          {(attachment.pins ?? []).map((pin, pinIndex) => (
            <div key={`pin-${pinIndex}`} className="grid gap-compact md:grid-cols-3 items-end">
              <FormField
                label={t('newSimWizard.networks.attachmentPinMac')}
                htmlFor={`attachment-pin-mac-${index}-${pinIndex}`}
              >
                <input
                  id={`attachment-pin-mac-${index}-${pinIndex}`}
                  data-testid={`attachment-pin-mac-${index}-${pinIndex}`}
                  className={inputClassName}
                  value={pin.mac}
                  onChange={(event) =>
                    onChange({
                      pins: (attachment.pins ?? []).map((current, i) =>
                        i === pinIndex ? { ...current, mac: event.target.value } : current,
                      ),
                    })
                  }
                />
              </FormField>
              <FormField
                label={t('newSimWizard.networks.attachmentPinPort')}
                htmlFor={`attachment-pin-port-${index}-${pinIndex}`}
              >
                <select
                  id={`attachment-pin-port-${index}-${pinIndex}`}
                  data-testid={`attachment-pin-port-${index}-${pinIndex}`}
                  className={inputClassName}
                  value={pin.interface}
                  onChange={(event) =>
                    onChange({
                      pins: (attachment.pins ?? []).map((current, i) =>
                        i === pinIndex
                          ? { ...current, device: pool.device, interface: event.target.value }
                          : current,
                      ),
                    })
                  }
                >
                  {pool.ports.map((port) => (
                    <option key={port} value={port}>
                      {port}
                    </option>
                  ))}
                </select>
              </FormField>
              <Button
                variant="outline"
                data-testid={`attachment-pin-remove-${index}-${pinIndex}`}
                onClick={() =>
                  onChange({
                    pins: (attachment.pins ?? []).filter((_, i) => i !== pinIndex),
                  })
                }
              >
                {t('newSimWizard.networks.remove')}
              </Button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};
