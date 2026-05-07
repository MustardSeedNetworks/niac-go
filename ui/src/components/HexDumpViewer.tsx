import { type FC, memo, useMemo } from 'react';

interface HexDumpViewerProps {
  /** Hex-encoded raw packet data */
  rawData: string;
  /** Number of bytes per row (default: 16) */
  bytesPerRow?: number;
  /** Header length in bytes for color coding */
  headerLength?: number;
  /** Optional byte range to highlight (from protocol tree field selection) */
  highlightRange?: [number, number];
}

/**
 * Convert hex string to byte array
 */
function hexToBytes(hex: string): number[] {
  const bytes: number[] = [];
  const cleanHex = hex.replace(/\s/g, '');

  for (let i = 0; i < cleanHex.length; i += 2) {
    const byte = Number.parseInt(cleanHex.slice(i, i + 2), 16);
    if (!Number.isNaN(byte)) {
      bytes.push(byte);
    }
  }

  return bytes;
}

/**
 * Convert byte to printable ASCII character or dot
 */
function byteToChar(byte: number): string {
  // Printable ASCII range: 32-126
  if (byte >= 32 && byte <= 126) {
    return String.fromCharCode(byte);
  }
  return '.';
}

/**
 * Format offset as 8-digit hex address
 */
function formatOffset(offset: number): string {
  return offset.toString(16).padStart(8, '0').toUpperCase();
}

/**
 * Format single byte as 2-digit hex
 */
function formatByte(byte: number): string {
  return byte.toString(16).padStart(2, '0').toUpperCase();
}

interface HexRowProps {
  offset: number;
  bytes: number[];
  bytesPerRow: number;
  headerLength: number;
  startIndex: number;
  highlightStart: number;
  highlightEnd: number;
}

/**
 * Individual hex dump row - memoized for performance
 */
const HexRow = memo(
  ({
    offset,
    bytes,
    bytesPerRow,
    headerLength,
    startIndex,
    highlightStart,
    highlightEnd,
  }: HexRowProps) => {
    // Build hex column with color coding
    const hexCells = bytes.map((byte, idx) => {
      const globalIndex = startIndex + idx;
      const isHeader = globalIndex < headerLength;
      const isHighlighted =
        highlightStart >= 0 && globalIndex >= highlightStart && globalIndex < highlightEnd;

      let className = isHeader ? 'text-cyan-400' : 'text-gray-300';
      if (isHighlighted) {
        className = 'text-yellow-200 bg-yellow-700/40 rounded-sm';
      }

      return (
        <span key={globalIndex} className={`${className} ${idx === 7 ? 'mr-1' : ''}`}>
          {formatByte(byte)}
          {idx < bytes.length - 1 ? ' ' : ''}
        </span>
      );
    });

    // Pad with spaces if row is not full
    const paddingCount = bytesPerRow - bytes.length;
    const padding =
      paddingCount > 0 ? <span className="text-gray-700">{'   '.repeat(paddingCount)}</span> : null;

    // Build ASCII column
    const asciiChars = bytes.map((byte, idx) => {
      const globalIndex = startIndex + idx;
      const isHeader = globalIndex < headerLength;
      const isHighlighted =
        highlightStart >= 0 && globalIndex >= highlightStart && globalIndex < highlightEnd;
      const char = byteToChar(byte);

      let asciiClass = isHeader ? 'text-cyan-400' : 'text-gray-300';
      if (isHighlighted) {
        asciiClass = 'text-yellow-200 bg-yellow-700/40';
      }

      return (
        <span key={globalIndex} className={asciiClass}>
          {char}
        </span>
      );
    });

    return (
      <div className="flex font-mono text-xs leading-5">
        {/* Offset column */}
        <span className="text-violet-400 w-20 flex-shrink-0">{formatOffset(offset)}:</span>

        {/* Hex column */}
        <span className="flex-1 min-w-0">
          {hexCells}
          {padding}
        </span>

        {/* Separator */}
        <span className="text-gray-600 mx-2">|</span>

        {/* ASCII column */}
        <span className="w-16 flex-shrink-0 text-right">{asciiChars}</span>
      </div>
    );
  },
);

HexRow.displayName = 'HexRow';

/**
 * Hex Dump Viewer Component
 *
 * Displays raw packet data in a traditional hex dump format:
 * - Offset address (left column)
 * - Hex bytes (middle column, 16 bytes per row with gap at byte 8)
 * - ASCII representation (right column)
 *
 * Header bytes are color-coded in cyan for easy identification.
 */
export const HexDumpViewer: FC<HexDumpViewerProps> = memo(
  ({
    rawData,
    bytesPerRow = 16,
    headerLength = 14, // Default Ethernet header length
    highlightRange,
  }) => {
    const highlightStart = highlightRange?.[0] ?? -1;
    const highlightEnd = highlightRange?.[1] ?? -1;
    // Parse hex data into rows
    const rows = useMemo(() => {
      if (!rawData) {
        return [];
      }

      const bytes = hexToBytes(rawData);
      const result: { offset: number; bytes: number[]; startIndex: number }[] = [];

      for (let i = 0; i < bytes.length; i += bytesPerRow) {
        result.push({
          offset: i,
          bytes: bytes.slice(i, i + bytesPerRow),
          startIndex: i,
        });
      }

      return result;
    }, [rawData, bytesPerRow]);

    if (!rawData) {
      return (
        <div className="h-full flex items-center justify-center text-gray-400">
          <p className="text-sm">Select a packet to view hex dump</p>
        </div>
      );
    }

    const totalBytes = hexToBytes(rawData).length;

    return (
      <div className="h-full flex flex-col">
        {/* Header with legend */}
        <div className="flex items-center justify-between mb-2 pb-2 border-b border-white/10">
          <div className="flex items-center gap-4 text-xs">
            <span className="text-gray-400">{totalBytes} bytes</span>
            <div className="flex items-center gap-2">
              <span className="w-3 h-3 bg-cyan-400/30 rounded" />
              <span className="text-gray-400">Header</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="w-3 h-3 bg-gray-300/30 rounded" />
              <span className="text-gray-400">Payload</span>
            </div>
            {highlightStart >= 0 && (
              <div className="flex items-center gap-2">
                <span className="w-3 h-3 bg-yellow-700/40 rounded" />
                <span className="text-gray-400">Selected</span>
              </div>
            )}
          </div>
        </div>

        {/* Hex dump content */}
        <div className="flex-1 overflow-y-auto rounded-lg bg-gray-950/70 p-3 border border-white/5">
          <div className="space-y-0.5">
            {rows.map((row) => (
              <HexRow
                key={row.offset}
                offset={row.offset}
                bytes={row.bytes}
                bytesPerRow={bytesPerRow}
                headerLength={headerLength}
                startIndex={row.startIndex}
                highlightStart={highlightStart}
                highlightEnd={highlightEnd}
              />
            ))}
          </div>
        </div>
      </div>
    );
  },
);

HexDumpViewer.displayName = 'HexDumpViewer';
