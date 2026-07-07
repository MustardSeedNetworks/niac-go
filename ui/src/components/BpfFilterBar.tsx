import { AlertCircle, CheckCircle2, Filter, X } from 'lucide-react';
import { type FC, memo, useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { clearCaptureFilter, getCaptureFilter, setCaptureFilter } from '../api/capture';
import { iconSizes } from '../constants/sizes';
import { Button } from '../ui/Button';
import { InfoPopover } from '../ui/InfoPopover';
import { SmallText } from '../ui/Typography';
import { getErrorMessage } from '../utils/format';

/** Common BPF filter presets. */
const BPF_PRESETS = [
  { labelKey: 'packets.bpfFilter.presetTcp', filter: 'tcp' },
  { labelKey: 'packets.bpfFilter.presetUdp', filter: 'udp' },
  { labelKey: 'packets.bpfFilter.presetPort80', filter: 'port 80' },
  { labelKey: 'packets.bpfFilter.presetPort443', filter: 'port 443' },
  { labelKey: 'packets.bpfFilter.presetNotArp', filter: 'not arp' },
  { labelKey: 'packets.bpfFilter.presetIcmp', filter: 'icmp' },
] as const;

/**
 * BPF Capture Filter Bar
 *
 * Allows setting a BPF filter on the live capture engine.
 * Shows the active filter and provides presets for common filters.
 */
export const BpfFilterBar: FC = memo(() => {
  const { t } = useTranslation('common');
  const { t: tHelp } = useTranslation('help');
  const { t: tPages } = useTranslation('pages');
  const [input, setInput] = useState('');
  const [activeFilter, setActiveFilter] = useState('');
  const [isActive, setIsActive] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  // Fetch current filter state on mount
  useEffect(() => {
    let cancelled = false;

    async function fetchState() {
      try {
        const state = await getCaptureFilter();
        if (!cancelled) {
          setActiveFilter(state.filter);
          setIsActive(state.active);
          if (state.filter) {
            setInput(state.filter);
          }
        }
      } catch {
        // Ignore fetch errors on mount (server may not be running)
      }
    }

    fetchState();

    return () => {
      cancelled = true;
    };
  }, []);

  // Apply the filter
  const handleApply = useCallback(async () => {
    const trimmed = input.trim();
    if (!trimmed) return;

    setIsLoading(true);
    setError(null);

    try {
      const result = await setCaptureFilter(trimmed);
      setActiveFilter(result.filter);
      setIsActive(result.active);
    } catch (err) {
      setError(getErrorMessage(err) || tPages('packets.bpfFilter.invalidFilterError'));
    } finally {
      setIsLoading(false);
    }
  }, [input, tPages]);

  // Clear the filter
  const handleClear = useCallback(async () => {
    setIsLoading(true);
    setError(null);

    try {
      await clearCaptureFilter();
      setActiveFilter('');
      setIsActive(false);
      setInput('');
    } catch (err) {
      setError(getErrorMessage(err) || tPages('packets.bpfFilter.clearFilterError'));
    } finally {
      setIsLoading(false);
    }
  }, [tPages]);

  // Apply preset
  const handlePreset = useCallback((filter: string) => {
    setInput(filter);
    setError(null);
  }, []);

  // Submit on Enter
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        handleApply();
      }
    },
    [handleApply],
  );

  return (
    <div className="stack-sm">
      <SmallText className="text-text-muted px-1">{t('filters.captureFilterCaption')}</SmallText>
      <div className="flex items-center gap-compact">
        {/* Filter icon and active indicator */}
        <div className="flex items-center gap-1.5 flex-shrink-0">
          <Filter className={`${iconSizes.md} text-text-muted`} />
          {isActive && <CheckCircle2 className="h-3.5 w-3.5 text-status-success" />}
          <InfoPopover label={t('jargon.ariaLabel', { term: 'BPF' })} title="BPF">
            {tHelp('jargon.bpf')}
          </InfoPopover>
        </div>

        {/* Input */}
        <input
          type="text"
          value={input}
          onChange={(e) => {
            setInput(e.target.value);
            setError(null);
          }}
          onKeyDown={handleKeyDown}
          placeholder={tPages('packets.bpfFilter.inputPlaceholder')}
          className={`flex-1 bg-bg-base/70 border rounded-lg px-3 py-compact-md text-sm font-mono text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-1 ${
            error
              ? 'border-status-error/60 focus:ring-status-error/40'
              : isActive
                ? 'border-status-success/40 focus:ring-status-success/40'
                : 'border-surface-border focus:ring-brand-primary/40'
          }`}
          disabled={isLoading}
        />

        {/* Apply button */}
        <Button
          variant="ghost"
          size="sm"
          onClick={handleApply}
          disabled={isLoading || !input.trim()}
        >
          {t('buttons.apply')}
        </Button>

        {/* Clear button */}
        {isActive && (
          <Button
            variant="ghost"
            size="sm"
            onClick={handleClear}
            disabled={isLoading}
            leftIcon={<X className="h-3.5 w-3.5" />}
          >
            {t('buttons.clear')}
          </Button>
        )}
      </div>

      {/* Error message */}
      {error && (
        <div className="flex items-center gap-1.5 px-1">
          <AlertCircle className={`${iconSizes.sm} text-status-error flex-shrink-0`} />
          <SmallText className="text-status-error">{error}</SmallText>
        </div>
      )}

      {/* Active filter indicator */}
      {isActive && activeFilter && (
        <div className="flex items-center gap-1.5 px-1">
          <SmallText className="text-status-success">
            {tPages('packets.bpfFilter.activeLabel')}{' '}
            <span className="font-mono">{activeFilter}</span>
          </SmallText>
        </div>
      )}

      {/* Presets */}
      <div className="flex flex-wrap gap-1.5 px-1">
        {BPF_PRESETS.map((preset) => (
          <button
            key={preset.filter}
            type="button"
            onClick={() => handlePreset(preset.filter)}
            className="px-cell py-0.5 text-xs rounded bg-bg-elevated/60 text-text-muted hover:text-text-primary hover:bg-bg-elevated/60 border border-surface-border transition-colors"
          >
            {tPages(preset.labelKey)}
          </button>
        ))}
      </div>
    </div>
  );
});

BpfFilterBar.displayName = 'BpfFilterBar';
