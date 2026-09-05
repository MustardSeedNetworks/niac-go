import {
  Download,
  FileBox,
  FileDown,
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
import { useNavigate, useSearchParams } from 'react-router';
import { fetchCaptureStatus, stopStandaloneCapture } from '../api/client';
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
import { useErrorToast } from '../hooks/useErrorToast';
import { isPacketStreamEvent, usePacketStream } from '../hooks/useEventSource';
import { useSimulationStatus } from '../hooks/useSimulationStatus';
import { Button } from '../ui/Button';
import { Card, CardContent } from '../ui/Card';
import { Inspector, InspectorPane, InspectorPanes, InspectorRecords } from '../ui/Inspector';
import { Tag } from '../ui/Tag';
import { H2, SmallText } from '../ui/Typography';
import { canFollowStream, getStreamFilter } from '../utils/conversations';
import { buildProtocolLayers, computeHeaderBoundary } from '../utils/protocol-layers';
import { PcapAnalyzerPage } from './PcapAnalyzerPage';
import { packetFromStreamEvent } from './packets/packet-from-stream';
import { StandaloneCaptureStarter } from './packets/StandaloneCaptureStarter';
import { useCaptureExport, useFilteredJsonExport } from './packets/useCaptureExport';

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
  const { t } = useTranslation('pages');
  if (connected) {
    return (
      <div className="flex items-center gap-compact">
        <Wifi className={`${iconSizes.md} text-status-success`} />
        <Tag colorScheme="green">{t('packets.inspector.connectedStatus')}</Tag>
      </div>
    );
  }

  // SSE auto-reconnects, so disconnected state is brief
  return (
    <div className="flex items-center gap-compact">
      <WifiOff className={`${iconSizes.md} text-status-warning animate-pulse`} />
      <Tag colorScheme="yellow">{t('packets.inspector.connectingStatus')}</Tag>
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
  const { t } = useTranslation('pages');
  const { t: tCommon } = useTranslation('common');
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const view = searchParams.get('view') === 'files' ? 'files' : 'live';
  // The page streams from /api/v1/stream/packets, which the daemon
  // populates from either a running simulation OR a standalone capture
  // session. Polling both lets us pick whichever is producing traffic
  // and show its interface in the header.
  const { data: simStatus } = useSimulationStatus();
  // A failed poll read as "not capturing" — so a running capture appeared to stop.
  const { data: captureStatus, refetch: refetchCapture } = useApiResource(fetchCaptureStatus, [], {
    intervalMs: POLL_INTERVALS.fast,
    errorToast: { title: t('packets.captureStatusFailed') },
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
  const showError = useErrorToast();

  // Coloring rules
  const {
    rules: coloringRules,
    setRules: setColoringRules,
    getRowStyle,
    resetRules: resetColoringRules,
  } = useColoringRules();

  // Apply display filter
  const { filteredPackets } = useDisplayFilter(packets, filterExpression);

  // Gated the same way the packet stream is: a session listed but no longer
  // running has no ring, and the export would 404 into a toast.
  const exportSessionId = simRunning ? simStatus?.sessionId : undefined;
  const handleExportPcap = useCaptureExport(exportSessionId);
  const handleExport = useFilteredJsonExport(filteredPackets);

  // Ref to track pause state in callback
  const isPausedRef = useRef(isPaused);
  useEffect(() => {
    isPausedRef.current = isPaused;
  }, [isPaused]);

  // SSE connection (auto-reconnects)
  const { connected, reconnect } = usePacketStream({
    sessionId: simRunning ? simStatus?.sessionId : undefined,
    onMessage: useCallback((data: unknown) => {
      // Skip if paused
      if (isPausedRef.current) {
        return;
      }

      if (!isPacketStreamEvent(data)) return;

      const incoming = data.data;

      const packet = packetFromStreamEvent(incoming, data.timestamp, generatePacketId());

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
  const followable = useMemo(() => canFollowStream(selectedPacket), [selectedPacket]);

  // Header/payload byte boundary for the hex dump, derived from the
  // selected packet's dissected protocol layers rather than a hardcoded
  // Ethernet-only constant, so IP/TCP/UDP header bytes are colored
  // correctly instead of bleeding into "Payload".
  const hexHeaderLength = useMemo(() => {
    if (!selectedPacket) {
      return undefined;
    }
    const layers = buildProtocolLayers(selectedPacket.headers, {
      timestamp: selectedPacket.timestamp,
      protocol: selectedPacket.protocol,
      sourceIp: selectedPacket.sourceIp,
      destIp: selectedPacket.destIp,
      sourcePort: selectedPacket.sourcePort,
      destPort: selectedPacket.destPort,
      size: selectedPacket.size,
    });
    return computeHeaderBoundary(layers);
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

  return (
    <div className="stack-xl">
      {/* Live / PCAP Files tab control */}
      <div
        className="inline-flex rounded-lg border border-surface-border bg-bg-base/40 p-0.5"
        role="tablist"
        aria-label={t('packets.inspector.viewTabsAriaLabel')}
      >
        <button
          type="button"
          role="tab"
          aria-selected={view === 'live'}
          onClick={() => setSearchParams({}, { replace: true })}
          className={`flex items-center gap-1.5 rounded px-3 py-compact-md text-xs font-medium transition-colors ${
            view === 'live'
              ? 'bg-brand-primary/20 text-brand-accent'
              : 'text-text-muted hover:bg-surface-hover hover:text-text-primary'
          }`}
        >
          <Radio className="w-3.5 h-3.5" />
          {t('packets.inspector.liveCaptureTab')}
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={view === 'files'}
          onClick={() => setSearchParams({ view: 'files' }, { replace: true })}
          className={`flex items-center gap-1.5 rounded px-3 py-compact-md text-xs font-medium transition-colors ${
            view === 'files'
              ? 'bg-brand-primary/20 text-brand-accent'
              : 'text-text-muted hover:bg-surface-hover hover:text-text-primary'
          }`}
        >
          <FileBox className="w-3.5 h-3.5" />
          {t('packets.inspector.pcapFilesTab')}
        </button>
      </div>

      {/* PCAP Files tab — renders the standalone analyzer */}
      {view === 'files' && <PcapAnalyzerPage />}

      {view === 'live' && !streamActive && (
        <StandaloneCaptureStarter
          onStarted={() => {
            refetchCapture();
          }}
          navigateToSim={() => navigate('/runtime')}
        />
      )}

      {/* Live capture content (everything below is the original page) */}
      {view === 'live' && streamActive && (
        <>
          {/* Header with controls */}
          <Card className="border-surface-border bg-bg-surface/70">
            <CardContent>
              <div className="flex flex-col gap-comfortable sm:flex-row sm:items-center sm:justify-between">
                {/* Title and connection status */}
                <div className="flex items-center gap-comfortable">
                  <H2>{t('packets.title')}</H2>
                  <SmallText className="text-text-muted">
                    {t('packets.inspector.onInterfacePrefix')}{' '}
                    <span className="font-mono text-text-primary">{activeInterface ?? '—'}</span>
                  </SmallText>
                  {captureRunning && !simRunning && (
                    <Tag colorScheme="violet">{t('packets.inspector.standaloneTag')}</Tag>
                  )}
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
                    {isPaused ? tCommon('buttons.resume') : tCommon('buttons.pause')}
                  </Button>

                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={handleClear}
                    leftIcon={<Trash2 className={iconSizes.md} />}
                    disabled={packets.length === 0}
                  >
                    {tCommon('buttons.clear')}
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
                        } catch (err) {
                          showError(err);
                        }
                      }}
                    >
                      {t('packets.inspector.stopCaptureButton')}
                    </Button>
                  )}

                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={handleExport}
                    leftIcon={<Download className={iconSizes.md} />}
                    disabled={filteredPackets.length === 0}
                    title={t('packets.inspector.exportButtonTitle')}
                  >
                    {t('packets.inspector.exportButton')}
                  </Button>

                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={handleExportPcap}
                    leftIcon={<FileDown className={iconSizes.md} />}
                    disabled={!exportSessionId}
                    title={t('packets.inspector.exportPcapButtonTitle')}
                  >
                    {t('packets.inspector.exportPcapButton')}
                  </Button>

                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setShowColoringRules(true)}
                    leftIcon={<Palette className={iconSizes.md} />}
                  >
                    {t('packets.inspector.colorsButton')}
                  </Button>

                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={handleFollowStream}
                    leftIcon={<Share2 className={iconSizes.md} />}
                    disabled={!followable}
                    title={
                      followable ? undefined : t('packets.inspector.followStreamDisabledTitle')
                    }
                  >
                    {t('packets.inspector.followStreamTitle')}
                  </Button>

                  {/* SSE auto-reconnects, but manual reconnect is available */}
                  {!connected && (
                    <Button tone="violet" size="sm" onClick={reconnect}>
                      {t('packets.inspector.reconnectButton')}
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
                  <SmallText className="text-text-muted">
                    {t('packets.inspector.autoScrollLabel')}
                  </SmallText>
                </label>

                {/* Packet count */}
                <SmallText className="text-text-muted">
                  {t('packets.inspector.packetsCountLabel', {
                    shown: filteredPackets.length,
                    total: packets.length,
                  })}
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

          <Inspector>
            <InspectorRecords>
              <Card className="border-surface-border bg-bg-surface/70 h-full">
                <CardContent className="h-full flex flex-col">
                  <div className="flex-between mb-heading">
                    <SmallText className="text-text-muted uppercase tracking-wide font-semibold">
                      {t('packets.inspector.packetListLabel')}
                    </SmallText>
                    {isPaused && (
                      <Tag colorScheme="yellow" className="text-xs">
                        {t('packets.inspector.pausedTag')}
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
            </InspectorRecords>

            <InspectorPanes>
              <InspectorPane label={t('packets.inspector.hexDumpLabel')}>
                <HexDumpViewer
                  rawData={selectedPacket?.rawData ?? ''}
                  headerLength={hexHeaderLength}
                  highlightRange={highlightRange}
                />
              </InspectorPane>

              <InspectorPane label={t('packets.inspector.packetDetailsLabel')} scroll>
                <PacketDetails
                  packet={selectedPacket}
                  onFieldSelect={(start, end) => setHighlightRange([start, end])}
                />
              </InspectorPane>
            </InspectorPanes>
          </Inspector>

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

// Keep the import live even if the type is only used implicitly via
// the API client return types.
export type { StandaloneCaptureStatus };
