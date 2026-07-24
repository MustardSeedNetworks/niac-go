import { type FC, memo, useMemo } from 'react';
import { useTranslation } from 'react-i18next';

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
    const byte = Number.parseInt(cleanHex.substring(i, i + 2), 16);
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

      let className = isHeader ? 'text-status-info' : 'text-text-secondary';
      if (isHighlighted) {
        className = 'text-status-warning bg-status-warning/40 rounded-sm';
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
      paddingCount > 0 ? (
        <span className="text-text-disabled">{'   '.repeat(paddingCount)}</span>
      ) : null;

    // Build ASCII column
    const asciiChars = bytes.map((byte, idx) => {
      const globalIndex = startIndex + idx;
      const isHeader = globalIndex < headerLength;
      const isHighlighted =
        highlightStart >= 0 && globalIndex >= highlightStart && globalIndex < highlightEnd;
      const char = byteToChar(byte);

      let asciiClass = isHeader ? 'text-status-info' : 'text-text-secondary';
      if (isHighlighted) {
        asciiClass = 'text-status-warning bg-status-warning/40';
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
        <span className="text-brand-accent w-20 flex-shrink-0">{formatOffset(offset)}:</span>

        {/* Hex column */}
        <span className="flex-1 min-w-0">
          {hexCells}
          {padding}
        </span>

        {/* Separator */}
        <span className="text-text-disabled mx-2">|</span>

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
    const { t } = useTranslation('pages');
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
        <div className="h-full flex-center text-text-muted">
          <p className="text-sm">{t('packets.hexDump.selectPacketPlaceholder')}</p>
        </div>
      );
    }

    const totalBytes = hexToBytes(rawData).length;

    return (
      <div className="h-full flex flex-col">
        {/* Header with legend */}
        <div className="flex-between mb-2 pb-inline border-b border-surface-border">
          <div className="flex items-center gap-comfortable text-xs">
            <span className="text-text-muted">
              {t('packets.stats.totalBytesHelper', { value: totalBytes })}
            </span>
            <div className="flex items-center gap-compact">
              <span className="w-3 h-3 bg-status-info/30 rounded" />
              <span className="text-text-muted">{t('packets.hexDump.headerLegend')}</span>
            </div>
            <div className="flex items-center gap-compact">
              <span className="w-3 h-3 bg-bg-muted/30 rounded" />
              <span className="text-text-muted">{t('packets.hexDump.payloadLegend')}</span>
            </div>
            {highlightStart >= 0 && (
              <div className="flex items-center gap-compact">
                <span className="w-3 h-3 bg-status-warning/40 rounded" />
                <span className="text-text-muted">{t('packets.hexDump.selectedLegend')}</span>
              </div>
            )}
          </div>
        </div>

        {/* Hex dump content */}
        <div className="flex-1 overflow-y-auto rounded-lg bg-bg-base/70 pad-sm border border-surface-border">
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
