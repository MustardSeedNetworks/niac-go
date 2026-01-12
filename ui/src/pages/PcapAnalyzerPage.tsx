import { type FC, useState, useCallback, useMemo } from 'react';
import {
  Card,
  CardContent,
  Button,
  Tag,
  SmallText,
  H2,
} from '../ui';
import { FileSearch, Download, Trash2, Info } from 'lucide-react';
import { PcapUploader } from '../components/pcap/PcapUploader';
import { PcapPacketList } from '../components/pcap/PcapPacketList';
import { PcapStats } from '../components/pcap/PcapStats';
import { HexDumpViewer } from '../components/HexDumpViewer';
import { PacketDetails } from '../components/PacketDetails';
import { uploadPcap, fetchPcapAnalysis } from '../api/client';
import type { PcapPacket, PcapAnalysisResult } from '../api/types';
import type { Packet } from '../components/PacketList';
import { getErrorMessage, fileToBase64 } from '../utils';

/**
 * Convert PcapPacket to Packet for PacketDetails component
 */
function pcapPacketToPacket(pcapPacket: PcapPacket): Packet {
  return {
    id: pcapPacket.id,
    timestamp: pcapPacket.timestamp,
    protocol: pcapPacket.protocol,
    sourceIP: pcapPacket.sourceIP,
    destIP: pcapPacket.destIP,
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
  // File and analysis state
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [analysisResult, setAnalysisResult] = useState<PcapAnalysisResult | null>(null);
  const [isAnalyzing, setIsAnalyzing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  // Selected packet state
  const [selectedPacket, setSelectedPacket] = useState<PcapPacket | null>(null);

  // Filter state
  const [protocolFilter, setProtocolFilter] = useState('All');
  const [sourceFilter, setSourceFilter] = useState('');
  const [destFilter, setDestFilter] = useState('');
  const [searchQuery, setSearchQuery] = useState('');

  // View mode state
  const [viewMode, setViewMode] = useState<'packets' | 'stats'>('packets');

  // Handle file selection
  const handleFileSelect = useCallback((file: File) => {
    setSelectedFile(file);
    setAnalysisResult(null);
    setSelectedPacket(null);
    setError(null);
    setSuccess(null);
    // Reset filters
    setProtocolFilter('All');
    setSourceFilter('');
    setDestFilter('');
    setSearchQuery('');
  }, []);

  // Handle analysis
  const handleAnalyze = useCallback(async () => {
    if (!selectedFile) return;

    setIsAnalyzing(true);
    setError(null);
    setSuccess(null);

    try {
      // Convert file to base64 and upload to backend
      const base64Data = await fileToBase64(selectedFile);
      const uploadResponse = await uploadPcap({ filename: selectedFile.name, data: base64Data });

      if (!uploadResponse.success) {
        throw new Error(uploadResponse.message || 'Upload failed');
      }

      // Fetch the analysis result
      const result = await fetchPcapAnalysis(uploadResponse.analysisId);

      setAnalysisResult(result);
      setSuccess(`Successfully analyzed ${result.packets.length} packets`);

      // Auto-select first packet
      if (result.packets.length > 0) {
        setSelectedPacket(result.packets[0]);
      }
    } catch (err) {
      const errorMessage = getErrorMessage(err) || 'Failed to analyze PCAP file';
      setError(errorMessage);
    } finally {
      setIsAnalyzing(false);
    }
  }, [selectedFile]);

  // Handle packet selection
  const handleSelectPacket = useCallback((packet: PcapPacket) => {
    setSelectedPacket(packet);
  }, []);

  // Handle clear
  const handleClear = useCallback(() => {
    setSelectedFile(null);
    setAnalysisResult(null);
    setSelectedPacket(null);
    setError(null);
    setSuccess(null);
    setProtocolFilter('All');
    setSourceFilter('');
    setDestFilter('');
    setSearchQuery('');
  }, []);

  // Export packets as JSON
  const handleExport = useCallback(() => {
    if (!analysisResult) return;

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

  // Convert selected packet for PacketDetails component
  const selectedPacketForDetails = useMemo(() => {
    return selectedPacket ? pcapPacketToPacket(selectedPacket) : null;
  }, [selectedPacket]);

  return (
    <div className="space-y-6">
      {/* Upload Section */}
      {!analysisResult && (
        <PcapUploader
          onFileSelect={handleFileSelect}
          onAnalyze={handleAnalyze}
          isAnalyzing={isAnalyzing}
          selectedFile={selectedFile}
          error={error}
          success={success}
        />
      )}

      {/* Results Section */}
      {analysisResult && (
        <>
          {/* Controls Header */}
          <Card className="border-white/5 bg-gray-900/70">
            <CardContent>
              <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
                {/* Title and file info */}
                <div className="flex items-center gap-4">
                  <H2 className="mb-0 flex items-center gap-2">
                    <FileSearch className="h-5 w-5 text-violet-400" />
                    PCAP Analysis
                  </H2>
                  <Tag colorScheme="green">{analysisResult.packets.length} packets</Tag>
                </div>

                {/* Control buttons */}
                <div className="flex flex-wrap items-center gap-2">
                  {/* View Mode Toggle */}
                  <div className="flex rounded-lg border border-white/10 bg-gray-950/50 p-1">
                    <button
                      type="button"
                      onClick={() => setViewMode('packets')}
                      className={`px-3 py-1.5 text-sm rounded-md transition-colors ${
                        viewMode === 'packets'
                          ? 'bg-violet-600 text-white'
                          : 'text-gray-400 hover:text-white'
                      }`}
                    >
                      Packets
                    </button>
                    <button
                      type="button"
                      onClick={() => setViewMode('stats')}
                      className={`px-3 py-1.5 text-sm rounded-md transition-colors ${
                        viewMode === 'stats'
                          ? 'bg-violet-600 text-white'
                          : 'text-gray-400 hover:text-white'
                      }`}
                    >
                      Statistics
                    </button>
                  </div>

                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={handleExport}
                    leftIcon={<Download className="h-4 w-4" />}
                  >
                    Export
                  </Button>

                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={handleClear}
                    leftIcon={<Trash2 className="h-4 w-4" />}
                  >
                    Clear
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Packets View */}
          {viewMode === 'packets' && (
            <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
              {/* Packet List */}
              <div className="lg:col-span-7 xl:col-span-8">
                <div className="h-[600px]">
                  <PcapPacketList
                    packets={analysisResult.packets}
                    selectedPacketId={selectedPacket?.id ?? null}
                    onSelectPacket={handleSelectPacket}
                    protocolFilter={protocolFilter}
                    sourceFilter={sourceFilter}
                    destFilter={destFilter}
                    searchQuery={searchQuery}
                    onProtocolFilterChange={setProtocolFilter}
                    onSourceFilterChange={setSourceFilter}
                    onDestFilterChange={setDestFilter}
                    onSearchQueryChange={setSearchQuery}
                  />
                </div>
              </div>

              {/* Right panel - Details */}
              <div className="lg:col-span-5 xl:col-span-4 space-y-6">
                {/* Hex Dump Viewer */}
                <Card className="border-white/5 bg-gray-900/70 h-[280px]">
                  <CardContent className="h-full flex flex-col">
                    <SmallText className="text-gray-400 uppercase tracking-wide font-semibold mb-3">
                      Hex Dump
                    </SmallText>
                    <div className="flex-1 min-h-0">
                      <HexDumpViewer
                        rawData={selectedPacket?.rawData ?? ''}
                        headerLength={14}
                      />
                    </div>
                  </CardContent>
                </Card>

                {/* Packet Details */}
                <Card className="border-white/5 bg-gray-900/70 h-[280px]">
                  <CardContent className="h-full flex flex-col">
                    <SmallText className="text-gray-400 uppercase tracking-wide font-semibold mb-3">
                      Packet Details
                    </SmallText>
                    <div className="flex-1 min-h-0 overflow-y-auto">
                      <PacketDetails packet={selectedPacketForDetails} />
                    </div>
                  </CardContent>
                </Card>
              </div>
            </div>
          )}

          {/* Statistics View */}
          {viewMode === 'stats' && (
            <PcapStats
              stats={analysisResult.stats}
              filename={analysisResult.filename}
              fileSize={analysisResult.fileSize}
            />
          )}
        </>
      )}

      {/* Info Banner when no file is loaded */}
      {!analysisResult && !selectedFile && (
        <Card className="border-white/5 bg-gray-900/70">
          <CardContent>
            <div className="flex items-start gap-3">
              <Info className="h-5 w-5 text-blue-400 flex-shrink-0 mt-0.5" />
              <div>
                <p className="font-medium text-white">About PCAP Analyzer</p>
                <SmallText className="text-gray-400">
                  Upload a PCAP or PCAPNG file to analyze network traffic. The analyzer will parse
                  packets and provide detailed statistics, protocol breakdown, and packet-level
                  inspection including hex dump viewing.
                </SmallText>
                <div className="mt-3 flex flex-wrap gap-2">
                  <Tag colorScheme="gray">Supports .pcap</Tag>
                  <Tag colorScheme="gray">Supports .pcapng</Tag>
                  <Tag colorScheme="gray">Max 100MB</Tag>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
};

export default PcapAnalyzerPage;
