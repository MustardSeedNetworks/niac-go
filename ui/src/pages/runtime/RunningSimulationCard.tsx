import { Activity, Download, FileCog } from 'lucide-react';
import type { FC } from 'react';
import { useNavigate } from 'react-router-dom';
import { fetchConfig } from '../../api/client';
import { StatBlock } from '../../components/StatBlock';
import { iconSizes } from '../../constants/sizes';
import { Button } from '../../ui/Button';
import { Card, CardContent } from '../../ui/Card';
import { Tag } from '../../ui/Tag';
import { H2, SmallText } from '../../ui/Typography';
import { formatTime, formatUptime } from '../../utils/format';

export interface RunningSimulationCardProps {
  simStatus: {
    interface?: string;
    configName?: string;
    configPath?: string;
    deviceCount: number;
    uptimeSeconds: number;
    startedAt?: string;
  };
  stopping: boolean;
  onStop: () => void;
  message: { tone: 'success' | 'error'; text: string } | null;
}

/**
 * RunningSimulationCard — the green "simulation is live" card. Lives
 * alongside RuntimeControlPage in pages/runtime/ so the page-level
 * file reads as a clean Start/Stop flow without 100+ lines of
 * Stat-block markup.
 */
export const RunningSimulationCard: FC<RunningSimulationCardProps> = ({
  simStatus,
  stopping,
  onStop,
  message,
}) => {
  const navigate = useNavigate();

  const handleDownload = async () => {
    try {
      const doc = await fetchConfig();
      const blob = new Blob([doc.content], { type: 'application/x-yaml' });
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = doc.filename || 'niac-config.yaml';
      link.click();
      URL.revokeObjectURL(url);
    } catch (err) {
      console.error('Failed to download config:', err);
    }
  };

  return (
    <Card className="border-green-500/30 bg-gradient-to-br from-green-900/30 to-gray-900/70">
      <CardContent className="space-y-5">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="h-3 w-3 animate-pulse rounded-full bg-green-400" />
            <H2>Simulation Running</H2>
          </div>
          <Tag colorScheme="green">ACTIVE</Tag>
        </div>

        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          <StatBlock
            label="Interface"
            value={simStatus.interface || '—'}
            helper="Network interface"
          />
          <StatBlock
            label="Config"
            value={simStatus.configName || '—'}
            helper={simStatus.configPath || 'Configuration file'}
          />
          <StatBlock
            label="Devices"
            value={simStatus.deviceCount.toString()}
            helper="Simulated devices"
          />
          <StatBlock
            label="Uptime"
            value={formatUptime(simStatus.uptimeSeconds)}
            helper="Time running"
          />
          <StatBlock
            label="Started"
            value={simStatus.startedAt ? formatTime(simStatus.startedAt) : '—'}
            helper="Start time"
          />
        </div>

        {message && (
          <SmallText className={message.tone === 'success' ? 'text-emerald-300' : 'text-red-400'}>
            {message.text}
          </SmallText>
        )}

        <div className="flex flex-wrap gap-3">
          <Button
            variant="outline"
            disabled={stopping}
            onClick={onStop}
            leftIcon={<Activity className={iconSizes.md} />}
          >
            {stopping ? 'Stopping…' : 'Stop Simulation'}
          </Button>
          <Button
            variant="ghost"
            leftIcon={<FileCog className={iconSizes.md} />}
            onClick={() => navigate('/devices')}
          >
            View Devices
          </Button>
          <Button
            variant="ghost"
            leftIcon={<Download className={iconSizes.md} />}
            onClick={handleDownload}
            title="Download the running config as YAML (the equivalent of niac config dump)."
          >
            Download YAML
          </Button>
        </div>
      </CardContent>
    </Card>
  );
};
