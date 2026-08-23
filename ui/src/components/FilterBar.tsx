import { type FC, memo, useCallback, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { SmallText } from '../ui/Typography';
import {
  type AutocompleteSuggestion,
  getAutocompleteSuggestions,
} from '../utils/filter/autocomplete';
import { validate } from '../utils/filter/parser';

interface FilterBarProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
}

const QUICK_PROTOCOLS = ['tcp', 'udp', 'icmp', 'arp', 'dns', 'http'] as const;

/**
 * Matches the trailing protocol group the chips themselves build: a bare atom
 * or a parenthesised OR of atoms, optionally preceded by a typed expression.
 */
const PROTOCOL_GROUP =
  /(?:^|\s&&\s)\(?((?:tcp|udp|icmp|arp|dns|http)(?:\s\|\|\s(?:tcp|udp|icmp|arp|dns|http))*)\)?$/;

interface SplitFilter {
  /** Whatever the user typed, with the protocol group removed. */
  rest: string;
  protocols: string[];
}

/**
 * Separate the chip-managed protocol group from the rest of the expression.
 *
 * Only the shapes the chips produce are recognised; anything else is left
 * untouched as `rest`, so toggling never rewrites an expression it did not
 * build.
 */
function splitFilter(value: string): SplitFilter {
  const trimmed = value.trim();
  if (!trimmed) {
    return { rest: '', protocols: [] };
  }
  const match = PROTOCOL_GROUP.exec(trimmed);
  const group = match?.[1];
  if (!match || !group) {
    return { rest: trimmed, protocols: [] };
  }
  return {
    rest: trimmed
      .slice(0, match.index)
      .replace(/\s*&&\s*$/, '')
      .trim(),
    protocols: group.split(' || '),
  };
}

/**
 * Protocol atoms combine with `||`: selecting TCP and UDP means either, and
 * `tcp && udp` can never match a frame (#1481).
 */
function buildFilter(rest: string, protocols: string[]): string {
  const group =
    protocols.length === 0
      ? ''
      : protocols.length === 1
        ? protocols.join('')
        : `(${protocols.join(' || ')})`;
  if (!rest) {
    return group;
  }
  return group ? `${rest} && ${group}` : rest;
}

/**
 * Expression-based filter bar with validation, autocomplete, and quick-insert buttons.
 * Shows green border when valid, red when invalid.
 */
export const FilterBar: FC<FilterBarProps> = memo(({ value, onChange, placeholder }) => {
  const { t } = useTranslation('common');
  const { t: tPages } = useTranslation('pages');
  const [isFocused, setIsFocused] = useState(false);
  const [suggestions, setSuggestions] = useState<AutocompleteSuggestion[]>([]);
  const [selectedSuggestion, setSelectedSuggestion] = useState(-1);
  const inputRef = useRef<HTMLInputElement>(null);

  const validationError = value.trim() ? validate(value) : null;
  const isValid = !validationError;

  const borderColor = !value.trim()
    ? 'border-surface-border'
    : isValid
      ? 'border-status-success/60'
      : 'border-status-error/60';

  // Update suggestions on input change
  useEffect(() => {
    if (!isFocused || !value.trim()) {
      setSuggestions([]);
      return;
    }
    const cursorPos = inputRef.current?.selectionStart ?? value.length;
    const results = getAutocompleteSuggestions(value, cursorPos);
    setSuggestions(results.slice(0, 8));
    setSelectedSuggestion(-1);
  }, [value, isFocused]);

  const applySuggestion = useCallback(
    (suggestion: AutocompleteSuggestion) => {
      const cursorPos = inputRef.current?.selectionStart ?? value.length;
      const textBefore = value.substring(0, cursorPos);
      const textAfter = value.substring(cursorPos);

      // Find the current word being typed
      const currentWord = textBefore.match(/([a-zA-Z0-9_.]+)$/)?.[1];
      const wordStart = currentWord ? cursorPos - currentWord.length : cursorPos;

      const newValue = value.substring(0, wordStart) + suggestion.insertText + textAfter;
      onChange(newValue);
      setSuggestions([]);

      // Refocus input
      setTimeout(() => {
        inputRef.current?.focus();
        const newPos = wordStart + suggestion.insertText.length;
        inputRef.current?.setSelectionRange(newPos, newPos);
      }, 0);
    },
    [value, onChange],
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (suggestions.length === 0) return;

      if (e.key === 'ArrowDown') {
        e.preventDefault();
        setSelectedSuggestion((prev) => Math.min(prev + 1, suggestions.length - 1));
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        setSelectedSuggestion((prev) => Math.max(prev - 1, -1));
      } else if (e.key === 'Tab' || e.key === 'Enter') {
        const chosen = suggestions[selectedSuggestion];
        if (chosen) {
          e.preventDefault();
          applySuggestion(chosen);
        }
      } else if (e.key === 'Escape') {
        setSuggestions([]);
      }
    },
    [suggestions, selectedSuggestion, applySuggestion],
  );

  const activeProtocols = splitFilter(value).protocols;

  const handleQuickToggle = useCallback(
    (protocol: string) => {
      const { rest, protocols } = splitFilter(value);
      const next = protocols.includes(protocol)
        ? protocols.filter((entry) => entry !== protocol)
        : [...protocols, protocol];
      onChange(buildFilter(rest, next));
      inputRef.current?.focus();
    },
    [value, onChange],
  );

  return (
    <div className="relative w-full">
      <SmallText className="text-text-muted px-1">{t('filters.displayFilterCaption')}</SmallText>
      <div className="flex items-center gap-compact">
        <div className="relative flex-1">
          <input
            ref={inputRef}
            type="text"
            value={value}
            onChange={(e) => onChange(e.target.value)}
            onFocus={() => setIsFocused(true)}
            onBlur={() => setTimeout(() => setIsFocused(false), 200)}
            onKeyDown={handleKeyDown}
            placeholder={placeholder ?? tPages('packets.filterBar.inputPlaceholder')}
            className={`w-full rounded-lg border ${borderColor} bg-bg-base/60 px-3 py-row text-sm text-text-primary placeholder:text-text-muted focus:outline-none font-mono transition-colors`}
          />
          {validationError && isFocused && (
            <div className="absolute top-full left-0 mt-tight px-cell py-compact bg-status-error/90 border border-status-error/50 rounded text-xs text-status-error z-20 max-w-md">
              {validationError}
            </div>
          )}

          {/* Autocomplete dropdown */}
          {suggestions.length > 0 && isFocused && (
            <div className="absolute top-full left-0 mt-tight w-full bg-bg-surface border border-surface-border rounded-lg shadow-lg z-30 max-h-48 overflow-y-auto">
              {suggestions.map((suggestion, idx) => (
                <button
                  key={suggestion.text}
                  type="button"
                  className={`w-full text-left px-3 py-compact-md text-sm hover:bg-bg-elevated ${
                    idx === selectedSuggestion
                      ? 'bg-bg-elevated text-text-primary'
                      : 'text-text-secondary'
                  }`}
                  onMouseDown={(e) => {
                    e.preventDefault();
                    applySuggestion(suggestion);
                  }}
                >
                  <span className="font-mono text-brand-accent">{suggestion.text}</span>
                  <span className="ml-inline text-text-muted text-xs">
                    {suggestion.description}
                  </span>
                </button>
              ))}
            </div>
          )}
        </div>

        {/* Quick protocol insert buttons */}
        <div className="flex items-center gap-tight">
          {QUICK_PROTOCOLS.map((protocol) => (
            <button
              key={protocol}
              type="button"
              aria-pressed={activeProtocols.includes(protocol)}
              onClick={() => handleQuickToggle(protocol)}
              className={`px-cell py-compact text-xs rounded border transition-colors uppercase ${
                activeProtocols.includes(protocol)
                  ? 'border-brand-primary bg-brand-primary/15 text-text-primary'
                  : 'border-surface-border text-text-muted hover:text-text-primary bg-bg-surface/50'
              }`}
            >
              {protocol}
            </button>
          ))}
        </div>
      </div>
    </div>
  );
});

FilterBar.displayName = 'FilterBar';
