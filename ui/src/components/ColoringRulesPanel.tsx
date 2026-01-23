import { ArrowDown, ArrowUp, Plus, RotateCcw, Trash2, X } from 'lucide-react';
import { type FC, memo, useCallback, useState } from 'react';
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
    <div className="flex items-center gap-2 py-2 px-2 rounded-lg bg-gray-950/40 border border-white/5">
      {/* Enable toggle */}
      <input
        type="checkbox"
        checked={rule.enabled}
        onChange={(e) => onChange({ ...rule, enabled: e.target.checked })}
        className="rounded border-gray-600 bg-gray-800 text-violet-500 focus:ring-violet-400 focus:ring-offset-gray-900 flex-shrink-0"
      />

      {/* Color preview */}
      <div
        className="w-6 h-6 rounded border border-white/10 flex-shrink-0"
        style={{ backgroundColor: rule.background, color: rule.foreground }}
      >
        <span className="text-xs font-bold flex items-center justify-center h-full">A</span>
      </div>

      {/* Name */}
      <input
        type="text"
        value={rule.name}
        onChange={(e) => onChange({ ...rule, name: e.target.value })}
        className="w-24 bg-transparent border-b border-white/10 text-sm text-white focus:outline-none focus:border-violet-400 px-1"
        placeholder="Name"
      />

      {/* Filter expression */}
      <input
        type="text"
        value={rule.filter}
        onChange={(e) => onChange({ ...rule, filter: e.target.value })}
        className={`flex-1 bg-transparent border-b text-sm font-mono text-white focus:outline-none px-1 ${
          filterError
            ? 'border-red-500/60 focus:border-red-400'
            : 'border-white/10 focus:border-violet-400'
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
        className="p-1 text-gray-500 hover:text-white disabled:opacity-30 disabled:cursor-not-allowed"
        title="Move up"
      >
        <ArrowUp className="h-3.5 w-3.5" />
      </button>
      <button
        type="button"
        onClick={onMoveDown}
        disabled={isLast}
        className="p-1 text-gray-500 hover:text-white disabled:opacity-30 disabled:cursor-not-allowed"
        title="Move down"
      >
        <ArrowDown className="h-3.5 w-3.5" />
      </button>

      {/* Delete */}
      <button
        type="button"
        onClick={onDelete}
        className="p-1 text-gray-500 hover:text-red-400"
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
      <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
        <div className="w-full max-w-3xl mx-4 bg-gray-900 border border-white/10 rounded-xl shadow-2xl max-h-[80vh] flex flex-col">
          {/* Header */}
          <div className="flex items-center justify-between px-5 py-4 border-b border-white/10">
            <h3 className="text-lg font-semibold text-white">Coloring Rules</h3>
            <button type="button" onClick={onClose} className="p-1 text-gray-400 hover:text-white">
              <X className="h-5 w-5" />
            </button>
          </div>

          {/* Rules list */}
          <div className="flex-1 overflow-y-auto px-5 py-4 space-y-2">
            <SmallText className="text-gray-400 mb-3 block">
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
              <div className="text-center py-8 text-gray-500">
                <p>No coloring rules defined.</p>
              </div>
            )}
          </div>

          {/* Footer */}
          <div className="flex items-center justify-between px-5 py-4 border-t border-white/10">
            <div className="flex items-center gap-2">
              <Button
                variant="ghost"
                size="sm"
                onClick={handleAdd}
                leftIcon={<Plus className="h-4 w-4" />}
              >
                Add Rule
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={handleReset}
                leftIcon={<RotateCcw className="h-4 w-4" />}
              >
                Reset Defaults
              </Button>
            </div>

            <div className="flex items-center gap-2">
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
