import { useTranslation } from 'react-i18next';
import { Button } from '../../ui/Button';
import { SmallText } from '../../ui/Typography';
import { FormField } from '../form/FormField';
import type {
  AuthoredAttachment,
  AuthoredAttachmentPool,
  DeviceAddressing,
} from './network-addressing';

interface AttachmentPoolEditorProps {
  index: number;
  attachment: AuthoredAttachment;
  devices: DeviceAddressing[];
  onChange: (patch: Partial<AuthoredAttachment>) => void;
  inputClassName: string;
}

const portKey = (device: string, port: string) => `${device}|${port}`;

/**
 * Authors the pool of free ports an attachment offers, and the clients pinned
 * to them. A pool may list spare ports on several switches, so a pin can move a
 * tester from one switch to another on the running session.
 *
 * The VLAN each port carries is shown and never edited here: it is the port's
 * own, decided by the device, and it is a different namespace from the VLAN the
 * NIAC host is cabled on. Offering to change it would invite the two to be
 * confused.
 */
export function AttachmentPoolEditor({
  index,
  attachment,
  devices,
  onChange,
  inputClassName,
}: AttachmentPoolEditorProps) {
  const { t } = useTranslation('pages');
  const groups = attachment.at ?? [{ device: '', ports: [] }];
  const pins = attachment.pins ?? [];
  const poolPorts = groups.flatMap((group) =>
    group.ports.map((port) => ({ device: group.device, port })),
  );

  // A pin survives only while its port is still in the pool.
  const commit = (next: AuthoredAttachmentPool[]) =>
    onChange({
      at: next,
      pins: pins.filter((pin) =>
        next.some((group) => group.device === pin.device && group.ports.includes(pin.interface)),
      ),
    });

  const setDevice = (groupIndex: number, name: string) =>
    commit(groups.map((group, i) => (i === groupIndex ? { device: name, ports: [] } : group)));

  const togglePort = (groupIndex: number, name: string) =>
    commit(
      groups.map((group, i) => {
        if (i !== groupIndex) return group;
        const ports = group.ports.includes(name)
          ? group.ports.filter((port) => port !== name)
          : [...group.ports, name];
        return { ...group, ports };
      }),
    );

  const chosen = new Set(groups.map((group) => group.device));
  const lastGroup = groups.at(-1);

  return (
    <div className="stack md:col-span-2">
      {groups.map((group, groupIndex) => {
        const device = devices.find((candidate) => candidate.device === group.device);
        const free = (device?.ports ?? []).filter((port) => !port.occupied && port.vlan !== null);
        const id = `${index}-${groupIndex}`;
        return (
          <div key={`group-${groupIndex}`} className="stack-xs">
            <div className="flex items-end gap-default">
              <FormField
                label={t('newSimWizard.networks.attachmentDevice')}
                htmlFor={`attachment-device-${id}`}
              >
                <select
                  id={`attachment-device-${id}`}
                  data-testid={`attachment-device-${id}`}
                  className={inputClassName}
                  value={group.device}
                  onChange={(event) => setDevice(groupIndex, event.target.value)}
                >
                  <option value="">{t('newSimWizard.networks.attachmentDeviceEmpty')}</option>
                  {devices
                    .filter(
                      (candidate) =>
                        candidate.device === group.device || !chosen.has(candidate.device),
                    )
                    .map((candidate) => (
                      <option key={candidate.device} value={candidate.device}>
                        {candidate.device}
                      </option>
                    ))}
                </select>
              </FormField>
              {groups.length > 1 && (
                <Button
                  variant="outline"
                  data-testid={`attachment-group-remove-${id}`}
                  onClick={() => commit(groups.filter((_, i) => i !== groupIndex))}
                >
                  {t('newSimWizard.networks.remove')}
                </Button>
              )}
            </div>

            {group.device !== '' && free.length === 0 && (
              <SmallText className="text-status-warning" data-testid={`attachment-no-free-${id}`}>
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
                    data-testid={`attachment-port-row-${id}-${port.name}`}
                    className="flex items-center gap-default text-sm text-text-primary"
                  >
                    <input
                      type="checkbox"
                      data-testid={`attachment-port-${id}-${port.name}`}
                      checked={group.ports.includes(port.name)}
                      onChange={() => togglePort(groupIndex, port.name)}
                    />
                    <span>{port.name}</span>
                    <SmallText className="text-text-muted">
                      {t('newSimWizard.networks.attachmentPortVlan', { vlan: port.vlan })}
                    </SmallText>
                  </label>
                ))}
              </div>
            )}
          </div>
        );
      })}

      {lastGroup !== undefined && lastGroup.ports.length > 0 && (
        <div>
          <Button
            variant="outline"
            data-testid={`attachment-group-add-${index}`}
            onClick={() => commit([...groups, { device: '', ports: [] }])}
          >
            {t('newSimWizard.networks.attachmentDeviceAdd')}
          </Button>
        </div>
      )}

      {poolPorts.length > 0 && (
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
                    ...pins,
                    {
                      mac: '',
                      device: poolPorts[0]?.device ?? '',
                      interface: poolPorts[0]?.port ?? '',
                    },
                  ],
                })
              }
            >
              {t('newSimWizard.networks.attachmentPinAdd')}
            </Button>
          </div>
          {pins.map((pin, pinIndex) => (
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
                      pins: pins.map((current, i) =>
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
                  value={portKey(pin.device, pin.interface)}
                  onChange={(event) => {
                    const target = poolPorts.find(
                      (candidate) =>
                        portKey(candidate.device, candidate.port) === event.target.value,
                    );
                    if (!target) return;
                    onChange({
                      pins: pins.map((current, i) =>
                        i === pinIndex
                          ? { ...current, device: target.device, interface: target.port }
                          : current,
                      ),
                    });
                  }}
                >
                  {poolPorts.map((candidate) => (
                    <option
                      key={portKey(candidate.device, candidate.port)}
                      value={portKey(candidate.device, candidate.port)}
                    >
                      {candidate.device} {candidate.port}
                    </option>
                  ))}
                </select>
              </FormField>
              <Button
                variant="outline"
                data-testid={`attachment-pin-remove-${index}-${pinIndex}`}
                onClick={() =>
                  onChange({
                    pins: pins.filter((_, i) => i !== pinIndex),
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
}
