import { RefreshCw, RotateCcw, Save, Settings2 } from 'lucide-react';
import { type ChangeEvent, type FC, memo, useCallback } from 'react';
import type {
  DebugLevel,
  DebugProtocol,
  ProtocolCategory,
  ProtocolDebugConfig,
} from '../../api/types';
import { iconSizes } from '../../constants/sizes';
import {
  CATEGORY_ORDER,
  DEBUG_LEVELS,
  PROTOCOL_CATEGORIES,
  useProtocolDebugLevels,
} from '../../hooks/useProtocolDebugLevels';
import { Button } from '../../ui/Button';
import { Card, CardContent } from '../../ui/Card';
import { Tag } from '../../ui/Tag';
import { H2, SmallText } from '../../ui/Typography';

type DebugLevelKey = Lowercase<DebugLevel>;

// Colors for different debug levels
const LEVEL_COLORS: Record<DebugLevelKey, string> = {
  off: 'bg-bg-elevated',
  error: 'bg-status-error',
  warn: 'bg-status-warning',
  info: 'bg-status-info',
  debug: 'bg-status-success',
  trace: 'bg-brand-primary',
};

// Category colors for visual grouping
const CATEGORY_COLORS: Record<ProtocolCategory, string> = {
  discovery: 'border-status-info/30',
  switching: 'border-status-success/30',
  routing: 'border-status-info/30',
  redundancy: 'border-status-warning/30',
  multicast: 'border-brand-primary/30',
  monitoring: 'border-pink-500/30',
};

const CATEGORY_HEADER_COLORS: Record<ProtocolCategory, string> = {
  discovery: 'text-status-info',
  switching: 'text-status-success',
  routing: 'text-status-info',
  redundancy: 'text-status-warning',
  multicast: 'text-brand-accent',
  monitoring: 'text-pink-400',
};

/**
 * Protocol Debug Level Slider Component
 */
interface ProtocolSliderProps {
  config: ProtocolDebugConfig;
  onChange: (protocol: DebugProtocol, level: DebugLevel) => void;
  disabled?: boolean;
}

const ProtocolSlider: FC<ProtocolSliderProps> = memo(({ config, onChange, disabled }) => {
  const levelIndex = DEBUG_LEVELS.indexOf(config.level);

  const handleChange = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => {
      const newIndex = Number.parseInt(e.target.value, 10);
      const newLevel = DEBUG_LEVELS[newIndex];
      onChange(config.protocol, newLevel);
    },
    [config.protocol, onChange],
  );

  return (
    <div className="rounded-lg border border-surface-border bg-bg-base/50 p-4">
      <div className="flex items-center justify-between mb-3">
        <span className="font-semibold text-text-primary">{config.protocol}</span>
        <Tag
          colorScheme={
            config.level === 'OFF'
              ? 'gray'
              : config.level === 'ERROR'
                ? 'red'
                : config.level === 'WARN'
                  ? 'yellow'
                  : config.level === 'DEBUG' || config.level === 'TRACE'
                    ? 'green'
                    : 'blue'
          }
        >
          {config.level}
        </Tag>
      </div>

      <div className="relative">
        {/* Slider track background with gradient */}
        <div className="absolute top-1/2 left-0 right-0 h-2 -translate-y-1/2 rounded-full bg-gradient-to-r from-gray-700 via-yellow-600 to-purple-600 opacity-30" />

        {/* Active track */}
        <div
          className={`absolute top-1/2 left-0 h-2 -translate-y-1/2 rounded-full transition-all ${
            LEVEL_COLORS[config.level.toLowerCase() as DebugLevelKey]
          }`}
          style={{
            width: `${(levelIndex / (DEBUG_LEVELS.length - 1)) * 100}%`,
          }}
        />

        {/* Range input */}
        <input
          type="range"
          min="0"
          max={DEBUG_LEVELS.length - 1}
          value={levelIndex}
          onChange={handleChange}
          disabled={disabled}
          className="relative z-10 w-full h-6 appearance-none bg-transparent cursor-pointer disabled:cursor-not-allowed disabled:opacity-50
            [&::-webkit-slider-thumb]:appearance-none
            [&::-webkit-slider-thumb]:h-5
            [&::-webkit-slider-thumb]:w-5
            [&::-webkit-slider-thumb]:rounded-full
            [&::-webkit-slider-thumb]:bg-white
            [&::-webkit-slider-thumb]:shadow-md
            [&::-webkit-slider-thumb]:border-2
            [&::-webkit-slider-thumb]:border-brand-accent
            [&::-webkit-slider-thumb]:transition-transform
            [&::-webkit-slider-thumb]:hover:scale-110
            [&::-moz-range-thumb]:h-5
            [&::-moz-range-thumb]:w-5
            [&::-moz-range-thumb]:rounded-full
            [&::-moz-range-thumb]:bg-white
            [&::-moz-range-thumb]:shadow-md
            [&::-moz-range-thumb]:border-2
            [&::-moz-range-thumb]:border-brand-accent"
          aria-label={`${config.protocol} debug level`}
          aria-valuetext={config.level}
        />
      </div>

      {/* Level labels */}
      <div className="flex justify-between mt-1 text-xs text-text-muted">
        {DEBUG_LEVELS.map((level) => (
          <span
            key={level}
            className={`${levelIndex === DEBUG_LEVELS.indexOf(level) ? 'text-text-primary font-medium' : ''}`}
          >
            {level.substring(0, 3)}
          </span>
        ))}
      </div>
    </div>
  );
});

ProtocolSlider.displayName = 'ProtocolSlider';

/**
 * Protocol Category Group Component
 */
interface CategoryGroupProps {
  category: ProtocolCategory;
  protocols: ProtocolDebugConfig[];
  onLevelChange: (protocol: DebugProtocol, level: DebugLevel) => void;
  disabled?: boolean;
}

const CategoryGroup: FC<CategoryGroupProps> = memo(
  ({ category, protocols, onLevelChange, disabled }) => {
    if (protocols.length === 0) {
      return null;
    }

    return (
      <div className={`rounded-xl border ${CATEGORY_COLORS[category]} bg-bg-surface/50 p-4`}>
        <h3
          className={`text-sm font-semibold uppercase tracking-wide mb-4 ${CATEGORY_HEADER_COLORS[category]}`}
        >
          {PROTOCOL_CATEGORIES[category]}
        </h3>
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {protocols.map((config) => (
            <ProtocolSlider
              key={config.protocol}
              config={config}
              onChange={onLevelChange}
              disabled={disabled}
            />
          ))}
        </div>
      </div>
    );
  },
);

CategoryGroup.displayName = 'CategoryGroup';

/**
 * Main Protocol Debug Levels Component
 *
 * Displays sliders for all 19 protocols grouped by category.
 * Supports saving changes to the API and resetting to defaults.
 */
export const ProtocolDebugLevels: FC = () => {
  const {
    loading,
    saving,
    error,
    successMessage,
    hasChanges,
    updateLevel,
    saveChanges,
    resetToDefaults,
    discardChanges,
    getProtocolsByCategory,
  } = useProtocolDebugLevels();

  const protocolsByCategory = getProtocolsByCategory();

  const handleReset = useCallback(async () => {
    // TODO: Replace with custom Modal component
    if (window.confirm('Reset all protocols to default debug levels? This cannot be undone.')) {
      await resetToDefaults();
    }
  }, [resetToDefaults]);

  const handleDiscard = useCallback(() => {
    // TODO: Replace with custom Modal component
    if (window.confirm('Discard unsaved changes?')) {
      discardChanges();
    }
  }, [discardChanges]);

  if (loading) {
    return (
      <Card className="border-surface-border bg-bg-surface/70">
        <CardContent className="py-12 text-center">
          <RefreshCw className={`mx-auto ${iconSizes['2xl']} animate-spin text-text-muted`} />
          <SmallText className="mt-3 text-text-muted">Loading protocol debug settings...</SmallText>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="border-surface-border bg-bg-surface/70">
      <CardContent className="space-y-6">
        {/* Header */}
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <Settings2 className={`${iconSizes.xl} text-brand-accent`} />
            <H2>Protocol Debug Levels</H2>
          </div>

          {/* Action Buttons */}
          <div className="flex flex-wrap items-center gap-2">
            {hasChanges && (
              <>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={handleDiscard}
                  disabled={saving}
                  aria-label="Discard changes"
                >
                  Discard
                </Button>
                <Button
                  tone="violet"
                  size="sm"
                  onClick={saveChanges}
                  disabled={saving}
                  leftIcon={<Save className={iconSizes.md} />}
                  aria-label="Save changes"
                >
                  {saving ? 'Saving...' : 'Save Changes'}
                </Button>
              </>
            )}
            <Button
              variant="outline"
              size="sm"
              onClick={handleReset}
              disabled={saving}
              leftIcon={<RotateCcw className={iconSizes.md} />}
              aria-label="Reset all protocols to defaults"
            >
              Reset All
            </Button>
          </div>
        </div>

        {/* Status Messages */}
        {error && (
          <div
            className="rounded-lg border border-status-error/30 bg-status-error/20 px-4 py-3 text-sm text-status-error"
            role="alert"
          >
            {error}
          </div>
        )}
        {successMessage && (
          <output className="rounded-lg border border-status-success/30 bg-status-success/20 px-4 py-3 text-sm text-status-success">
            {successMessage}
          </output>
        )}
        {hasChanges && !error && !successMessage && (
          <output className="rounded-lg border border-status-warning/30 bg-status-warning/20 px-4 py-3 text-sm text-status-warning">
            You have unsaved changes
          </output>
        )}

        {/* Description */}
        <SmallText className="text-text-muted">
          Adjust debug output verbosity for each network protocol. Higher levels (DEBUG, TRACE)
          produce more detailed logs but may impact performance. Changes take effect immediately
          when saved.
        </SmallText>

        {/* Protocol Groups */}
        <div className="space-y-4">
          {CATEGORY_ORDER.map((category) => {
            const protocols = protocolsByCategory.get(category) || [];
            return (
              <CategoryGroup
                key={category}
                category={category}
                protocols={protocols}
                onLevelChange={updateLevel}
                disabled={saving}
              />
            );
          })}
        </div>

        {/* Legend */}
        <div className="rounded-lg border border-surface-border bg-bg-base/50 p-4">
          <SmallText className="font-semibold uppercase tracking-wide text-text-muted mb-3">
            Level Guide
          </SmallText>
          <div className="flex flex-wrap gap-4 text-sm">
            <div className="flex items-center gap-2">
              <span className={`h-3 w-3 rounded-full ${LEVEL_COLORS.off}`} />
              <span className="text-text-secondary">
                <strong>OFF</strong> - No output
              </span>
            </div>
            <div className="flex items-center gap-2">
              <span className={`h-3 w-3 rounded-full ${LEVEL_COLORS.error}`} />
              <span className="text-text-secondary">
                <strong>ERROR</strong> - Critical issues only
              </span>
            </div>
            <div className="flex items-center gap-2">
              <span className={`h-3 w-3 rounded-full ${LEVEL_COLORS.warn}`} />
              <span className="text-text-secondary">
                <strong>WARN</strong> - Warnings and errors
              </span>
            </div>
            <div className="flex items-center gap-2">
              <span className={`h-3 w-3 rounded-full ${LEVEL_COLORS.info}`} />
              <span className="text-text-secondary">
                <strong>INFO</strong> - General information
              </span>
            </div>
            <div className="flex items-center gap-2">
              <span className={`h-3 w-3 rounded-full ${LEVEL_COLORS.debug}`} />
              <span className="text-text-secondary">
                <strong>DEBUG</strong> - Detailed debugging
              </span>
            </div>
            <div className="flex items-center gap-2">
              <span className={`h-3 w-3 rounded-full ${LEVEL_COLORS.trace}`} />
              <span className="text-text-secondary">
                <strong>TRACE</strong> - Full packet traces
              </span>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
};

export default ProtocolDebugLevels;
