import { Network, Plus, Trash2, Wand2 } from 'lucide-react';
import { type FC, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { iconSizes } from '../../constants/sizes';
import { Button } from '../../ui/Button';
import { Card, CardContent } from '../../ui/Card';
import { SmallText } from '../../ui/Typography';
import { spliceConfigSection } from '../../utils/config-section';
import { FormField } from '../form/FormField';
import { AttachmentPoolEditor } from './AttachmentPoolEditor';
import { setDeviceAddress } from './device-addressing';
import {
  type AuthoredAttachment,
  type AuthoredNetwork,
  nextFreeAddress,
  parseNetworkModel,
  serializeAttachments,
  serializeNetworks,
  takenAddresses,
} from './network-addressing';

interface NetworksStepProps {
  content: string;
  onChange: (content: string) => void;
}

const inputClassName =
  'w-full rounded border border-surface-border bg-bg-surface px-3 py-row text-sm text-text-primary';

/**
 * Step 3 authors the routed networks, the attachments that expose them, and
 * each device's address.
 *
 * It edits the draft's YAML through byte-range splices rather than holding its
 * own state and re-serializing the document, so a config the author uploaded
 * or generated keeps its comments and its spacing.
 */
export const NetworksStep: FC<NetworksStepProps> = ({ content, onChange }) => {
  const { t } = useTranslation('pages');
  const model = useMemo(() => parseNetworkModel(content), [content]);

  // A pool with no ports yet is not expressible in the document: the daemon
  // refuses an empty `ports:` list, and this step's whole design is that the
  // draft stays loadable after every keystroke. So the half-built pool lives
  // here until it has a port, and the document takes it from there.
  const [drafts, setDrafts] = useState<Record<number, Partial<AuthoredAttachment>>>({});

  const writeNetworks = (networks: AuthoredNetwork[]) =>
    onChange(spliceConfigSection(content, 'networks', serializeNetworks(networks)));

  const writeAttachments = (attachments: AuthoredAttachment[]) =>
    onChange(spliceConfigSection(content, 'attachments', serializeAttachments(attachments)));

  const updateNetwork = (index: number, patch: Partial<AuthoredNetwork>) =>
    writeNetworks(model.networks.map((n, i) => (i === index ? { ...n, ...patch } : n)));

  const updateAttachment = (index: number, patch: Partial<AuthoredAttachment>) =>
    writeAttachments(model.attachments.map((a, i) => (i === index ? { ...a, ...patch } : a)));

  const poolOf = (index: number, attachment: AuthoredAttachment) =>
    attachment.at ?? drafts[index]?.at;

  // The document keeps what it can express; the draft keeps the rest, so a pin
  // still being typed is not lost on the next re-parse.
  const attachmentWithDraft = (index: number, attachment: AuthoredAttachment) => ({
    ...attachment,
    at: poolOf(index, attachment),
    pins: attachment.pins ?? drafts[index]?.pins,
  });

  const selectAttachmentForm = (index: number, form: string) => {
    if (form === 'ports') {
      setDrafts({ ...drafts, [index]: { at: { device: '', ports: [] } } });
      return;
    }
    const { [index]: _removed, ...rest } = drafts;
    setDrafts(rest);
    updateAttachment(index, {
      connect: model.networks[0]?.name ?? '',
      at: undefined,
      pins: [],
    });
  };

  const updateAttachmentPool = (index: number, patch: Partial<AuthoredAttachment>) => {
    setDrafts({ ...drafts, [index]: { ...drafts[index], ...patch } });
    updateAttachment(index, patch);
  };

  const assign = (device: string, networkName: string) => {
    const network = model.networks.find((candidate) => candidate.name === networkName);
    if (!network) return;
    const address = nextFreeAddress(network.subnet, takenAddresses(content));
    if (!address) return;
    onChange(setDeviceAddress(content, device, network.name, address));
  };

  const assignAll = () => {
    let next = content;
    for (const entry of model.devices) {
      if (entry.address) continue;
      const network = model.networks[0];
      if (!network) break;
      const address = nextFreeAddress(network.subnet, takenAddresses(next));
      if (!address) break;
      next = setDeviceAddress(next, entry.device, network.name, address);
    }
    onChange(next);
  };

  const unaddressed = model.devices.filter((entry) => !entry.address).length;

  return (
    <div className="stack">
      <Card className="border-surface-border bg-bg-surface/70">
        <CardContent className="stack">
          <div className="flex items-center justify-between gap-default">
            <div className="flex items-center gap-default">
              <Network className={`${iconSizes.lg} text-brand-accent`} />
              <SmallText className="font-medium text-text-primary">
                {t('newSimWizard.networks.title')}
              </SmallText>
            </div>
            <Button
              variant="outline"
              data-testid="networks-add"
              onClick={() =>
                writeNetworks([
                  ...model.networks,
                  { name: `network-${model.networks.length + 1}`, subnet: '10.0.0.0/24' },
                ])
              }
            >
              <Plus className={iconSizes.sm} /> {t('newSimWizard.networks.add')}
            </Button>
          </div>

          {model.networks.length === 0 ? (
            <SmallText className="text-text-muted">{t('newSimWizard.networks.empty')}</SmallText>
          ) : (
            <div className="stack" data-testid="networks-list">
              {model.networks.map((network, index) => (
                <div
                  key={`${network.name}-${index}`}
                  className="grid gap-comfortable md:grid-cols-4 items-end"
                >
                  <FormField
                    label={t('newSimWizard.networks.name')}
                    htmlFor={`network-name-${index}`}
                  >
                    <input
                      id={`network-name-${index}`}
                      className={inputClassName}
                      value={network.name}
                      onChange={(event) => updateNetwork(index, { name: event.target.value })}
                    />
                  </FormField>
                  <FormField
                    label={t('newSimWizard.networks.subnet')}
                    htmlFor={`network-subnet-${index}`}
                  >
                    <input
                      id={`network-subnet-${index}`}
                      className={inputClassName}
                      value={network.subnet}
                      onChange={(event) => updateNetwork(index, { subnet: event.target.value })}
                    />
                  </FormField>
                  <FormField
                    label={t('newSimWizard.networks.virtualVlan')}
                    htmlFor={`network-vlan-${index}`}
                  >
                    <input
                      id={`network-vlan-${index}`}
                      className={inputClassName}
                      inputMode="numeric"
                      value={network.virtualVlan ?? ''}
                      onChange={(event) =>
                        updateNetwork(index, {
                          virtualVlan: event.target.value ? Number(event.target.value) : undefined,
                        })
                      }
                    />
                  </FormField>
                  <Button
                    variant="outline"
                    data-testid={`networks-remove-${index}`}
                    onClick={() => writeNetworks(model.networks.filter((_, i) => i !== index))}
                  >
                    <Trash2 className={iconSizes.sm} /> {t('newSimWizard.networks.remove')}
                  </Button>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <Card className="border-surface-border bg-bg-surface/70">
        <CardContent className="stack">
          <div className="flex items-center justify-between gap-default">
            <SmallText className="font-medium text-text-primary">
              {t('newSimWizard.networks.attachmentsTitle')}
            </SmallText>
            <Button
              variant="outline"
              data-testid="attachments-add"
              disabled={model.networks.length === 0}
              onClick={() =>
                writeAttachments([
                  ...model.attachments,
                  {
                    name: `attachment-${model.attachments.length + 1}`,
                    connect: model.networks[0]?.name ?? '',
                  },
                ])
              }
            >
              <Plus className={iconSizes.sm} /> {t('newSimWizard.networks.add')}
            </Button>
          </div>
          <SmallText className="text-text-muted">
            {t('newSimWizard.networks.attachmentsHelp')}
          </SmallText>

          {model.attachments.map((attachment, index) => (
            <div
              key={`${attachment.name}-${index}`}
              className="grid gap-comfortable md:grid-cols-3 items-end"
            >
              <FormField
                label={t('newSimWizard.networks.attachmentName')}
                htmlFor={`attachment-name-${index}`}
              >
                <input
                  id={`attachment-name-${index}`}
                  className={inputClassName}
                  value={attachment.name}
                  onChange={(event) => updateAttachment(index, { name: event.target.value })}
                />
              </FormField>
              <FormField
                label={t('newSimWizard.networks.attachmentForm')}
                htmlFor={`attachment-form-${index}`}
              >
                <select
                  id={`attachment-form-${index}`}
                  data-testid={`attachment-form-${index}`}
                  className={inputClassName}
                  value={poolOf(index, attachment) ? 'ports' : 'network'}
                  onChange={(event) => selectAttachmentForm(index, event.target.value)}
                >
                  <option value="network">
                    {t('newSimWizard.networks.attachmentFormNetwork')}
                  </option>
                  <option value="ports">{t('newSimWizard.networks.attachmentFormPorts')}</option>
                </select>
              </FormField>
              {poolOf(index, attachment) === undefined && (
                <FormField
                  label={t('newSimWizard.networks.attachmentConnect')}
                  htmlFor={`attachment-connect-${index}`}
                >
                  <select
                    id={`attachment-connect-${index}`}
                    className={inputClassName}
                    value={attachment.connect}
                    onChange={(event) => updateAttachment(index, { connect: event.target.value })}
                  >
                    {model.networks.map((network) => (
                      <option key={network.name} value={network.name}>
                        {network.name}
                      </option>
                    ))}
                  </select>
                </FormField>
              )}
              <Button
                variant="outline"
                data-testid={`attachments-remove-${index}`}
                onClick={() => writeAttachments(model.attachments.filter((_, i) => i !== index))}
              >
                <Trash2 className={iconSizes.sm} /> {t('newSimWizard.networks.remove')}
              </Button>
              {poolOf(index, attachment) !== undefined && (
                <AttachmentPoolEditor
                  index={index}
                  attachment={attachmentWithDraft(index, attachment)}
                  devices={model.devices}
                  onChange={(patch) => updateAttachmentPool(index, patch)}
                  inputClassName={inputClassName}
                />
              )}
            </div>
          ))}
        </CardContent>
      </Card>

      <Card className="border-surface-border bg-bg-surface/70">
        <CardContent className="stack">
          <div className="flex items-center justify-between gap-default">
            <SmallText className="font-medium text-text-primary">
              {t('newSimWizard.networks.addressingTitle')}
            </SmallText>
            <Button
              variant="outline"
              data-testid="addressing-assign-all"
              disabled={model.networks.length === 0 || unaddressed === 0}
              onClick={assignAll}
            >
              <Wand2 className={iconSizes.sm} /> {t('newSimWizard.networks.assignAll')}
            </Button>
          </div>

          {model.devices.length === 0 ? (
            <SmallText className="text-text-muted">
              {t('newSimWizard.networks.noDevices')}
            </SmallText>
          ) : (
            <div className="stack" data-testid="addressing-list">
              {model.devices.map((entry) => (
                <div key={entry.device} className="grid gap-comfortable md:grid-cols-3 items-end">
                  <SmallText className="font-mono text-text-primary">{entry.device}</SmallText>
                  <SmallText
                    data-testid={`addressing-address-${entry.device}`}
                    className="font-mono text-text-secondary"
                  >
                    {entry.address ?? t('newSimWizard.networks.unaddressed')}
                  </SmallText>
                  <select
                    aria-label={t('newSimWizard.networks.assignTo', { device: entry.device })}
                    data-testid={`addressing-network-${entry.device}`}
                    className={inputClassName}
                    value={entry.network ?? ''}
                    onChange={(event) => assign(entry.device, event.target.value)}
                  >
                    <option value="">{t('newSimWizard.networks.pickNetwork')}</option>
                    {model.networks.map((network) => (
                      <option key={network.name} value={network.name}>
                        {network.name}
                      </option>
                    ))}
                  </select>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
};
