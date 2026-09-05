import { Activity } from 'lucide-react';
import { type FC, useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { fetchUsableInterfaces, startStandaloneCapture } from '../../api/client';
import { iconSizes } from '../../constants/sizes';
import { useApiResource } from '../../hooks/useApiResource';
import { useErrorToast } from '../../hooks/useErrorToast';
import { Button } from '../../ui/Button';
import { Card, CardContent } from '../../ui/Card';
import { SmallText } from '../../ui/Typography';

/**
 * StandaloneCaptureStarter is the empty-state UI for PCAP Inspector
 * when neither a simulation nor a standalone capture session is
 * running. It lets the user start a sniff-only capture on an
 * interface of their choice, with an optional BPF filter.
 *
 * Once StartStandaloneCapture succeeds, the parent's polling picks up
 * captureStatus.running and re-renders into the regular packet stream
 * UI on the next tick. Calling onStarted forces an immediate refetch
 * so the transition doesn't wait on the poll interval.
 */
export const StandaloneCaptureStarter: FC<{
  onStarted: () => void;
  navigateToSim: () => void;
}> = ({ onStarted, navigateToSim }) => {
  const { t } = useTranslation('pages');
  // An empty picker and an unreachable daemon looked identical here.
  const { data: interfacesResp } = useApiResource(fetchUsableInterfaces, [], {
    errorToast: { title: t('packets.interfacesFailed') },
  });
  const interfaces = interfacesResp?.interfaces ?? [];
  const [selectedIface, setSelectedIface] = useState('');
  const [bpfFilter, setBpfFilter] = useState('');
  const [busy, setBusy] = useState(false);
  const showError = useErrorToast();

  // Default to the first usable interface once the list arrives.
  useEffect(() => {
    if (!selectedIface && interfaces.length > 0) {
      setSelectedIface(interfaces[0]?.name ?? '');
    }
  }, [interfaces, selectedIface]);

  const handleStart = useCallback(async () => {
    if (!selectedIface || busy) {
      return;
    }
    setBusy(true);
    try {
      await startStandaloneCapture({
        interface: selectedIface,
        filter: bpfFilter.trim() || undefined,
      });
      onStarted();
    } catch (err) {
      showError(err);
      setBusy(false);
    }
  }, [bpfFilter, busy, onStarted, selectedIface, showError]);

  return (
    <Card className="border-surface-border bg-bg-surface/70">
      <CardContent className="stack-lg">
        <div className="flex items-start gap-default">
          <Activity className={`mt-tight ${iconSizes.lg} text-brand-accent`} />
          <div className="flex-1">
            <p className="font-semibold text-text-primary">
              {t('packets.inspector.standaloneCapture')}
            </p>
            <SmallText className="text-text-muted">
              {t('packets.inspector.standaloneDescriptionPart1')}{' '}
              <button
                type="button"
                onClick={navigateToSim}
                className="text-brand-accent underline hover:text-brand-accent"
              >
                {t('packets.inspector.standaloneStartSimLink')}
              </button>{' '}
              {t('packets.inspector.standaloneDescriptionPart2')}
            </SmallText>
          </div>
        </div>
        <div className="grid gap-default sm:grid-cols-[1fr_2fr_auto] sm:items-end">
          <label className="flex flex-col gap-tight text-sm text-text-muted">
            {t('packets.inspector.interfaceLabel')}
            <select
              value={selectedIface}
              onChange={(e) => setSelectedIface(e.target.value)}
              disabled={busy || interfaces.length === 0}
              className="rounded-lg border border-surface-border bg-bg-base/60 px-3 py-row text-sm text-text-primary focus:border-brand-accent focus:outline-none"
            >
              {interfaces.length === 0 ? (
                <option value="">{t('packets.inspector.noInterfaces')}</option>
              ) : (
                interfaces.map((iface) => (
                  <option key={iface.name} value={iface.name}>
                    {iface.name}
                    {iface.addresses && iface.addresses.length > 0
                      ? ` (${iface.addresses[0]})`
                      : ''}
                  </option>
                ))
              )}
            </select>
          </label>
          <label className="flex flex-col gap-tight text-sm text-text-muted">
            {t('packets.inspector.standaloneBpfLabel')}
            <input
              type="text"
              value={bpfFilter}
              onChange={(e) => setBpfFilter(e.target.value)}
              disabled={busy}
              placeholder={t('packets.inspector.standaloneBpfPlaceholder')}
              className="rounded-lg border border-surface-border bg-bg-base/60 px-3 py-row font-mono text-sm text-text-primary focus:border-brand-accent focus:outline-none"
            />
          </label>
          <Button tone="violet" onClick={handleStart} disabled={!selectedIface || busy}>
            {busy
              ? t('packets.inspector.startingLabel')
              : t('packets.inspector.startCaptureButton')}
          </Button>
        </div>
      </CardContent>
    </Card>
  );
};
