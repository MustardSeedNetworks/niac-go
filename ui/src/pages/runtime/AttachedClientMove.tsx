import { type FC, type FormEvent, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { pinSessionClient } from '../../api/client';
import { type ApiErrorDetail, isApiError } from '../../api/errors';
import type { AttachmentPort, CompiledAttachment, ObservedClient } from '../../api/types';
import { selectClassName } from '../../components/device-editor/types';
import { ApiErrorMessage } from '../../ui/ApiErrorMessage';
import { Button } from '../../ui/Button';
import { SmallText } from '../../ui/Typography';

interface AttachedClientMoveProps {
  sessionId: string;
  pool: CompiledAttachment;
  clients: readonly ObservedClient[];
  onMoved: () => void;
}

const portKey = (device: string, iface: string) => `${device}|${iface}`;

/**
 * The ports a client can move to: every pool port nobody else holds. Another
 * observed client's port and another MAC's pin are both taken; the client's
 * own port is left out because moving there changes nothing.
 */
const freePorts = (
  pool: CompiledAttachment,
  clients: readonly ObservedClient[],
  mac: string,
): AttachmentPort[] => {
  const taken = new Set<string>();
  for (const client of clients) {
    if (client.device && client.interface) taken.add(portKey(client.device, client.interface));
  }
  for (const pin of pool.pins ?? []) {
    if (pin.mac !== mac) taken.add(portKey(pin.device, pin.interface));
  }
  return (pool.ports ?? []).filter((port) => !taken.has(portKey(port.device, port.interface)));
};

/** Pins one attached client to another port of the session's pool. */
export const AttachedClientMove: FC<AttachedClientMoveProps> = ({
  sessionId,
  pool,
  clients,
  onMoved,
}) => {
  const { t } = useTranslation('pages');
  const [mac, setMac] = useState(clients[0]?.mac ?? '');
  const [port, setPort] = useState('');
  const [moving, setMoving] = useState(false);
  const [moved, setMoved] = useState<string | null>(null);
  const [failure, setFailure] = useState<{
    message: string;
    details: readonly ApiErrorDetail[];
  } | null>(null);

  // The selected client can expire out of the list between polls.
  const selectedMac = clients.some((client) => client.mac === mac) ? mac : (clients[0]?.mac ?? '');
  const ports = freePorts(pool, clients, selectedMac);
  const target = ports.find((candidate) => portKey(candidate.device, candidate.interface) === port);

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault();
    if (!target) return;
    setMoving(true);
    setMoved(null);
    setFailure(null);
    try {
      await pinSessionClient(sessionId, {
        mac: selectedMac,
        device: target.device,
        interface: target.interface,
      });
      setMoved(
        t('runtime.clients.move.moved', {
          mac: selectedMac,
          device: target.device,
          port: target.interface,
        }),
      );
      setPort('');
      onMoved();
    } catch (caught) {
      setFailure({
        message: t('runtime.clients.move.failed', { error: (caught as Error).message }),
        details: isApiError(caught) ? caught.details : [],
      });
    } finally {
      setMoving(false);
    }
  };

  return (
    <form className="stack-sm" onSubmit={handleSubmit} data-testid="attached-client-move">
      <SmallText>{t('runtime.clients.move.help')}</SmallText>
      <div className="flex flex-wrap items-end gap-default">
        <div className="min-w-[12rem]">
          <label htmlFor="attached-client-move-mac" className="block text-xs text-text-muted">
            {t('runtime.clients.move.client')}
          </label>
          <select
            id="attached-client-move-mac"
            data-testid="attached-client-move-mac"
            className={`${selectClassName} font-mono`}
            value={selectedMac}
            onChange={(event) => {
              setMac(event.target.value);
              setPort('');
            }}
          >
            {clients.map((client) => (
              <option key={client.mac} value={client.mac}>
                {client.mac}
              </option>
            ))}
          </select>
        </div>
        <div className="min-w-[16rem]">
          <label htmlFor="attached-client-move-port" className="block text-xs text-text-muted">
            {t('runtime.clients.move.port')}
          </label>
          <select
            id="attached-client-move-port"
            data-testid="attached-client-move-port"
            className={selectClassName}
            value={target ? port : ''}
            onChange={(event) => setPort(event.target.value)}
            disabled={ports.length === 0}
          >
            <option value="">
              {ports.length === 0
                ? t('runtime.clients.move.noFreePort')
                : t('runtime.clients.move.choosePort')}
            </option>
            {ports.map((candidate) => (
              <option
                key={portKey(candidate.device, candidate.interface)}
                value={portKey(candidate.device, candidate.interface)}
              >
                {candidate.device} {candidate.interface}
              </option>
            ))}
          </select>
        </div>
        <Button
          type="submit"
          action="start"
          size="sm"
          disabled={!target || moving}
          loading={moving}
          data-testid="attached-client-move-submit"
        >
          {t('runtime.clients.move.submit')}
        </Button>
      </div>
      {moved && (
        <SmallText className="text-status-success" role="status">
          {moved}
        </SmallText>
      )}
      {failure && <ApiErrorMessage message={failure.message} details={failure.details} />}
    </form>
  );
};
