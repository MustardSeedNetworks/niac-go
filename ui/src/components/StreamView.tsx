import { X } from 'lucide-react';
import { type FC, memo, useMemo, useState } from 'react';
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
    const byte = Number.parseInt(hex.slice(i, i + 2), 16);
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
    bytes.push(hex.slice(i, i + 2));
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
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
      <div className="w-full max-w-4xl mx-4 bg-gray-900 border border-white/10 rounded-xl shadow-2xl max-h-[85vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-white/10">
          <div>
            <h3 className="text-lg font-semibold text-white">Follow Stream</h3>
            <SmallText className="text-gray-400">
              {packets.length} packets |{' '}
              <span className="text-blue-400">{totalClientBytes} B client</span> /{' '}
              <span className="text-red-400">{totalServerBytes} B server</span>
            </SmallText>
          </div>
          <div className="flex items-center gap-3">
            {/* Display mode toggle */}
            <div className="flex rounded-lg border border-white/10 bg-gray-950/50 p-1">
              <button
                type="button"
                onClick={() => setDisplayMode('ascii')}
                className={`px-3 py-1 text-xs rounded-md transition-colors ${
                  displayMode === 'ascii'
                    ? 'bg-violet-600 text-white'
                    : 'text-gray-400 hover:text-white'
                }`}
              >
                ASCII
              </button>
              <button
                type="button"
                onClick={() => setDisplayMode('hex')}
                className={`px-3 py-1 text-xs rounded-md transition-colors ${
                  displayMode === 'hex'
                    ? 'bg-violet-600 text-white'
                    : 'text-gray-400 hover:text-white'
                }`}
              >
                Hex
              </button>
            </div>

            <button type="button" onClick={onClose} className="p-1 text-gray-400 hover:text-white">
              <X className="h-5 w-5" />
            </button>
          </div>
        </div>

        {/* Stream content */}
        <div className="flex-1 overflow-y-auto px-5 py-4">
          {segments.length === 0 ? (
            <div className="text-center py-8 text-gray-500">
              <p>No payload data available for this stream</p>
            </div>
          ) : (
            <div className="space-y-1 font-mono text-xs">
              {segments.map((segment, idx) => (
                <div
                  key={`${segment.timestamp}-${idx}`}
                  className={`px-3 py-1.5 rounded whitespace-pre-wrap break-all ${
                    segment.isClient
                      ? 'bg-blue-950/30 text-blue-300 border-l-2 border-blue-500'
                      : 'bg-red-950/30 text-red-300 border-l-2 border-red-500'
                  }`}
                >
                  {displayMode === 'ascii' ? hexToAscii(segment.data) : formatHex(segment.data)}
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end px-5 py-3 border-t border-white/10">
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
