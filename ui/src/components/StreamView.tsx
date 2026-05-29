import { X } from 'lucide-react';
import { type FC, memo, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { PcapPacket } from '../api/types';
import { Button } from '../ui/Button';
import { SmallText } from '../ui/Typography';
import type { Packet } from './PacketList';

type PacketLike = Packet | PcapPacket;

type DisplayMode = 'ascii' | 'hex';

interface StreamViewProps {
  packets: PacketLike[];
  /** The source endpoint (ip:port) to mark as "client" direction */
  clientEndpoint: string;
  onClose: () => void;
}

function getPacketEndpoint(packet: PacketLike): string {
  return `${packet.sourceIp}:${packet.sourcePort ?? 0}`;
}

function getRawData(packet: PacketLike): string {
  if ('rawData' in packet && packet.rawData) return packet.rawData;
  return '';
}

/**
 * Convert hex string to ASCII, replacing non-printable chars with dots.
 */
function hexToAscii(hex: string): string {
  let result = '';
  for (let i = 0; i < hex.length; i += 2) {
    const byte = Number.parseInt(hex.substring(i, i + 2), 16);
    if (byte >= 32 && byte <= 126) {
      result += String.fromCharCode(byte);
    } else {
      result += '.';
    }
  }
  return result;
}

/**
 * Format hex string with spaces between bytes.
 */
function formatHex(hex: string): string {
  const bytes: string[] = [];
  for (let i = 0; i < hex.length; i += 2) {
    bytes.push(hex.substring(i, i + 2));
  }
  return bytes.join(' ');
}

interface StreamSegment {
  isClient: boolean;
  data: string; // raw hex
  timestamp: string;
}

/**
 * Stream View Modal
 *
 * Shows stream data from a TCP/UDP conversation with:
 * - Alternating colors for client (blue) and server (red) traffic
 * - ASCII / Hex display toggle
 * - Packet-by-packet segmentation
 */
export const StreamView: FC<StreamViewProps> = memo(({ packets, clientEndpoint, onClose }) => {
  const { t } = useTranslation('pages');
  const [displayMode, setDisplayMode] = useState<DisplayMode>('ascii');

  const segments = useMemo<StreamSegment[]>(() => {
    return packets
      .filter((p) => getRawData(p).length > 0)
      .map((p) => ({
        isClient: getPacketEndpoint(p) === clientEndpoint,
        data: getRawData(p),
        timestamp: p.timestamp,
      }));
  }, [packets, clientEndpoint]);

  const totalClientBytes = useMemo(
    () => segments.filter((s) => s.isClient).reduce((sum, s) => sum + s.data.length / 2, 0),
    [segments],
  );

  const totalServerBytes = useMemo(
    () => segments.filter((s) => !s.isClient).reduce((sum, s) => sum + s.data.length / 2, 0),
    [segments],
  );

  return (
    <div className="fixed inset-0 z-50 flex-center bg-scrim/60">
      <div className="w-full max-w-4xl mx-4 bg-bg-surface border border-surface-border rounded-xl shadow-2xl max-h-[85vh] flex flex-col">
        {/* Header */}
        <div className="flex-between px-5 py-4 border-b border-surface-border">
          <div>
            <h3 className="heading-3 text-text-primary">
              {t('packets.inspector.followStreamTitle')}
            </h3>
            <SmallText className="text-text-muted">
              {packets.length} packets |{' '}
              <span className="text-status-info">{totalClientBytes} B client</span> /{' '}
              <span className="text-status-error">{totalServerBytes} B server</span>
            </SmallText>
          </div>
          <div className="flex items-center gap-default">
            {/* Display mode toggle */}
            <div className="flex rounded-lg border border-surface-border bg-bg-base/50 p-1">
              <button
                type="button"
                onClick={() => setDisplayMode('ascii')}
                className={`px-3 py-compact text-xs rounded-md transition-colors ${
                  displayMode === 'ascii'
                    ? 'bg-brand-primary text-text-primary'
                    : 'text-text-muted hover:text-text-primary'
                }`}
              >
                ASCII
              </button>
              <button
                type="button"
                onClick={() => setDisplayMode('hex')}
                className={`px-3 py-compact text-xs rounded-md transition-colors ${
                  displayMode === 'hex'
                    ? 'bg-brand-primary text-text-primary'
                    : 'text-text-muted hover:text-text-primary'
                }`}
              >
                Hex
              </button>
            </div>

            <button
              type="button"
              onClick={onClose}
              className="p-1 text-text-muted hover:text-text-primary"
            >
              <X className="h-5 w-5" />
            </button>
          </div>
        </div>

        {/* Stream content */}
        <div className="flex-1 overflow-y-auto px-5 py-4">
          {segments.length === 0 ? (
            <div className="text-center py-8 text-text-muted">
              <p>{t('packets.inspector.noPayload')}</p>
            </div>
          ) : (
            <div className="stack-xs font-mono text-xs">
              {segments.map((segment, idx) => (
                <div
                  key={`${segment.timestamp}-${idx}`}
                  className={`px-3 py-compact-md rounded whitespace-pre-wrap break-all ${
                    segment.isClient
                      ? 'bg-status-info/30 text-status-info border-l-2 border-status-info'
                      : 'bg-status-error/30 text-status-error border-l-2 border-status-error'
                  }`}
                >
                  {displayMode === 'ascii' ? hexToAscii(segment.data) : formatHex(segment.data)}
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end px-5 py-row-lg border-t border-surface-border">
          <Button variant="ghost" size="sm" onClick={onClose}>
            Close
          </Button>
        </div>
      </div>
    </div>
  );
});

StreamView.displayName = 'StreamView';

export default StreamView;
