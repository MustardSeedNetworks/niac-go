import { Activity } from 'lucide-react';
import { type FC, useState } from 'react';
import { fetchProtocolDebugLevels, updateProtocolDebugLevels } from '../../api/client';
import type { DebugLevel } from '../../api/types';
import { iconSizes } from '../../constants/sizes';
import { useApiResource } from '../../hooks/useApiResource';
import { Button } from '../../ui/Button';
import { Card, CardContent } from '../../ui/Card';
import { H2, SmallText } from '../../ui/Typography';

const DEBUG_LEVELS: DebugLevel[] = ['OFF', 'ERROR', 'WARN', 'INFO', 'DEBUG', 'TRACE'];

/**
 * GlobalDebugLevelCard — applies the same DebugLevel to every protocol
 * in the running stack via PUT /api/v1/debug/levels. CLI parity for
 * the --debug / --verbose / --quiet family of flags. The Protocol
 * Debug page is the canonical place for per-protocol fine-tuning;
 * this is the 90% case for users who just want "loud" or "quiet"
 * globally.
 */
export const GlobalDebugLevelCard: FC = () => {
  const { data, refetch } = useApiResource(fetchProtocolDebugLevels, [], {
    intervalMs: 0,
  });
  const [pending, setPending] = useState<DebugLevel | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const current: DebugLevel = pending ?? data?.defaultLevel ?? 'INFO';

  const apply = async (level: DebugLevel) => {
    if (!data) return;
    setBusy(true);
    setError(null);
    try {
      await updateProtocolDebugLevels({
        protocols: data.protocols.map((p) => ({ protocol: p.protocol, level })),
      });
      setPending(null);
      await refetch();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <Card className="border-surface-border bg-bg-surface/70">
      <CardContent className="space-y-3">
        <H2 className="flex items-center gap-2 text-lg">
          <Activity className={`${iconSizes.lg} text-brand-accent`} />
          Debug level
        </H2>
        <SmallText className="text-text-muted">
          Sets every protocol to the same level. Use Protocol Debug for per-protocol tuning.
        </SmallText>
        <div className="flex flex-wrap items-center gap-3">
          <select
            value={current}
            onChange={(e) => setPending(e.target.value as DebugLevel)}
            disabled={busy || !data}
            className="rounded border border-surface-border bg-bg-base/60 px-3 py-1.5 text-sm text-text-primary focus:border-brand-accent focus:outline-none disabled:opacity-50"
            aria-label="Global debug level"
            title="Applies to every protocol in the running stack. OFF silences everything; TRACE is the loudest."
          >
            {DEBUG_LEVELS.map((lvl) => (
              <option key={lvl} value={lvl}>
                {lvl}
              </option>
            ))}
          </select>
          <Button
            tone="violet"
            disabled={busy || pending === null || !data}
            onClick={() => void apply(current)}
            title="PUT /api/v1/debug/levels with every protocol set to the chosen level."
          >
            {busy ? 'Applying…' : 'Apply'}
          </Button>
          {error && (
            <SmallText className="text-status-error" role="alert">
              {error}
            </SmallText>
          )}
        </div>
      </CardContent>
    </Card>
  );
};
