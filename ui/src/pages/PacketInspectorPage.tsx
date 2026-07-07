import {
  Activity,
  Download,
  FileBox,
  Palette,
  Pause,
  Play,
  Radio,
  Share2,
  Trash2,
  Wifi,
  WifiOff,
} from 'lucide-react';
import { type FC, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import {
  fetchCaptureStatus,
  fetchUsableInterfaces,
  startStandaloneCapture,
  stopStandaloneCapture,
} from '../api/client';
import type { StandaloneCaptureStatus } from '../api/types';
import { BpfFilterBar } from '../components/BpfFilterBar';
import { ColoringRulesPanel } from '../components/ColoringRulesPanel';
import { FilterBar } from '../components/FilterBar';
import { HexDumpViewer } from '../components/HexDumpViewer';
import { PacketDetails } from '../components/PacketDetails';
import { type Packet, PacketList } from '../components/PacketList';
import { StreamView } from '../components/StreamView';
import { POLL_INTERVALS } from '../constants/polling';
import { iconSizes } from '../constants/sizes';
import { useApiResource } from '../hooks/useApiResource';
import { useColoringRules } from '../hooks/useColoringRules';
import { useDisplayFilter } from '../hooks/useDisplayFilter';
import { usePacketStream } from '../hooks/useEventSource';
import { useSimulationStatus } from '../hooks/useSimulationStatus';
import { Button } from '../ui/Button';
import { Card, CardContent } from '../ui/Card';
import { Tag } from '../ui/Tag';
import { H2, SmallText } from '../ui/Typography';
import { getStreamFilter } from '../utils/conversations';
import { PcapAnalyzerPage } from './PcapAnalyzerPage';

/** Maximum number of packets to buffer */
const MAX_PACKETS = 100;

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
      <div className="flex items-center gap-compact">
        <Wifi className={`${iconSizes.md} text-status-success`} />
        <Tag colorScheme="green">Connected</Tag>
      </div>
    );
  }

  // SSE auto-reconnects, so disconnected state is brief
  return (
    <div className="flex items-center gap-compact">
      <WifiOff className={`${iconSizes.md} text-status-warning animate-pulse`} />
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
  const navigate = useNavigate();
  const [view, setView] = useState<'live' | 'files'>('live');
  // The page streams from /api/v1/stream/packets, which the daemon
  // populates from either a running simulation OR a standalone capture
  // session. Polling both lets us pick whichever is producing traffic
  // and show its interface in the header.
  const { data: simStatus } = useSimulationStatus();
  const { data: captureStatus, refetch: refetchCapture } = useApiResource(fetchCaptureStatus, [], {
    intervalMs: POLL_INTERVALS.fast,
  });
  const simRunning = simStatus?.running === true;
  const captureRunning = captureStatus?.running === true;
  const streamActive = simRunning || captureRunning;
  const activeInterface = simRunning
    ? simStatus?.interface
    : captureRunning
      ? captureStatus?.interface
      : undefined;

  // Packet buffer state
  const [packets, setPackets] = useState<Packet[]>([]);
  const [selectedPacket, setSelectedPacket] = useState<Packet | null>(null);
  const [highlightRange, setHighlightRange] = useState<[number, number] | undefined>(undefined);

  // Filter state - single expression-based filter
  const [filterExpression, setFilterExpression] = useState('');

  // UI state
  const [autoScroll, setAutoScroll] = useState(true);
  const [isPaused, setIsPaused] = useState(false);
  const [showColoringRules, setShowColoringRules] = useState(false);
  const [showStreamView, setShowStreamView] = useState(false);

  // Coloring rules
  const {
    rules: coloringRules,
    setRules: setColoringRules,
    getRowStyle,
    resetRules: resetColoringRules,
  } = useColoringRules();

  // Apply display filter
  const { filteredPackets } = useDisplayFilter(packets, filterExpression);

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

      // Validate incoming data is an object before casting
      if (typeof data !== 'object' || data === null) {
        return;
      }

      // Transform incoming data to Packet format
      const incoming = data as Record<string, unknown>;

      // SSE payloads bypass the shared toCamelCase converter in client.ts,
      // so fall back to snake_case keys when the camelCase key is missing.
      const packet: Packet = {
        id: generatePacketId(),
        timestamp: (incoming.timestamp as string) || new Date().toISOString(),
        protocol: (incoming.protocol as string) || 'Unknown',
        sourceIp: (incoming.sourceIp as string) || (incoming.source_ip as string) || 'Unknown',
        destIp: (incoming.destIp as string) || (incoming.dest_ip as string) || 'Unknown',
        sourcePort:
          (incoming.sourcePort as number | undefined) ??
          (incoming.source_port as number | undefined),
        destPort:
          (incoming.destPort as number | undefined) ?? (incoming.dest_port as number | undefined),
        size: (incoming.size as number) || 0,
        summary: (incoming.summary as string) || '',
        rawData:
          (incoming.rawData as string) ||
          (incoming.raw_data as string) ||
          (incoming.payload as string) ||
          '',
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
    setHighlightRange(undefined);
  }, []);

  // Follow Stream - filter to conversation and show stream view
  const handleFollowStream = useCallback(() => {
    if (!selectedPacket) return;
    const filter = getStreamFilter(selectedPacket);
    if (filter) {
      setFilterExpression(filter);
      setShowStreamView(true);
    }
  }, [selectedPacket]);

  // Can the selected packet be followed as a stream?
  const canFollowStream = useMemo(() => {
    if (!selectedPacket) return false;
    const proto = selectedPacket.protocol.toUpperCase();
    return (
      (proto === 'TCP' || proto === 'UDP') &&
      !!selectedPacket.sourcePort &&
      !!selectedPacket.destPort
    );
  }, [selectedPacket]);

  // Stream packets for StreamView (all packets in current conversation)
  const streamPackets = useMemo(() => {
    if (!selectedPacket || !showStreamView) return [];
    return filteredPackets;
  }, [selectedPacket, showStreamView, filteredPackets]);

  // Client endpoint for StreamView
  const clientEndpoint = useMemo(() => {
    if (!selectedPacket) return '';
    return `${selectedPacket.sourceIp}:${selectedPacket.sourcePort ?? 0}`;
  }, [selectedPacket]);

  if (!streamActive) {
    return (
      <StandaloneCaptureStarter
        onStarted={() => {
          // Force an immediate status refresh so the page transitions
          // out of the empty state without waiting for the poll tick.
          refetchCapture();
        }}
        navigateToSim={() => navigate('/runtime')}
      />
    );
  }

  return (
    <div className="stack-xl">
      {/* Live / PCAP Files tab control */}
      <div
        className="inline-flex rounded-lg border border-surface-border bg-bg-base/40 p-0.5"
        role="tablist"
        aria-label="Packets view"
      >
        <button
          type="button"
          role="tab"
          aria-selected={view === 'live'}
          onClick={() => setView('live')}
          className={`flex items-center gap-1.5 rounded px-3 py-compact-md text-xs font-medium transition-colors ${
            view === 'live'
              ? 'bg-brand-primary/20 text-brand-accent'
              : 'text-text-muted hover:bg-surface-hover hover:text-text-primary'
          }`}
        >
          <Radio className="w-3.5 h-3.5" />
          Live capture
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={view === 'files'}
          onClick={() => setView('files')}
          className={`flex items-center gap-1.5 rounded px-3 py-compact-md text-xs font-medium transition-colors ${
            view === 'files'
              ? 'bg-brand-primary/20 text-brand-accent'
              : 'text-text-muted hover:bg-surface-hover hover:text-text-primary'
          }`}
        >
          <FileBox className="w-3.5 h-3.5" />
          PCAP files
        </button>
      </div>

      {/* PCAP Files tab — renders the standalone analyzer */}
      {view === 'files' && <PcapAnalyzerPage />}

      {/* Live capture content (everything below is the original page) */}
      {view === 'live' && (
        <>
          {/* Header with controls */}
          <Card className="border-surface-border bg-bg-surface/70">
            <CardContent>
              <div className="flex flex-col gap-comfortable sm:flex-row sm:items-center sm:justify-between">
                {/* Title and connection status */}
                <div className="flex items-center gap-comfortable">
                  <H2>Packets</H2>
                  <SmallText className="text-text-muted">
                    on <span className="font-mono text-text-primary">{activeInterface ?? '—'}</span>
                  </SmallText>
                  {captureRunning && !simRunning && <Tag colorScheme="violet">Standalone</Tag>}
                  <ConnectionStatus connected={connected} />
                </div>

                {/* Control buttons */}
                <div className="flex flex-wrap items-center gap-compact">
                  <Button
                    variant={isPaused ? 'outline' : 'ghost'}
                    size="sm"
                    onClick={handlePauseToggle}
                    leftIcon={
                      isPaused ? (
                        <Play className={iconSizes.md} />
                      ) : (
                        <Pause className={iconSizes.md} />
                      )
                    }
                  >
                    {isPaused ? 'Resume' : 'Pause'}
                  </Button>

                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={handleClear}
                    leftIcon={<Trash2 className={iconSizes.md} />}
                    disabled={packets.length === 0}
                  >
                    Clear
                  </Button>

                  {captureRunning && !simRunning && (
                    <Button
                      variant="outline"
                      tone="red"
                      size="sm"
                      onClick={async () => {
                        try {
                          await stopStandaloneCapture();
                          refetchCapture();
                        } catch {
                          // Surface via the empty-state error path on next render.
                        }
                      }}
                    >
                      Stop capture
                    </Button>
                  )}

                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={handleExport}
                    leftIcon={<Download className={iconSizes.md} />}
                    disabled={packets.length === 0}
                  >
                    Export
                  </Button>

                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setShowColoringRules(true)}
                    leftIcon={<Palette className={iconSizes.md} />}
                  >
                    Colors
                  </Button>

                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={handleFollowStream}
                    leftIcon={<Share2 className={iconSizes.md} />}
                    disabled={!canFollowStream}
                  >
                    Follow Stream
                  </Button>

                  {/* SSE auto-reconnects, but manual reconnect is available */}
                  {!connected && (
                    <Button tone="violet" size="sm" onClick={reconnect}>
                      Reconnect
                    </Button>
                  )}
                </div>
              </div>

              {/* Filter row */}
              <div className="mt-content flex flex-col gap-default sm:flex-row sm:items-center">
                <div className="flex-1">
                  <FilterBar value={filterExpression} onChange={setFilterExpression} />
                </div>

                {/* Auto-scroll toggle */}
                <label className="flex items-center gap-compact cursor-pointer">
                  <input
                    type="checkbox"
                    checked={autoScroll}
                    onChange={(e) => setAutoScroll(e.target.checked)}
                    className="rounded border-border-muted bg-bg-elevated text-brand-primary focus:ring-brand-accent focus:ring-offset-surface-base"
                  />
                  <SmallText className="text-text-muted">Auto-scroll</SmallText>
                </label>

                {/* Packet count */}
                <SmallText className="text-text-muted">
                  {filteredPackets.length} / {packets.length} packets
                </SmallText>
              </div>
            </CardContent>
          </Card>

          {/* BPF Capture Filter */}
          <Card className="border-surface-border bg-bg-surface/70">
            <CardContent>
              <BpfFilterBar />
            </CardContent>
          </Card>

          {/* Main content grid */}
          <div className="grid grid-cols-1 lg:grid-cols-12 gap-spacious">
            {/* Packet List - Left sidebar */}
            <div className="lg:col-span-4 xl:col-span-3">
              <Card className="border-surface-border bg-bg-surface/70 h-[600px]">
                <CardContent className="h-full flex flex-col">
                  <div className="flex-between mb-heading">
                    <SmallText className="text-text-muted uppercase tracking-wide font-semibold">
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
                      packets={filteredPackets}
                      selectedPacketId={selectedPacket?.id ?? null}
                      onSelectPacket={handleSelectPacket}
                      autoScroll={autoScroll}
                      getRowStyle={getRowStyle}
                    />
                  </div>
                </CardContent>
              </Card>
            </div>

            {/* Right panel - Hex dump and details */}
            <div className="lg:col-span-8 xl:col-span-9 stack-xl">
              {/* Hex Dump Viewer */}
              <Card className="border-surface-border bg-bg-surface/70 h-[350px]">
                <CardContent className="h-full flex flex-col">
                  <SmallText className="text-text-muted uppercase tracking-wide font-semibold mb-heading">
                    Hex Dump
                  </SmallText>
                  <div className="flex-1 min-h-0">
                    <HexDumpViewer
                      rawData={selectedPacket?.rawData ?? ''}
                      headerLength={14}
                      highlightRange={highlightRange}
                    />
                  </div>
                </CardContent>
              </Card>

              {/* Packet Details */}
              <Card className="border-surface-border bg-bg-surface/70 h-[220px]">
                <CardContent className="h-full flex flex-col">
                  <SmallText className="text-text-muted uppercase tracking-wide font-semibold mb-heading">
                    Packet Details
                  </SmallText>
                  <div className="flex-1 min-h-0 overflow-y-auto">
                    <PacketDetails
                      packet={selectedPacket}
                      onFieldSelect={(start, end) => setHighlightRange([start, end])}
                    />
                  </div>
                </CardContent>
              </Card>
            </div>
          </div>

          {/* Coloring Rules Panel */}
          {showColoringRules && (
            <ColoringRulesPanel
              rules={coloringRules}
              onRulesChange={setColoringRules}
              onReset={resetColoringRules}
              onClose={() => setShowColoringRules(false)}
            />
          )}

          {/* Stream View */}
          {showStreamView && selectedPacket && (
            <StreamView
              packets={streamPackets}
              clientEndpoint={clientEndpoint}
              onClose={() => setShowStreamView(false)}
            />
          )}
        </>
      )}
    </div>
  );
};

/**
 * StandaloneCaptureStarter is the empty-state UI for PCAP Inspector
 * when neither a simulation nor a standalone capture session is
 * running. It lets the user start a sniff-only capture on an
 * interface of their choice, with an optional BPF filter.
 *
 * Once StartStandaloneCapture succeeds, the parent's polling picks up
 * captureStatus.running and re-renders into the regular packet stream
 * UI on the next tick. Calling onStarted forces an immediate refetch
 * so the transition doesn't wait on the poll interval.
 */
const StandaloneCaptureStarter: FC<{
  onStarted: () => void;
  navigateToSim: () => void;
}> = ({ onStarted, navigateToSim }) => {
  const { t } = useTranslation('pages');
  const { data: interfacesResp } = useApiResource(fetchUsableInterfaces, []);
  const interfaces = interfacesResp?.interfaces ?? [];
  const [selectedIface, setSelectedIface] = useState('');
  const [bpfFilter, setBpfFilter] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Default to the first usable interface once the list arrives.
  useEffect(() => {
    if (!selectedIface && interfaces.length > 0) {
      setSelectedIface(interfaces[0]?.name ?? '');
    }
  }, [interfaces, selectedIface]);

  const handleStart = useCallback(async () => {
    if (!selectedIface || busy) {
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await startStandaloneCapture({
        interface: selectedIface,
        filter: bpfFilter.trim() || undefined,
      });
      onStarted();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start capture');
      setBusy(false);
    }
  }, [bpfFilter, busy, onStarted, selectedIface]);

  return (
    <Card className="border-surface-border bg-bg-surface/70">
      <CardContent className="stack-lg">
        <div className="flex items-start gap-default">
          <Activity className={`mt-tight ${iconSizes.lg} text-brand-accent`} />
          <div className="flex-1">
            <p className="font-semibold text-text-primary">
              {t('packets.inspector.standaloneCapture')}
            </p>
            <SmallText className="text-text-muted">
              Sniff an interface without starting a simulation. Frames stream into the viewer below
              as soon as they hit the wire. Or{' '}
              <button
                type="button"
                onClick={navigateToSim}
                className="text-brand-accent underline hover:text-brand-accent"
              >
                start a simulation
              </button>{' '}
              if you want NIAC to also respond on the wire.
            </SmallText>
          </div>
        </div>
        <div className="grid gap-default sm:grid-cols-[1fr_2fr_auto] sm:items-end">
          <label className="flex flex-col gap-tight text-sm text-text-muted">
            Interface
            <select
              value={selectedIface}
              onChange={(e) => setSelectedIface(e.target.value)}
              disabled={busy || interfaces.length === 0}
              className="rounded-lg border border-surface-border bg-bg-base/60 px-3 py-row text-sm text-text-primary focus:border-brand-accent focus:outline-none"
            >
              {interfaces.length === 0 ? (
                <option value="">{t('packets.inspector.noInterfaces')}</option>
              ) : (
                interfaces.map((iface) => (
                  <option key={iface.name} value={iface.name}>
                    {iface.name}
                    {iface.addresses && iface.addresses.length > 0
                      ? ` (${iface.addresses[0]})`
                      : ''}
                  </option>
                ))
              )}
            </select>
          </label>
          <label className="flex flex-col gap-tight text-sm text-text-muted">
            BPF filter (optional)
            <input
              type="text"
              value={bpfFilter}
              onChange={(e) => setBpfFilter(e.target.value)}
              disabled={busy}
              placeholder="e.g. tcp port 80"
              className="rounded-lg border border-surface-border bg-bg-base/60 px-3 py-row font-mono text-sm text-text-primary focus:border-brand-accent focus:outline-none"
            />
          </label>
          <Button tone="violet" onClick={handleStart} disabled={!selectedIface || busy}>
            {busy ? 'Starting…' : 'Start capture'}
          </Button>
        </div>
        {error && (
          <SmallText className="text-status-error" role="alert">
            {error}
          </SmallText>
        )}
      </CardContent>
    </Card>
  );
};

// Keep the import live even if the type is only used implicitly via
// the API client return types.
export type { StandaloneCaptureStatus };
