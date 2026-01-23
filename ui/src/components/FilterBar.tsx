import { type FC, memo, useCallback, useEffect, useRef, useState } from 'react';
import { getAutocompleteSuggestions } from '../utils/filter/autocomplete';
import { validate } from '../utils/filter/parser';
import type { AutocompleteSuggestion } from '../utils/filter/types';

interface FilterBarProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
}

const QUICK_PROTOCOLS = ['tcp', 'udp', 'icmp', 'arp', 'dns', 'http'] as const;

/**
 * Expression-based filter bar with validation, autocomplete, and quick-insert buttons.
 * Shows green border when valid, red when invalid.
 */
export const FilterBar: FC<FilterBarProps> = memo(({ value, onChange, placeholder }) => {
  const [isFocused, setIsFocused] = useState(false);
  const [suggestions, setSuggestions] = useState<AutocompleteSuggestion[]>([]);
  const [selectedSuggestion, setSelectedSuggestion] = useState(-1);
  const inputRef = useRef<HTMLInputElement>(null);

  const validationError = value.trim() ? validate(value) : null;
  const isValid = !validationError;

  const borderColor = !value.trim()
    ? 'border-white/10'
    : isValid
      ? 'border-green-500/60'
      : 'border-red-500/60';

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
      const wordMatch = textBefore.match(/([a-zA-Z0-9_.]+)$/);
      const wordStart = wordMatch ? cursorPos - wordMatch[1].length : cursorPos;

      const newValue =
        value.substring(0, wordStart) + suggestion.insertText + textAfter;
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
        if (selectedSuggestion >= 0) {
          e.preventDefault();
          applySuggestion(suggestions[selectedSuggestion]);
        }
      } else if (e.key === 'Escape') {
        setSuggestions([]);
      }
    },
    [suggestions, selectedSuggestion, applySuggestion],
  );

  const handleQuickInsert = useCallback(
    (protocol: string) => {
      const newValue = value.trim() ? `${value} && ${protocol}` : protocol;
      onChange(newValue);
      inputRef.current?.focus();
    },
    [value, onChange],
  );

  return (
    <div className="relative w-full">
      <div className="flex items-center gap-2">
        <div className="relative flex-1">
          <input
            ref={inputRef}
            type="text"
            value={value}
            onChange={(e) => onChange(e.target.value)}
            onFocus={() => setIsFocused(true)}
            onBlur={() => setTimeout(() => setIsFocused(false), 200)}
            onKeyDown={handleKeyDown}
            placeholder={placeholder ?? 'Display filter (e.g., ip.src == 10.0.0.1 && tcp)'}
            className={`w-full rounded-lg border ${borderColor} bg-gray-950/60 px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none font-mono transition-colors`}
          />
          {validationError && isFocused && (
            <div className="absolute top-full left-0 mt-1 px-2 py-1 bg-red-900/90 border border-red-500/50 rounded text-xs text-red-200 z-20 max-w-md">
              {validationError}
            </div>
          )}

          {/* Autocomplete dropdown */}
          {suggestions.length > 0 && isFocused && (
            <div className="absolute top-full left-0 mt-1 w-full bg-gray-900 border border-white/10 rounded-lg shadow-lg z-30 max-h-48 overflow-y-auto">
              {suggestions.map((suggestion, idx) => (
                <button
                  key={suggestion.text}
                  type="button"
                  className={`w-full text-left px-3 py-1.5 text-sm hover:bg-gray-800 ${
                    idx === selectedSuggestion ? 'bg-gray-800 text-white' : 'text-gray-300'
                  }`}
                  onMouseDown={(e) => {
                    e.preventDefault();
                    applySuggestion(suggestion);
                  }}
                >
                  <span className="font-mono text-violet-300">{suggestion.text}</span>
                  <span className="ml-2 text-gray-500 text-xs">{suggestion.description}</span>
                </button>
              ))}
            </div>
          )}
        </div>

        {/* Quick protocol insert buttons */}
        <div className="flex items-center gap-1">
          {QUICK_PROTOCOLS.map((protocol) => (
            <button
              key={protocol}
              type="button"
              onClick={() => handleQuickInsert(protocol)}
              className="px-2 py-1 text-xs rounded border border-white/10 text-gray-400 hover:text-white hover:border-white/20 bg-gray-900/50 transition-colors uppercase"
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
