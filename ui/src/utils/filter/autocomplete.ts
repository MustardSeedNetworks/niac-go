import { BARE_PROTOCOLS, FILTER_FIELDS } from './types';

export interface AutocompleteSuggestion {
  text: string;
  description: string;
  insertText: string;
}

/**
 * Generate autocomplete suggestions based on cursor position in the filter expression.
 */
export function getAutocompleteSuggestions(
  input: string,
  cursorPosition: number,
): AutocompleteSuggestion[] {
  const textBeforeCursor = input.substring(0, cursorPosition);
  const currentWord = extractCurrentWord(textBeforeCursor);

  if (!currentWord) {
    // At start or after operator/logical - suggest fields and protocols
    return [...getFieldSuggestions(''), ...getProtocolSuggestions('')];
  }

  // Check if we're after an operator - suggest values
  const beforeWord = textBeforeCursor.substring(0, textBeforeCursor.length - currentWord.length).trimEnd();
  if (beforeWord.endsWith('==') || beforeWord.endsWith('!=') || beforeWord.endsWith('contains')) {
    return getValueSuggestions(currentWord, beforeWord);
  }

  // Suggest matching fields and protocols
  return [...getFieldSuggestions(currentWord), ...getProtocolSuggestions(currentWord)];
}

function extractCurrentWord(text: string): string {
  const match = text.match(/([a-zA-Z0-9_.]+)$/);
  return match ? match[1] : '';
}

function getFieldSuggestions(prefix: string): AutocompleteSuggestion[] {
  const lowPrefix = prefix.toLowerCase();
  return Object.entries(FILTER_FIELDS)
    .filter(([field]) => field.toLowerCase().startsWith(lowPrefix))
    .map(([field, description]) => ({
      text: field,
      description,
      insertText: field,
    }));
}

function getProtocolSuggestions(prefix: string): AutocompleteSuggestion[] {
  const lowPrefix = prefix.toLowerCase();
  return BARE_PROTOCOLS.filter((p) => p.startsWith(lowPrefix)).map((protocol) => ({
    text: protocol,
    description: `Match ${protocol.toUpperCase()} packets`,
    insertText: protocol,
  }));
}

function getValueSuggestions(prefix: string, context: string): AutocompleteSuggestion[] {
  // Extract the field name before the operator
  const fieldMatch = context.match(/([a-zA-Z0-9_.]+)\s*(?:==|!=|contains|>=?|<=?)\s*$/);
  if (!fieldMatch) return [];

  const field = fieldMatch[1];

  // Provide common value suggestions based on field
  switch (field) {
    case 'protocol':
      return BARE_PROTOCOLS.filter((p) => p.startsWith(prefix.toLowerCase())).map((p) => ({
        text: p.toUpperCase(),
        description: `Protocol ${p.toUpperCase()}`,
        insertText: p.toUpperCase(),
      }));
    default:
      return [];
  }
}
