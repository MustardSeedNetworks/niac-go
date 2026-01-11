import { type FC, useState, useCallback, useMemo } from 'react';
import {
  Card,
  CardContent,
  Button,
  Tag,
  SmallText,
  H2,
} from '@krisarmstrong/web-foundation';
import { FileSearch, Download, Trash2, Info } from 'lucide-react';
import { PcapUploader } from '../components/pcap/PcapUploader';
import { PcapPacketList } from '../components/pcap/PcapPacketList';
import { PcapStats } from '../components/pcap/PcapStats';
import { HexDumpViewer } from '../components/HexDumpViewer';
import { PacketDetails } from '../components/PacketDetails';
import type { PcapPacket, PcapStats as PcapStatsType, PcapAnalysisResult } from '../api/types';
import type { Packet } from '../components/PacketList';

/**
 * Convert File to base64 string
 * Used for API upload when backend is implemented
 */
// eslint-disable-next-line @typescript-eslint/no-unused-vars
async function fileToBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      const result = reader.result as string;
      // Remove data URL prefix (data:*/*;base64,)
      const base64 = result.split(',')[1] || result;
      resolve(base64);
    };
    reader.onerror = () => reject(new Error('Failed to read file'));
    reader.readAsDataURL(file);
  });
}

/**
 * Generate unique ID for packets
 */
function generatePacketId(): string {
  return `${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
}

/**
 * Mock PCAP parser - generates sample packets from file
 * This simulates what the backend would return
 * TODO: Replace with actual API call when backend is implemented
 */
async function mockParsePcap(file: File): Promise<PcapAnalysisResult> {
  // Simulate network delay
  await new Promise((resolve) => setTimeout(resolve, 1500));

  // Generate mock packets
  const numPackets = Math.floor(Math.random() * 100) + 50;
  const baseTime = Date.now() - 60000; // Start 1 minute ago

  const protocols = ['TCP', 'UDP', 'ICMP', 'ARP', 'DNS', 'HTTP', 'TLS', 'DHCP'];
  const sourceIPs = [
    '192.168.1.100',
    '192.168.1.101',
    '192.168.1.102',
    '10.0.0.1',
    '10.0.0.50',
    '172.16.0.10',
  ];
  const destIPs = [
    '8.8.8.8',
    '1.1.1.1',
    '192.168.1.1',
    '10.0.0.254',
    '172.16.0.1',
    '224.0.0.1',
  ];

  const packets: PcapPacket[] = [];
  const protocolCounts: Record<string, number> = {};
  const sourceCounts: Record<string, number> = {};
  const destCounts: Record<string, number> = {};
  let totalBytes = 0;

  for (let i = 0; i < numPackets; i++) {
    const protocol = protocols[Math.floor(Math.random() * protocols.length)];
    const sourceIP = sourceIPs[Math.floor(Math.random() * sourceIPs.length)];
    const destIP = destIPs[Math.floor(Math.random() * destIPs.length)];
    const length = Math.floor(Math.random() * 1400) + 60;

    // Update counts
    protocolCounts[protocol] = (protocolCounts[protocol] || 0) + 1;
    sourceCounts[sourceIP] = (sourceCounts[sourceIP] || 0) + 1;
    destCounts[destIP] = (destCounts[destIP] || 0) + 1;
    totalBytes += length;

    const packet: PcapPacket = {
      id: generatePacketId(),
      number: i + 1,
      timestamp: new Date(baseTime + i * 10).toISOString(),
      sourceIP,
      destIP,
      sourcePort: protocol === 'TCP' || protocol === 'UDP' ? Math.floor(Math.random() * 65535) : undefined,
      destPort: protocol === 'TCP' || protocol === 'UDP' ? Math.floor(Math.random() * 65535) : undefined,
      protocol,
      length,
      info: getPacketInfo(protocol, sourceIP, destIP),
      rawData: generateMockHexData(length),
    };

    packets.push(packet);
  }

  // Sort sources and destinations by count
  const topSources = Object.entries(sourceCounts)
    .map(([ip, count]) => ({ ip, count }))
    .sort((a, b) => b.count - a.count);

  const topDestinations = Object.entries(destCounts)
    .map(([ip, count]) => ({ ip, count }))
    .sort((a, b) => b.count - a.count);

  const stats: PcapStatsType = {
    totalPackets: numPackets,
    totalBytes,
    timeRange: {
      start: packets[0]?.timestamp || new Date().toISOString(),
      end: packets[packets.length - 1]?.timestamp || new Date().toISOString(),
      durationMs: numPackets * 10,
    },
    protocols: protocolCounts,
    topSources,
    topDestinations,
  };

  return {
    filename: file.name,
    fileSize: file.size,
    packets,
    stats,
  };
}

/**
 * Generate mock packet info based on protocol
 */
function getPacketInfo(protocol: string, source: string, dest: string): string {
  switch (protocol) {
    case 'TCP':
      return `${Math.floor(Math.random() * 65535)} > ${Math.floor(Math.random() * 65535)} [SYN] Seq=0 Win=65535 Len=0`;
    case 'UDP':
      return `Source port: ${Math.floor(Math.random() * 65535)} Destination port: ${Math.floor(Math.random() * 65535)}`;
    case 'ICMP':
      return 'Echo (ping) request';
    case 'ARP':
      return `Who has ${dest}? Tell ${source}`;
    case 'DNS':
      return 'Standard query 0x1234 A example.com';
    case 'HTTP':
      return 'GET /index.html HTTP/1.1';
    case 'TLS':
      return 'Client Hello';
    case 'DHCP':
      return 'DHCP Discover - Transaction ID 0x12345678';
    default:
      return `${protocol} packet`;
  }
}

/**
 * Generate mock hex data
 */
function generateMockHexData(length: number): string {
  const bytes = Math.min(length, 256);
  let hex = '';
  for (let i = 0; i < bytes; i++) {
    hex += Math.floor(Math.random() * 256).toString(16).padStart(2, '0');
  }
  return hex;
}

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
      // TODO: Replace with actual API call when backend is implemented
      // const base64Data = await fileToBase64(selectedFile);
      // const uploadResponse = await uploadPcap({ filename: selectedFile.name, data: base64Data });
      // const result = await fetchPcapAnalysis(uploadResponse.analysisId);

      // For now, use mock parser
      const result = await mockParsePcap(selectedFile);

      setAnalysisResult(result);
      setSuccess(`Successfully analyzed ${result.packets.length} packets`);

      // Auto-select first packet
      if (result.packets.length > 0) {
        setSelectedPacket(result.packets[0]);
      }
    } catch (err) {
      setError((err as Error).message || 'Failed to analyze PCAP file');
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
