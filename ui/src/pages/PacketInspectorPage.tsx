import { Download, Pause, Play, Trash2, Wifi, WifiOff } from 'lucide-react';
import { type FC, useCallback, useEffect, useRef, useState } from 'react';
import { HexDumpViewer } from '../components/HexDumpViewer';
import { PacketDetails } from '../components/PacketDetails';
import { type Packet, PacketList } from '../components/PacketList';
import { usePacketStream } from '../hooks/useEventSource';
import { Button } from '../ui/Button';
import { Card, CardContent } from '../ui/Card';
import { Tag } from '../ui/Tag';
import { H2, SmallText } from '../ui/Typography';

/** Maximum number of packets to buffer */
const MAX_PACKETS = 100;

/** Available protocol filters */
const PROTOCOL_FILTERS = ['All', 'ARP', 'ICMP', 'DNS', 'TCP', 'UDP', 'HTTP', 'DHCP'] as const;

/**
 * Generate unique packet ID
 */
function generatePacketId(): string {
  return `${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
}

/**
 * Connection Status Indicator Component
 * SSE auto-reconnects, so we just show connected/connecting states
 */
const ConnectionStatus: FC<{
  connected: boolean;
}> = ({ connected }) => {
  if (connected) {
    return (
      <div className="flex items-center gap-2">
        <Wifi className="h-4 w-4 text-green-400" />
        <Tag colorScheme="green">Connected</Tag>
      </div>
    );
  }

  // SSE auto-reconnects, so disconnected state is brief
  return (
    <div className="flex items-center gap-2">
      <WifiOff className="h-4 w-4 text-yellow-400 animate-pulse" />
      <Tag colorScheme="yellow">Connecting...</Tag>
    </div>
  );
};

/**
 * Packet Inspector Page
 *
 * Real-time packet hex dump viewer with:
 * - SSE streaming of live packets (auto-reconnects)
 * - Packet list with filtering
 * - Hex dump display with color-coded headers/payload
 * - Parsed packet details panel
 * - Auto-scroll toggle
 * - Protocol and search filtering
 */
export const PacketInspectorPage: FC = () => {
  // Packet buffer state
  const [packets, setPackets] = useState<Packet[]>([]);
  const [selectedPacket, setSelectedPacket] = useState<Packet | null>(null);

  // Filter state
  const [protocolFilter, setProtocolFilter] = useState<string>('All');
  const [searchQuery, setSearchQuery] = useState('');

  // UI state
  const [autoScroll, setAutoScroll] = useState(true);
  const [isPaused, setIsPaused] = useState(false);

  // Ref to track pause state in callback
  const isPausedRef = useRef(isPaused);
  useEffect(() => {
    isPausedRef.current = isPaused;
  }, [isPaused]);

  // SSE connection (auto-reconnects)
  const { connected, reconnect } = usePacketStream({
    onMessage: useCallback((data: unknown) => {
      // Skip if paused
      if (isPausedRef.current) {
        return;
      }

      // Transform incoming data to Packet format
      const incoming = data as Record<string, unknown>;

      const packet: Packet = {
        id: generatePacketId(),
        timestamp: (incoming.timestamp as string) || new Date().toISOString(),
        protocol: (incoming.protocol as string) || 'Unknown',
        sourceIp: (incoming.sourceIp as string) || (incoming.sourceIp as string) || 'Unknown',
        destIp: (incoming.destIp as string) || (incoming.destIp as string) || 'Unknown',
        sourcePort: incoming.sourcePort as number | undefined,
        destPort: incoming.destPort as number | undefined,
        size: (incoming.size as number) || 0,
        summary: (incoming.summary as string) || '',
        rawData: (incoming.rawData as string) || (incoming.payload as string) || '',
        headers: incoming.headers as Record<string, unknown> | undefined,
      };

      setPackets((prev) => {
        const updated = [...prev, packet];
        // Keep only the last MAX_PACKETS
        if (updated.length > MAX_PACKETS) {
          return updated.slice(-MAX_PACKETS);
        }
        return updated;
      });
    }, []),
  });

  // Clear all packets
  const handleClear = useCallback(() => {
    setPackets([]);
    setSelectedPacket(null);
  }, []);

  // Export packets as JSON
  const handleExport = useCallback(() => {
    if (packets.length === 0) {
      return;
    }

    const exportData = JSON.stringify(packets, null, 2);
    const blob = new Blob([exportData], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `packets-${new Date().toISOString().slice(0, 19).replace(/[:.]/g, '-')}.json`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  }, [packets]);

  // Toggle pause state
  const handlePauseToggle = useCallback(() => {
    setIsPaused((prev) => !prev);
  }, []);

  // Select packet handler
  const handleSelectPacket = useCallback((packet: Packet) => {
    setSelectedPacket(packet);
  }, []);

  return (
    <div className="space-y-6">
      {/* Header with controls */}
      <Card className="border-white/5 bg-gray-900/70">
        <CardContent>
          <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            {/* Title and connection status */}
            <div className="flex items-center gap-4">
              <H2 className="mb-0">Packet Inspector</H2>
              <ConnectionStatus connected={connected} />
            </div>

            {/* Control buttons */}
            <div className="flex flex-wrap items-center gap-2">
              <Button
                variant={isPaused ? 'outline' : 'ghost'}
                size="sm"
                onClick={handlePauseToggle}
                leftIcon={isPaused ? <Play className="h-4 w-4" /> : <Pause className="h-4 w-4" />}
              >
                {isPaused ? 'Resume' : 'Pause'}
              </Button>

              <Button
                variant="ghost"
                size="sm"
                onClick={handleClear}
                leftIcon={<Trash2 className="h-4 w-4" />}
                disabled={packets.length === 0}
              >
                Clear
              </Button>

              <Button
                variant="ghost"
                size="sm"
                onClick={handleExport}
                leftIcon={<Download className="h-4 w-4" />}
                disabled={packets.length === 0}
              >
                Export
              </Button>

              {/* SSE auto-reconnects, but manual reconnect is available */}
              {!connected && (
                <Button tone="violet" size="sm" onClick={reconnect}>
                  Reconnect
                </Button>
              )}
            </div>
          </div>

          {/* Filters row */}
          <div className="mt-4 flex flex-col gap-3 sm:flex-row sm:items-center">
            {/* Protocol filter */}
            <div className="flex items-center gap-2">
              <SmallText className="text-gray-400">Protocol:</SmallText>
              <select
                value={protocolFilter}
                onChange={(e) => setProtocolFilter(e.target.value)}
                aria-label="Protocol filter"
                className="rounded-lg border border-white/10 bg-gray-950/60 px-3 py-1.5 text-sm text-white focus:border-violet-400 focus:outline-none"
              >
                {PROTOCOL_FILTERS.map((protocol) => (
                  <option key={protocol} value={protocol}>
                    {protocol}
                  </option>
                ))}
              </select>
            </div>

            {/* Search box */}
            <div className="flex-1">
              <input
                type="text"
                placeholder="Search packets (IP, port, protocol...)"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                aria-label="Search packets"
                className="w-full rounded-lg border border-white/10 bg-gray-950/60 px-3 py-1.5 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
              />
            </div>

            {/* Auto-scroll toggle */}
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={autoScroll}
                onChange={(e) => setAutoScroll(e.target.checked)}
                className="rounded border-gray-600 bg-gray-800 text-violet-500 focus:ring-violet-400 focus:ring-offset-gray-900"
              />
              <SmallText className="text-gray-400">Auto-scroll</SmallText>
            </label>

            {/* Packet count */}
            <SmallText className="text-gray-400">
              {packets.length} / {MAX_PACKETS} packets
            </SmallText>
          </div>
        </CardContent>
      </Card>

      {/* Main content grid */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Packet List - Left sidebar */}
        <div className="lg:col-span-4 xl:col-span-3">
          <Card className="border-white/5 bg-gray-900/70 h-[600px]">
            <CardContent className="h-full flex flex-col">
              <div className="flex items-center justify-between mb-3">
                <SmallText className="text-gray-400 uppercase tracking-wide font-semibold">
                  Packet List
                </SmallText>
                {isPaused && (
                  <Tag colorScheme="yellow" className="text-xs">
                    Paused
                  </Tag>
                )}
              </div>
              <div className="flex-1 min-h-0">
                <PacketList
                  packets={packets}
                  selectedPacketId={selectedPacket?.id ?? null}
                  onSelectPacket={handleSelectPacket}
                  protocolFilter={protocolFilter}
                  searchQuery={searchQuery}
                  autoScroll={autoScroll}
                />
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Right panel - Hex dump and details */}
        <div className="lg:col-span-8 xl:col-span-9 space-y-6">
          {/* Hex Dump Viewer */}
          <Card className="border-white/5 bg-gray-900/70 h-[350px]">
            <CardContent className="h-full flex flex-col">
              <SmallText className="text-gray-400 uppercase tracking-wide font-semibold mb-3">
                Hex Dump
              </SmallText>
              <div className="flex-1 min-h-0">
                <HexDumpViewer
                  rawData={selectedPacket?.rawData ?? ''}
                  headerLength={14} // Ethernet header
                />
              </div>
            </CardContent>
          </Card>

          {/* Packet Details */}
          <Card className="border-white/5 bg-gray-900/70 h-[220px]">
            <CardContent className="h-full flex flex-col">
              <SmallText className="text-gray-400 uppercase tracking-wide font-semibold mb-3">
                Packet Details
              </SmallText>
              <div className="flex-1 min-h-0 overflow-y-auto">
                <PacketDetails packet={selectedPacket} />
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
};
