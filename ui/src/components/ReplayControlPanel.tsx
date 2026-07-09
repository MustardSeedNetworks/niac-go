import { type FC, useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { ReplayRateMode } from '../api/api-response-types';
import { fetchReplayStatus, startReplay, stopReplay } from '../api/client';
import { fetchLibraryPcaps } from '../api/library-client';
import { useApiResource } from '../hooks/useApiResource';
import { Button } from '../ui/Button';
import { Card, CardContent } from '../ui/Card';
import { Tag } from '../ui/Tag';
import { SmallText } from '../ui/Typography';
import { formatBytes, getErrorMessage } from '../utils/format';

const PERCENT_MAX = 100;

/**
 * Live packets/bytes-sent progress bar for an in-flight replay. Renders
 * "unknown total" copy instead of a fake 0% bar when the backend hasn't
 * reported a packet total yet (percentComplete omitted from the JSON).
 */
const ReplayProgress: FC<{
  packetsSent: number;
  bytesSent: number;
  packetsTotal: number;
  percentComplete?: number;
}> = ({ packetsSent, bytesSent, packetsTotal, percentComplete }) => {
  const { t } = useTranslation('pages');
  const hasTotal = packetsTotal > 0 && percentComplete !== undefined;
  const percent = hasTotal ? Math.min(PERCENT_MAX, Math.max(0, percentComplete)) : 0;

  return (
    <div className="mt-tight" data-testid="replay-progress">
      <div
        className="w-full h-2 bg-bg-elevated border border-border-default rounded-full overflow-hidden"
        role="progressbar"
        aria-valuenow={hasTotal ? percent : undefined}
        aria-valuemin={0}
        aria-valuemax={PERCENT_MAX}
        aria-label={t('traffic.page.replayProgressLabel', {
          sent: packetsSent,
          total: packetsTotal,
          percent: percent.toFixed(0),
        })}
      >
        <div
          className="h-full bg-status-info transition-[width] duration-300"
          style={{ width: hasTotal ? `${percent}%` : '100%' }}
        />
      </div>
      <SmallText className="text-text-muted">
        {hasTotal
          ? t('traffic.page.replayProgressLabel', {
              sent: packetsSent,
              total: packetsTotal,
              percent: percent.toFixed(0),
            })
          : t('traffic.page.replayProgressUnknown')}
        {' • '}
        {t('traffic.page.replayProgressBytes', { bytes: formatBytes(bytesSent) })}
      </SmallText>
    </div>
  );
};

export const ReplayControlPanel: FC = () => {
  // Hydrates from /api/v1/library/pcaps. The daemon's
  // validatePcapFilePath falls back to ~/.niac/library/pcaps/ when
  // the file isn't in the legacy config-dir allow-list, so passing
  // bare library names like "sample.pcap" to startReplay works
  // without any further translation in the UI.
  const { data: pcapFiles } = useApiResource(fetchLibraryPcaps, []);
  const { data: replayStatus, refetch: refetchStatus } = useApiResource(fetchReplayStatus, [], {
    intervalMs: 2000,
  });

  const [selectedFile, setSelectedFile] = useState('');
  const [loopMs, setLoopMs] = useState(0);
  const [scale, setScale] = useState(1.0);
  const [rateMode, setRateMode] = useState<ReplayRateMode>('');
  const [pps, setPps] = useState(0);
  const [mbpsCap, setMbpsCap] = useState(0);
  const [loopCount, setLoopCount] = useState(0);
  const [bpfFilter, setBpfFilter] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [message, setMessage] = useState<{
    type: 'success' | 'error';
    text: string;
  } | null>(null);

  const handleStart = async () => {
    if (!selectedFile) {
      setMessage({ type: 'error', text: 'Please select a PCAP file' });
      return;
    }

    setIsSubmitting(true);
    setMessage(null);

    try {
      await startReplay({
        file: selectedFile,
        loopMs: loopMs,
        scale: scale,
        rateMode: rateMode || undefined,
        pps: rateMode === 'pps' ? pps : undefined,
        mbpsCap: rateMode === 'mbps' ? mbpsCap : undefined,
        loopCount: loopCount || undefined,
        bpfFilter: bpfFilter.trim() || undefined,
      });
      setMessage({ type: 'success', text: 'Replay started successfully' });
      refetchStatus();
    } catch (err: unknown) {
      setMessage({
        type: 'error',
        text: getErrorMessage(err) || 'Failed to start replay',
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleStop = async () => {
    setIsSubmitting(true);
    try {
      await stopReplay();
      setMessage({ type: 'success', text: 'Replay stopped' });
      refetchStatus();
    } catch (err: unknown) {
      setMessage({
        type: 'error',
        text: getErrorMessage(err) || 'Failed to stop replay',
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="stack-lg">
      {/* Status Card */}
      {replayStatus?.running && (
        <Card>
          <CardContent>
            <div className="flex-between">
              <div>
                <div className="flex items-center gap-compact mb-tight">
                  <Tag colorScheme="green">Running</Tag>
                  <span className="font-medium">{replayStatus.file}</span>
                </div>
                <SmallText className="text-text-muted">
                  Started:{' '}
                  {replayStatus.startedAt
                    ? new Date(replayStatus.startedAt).toLocaleString()
                    : 'Unknown'}
                  {replayStatus.loopMs > 0 && ` • Looping every ${replayStatus.loopMs}ms`}
                  {replayStatus.scale !== 1.0 && ` • Scale: ${replayStatus.scale}x`}
                  {replayStatus.rateMode &&
                    replayStatus.rateMode !== 'timing' &&
                    ` • Rate: ${replayStatus.rateMode}`}
                  {(replayStatus.loopCount ?? 0) > 0 && ` • Loop ${replayStatus.loopCount}×`}
                  {replayStatus.bpfFilter && ` • Filter: ${replayStatus.bpfFilter}`}
                  {(replayStatus.passes ?? 0) > 1 && ` • Iteration ${replayStatus.passes}`}
                  {(replayStatus.packetsFiltered ?? 0) > 0 &&
                    ` • ${replayStatus.packetsFiltered} filtered`}
                </SmallText>
                <ReplayProgress
                  packetsSent={replayStatus.packetsSent}
                  bytesSent={replayStatus.bytesSent}
                  packetsTotal={replayStatus.packetsTotal}
                  percentComplete={replayStatus.percentComplete}
                />
              </div>
              <Button onClick={handleStop} disabled={isSubmitting} variant="secondary">
                Stop Replay
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Control Card */}
      <Card>
        <CardContent className="stack-lg">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-comfortable">
            {/* File Selector */}
            <div className="col-span-full">
              <label htmlFor="replay-file" className="block text-sm font-medium mb-2">
                PCAP File
              </label>
              <select
                id="replay-file"
                value={selectedFile}
                onChange={(e) => setSelectedFile(e.target.value)}
                disabled={replayStatus?.running}
                className="w-full px-3 py-row bg-bg-elevated border border-border-default rounded-md focus:outline-none focus:ring-2 focus:ring-status-info disabled:opacity-50"
              >
                <option value="">Select PCAP file...</option>
                {pcapFiles?.map((file) => (
                  <option key={file.name} value={file.name}>
                    {file.name} ({(file.sizeBytes / 1024).toFixed(1)} KB)
                  </option>
                ))}
              </select>
            </div>

            {/* Loop Interval */}
            <div>
              <label htmlFor="replay-loop" className="block text-sm font-medium mb-2">
                Loop Interval (ms)
              </label>
              <input
                id="replay-loop"
                type="number"
                min="0"
                step="1000"
                value={loopMs}
                onChange={(e) => setLoopMs(Number.parseInt(e.target.value, 10) || 0)}
                placeholder="0 = no loop"
                disabled={replayStatus?.running}
                className="w-full px-3 py-row bg-bg-elevated border border-border-default rounded-md focus:outline-none focus:ring-2 focus:ring-status-info disabled:opacity-50"
              />
              <SmallText className="text-text-muted">
                0 = Play once, &gt;0 = Loop with delay
              </SmallText>
            </div>

            {/* Time Scale */}
            <div>
              <label htmlFor="replay-scale" className="block text-sm font-medium mb-2">
                Time Scale
              </label>
              <input
                id="replay-scale"
                type="number"
                min="0.1"
                max="10"
                step="0.1"
                value={scale}
                onChange={(e) => setScale(Number.parseFloat(e.target.value) || 1.0)}
                disabled={replayStatus?.running}
                className="w-full px-3 py-row bg-bg-elevated border border-border-default rounded-md focus:outline-none focus:ring-2 focus:ring-status-info disabled:opacity-50"
              />
              <SmallText className="text-text-muted">
                1.0 = Original timing, 2.0 = 2x faster, 0.5 = 2x slower
              </SmallText>
            </div>

            {/* Rate Mode */}
            <div>
              <label htmlFor="replay-rate-mode" className="block text-sm font-medium mb-2">
                Rate Mode
              </label>
              <select
                id="replay-rate-mode"
                value={rateMode}
                onChange={(e) => setRateMode(e.target.value as ReplayRateMode)}
                disabled={replayStatus?.running}
                className="w-full px-3 py-row bg-bg-elevated border border-border-default rounded-md focus:outline-none focus:ring-2 focus:ring-status-info disabled:opacity-50"
              >
                <option value="">Timing (original)</option>
                <option value="topspeed">Topspeed</option>
                <option value="pps">Fixed rate (pps)</option>
                <option value="mbps">Throughput cap (Mbps)</option>
              </select>
              <SmallText className="text-text-muted">
                Timing honors capture timing (× Time Scale); the others override it
              </SmallText>
            </div>

            {/* Rate parameter: pps or Mbps, shown for the matching mode */}
            {rateMode === 'pps' && (
              <div>
                <label htmlFor="replay-pps" className="block text-sm font-medium mb-2">
                  Rate (pps)
                </label>
                <input
                  id="replay-pps"
                  type="number"
                  min="1"
                  step="100"
                  value={pps}
                  onChange={(e) => setPps(Number.parseFloat(e.target.value) || 0)}
                  disabled={replayStatus?.running}
                  className="w-full px-3 py-row bg-bg-elevated border border-border-default rounded-md focus:outline-none focus:ring-2 focus:ring-status-info disabled:opacity-50"
                />
                <SmallText className="text-text-muted">Packets per second</SmallText>
              </div>
            )}
            {rateMode === 'mbps' && (
              <div>
                <label htmlFor="replay-mbps" className="block text-sm font-medium mb-2">
                  Throughput cap (Mbps)
                </label>
                <input
                  id="replay-mbps"
                  type="number"
                  min="0.1"
                  step="1"
                  value={mbpsCap}
                  onChange={(e) => setMbpsCap(Number.parseFloat(e.target.value) || 0)}
                  disabled={replayStatus?.running}
                  className="w-full px-3 py-row bg-bg-elevated border border-border-default rounded-md focus:outline-none focus:ring-2 focus:ring-status-info disabled:opacity-50"
                />
                <SmallText className="text-text-muted">Megabits per second</SmallText>
              </div>
            )}

            {/* Loop Count */}
            <div>
              <label htmlFor="replay-loop-count" className="block text-sm font-medium mb-2">
                Loop Count
              </label>
              <input
                id="replay-loop-count"
                type="number"
                min="0"
                step="1"
                value={loopCount}
                onChange={(e) => setLoopCount(Number.parseInt(e.target.value, 10) || 0)}
                placeholder="0 = per Loop Interval"
                disabled={replayStatus?.running}
                className="w-full px-3 py-row bg-bg-elevated border border-border-default rounded-md focus:outline-none focus:ring-2 focus:ring-status-info disabled:opacity-50"
              />
              <SmallText className="text-text-muted">
                0 = unbounded, N = stop after N passes
              </SmallText>
            </div>

            {/* BPF Filter */}
            <div className="col-span-full">
              <label htmlFor="replay-bpf" className="block text-sm font-medium mb-2">
                BPF Filter
              </label>
              <input
                id="replay-bpf"
                type="text"
                value={bpfFilter}
                onChange={(e) => setBpfFilter(e.target.value)}
                placeholder="e.g. udp port 53 (blank = replay all)"
                disabled={replayStatus?.running}
                className="w-full px-3 py-row bg-bg-elevated border border-border-default rounded-md focus:outline-none focus:ring-2 focus:ring-status-info disabled:opacity-50"
              />
              <SmallText className="text-text-muted">
                tcpdump-style filter; replay only matching packets
              </SmallText>
            </div>
          </div>

          {/* Message Display */}
          {message && (
            <div
              role="alert"
              aria-live="polite"
              className={`pad-sm rounded ${
                message.type === 'success'
                  ? 'bg-status-success/10 text-status-success border border-status-success/20'
                  : 'bg-status-error/10 text-status-error border border-status-error/20'
              }`}
            >
              {message.text}
            </div>
          )}

          {/* Action Button */}
          {!replayStatus?.running && (
            <Button onClick={handleStart} disabled={isSubmitting} className="w-full">
              {isSubmitting ? 'Starting...' : 'Start Replay'}
            </Button>
          )}
        </CardContent>
      </Card>
    </div>
  );
};
