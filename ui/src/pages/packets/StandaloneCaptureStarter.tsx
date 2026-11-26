import { Activity } from 'lucide-react';
import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { startStandaloneCapture } from '../../api/client';
import { iconSizes } from '../../constants/sizes';
import { useUsableInterfacesResource } from '../../hooks/usePageResources';
import { useResourceError } from '../../hooks/useResourceError';
import { useToast } from '../../hooks/useToast';
import { Card } from '../../shared/Card';
import { FormField } from '../../shared/FormField';
import { Input } from '../../shared/Input';
import { Select } from '../../shared/Select';
import { ActionButton } from '../../ui/ActionButton';
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
export function StandaloneCaptureStarter({
  onStarted,
  navigateToSim,
}: {
  onStarted: () => void;
  navigateToSim: () => void;
}) {
  const { t } = useTranslation('pages');
  // An empty picker and an unreachable daemon looked identical here.
  const { data: interfacesResp, error } = useUsableInterfacesResource();
  useResourceError(error, t('packets.interfacesFailed'));
  const interfaces = interfacesResp?.interfaces ?? [];
  const [selectedIface, setSelectedIface] = useState('');
  const [bpfFilter, setBpfFilter] = useState('');
  const [busy, setBusy] = useState(false);
  const { showError } = useToast();

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
    <Card>
      <div className="stack-lg">
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
          <FormField label={t('packets.inspector.interfaceLabel')}>
            {(control) => (
              <Select
                {...control}
                value={selectedIface}
                onValueChange={setSelectedIface}
                disabled={busy || interfaces.length === 0}
                options={
                  interfaces.length === 0
                    ? [{ value: '', label: t('packets.inspector.noInterfaces') }]
                    : interfaces.map((iface) => ({
                        value: iface.name,
                        label: iface.addresses?.[0]
                          ? `${iface.name} (${iface.addresses[0]})`
                          : iface.name,
                      }))
                }
              />
            )}
          </FormField>
          <FormField label={t('packets.inspector.standaloneBpfLabel')}>
            {(control) => (
              <Input
                {...control}
                value={bpfFilter}
                onChange={(e) => setBpfFilter(e.target.value)}
                disabled={busy}
                placeholder={t('packets.inspector.standaloneBpfPlaceholder')}
                className="font-mono"
              />
            )}
          </FormField>
          <ActionButton
            variant="primary"
            action="start"
            onClick={handleStart}
            disabled={!selectedIface || busy}
          >
            {busy
              ? t('packets.inspector.startingLabel')
              : t('packets.inspector.startCaptureButton')}
          </ActionButton>
        </div>
      </div>
    </Card>
  );
}
