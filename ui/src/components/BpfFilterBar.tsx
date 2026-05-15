import { AlertCircle, CheckCircle2, Filter, X } from 'lucide-react';
import { type FC, memo, useCallback, useEffect, useState } from 'react';
import { clearCaptureFilter, getCaptureFilter, setCaptureFilter } from '../api/capture';
import { iconSizes } from '../constants/sizes';
import { Button } from '../ui/Button';
import { SmallText } from '../ui/Typography';
import { getErrorMessage } from '../utils/format';

/** Common BPF filter presets. */
const BPF_PRESETS = [
  { label: 'TCP', filter: 'tcp' },
  { label: 'UDP', filter: 'udp' },
  { label: 'Port 80', filter: 'port 80' },
  { label: 'Port 443', filter: 'port 443' },
  { label: 'Not ARP', filter: 'not arp' },
  { label: 'ICMP', filter: 'icmp' },
] as const;

/**
 * BPF Capture Filter Bar
 *
 * Allows setting a BPF filter on the live capture engine.
 * Shows the active filter and provides presets for common filters.
 */
export const BpfFilterBar: FC = memo(() => {
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
      setError(getErrorMessage(err) || 'Invalid BPF filter expression');
    } finally {
      setIsLoading(false);
    }
  }, [input]);

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
      setError(getErrorMessage(err) || 'Failed to clear filter');
    } finally {
      setIsLoading(false);
    }
  }, []);

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
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        {/* Filter icon and active indicator */}
        <div className="flex items-center gap-1.5 flex-shrink-0">
          <Filter className={`${iconSizes.md} text-gray-400`} />
          {isActive && <CheckCircle2 className="h-3.5 w-3.5 text-green-400" />}
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
          placeholder="BPF filter (e.g. tcp port 80)"
          className={`flex-1 bg-gray-950/70 border rounded-lg px-3 py-1.5 text-sm font-mono text-white placeholder-gray-500 focus:outline-none focus:ring-1 ${
            error
              ? 'border-red-500/60 focus:ring-red-500/40'
              : isActive
                ? 'border-green-500/40 focus:ring-green-500/40'
                : 'border-white/10 focus:ring-violet-500/40'
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
          Apply
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
            Clear
          </Button>
        )}
      </div>

      {/* Error message */}
      {error && (
        <div className="flex items-center gap-1.5 px-1">
          <AlertCircle className={`${iconSizes.sm} text-red-400 flex-shrink-0`} />
          <SmallText className="text-red-400">{error}</SmallText>
        </div>
      )}

      {/* Active filter indicator */}
      {isActive && activeFilter && (
        <div className="flex items-center gap-1.5 px-1">
          <SmallText className="text-green-400">
            Active: <span className="font-mono">{activeFilter}</span>
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
            className="px-2 py-0.5 text-xs rounded bg-gray-800/60 text-gray-400 hover:text-white hover:bg-gray-700/60 border border-white/5 transition-colors"
          >
            {preset.label}
          </button>
        ))}
      </div>
    </div>
  );
});

BpfFilterBar.displayName = 'BpfFilterBar';
