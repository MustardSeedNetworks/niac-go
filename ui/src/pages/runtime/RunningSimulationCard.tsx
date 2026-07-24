import { Activity, Download, FileCog } from 'lucide-react';
import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router';
import { fetchConfig } from '../../api/client';
import type { SimulationStatus } from '../../api/types';
import { StatBlock } from '../../components/StatBlock';
import { iconSizes } from '../../constants/sizes';
import { useErrorToast } from '../../hooks/useErrorToast';
import { Button } from '../../ui/Button';
import { Card, CardContent } from '../../ui/Card';
import { Tag } from '../../ui/Tag';
import { H2, SmallText } from '../../ui/Typography';
import { formatTime, formatUptime } from '../../utils/format';

export interface RunningSimulationCardProps {
  simStatus: Pick<
    SimulationStatus,
    | 'interface'
    | 'configName'
    | 'configPath'
    | 'deviceCount'
    | 'uptimeSeconds'
    | 'startedAt'
    | 'fabric'
  >;
  stopping: boolean;
  onStop: () => void;
  /** Success-only status text; failures are surfaced as toasts. */
  message: string | null;
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
  const { t } = useTranslation('pages');
  const { t: tCommon } = useTranslation('common');
  const navigate = useNavigate();
  const showError = useErrorToast();

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
      showError(err);
    }
  };

  return (
    <Card className="border-status-success/30 bg-gradient-to-br from-status-success/20 to-bg-surface/70">
      <CardContent className="space-y-5">
        <div className="flex-between">
          <div className="flex items-center gap-default">
            <div className="h-3 w-3 animate-pulse rounded-full bg-status-success" />
            <H2>{t('runtime.running.title')}</H2>
          </div>
          <Tag colorScheme="green">{t('runtime.running.active')}</Tag>
        </div>

        <div className="grid gap-comfortable md:grid-cols-2 lg:grid-cols-3">
          <StatBlock
            label={tCommon('labels.interface')}
            value={simStatus.interface || '—'}
            helper={t('runtime.running.interfaceHelper')}
          />
          <StatBlock
            label={tCommon('labels.config')}
            value={simStatus.configName || '—'}
            helper={simStatus.configPath || t('runtime.running.configHelper')}
          />
          <StatBlock
            label={tCommon('labels.devices')}
            value={simStatus.deviceCount.toString()}
            helper={t('runtime.running.devicesHelper')}
          />
          <StatBlock
            label={t('runtime.running.uptimeLabel')}
            value={formatUptime(simStatus.uptimeSeconds)}
            helper={t('runtime.running.uptimeHelper')}
          />
          <StatBlock
            label={t('runtime.running.startedLabel')}
            value={simStatus.startedAt ? formatTime(simStatus.startedAt) : '—'}
            helper={t('runtime.running.startedHelper')}
          />
        </div>

        {simStatus.fabric && (
          <div className="rounded-lg border border-surface-border bg-bg-base/40 pad-default stack">
            <div className="flex flex-wrap gap-default text-sm text-text-secondary">
              <span>
                {t('runtime.fabric.attachment')}: {simStatus.fabric.topology.binding.attachment}
              </span>
              <span>
                {t('runtime.fabric.physicalVlan')}:{' '}
                {simStatus.fabric.topology.binding.physicalVlan ?? t('runtime.fabric.untagged')}
              </span>
              <span>
                {t('runtime.fabric.forwarded')}: {simStatus.fabric.forwarded}
              </span>
              <span>
                {t('runtime.fabric.drops')}: {simStatus.fabric.drops}
              </span>
              <span>
                {t('runtime.fabric.received')}: {simStatus.fabric.received}
              </span>
              <span>
                {t('runtime.fabric.transmitted')}: {simStatus.fabric.transmitted}
              </span>
            </div>
            <div className="grid gap-default lg:grid-cols-2">
              <div>
                <SmallText className="text-text-muted uppercase">
                  {t('runtime.fabric.virtualNetworks')}
                </SmallText>
                <ul className="mt-tight text-sm text-text-primary">
                  {simStatus.fabric.topology.networks.map((network) => (
                    <li key={network.name}>
                      {network.name} · {network.prefix} · {t('runtime.fabric.virtualVlan')}:{' '}
                      {network.virtualVlan ?? t('runtime.fabric.none')}
                    </li>
                  ))}
                </ul>
              </div>
              <div>
                <SmallText className="text-text-muted uppercase">
                  {t('runtime.fabric.routes')}
                </SmallText>
                <ul className="mt-tight text-sm text-text-primary">
                  {simStatus.fabric.topology.routes.map((route) => (
                    <li key={`${route.device}-${route.destination}-${route.via}`}>
                      {route.device}: {route.destination} → {route.via}
                    </li>
                  ))}
                </ul>
              </div>
            </div>
          </div>
        )}

        {message && <SmallText className="text-status-success">{message}</SmallText>}

        <div className="flex flex-wrap gap-default">
          <Button
            variant="outline"
            disabled={stopping}
            onClick={onStop}
            leftIcon={<Activity className={iconSizes.md} />}
          >
            {stopping ? t('runtime.running.stoppingLabel') : t('runtime.running.stopButton')}
          </Button>
          <Button
            variant="ghost"
            leftIcon={<FileCog className={iconSizes.md} />}
            onClick={() => navigate('/devices')}
          >
            {t('runtime.running.viewDevices')}
          </Button>
          <Button
            variant="ghost"
            leftIcon={<Download className={iconSizes.md} />}
            onClick={handleDownload}
            title={t('runtime.running.downloadYamlTitle')}
          >
            {t('runtime.running.downloadYaml')}
          </Button>
        </div>
      </CardContent>
    </Card>
  );
};
