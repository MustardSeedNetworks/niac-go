import { type FC, useState } from 'react';
import { fetchFiles, fetchReplayStatus, startReplay, stopReplay } from '../api/client';
import { useApiResource } from '../hooks/useApiResource';
import { Button } from '../ui/Button';
import { Card, CardContent } from '../ui/Card';
import { Tag } from '../ui/Tag';
import { SmallText } from '../ui/Typography';
import { getErrorMessage } from '../utils/format';

export const ReplayControlPanel: FC = () => {
  const { data: pcapFiles } = useApiResource(() => fetchFiles('pcaps'), []);
  const { data: replayStatus, refetch: refetchStatus } = useApiResource(fetchReplayStatus, [], {
    intervalMs: 2000,
  });

  const [selectedFile, setSelectedFile] = useState('');
  const [loopMs, setLoopMs] = useState(0);
  const [scale, setScale] = useState(1.0);
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
    <div className="space-y-4">
      {/* Status Card */}
      {replayStatus?.running && (
        <Card>
          <CardContent>
            <div className="flex items-center justify-between">
              <div>
                <div className="flex items-center gap-2 mb-1">
                  <Tag colorScheme="green">Running</Tag>
                  <span className="font-medium">{replayStatus.file}</span>
                </div>
                <SmallText className="text-gray-400">
                  Started:{' '}
                  {replayStatus.startedAt
                    ? new Date(replayStatus.startedAt).toLocaleString()
                    : 'Unknown'}
                  {replayStatus.loopMs > 0 && ` • Looping every ${replayStatus.loopMs}ms`}
                  {replayStatus.scale !== 1.0 && ` • Scale: ${replayStatus.scale}x`}
                </SmallText>
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
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
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
                className="w-full px-3 py-2 bg-gray-800 border border-gray-700 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
              >
                <option value="">Select PCAP file...</option>
                {pcapFiles?.map((file) => (
                  <option key={file.path} value={file.path}>
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
                className="w-full px-3 py-2 bg-gray-800 border border-gray-700 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
              />
              <SmallText className="text-gray-400">
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
                className="w-full px-3 py-2 bg-gray-800 border border-gray-700 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
              />
              <SmallText className="text-gray-400">
                1.0 = Original timing, 2.0 = 2x faster, 0.5 = 2x slower
              </SmallText>
            </div>
          </div>

          {/* Message Display */}
          {message && (
            <div
              role="alert"
              aria-live="polite"
              className={`p-3 rounded ${
                message.type === 'success'
                  ? 'bg-green-500/10 text-green-400 border border-green-500/20'
                  : 'bg-red-500/10 text-red-400 border border-red-500/20'
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
