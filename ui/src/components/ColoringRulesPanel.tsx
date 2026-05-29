import { ArrowDown, ArrowUp, Plus, RotateCcw, Trash2, X } from 'lucide-react';
import { type FC, memo, useCallback, useState } from 'react';
import { iconSizes } from '../constants/sizes';
import { Button } from '../ui/Button';
import { SmallText } from '../ui/Typography';
import type { ColoringRule } from '../utils/coloring-rules';
import { generateRuleId } from '../utils/coloring-rules';
import { validate } from '../utils/filter/parser';

interface ColoringRulesPanelProps {
  rules: ColoringRule[];
  onRulesChange: (rules: ColoringRule[]) => void;
  onReset: () => void;
  onClose: () => void;
}

/**
 * Rule editor row
 */
const RuleRow: FC<{
  rule: ColoringRule;
  onChange: (rule: ColoringRule) => void;
  onDelete: () => void;
  onMoveUp: () => void;
  onMoveDown: () => void;
  isFirst: boolean;
  isLast: boolean;
}> = memo(({ rule, onChange, onDelete, onMoveUp, onMoveDown, isFirst, isLast }) => {
  const filterError = rule.filter ? validate(rule.filter) : null;

  return (
    <div className="flex items-center gap-compact py-row px-cell rounded-lg bg-bg-base/40 border border-surface-border">
      {/* Enable toggle */}
      <input
        type="checkbox"
        checked={rule.enabled}
        onChange={(e) => onChange({ ...rule, enabled: e.target.checked })}
        className="rounded border-border-muted bg-bg-elevated text-brand-primary focus:ring-brand-accent focus:ring-offset-gray-900 flex-shrink-0"
      />

      {/* Color preview */}
      <div
        className="w-6 h-6 rounded border border-surface-border flex-shrink-0"
        style={{ backgroundColor: rule.background, color: rule.foreground }}
      >
        <span className="text-xs font-bold flex-center h-full">A</span>
      </div>

      {/* Name */}
      <input
        type="text"
        value={rule.name}
        onChange={(e) => onChange({ ...rule, name: e.target.value })}
        className="w-24 bg-transparent border-b border-surface-border text-sm text-text-primary focus:outline-none focus:border-brand-accent px-1"
        placeholder="Name"
      />

      {/* Filter expression */}
      <input
        type="text"
        value={rule.filter}
        onChange={(e) => onChange({ ...rule, filter: e.target.value })}
        className={`flex-1 bg-transparent border-b text-sm font-mono text-text-primary focus:outline-none px-1 ${
          filterError
            ? 'border-status-error/60 focus:border-status-error'
            : 'border-surface-border focus:border-brand-accent'
        }`}
        placeholder="Filter expression"
      />

      {/* Foreground color */}
      <input
        type="color"
        value={rule.foreground}
        onChange={(e) => onChange({ ...rule, foreground: e.target.value })}
        className="w-6 h-6 rounded cursor-pointer border-0 bg-transparent flex-shrink-0"
        title="Text color"
      />

      {/* Background color */}
      <input
        type="color"
        value={rule.background}
        onChange={(e) => onChange({ ...rule, background: e.target.value })}
        className="w-6 h-6 rounded cursor-pointer border-0 bg-transparent flex-shrink-0"
        title="Background color"
      />

      {/* Move buttons */}
      <button
        type="button"
        onClick={onMoveUp}
        disabled={isFirst}
        className="p-1 text-text-muted hover:text-text-primary disabled:opacity-30 disabled:cursor-not-allowed"
        title="Move up"
      >
        <ArrowUp className={iconSizes.sm} />
      </button>
      <button
        type="button"
        onClick={onMoveDown}
        disabled={isLast}
        className="p-1 text-text-muted hover:text-text-primary disabled:opacity-30 disabled:cursor-not-allowed"
        title="Move down"
      >
        <ArrowDown className={iconSizes.sm} />
      </button>

      {/* Delete */}
      <button
        type="button"
        onClick={onDelete}
        className="p-1 text-text-muted hover:text-status-error"
        title="Delete rule"
      >
        <Trash2 className="h-3.5 w-3.5" />
      </button>
    </div>
  );
});

RuleRow.displayName = 'RuleRow';

/**
 * Coloring Rules Panel
 *
 * Modal/panel for managing packet coloring rules.
 * Rules are evaluated top-to-bottom; first match wins.
 */
export const ColoringRulesPanel: FC<ColoringRulesPanelProps> = memo(
  ({ rules, onRulesChange, onReset, onClose }) => {
    const [localRules, setLocalRules] = useState<ColoringRule[]>([...rules]);

    const handleRuleChange = useCallback((index: number, updated: ColoringRule) => {
      setLocalRules((prev) => {
        const next = [...prev];
        next[index] = updated;
        return next;
      });
    }, []);

    const handleDelete = useCallback((index: number) => {
      setLocalRules((prev) => prev.filter((_, i) => i !== index));
    }, []);

    const handleMoveUp = useCallback((index: number) => {
      if (index <= 0) return;
      setLocalRules((prev) => {
        const next = [...prev];
        [next[index - 1], next[index]] = [next[index], next[index - 1]];
        return next;
      });
    }, []);

    const handleMoveDown = useCallback((index: number) => {
      setLocalRules((prev) => {
        if (index >= prev.length - 1) return prev;
        const next = [...prev];
        [next[index], next[index + 1]] = [next[index + 1], next[index]];
        return next;
      });
    }, []);

    const handleAdd = useCallback(() => {
      setLocalRules((prev) => [
        ...prev,
        {
          id: generateRuleId(),
          name: 'New Rule',
          filter: '',
          foreground: '#ffffff',
          background: '#374151',
          enabled: true,
        },
      ]);
    }, []);

    const handleApply = useCallback(() => {
      onRulesChange(localRules);
      onClose();
    }, [localRules, onRulesChange, onClose]);

    const handleReset = useCallback(() => {
      onReset();
      onClose();
    }, [onReset, onClose]);

    return (
      <div className="fixed inset-0 z-50 flex-center bg-scrim/60">
        <div className="w-full max-w-3xl mx-4 bg-bg-surface border border-surface-border rounded-xl shadow-2xl max-h-[80vh] flex flex-col">
          {/* Header */}
          <div className="flex-between px-5 py-4 border-b border-surface-border">
            <h3 className="heading-3 text-text-primary">Coloring Rules</h3>
            <button
              type="button"
              onClick={onClose}
              className="p-1 text-text-muted hover:text-text-primary"
            >
              <X className="h-5 w-5" />
            </button>
          </div>

          {/* Rules list */}
          <div className="flex-1 overflow-y-auto px-5 py-4 stack-sm">
            <SmallText className="text-text-muted mb-heading block">
              Rules are evaluated top-to-bottom. First matching rule determines the row color.
            </SmallText>

            {localRules.map((rule, index) => (
              <RuleRow
                key={rule.id}
                rule={rule}
                onChange={(updated) => handleRuleChange(index, updated)}
                onDelete={() => handleDelete(index)}
                onMoveUp={() => handleMoveUp(index)}
                onMoveDown={() => handleMoveDown(index)}
                isFirst={index === 0}
                isLast={index === localRules.length - 1}
              />
            ))}

            {localRules.length === 0 && (
              <div className="text-center py-8 text-text-muted">
                <p>No coloring rules defined.</p>
              </div>
            )}
          </div>

          {/* Footer */}
          <div className="flex-between px-5 py-4 border-t border-surface-border">
            <div className="flex items-center gap-compact">
              <Button
                variant="ghost"
                size="sm"
                onClick={handleAdd}
                leftIcon={<Plus className={iconSizes.md} />}
              >
                Add Rule
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={handleReset}
                leftIcon={<RotateCcw className={iconSizes.md} />}
              >
                Reset Defaults
              </Button>
            </div>

            <div className="flex items-center gap-compact">
              <Button variant="ghost" size="sm" onClick={onClose}>
                Cancel
              </Button>
              <Button tone="violet" size="sm" onClick={handleApply}>
                Apply
              </Button>
            </div>
          </div>
        </div>
      </div>
    );
  },
);

ColoringRulesPanel.displayName = 'ColoringRulesPanel';
