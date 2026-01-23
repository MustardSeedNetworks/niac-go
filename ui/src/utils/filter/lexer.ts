import type { Token, TokenType } from './types';

const OPERATORS = ['==', '!=', '>=', '<=', '>', '<', 'contains'] as const;
const LOGICAL_OPS = ['&&', '||'] as const;

/**
 * Tokenize a filter expression string into a sequence of tokens.
 */
export function tokenize(input: string): Token[] {
  const tokens: Token[] = [];
  let pos = 0;

  while (pos < input.length) {
    // Skip whitespace
    if (/\s/.test(input[pos])) {
      pos++;
      continue;
    }

    // Single-character tokens
    const singleChar = matchSingleChar(input, pos);
    if (singleChar) {
      tokens.push(singleChar);
      pos++;
      continue;
    }

    // Logical operators
    const logicalMatch = matchLogical(input, pos);
    if (logicalMatch) {
      tokens.push({ type: 'LOGICAL', value: logicalMatch, position: pos });
      pos += logicalMatch.length;
      continue;
    }

    // Comparison operators
    const opMatch = matchOperator(input, pos);
    if (opMatch) {
      tokens.push({ type: 'OPERATOR', value: opMatch, position: pos });
      pos += opMatch.length;
      continue;
    }

    // Quoted string literal
    if (input[pos] === '"' || input[pos] === "'") {
      pos = tokenizeQuotedString(input, pos, tokens);
      continue;
    }

    // Field names (dotted identifiers) or bare literals/keywords
    if (/[a-zA-Z0-9_.]/.test(input[pos])) {
      pos = tokenizeIdentifier(input, pos, tokens);
      continue;
    }

    // Unknown character - skip
    pos++;
  }

  tokens.push({ type: 'EOF', value: '', position: pos });
  return tokens;
}

function matchSingleChar(input: string, pos: number): Token | null {
  if (input[pos] === '(') return { type: 'LPAREN', value: '(', position: pos };
  if (input[pos] === ')') return { type: 'RPAREN', value: ')', position: pos };
  if (input[pos] === '!' && input[pos + 1] !== '=')
    return { type: 'NOT', value: '!', position: pos };
  return null;
}

function tokenizeQuotedString(input: string, startPos: number, tokens: Token[]): number {
  const quote = input[startPos];
  let pos = startPos + 1;
  let value = '';
  while (pos < input.length && input[pos] !== quote) {
    if (input[pos] === '\\' && pos + 1 < input.length) {
      pos++;
    }
    value += input[pos];
    pos++;
  }
  if (pos < input.length) {
    pos++; // skip closing quote
  }
  tokens.push({ type: 'LITERAL', value, position: startPos });
  return pos;
}

function tokenizeIdentifier(input: string, startPos: number, tokens: Token[]): number {
  let pos = startPos;
  let value = '';
  while (pos < input.length && /[a-zA-Z0-9_./:-]/.test(input[pos])) {
    value += input[pos];
    pos++;
  }
  if (value === 'contains') {
    tokens.push({ type: 'OPERATOR', value: 'contains', position: startPos });
  } else {
    const tokenType = classifyIdentifier(value, tokens);
    tokens.push({ type: tokenType, value, position: startPos });
  }
  return pos;
}

function matchOperator(input: string, pos: number): string | null {
  for (const op of OPERATORS) {
    if (op === 'contains') continue; // handled as keyword
    if (input.substring(pos, pos + op.length) === op) {
      return op;
    }
  }
  return null;
}

function matchLogical(input: string, pos: number): string | null {
  for (const op of LOGICAL_OPS) {
    if (input.substring(pos, pos + op.length) === op) {
      return op;
    }
  }
  return null;
}

/**
 * Classify an identifier as FIELD or LITERAL based on context.
 * After an operator, it's a literal. Otherwise, it's a field.
 */
function classifyIdentifier(_value: string, precedingTokens: Token[]): TokenType {
  const lastToken = precedingTokens.length > 0 ? precedingTokens[precedingTokens.length - 1] : null;
  if (lastToken?.type === 'OPERATOR') {
    return 'LITERAL';
  }
  return 'FIELD';
}
