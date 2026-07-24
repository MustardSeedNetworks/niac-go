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
  const { t } = useTranslation('pages');
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
      setMessage({ type: 'error', text: t('traffic.page.replayNoFileError') });
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
      setMessage({ type: 'success', text: t('traffic.page.replayStartSuccess') });
      refetchStatus();
    } catch (err: unknown) {
      setMessage({
        type: 'error',
        text: getErrorMessage(err) || t('traffic.page.replayStartError'),
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleStop = async () => {
    setIsSubmitting(true);
    try {
      await stopReplay();
      setMessage({ type: 'success', text: t('traffic.page.replayStopSuccess') });
      refetchStatus();
    } catch (err: unknown) {
      setMessage({
        type: 'error',
        text: getErrorMessage(err) || t('traffic.page.replayStopError'),
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
                  <Tag colorScheme="green">{t('traffic.page.replayStatusRunning')}</Tag>
                  <span className="font-medium">{replayStatus.file}</span>
                </div>
                <SmallText className="text-text-muted">
                  {t('traffic.page.replayStarted', {
                    time: replayStatus.startedAt
                      ? new Date(replayStatus.startedAt).toLocaleString()
                      : t('traffic.page.replayStartedUnknown'),
                  })}
                  {replayStatus.loopMs > 0 &&
                    ` • ${t('traffic.page.replayLoopingEvery', { ms: replayStatus.loopMs })}`}
                  {replayStatus.scale !== 1.0 &&
                    ` • ${t('traffic.page.replayScaleTag', { scale: replayStatus.scale })}`}
                  {replayStatus.rateMode &&
                    replayStatus.rateMode !== 'timing' &&
                    ` • ${t('traffic.page.replayRateTag', { mode: replayStatus.rateMode })}`}
                  {(replayStatus.loopCount ?? 0) > 0 &&
                    ` • ${t('traffic.page.replayLoopTag', { value: replayStatus.loopCount })}`}
                  {replayStatus.bpfFilter &&
                    ` • ${t('traffic.page.replayFilterTag', { filter: replayStatus.bpfFilter })}`}
                  {(replayStatus.passes ?? 0) > 1 &&
                    ` • ${t('traffic.page.replayIterationTag', { n: replayStatus.passes })}`}
                  {(replayStatus.packetsFiltered ?? 0) > 0 &&
                    ` • ${t('traffic.page.replayFilteredTag', { n: replayStatus.packetsFiltered })}`}
                </SmallText>
                <ReplayProgress
                  packetsSent={replayStatus.packetsSent}
                  bytesSent={replayStatus.bytesSent}
                  packetsTotal={replayStatus.packetsTotal}
                  percentComplete={replayStatus.percentComplete}
                />
              </div>
              <Button onClick={handleStop} disabled={isSubmitting} variant="secondary">
                {t('traffic.page.replayStopButton')}
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
                {t('traffic.page.replayFileLabel')}
              </label>
              <select
                id="replay-file"
                value={selectedFile}
                onChange={(e) => setSelectedFile(e.target.value)}
                disabled={replayStatus?.running}
                className="w-full px-3 py-row bg-bg-elevated border border-border-default rounded-md focus:outline-none focus:ring-2 focus:ring-status-info disabled:opacity-50"
              >
                <option value="">{t('traffic.page.replayFileSelectPrompt')}</option>
                {pcapFiles?.map((file) => (
                  <option key={file.name} value={file.name}>
                    {t('traffic.page.replayFileOption', {
                      name: file.name,
                      size: (file.sizeBytes / 1024).toFixed(1),
                    })}
                  </option>
                ))}
              </select>
            </div>

            {/* Loop Interval */}
            <div>
              <label htmlFor="replay-loop" className="block text-sm font-medium mb-2">
                {t('traffic.page.replayLoopIntervalLabel')}
              </label>
              <input
                id="replay-loop"
                type="number"
                min="0"
                step="1000"
                value={loopMs}
                onChange={(e) => setLoopMs(Number.parseInt(e.target.value, 10) || 0)}
                placeholder={t('traffic.page.replayLoopIntervalPlaceholder')}
                disabled={replayStatus?.running}
                className="w-full px-3 py-row bg-bg-elevated border border-border-default rounded-md focus:outline-none focus:ring-2 focus:ring-status-info disabled:opacity-50"
              />
              <SmallText className="text-text-muted">
                {t('traffic.page.replayLoopIntervalHelp')}
              </SmallText>
            </div>

            {/* Time Scale */}
            <div>
              <label htmlFor="replay-scale" className="block text-sm font-medium mb-2">
                {t('traffic.page.replayTimeScaleLabel')}
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
                {t('traffic.page.replayTimeScaleHelp')}
              </SmallText>
            </div>

            {/* Rate Mode */}
            <div>
              <label htmlFor="replay-rate-mode" className="block text-sm font-medium mb-2">
                {t('traffic.page.replayRateModeLabel')}
              </label>
              <select
                id="replay-rate-mode"
                value={rateMode}
                onChange={(e) => setRateMode(e.target.value as ReplayRateMode)}
                disabled={replayStatus?.running}
                className="w-full px-3 py-row bg-bg-elevated border border-border-default rounded-md focus:outline-none focus:ring-2 focus:ring-status-info disabled:opacity-50"
              >
                <option value="">{t('traffic.page.replayRateModeTiming')}</option>
                <option value="topspeed">{t('traffic.page.replayRateModeTopspeed')}</option>
                <option value="pps">{t('traffic.page.replayRateModePps')}</option>
                <option value="mbps">{t('traffic.page.replayRateModeMbps')}</option>
              </select>
              <SmallText className="text-text-muted">
                {t('traffic.page.replayRateModeHelp')}
              </SmallText>
            </div>

            {/* Rate parameter: pps or Mbps, shown for the matching mode */}
            {rateMode === 'pps' && (
              <div>
                <label htmlFor="replay-pps" className="block text-sm font-medium mb-2">
                  {t('traffic.page.replayPpsLabel')}
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
                <SmallText className="text-text-muted">{t('traffic.page.replayPpsHelp')}</SmallText>
              </div>
            )}
            {rateMode === 'mbps' && (
              <div>
                <label htmlFor="replay-mbps" className="block text-sm font-medium mb-2">
                  {t('traffic.page.replayMbpsLabel')}
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
                <SmallText className="text-text-muted">
                  {t('traffic.page.replayMbpsHelp')}
                </SmallText>
              </div>
            )}

            {/* Loop Count */}
            <div>
              <label htmlFor="replay-loop-count" className="block text-sm font-medium mb-2">
                {t('traffic.page.replayLoopCountLabel')}
              </label>
              <input
                id="replay-loop-count"
                type="number"
                min="0"
                step="1"
                value={loopCount}
                onChange={(e) => setLoopCount(Number.parseInt(e.target.value, 10) || 0)}
                placeholder={t('traffic.page.replayLoopCountPlaceholder')}
                disabled={replayStatus?.running}
                className="w-full px-3 py-row bg-bg-elevated border border-border-default rounded-md focus:outline-none focus:ring-2 focus:ring-status-info disabled:opacity-50"
              />
              <SmallText className="text-text-muted">
                {t('traffic.page.replayLoopCountHelp')}
              </SmallText>
            </div>

            {/* BPF Filter */}
            <div className="col-span-full">
              <label htmlFor="replay-bpf" className="block text-sm font-medium mb-2">
                {t('traffic.page.replayBpfLabel')}
              </label>
              <input
                id="replay-bpf"
                type="text"
                value={bpfFilter}
                onChange={(e) => setBpfFilter(e.target.value)}
                placeholder={t('traffic.page.replayBpfPlaceholder')}
                disabled={replayStatus?.running}
                className="w-full px-3 py-row bg-bg-elevated border border-border-default rounded-md focus:outline-none focus:ring-2 focus:ring-status-info disabled:opacity-50"
              />
              <SmallText className="text-text-muted">{t('traffic.page.replayBpfHelp')}</SmallText>
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
              {isSubmitting
                ? t('traffic.page.replayStarting')
                : t('traffic.page.replayStartButton')}
            </Button>
          )}
        </CardContent>
      </Card>
    </div>
  );
};
