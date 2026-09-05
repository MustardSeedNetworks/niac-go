import { Download, FileSearch, Info, Palette, Share2, Trash2 } from 'lucide-react';
import { type FC, useCallback, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { fetchPcapAnalysis, uploadPcapWithProgress } from '../api/client';
import { isApiError } from '../api/errors';
import type { PcapAnalysisResult, PcapPacket } from '../api/types';
import { ColoringRulesPanel } from '../components/ColoringRulesPanel';
import { ConversationList } from '../components/ConversationList';
import { FilterBar } from '../components/FilterBar';
import { HexDumpViewer } from '../components/HexDumpViewer';
import { PacketDetails } from '../components/PacketDetails';
import type { Packet } from '../components/PacketList';
import { PcapPacketList } from '../components/pcap/PcapPacketList';
import { PcapStats } from '../components/pcap/PcapStats';
import { PcapUploader } from '../components/pcap/PcapUploader';
import { StreamView } from '../components/StreamView';
import { iconSizes } from '../constants/sizes';
import { useColoringRules } from '../hooks/useColoringRules';
import { useDisplayFilter } from '../hooks/useDisplayFilter';
import { useErrorToast } from '../hooks/useErrorToast';
import { Button } from '../ui/Button';
import { Card, CardContent } from '../ui/Card';
import { InfoPopover } from '../ui/InfoPopover';
import { Inspector, InspectorPane, InspectorPanes, InspectorRecords } from '../ui/Inspector';
import { Tag } from '../ui/Tag';
import { H2, SmallText } from '../ui/Typography';
import { canFollowStream, getStreamFilter } from '../utils/conversations';
import { fileToBase64 } from '../utils/file';

/**
 * Convert PcapPacket to Packet for PacketDetails component
 */
function pcapPacketToPacket(pcapPacket: PcapPacket): Packet {
  return {
    id: pcapPacket.id,
    timestamp: pcapPacket.timestamp,
    protocol: pcapPacket.protocol,
    sourceIp: pcapPacket.sourceIp,
    destIp: pcapPacket.destIp,
    sourcePort: pcapPacket.sourcePort,
    destPort: pcapPacket.destPort,
    size: pcapPacket.length,
    summary: pcapPacket.info,
    rawData: pcapPacket.rawData || '',
    headers: pcapPacket.headers,
  };
}

/**
 * PCAP Analyzer Page
 *
 * Full-featured PCAP file analyzer with:
 * - Drag-and-drop file upload
 * - Packet list with filtering
 * - Packet details and hex dump viewer
 * - Summary statistics and protocol breakdown
 */
export const PcapAnalyzerPage: FC = () => {
  const { t } = useTranslation('common');
  const { t: tHelp } = useTranslation('help');
  const { t: tPages } = useTranslation('pages');

  // File and analysis state
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [analysisResult, setAnalysisResult] = useState<PcapAnalysisResult | null>(null);
  const [isAnalyzing, setIsAnalyzing] = useState(false);
  // Field-scoped: the uploader's own rejected-file reason (bad type/size).
  // Stays inline next to the uploader, unlike the toast-surfaced analyze
  // failure below.
  const [validationError, setValidationError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [uploadProgress, setUploadProgress] = useState<number | null>(null);
  const showError = useErrorToast();

  // Selected packet state
  const [selectedPacket, setSelectedPacket] = useState<PcapPacket | null>(null);
  const [highlightRange, setHighlightRange] = useState<[number, number] | undefined>(undefined);

  // Filter state - single expression replaces 4 separate filters
  const [filterExpression, setFilterExpression] = useState('');

  // View mode state
  const [viewMode, setViewMode] = useState<'packets' | 'stats' | 'conversations'>('packets');
  const [showColoringRules, setShowColoringRules] = useState(false);
  const [showStreamView, setShowStreamView] = useState(false);

  // Coloring rules
  const {
    rules: coloringRules,
    setRules: setColoringRules,
    getRowStyle,
    resetRules: resetColoringRules,
  } = useColoringRules();

  // Apply display filter to packets
  const { filteredPackets } = useDisplayFilter(analysisResult?.packets ?? [], filterExpression);

  // Handle file selection
  const handleFileSelect = useCallback((file: File | null) => {
    setSelectedFile(file);
    setAnalysisResult(null);
    setSelectedPacket(null);
    setValidationError(null);
    setSuccess(null);
    setFilterExpression('');
  }, []);

  // Handle rejected file (invalid type/size) — surface the reason instead of failing silently
  const handleValidationError = useCallback((message: string) => {
    setSelectedFile(null);
    setSuccess(null);
    setValidationError(message);
  }, []);

  // Handle analysis
  const handleAnalyze = useCallback(async () => {
    if (!selectedFile) {
      return;
    }

    setIsAnalyzing(true);
    setValidationError(null);
    setSuccess(null);
    setUploadProgress(0);

    try {
      // Convert file to base64 and upload to backend
      const base64Data = await fileToBase64(selectedFile);
      const uploadResponse = await uploadPcapWithProgress(
        { filename: selectedFile.name, data: base64Data },
        setUploadProgress,
      );

      if (!uploadResponse.success) {
        throw new Error(uploadResponse.message || tPages('libraryPcaps.analyzer.uploadFailed'));
      }

      // Fetch the analysis result
      const result = await fetchPcapAnalysis(uploadResponse.analysisId);

      setAnalysisResult(result);
      setSuccess(tPages('libraryPcaps.analyzer.analyzeSuccess', { count: result.packets.length }));

      // Auto-select first packet
      const firstPacket = result.packets[0];
      if (firstPacket) {
        setSelectedPacket(firstPacket);
      }
    } catch (err) {
      // The server enforces its own upload size cap (independent of the
      // client-side MAX_FILE_SIZE check) — surface that specific failure
      // as a friendly, actionable inline message instead of a generic
      // toast, since the fix is "pick a smaller file."
      if (isApiError(err) && err.code === 'request_too_large') {
        const limit = err.details.find((detail) => detail.field === 'body')?.value ?? '100MB';
        setValidationError(tPages('libraryPcaps.uploader.serverFileTooLarge', { size: limit }));
      } else {
        showError(err);
      }
    } finally {
      setIsAnalyzing(false);
      setUploadProgress(null);
    }
  }, [selectedFile, showError, tPages]);

  // Handle packet selection
  const handleSelectPacket = useCallback((packet: PcapPacket) => {
    setSelectedPacket(packet);
    setHighlightRange(undefined);
  }, []);

  // Handle clear
  const handleClear = useCallback(() => {
    setSelectedFile(null);
    setAnalysisResult(null);
    setSelectedPacket(null);
    setValidationError(null);
    setSuccess(null);
    setFilterExpression('');
  }, []);

  // Export packets as JSON
  const handleExport = useCallback(() => {
    if (!analysisResult) {
      return;
    }

    const exportData = JSON.stringify(analysisResult, null, 2);
    const blob = new Blob([exportData], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `pcap-analysis-${new Date().toISOString().slice(0, 19).replace(/[:.]/g, '-')}.json`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  }, [analysisResult]);

  // Handle conversation selection - switch to packets view with filter
  const handleSelectConversation = useCallback((filter: string) => {
    setFilterExpression(filter);
    setViewMode('packets');
  }, []);

  // Follow Stream
  const handleFollowStream = useCallback(() => {
    if (!selectedPacket) return;
    const filter = getStreamFilter(selectedPacket);
    if (filter) {
      setFilterExpression(filter);
      setShowStreamView(true);
    }
  }, [selectedPacket]);

  const followable = useMemo(() => canFollowStream(selectedPacket), [selectedPacket]);

  const streamPackets = useMemo(() => {
    if (!selectedPacket || !showStreamView) return [];
    return filteredPackets;
  }, [selectedPacket, showStreamView, filteredPackets]);

  const clientEndpoint = useMemo(() => {
    if (!selectedPacket) return '';
    return `${selectedPacket.sourceIp}:${selectedPacket.sourcePort ?? 0}`;
  }, [selectedPacket]);

  // Convert selected packet for PacketDetails component
  const selectedPacketForDetails = useMemo(
    () => (selectedPacket ? pcapPacketToPacket(selectedPacket) : null),
    [selectedPacket],
  );

  return (
    <div className="stack-xl">
      {/* Upload Section */}
      {!analysisResult && (
        <PcapUploader
          onFileSelect={handleFileSelect}
          onAnalyze={handleAnalyze}
          onValidationError={handleValidationError}
          isAnalyzing={isAnalyzing}
          selectedFile={selectedFile}
          error={validationError}
          success={success}
          uploadProgress={uploadProgress}
        />
      )}

      {/* Results Section */}
      {analysisResult && (
        <>
          {/* Controls Header */}
          <Card className="border-surface-border bg-bg-surface/70">
            <CardContent>
              <div className="flex flex-col gap-comfortable sm:flex-row sm:items-center sm:justify-between">
                {/* Title and file info */}
                <div className="flex items-center gap-comfortable">
                  <H2 className="flex items-center gap-compact">
                    <FileSearch className={`${iconSizes.lg} text-brand-accent`} />
                    {tPages('libraryPcaps.analyzer.pageHeading')}
                    <InfoPopover label={t('jargon.ariaLabel', { term: 'PCAP' })} title="PCAP">
                      {tHelp('jargon.pcap')}
                    </InfoPopover>
                  </H2>
                  <Tag colorScheme="green">
                    {tPages('libraryPcaps.analyzer.packetsCountTag', {
                      count: analysisResult.packets.length,
                    })}
                  </Tag>
                  {analysisResult.truncated && (
                    <Tag colorScheme="yellow">
                      {tPages('libraryPcaps.analyzer.truncatedNotice', {
                        shown: analysisResult.packets.length,
                      })}
                    </Tag>
                  )}
                </div>

                {/* Control buttons */}
                <div className="flex flex-wrap items-center gap-compact">
                  {/* View Mode Toggle */}
                  <div className="flex rounded-lg border border-surface-border bg-bg-base/50 p-1">
                    <button
                      type="button"
                      onClick={() => setViewMode('packets')}
                      title={tPages('libraryPcaps.analyzer.tabPacketsTitle')}
                      className={`px-3 py-compact-md text-sm rounded-md transition-colors ${
                        viewMode === 'packets'
                          ? 'bg-brand-primary text-text-primary'
                          : 'text-text-muted hover:text-text-primary'
                      }`}
                    >
                      {tPages('libraryPcaps.analyzer.tabPackets')}
                    </button>
                    <button
                      type="button"
                      onClick={() => setViewMode('stats')}
                      title={tPages('libraryPcaps.analyzer.tabStatsTitle')}
                      className={`px-3 py-compact-md text-sm rounded-md transition-colors ${
                        viewMode === 'stats'
                          ? 'bg-brand-primary text-text-primary'
                          : 'text-text-muted hover:text-text-primary'
                      }`}
                    >
                      {tPages('libraryPcaps.analyzer.tabStats')}
                    </button>
                    <button
                      type="button"
                      onClick={() => setViewMode('conversations')}
                      title={tPages('libraryPcaps.analyzer.tabConversationsTitle')}
                      className={`px-3 py-compact-md text-sm rounded-md transition-colors ${
                        viewMode === 'conversations'
                          ? 'bg-brand-primary text-text-primary'
                          : 'text-text-muted hover:text-text-primary'
                      }`}
                    >
                      {tPages('libraryPcaps.analyzer.tabConversations')}
                    </button>
                  </div>

                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={handleExport}
                    leftIcon={<Download className={iconSizes.md} />}
                  >
                    {t('buttons.export')}
                  </Button>

                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setShowColoringRules(true)}
                    leftIcon={<Palette className={iconSizes.md} />}
                  >
                    {tPages('libraryPcaps.analyzer.colorsButton')}
                  </Button>

                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={handleFollowStream}
                    leftIcon={<Share2 className={iconSizes.md} />}
                    disabled={!followable}
                  >
                    {tPages('libraryPcaps.analyzer.followStreamButton')}
                  </Button>

                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={handleClear}
                    leftIcon={<Trash2 className={iconSizes.md} />}
                  >
                    {t('buttons.clear')}
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Packets View */}
          {viewMode === 'packets' && (
            <Inspector
              filter={<FilterBar value={filterExpression} onChange={setFilterExpression} />}
            >
              <InspectorRecords>
                <PcapPacketList
                  packets={filteredPackets}
                  totalPackets={analysisResult.packets.length}
                  selectedPacketId={selectedPacket?.id ?? null}
                  onSelectPacket={handleSelectPacket}
                  getRowStyle={getRowStyle}
                />
              </InspectorRecords>

              <InspectorPanes>
                <InspectorPane label={tPages('libraryPcaps.analyzer.hexDumpLabel')}>
                  <HexDumpViewer
                    rawData={selectedPacket?.rawData ?? ''}
                    headerLength={14}
                    highlightRange={highlightRange}
                  />
                </InspectorPane>

                <InspectorPane label={tPages('libraryPcaps.analyzer.packetDetailsLabel')} scroll>
                  <PacketDetails
                    packet={selectedPacketForDetails}
                    onFieldSelect={(start, end) => setHighlightRange([start, end])}
                  />
                </InspectorPane>
              </InspectorPanes>
            </Inspector>
          )}

          {/* Statistics View */}
          {viewMode === 'stats' && (
            <PcapStats
              stats={analysisResult.stats}
              filename={analysisResult.filename}
              fileSize={analysisResult.fileSize}
            />
          )}

          {/* Conversations View */}
          {viewMode === 'conversations' && (
            <ConversationList
              packets={analysisResult.packets}
              onSelectConversation={handleSelectConversation}
            />
          )}
        </>
      )}

      {/* Info Banner when no file is loaded */}
      {!(analysisResult || selectedFile) && (
        <Card className="border-surface-border bg-bg-surface/70">
          <CardContent>
            <div className="flex items-start gap-default">
              <Info className={`${iconSizes.lg} text-status-info flex-shrink-0 mt-0.5`} />
              <div>
                <p className="font-medium text-text-primary">
                  {tPages('libraryPcaps.analyzer.aboutTitle')}
                </p>
                <SmallText className="text-text-muted">
                  {tPages('libraryPcaps.analyzer.aboutDescription')}
                </SmallText>
                <div className="mt-heading flex flex-wrap gap-compact">
                  <Tag colorScheme="gray">{tPages('libraryPcaps.analyzer.supportsPcap')}</Tag>
                  <Tag colorScheme="gray">{tPages('libraryPcaps.analyzer.supportsPcapng')}</Tag>
                  <Tag colorScheme="gray">{tPages('libraryPcaps.analyzer.maxSizeTag')}</Tag>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

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
    </div>
  );
};

export default PcapAnalyzerPage;
