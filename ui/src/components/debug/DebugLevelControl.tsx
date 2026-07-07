import { Activity } from 'lucide-react';
import { type FC, useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { fetchDebugLevel, updateDebugLevel } from '../../api/client';
import type { DebugLevel } from '../../api/types';
import { iconSizes } from '../../constants/sizes';
import { H2, SmallText } from '../../ui/Typography';
import { getErrorMessage } from '../../utils/format';

interface LevelOption {
  value: DebugLevel;
  label: string;
  hint: string;
}

// Ordered least→most verbose. These five map 1:1 onto the backend ladder
// (0..4); there is no per-protocol surface — NIAC's handlers read one global
// level live, so a change takes effect with no restart.
function useLevels(): LevelOption[] {
  const { t } = useTranslation('pages');
  return useMemo(
    () => [
      { value: 'off', label: t('debug.levelOff'), hint: t('debug.levelOffHint') },
      { value: 'basic', label: t('debug.levelBasic'), hint: t('debug.levelBasicHint') },
      { value: 'info', label: t('debug.levelInfo'), hint: t('debug.levelInfoHint') },
      { value: 'verbose', label: t('debug.levelVerbose'), hint: t('debug.levelVerboseHint') },
      { value: 'trace', label: t('debug.levelTrace'), hint: t('debug.levelTraceHint') },
    ],
    [t],
  );
}

/**
 * DebugLevelControl — the single global debug-verbosity control. Reads
 * GET /api/v1/debug/level and writes PUT /api/v1/debug/level; the running
 * stack honors the new level live.
 */
export const DebugLevelControl: FC = () => {
  const { t } = useTranslation('pages');
  const LEVELS = useLevels();
  const labelFor = useCallback(
    (value: DebugLevel): string => LEVELS.find((o) => o.value === value)?.label ?? value,
    [LEVELS],
  );
  const [level, setLevel] = useState<DebugLevel | null>(null);
  const [defaultLevel, setDefaultLevel] = useState<DebugLevel>('basic');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    fetchDebugLevel()
      .then((res) => {
        if (cancelled) {
          return;
        }
        setLevel(res.level);
        setDefaultLevel(res.defaultLevel);
        setError(null);
      })
      .catch((err) => {
        if (!cancelled) {
          setError(getErrorMessage(err));
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, []);

  const apply = useCallback(
    async (next: DebugLevel) => {
      if (next === level || saving) {
        return;
      }

      const previous = level;
      setSaving(true);
      setError(null);
      setLevel(next); // optimistic

      try {
        const res = await updateDebugLevel({ level: next });
        setLevel(res.level);
        setDefaultLevel(res.defaultLevel);
      } catch (err) {
        setLevel(previous); // rollback
        setError(getErrorMessage(err));
      } finally {
        setSaving(false);
      }
    },
    [level, saving],
  );

  const disabled = loading || saving || level === null;

  return (
    <div className="stack" data-testid="debug-level-control">
      <H2 className="flex items-center gap-compact text-lg">
        <Activity className={`${iconSizes.lg} text-brand-accent`} />
        {t('debug.debugLevelTitle')}
      </H2>
      <SmallText className="text-text-muted">{t('debug.debugLevelDescription')}</SmallText>
      <fieldset className="border-0 p-0">
        <legend className="sr-only">{t('debug.debugLevelTitle')}</legend>
        <div className="flex flex-wrap gap-compact">
          {LEVELS.map((opt) => {
            const active = level === opt.value;

            return (
              <label key={opt.value} title={opt.hint} className="cursor-pointer">
                <input
                  type="radio"
                  name="debug-level"
                  value={opt.value}
                  checked={active}
                  disabled={disabled}
                  onChange={() => void apply(opt.value)}
                  data-testid={`debug-level-${opt.value}`}
                  className="peer sr-only"
                />
                <span
                  className={`block rounded border px-3 py-compact-md text-sm transition-colors peer-focus-visible:ring-2 peer-focus-visible:ring-brand-accent ${
                    active
                      ? 'border-brand-accent bg-brand-accent/15 text-text-primary'
                      : 'border-surface-border bg-bg-base/60 text-text-muted hover:text-text-primary'
                  } ${disabled ? 'opacity-50' : ''}`}
                >
                  {opt.label}
                </span>
              </label>
            );
          })}
        </div>
      </fieldset>
      <SmallText className="text-text-muted">
        {t('debug.debugLevelDefault', { level: labelFor(defaultLevel) })}
      </SmallText>
      {error && (
        <SmallText className="text-status-error" role="alert">
          {error}
        </SmallText>
      )}
    </div>
  );
};
