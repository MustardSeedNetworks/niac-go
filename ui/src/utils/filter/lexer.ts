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

    // Parentheses
    if (input[pos] === '(') {
      tokens.push({ type: 'LPAREN', value: '(', position: pos });
      pos++;
      continue;
    }
    if (input[pos] === ')') {
      tokens.push({ type: 'RPAREN', value: ')', position: pos });
      pos++;
      continue;
    }

    // NOT operator
    if (input[pos] === '!' && input[pos + 1] !== '=') {
      tokens.push({ type: 'NOT', value: '!', position: pos });
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
      const quote = input[pos];
      const startPos = pos;
      pos++;
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
      continue;
    }

    // Field names (dotted identifiers) or bare literals/keywords
    if (/[a-zA-Z0-9_.]/.test(input[pos])) {
      const startPos = pos;
      let value = '';
      while (pos < input.length && /[a-zA-Z0-9_./:-]/.test(input[pos])) {
        value += input[pos];
        pos++;
      }

      // Check if this is the keyword 'contains'
      if (value === 'contains') {
        tokens.push({ type: 'OPERATOR', value: 'contains', position: startPos });
        continue;
      }

      // Determine if this looks like a field name or a literal
      const tokenType = classifyIdentifier(value, tokens);
      tokens.push({ type: tokenType, value, position: startPos });
      continue;
    }

    // Unknown character - skip
    pos++;
  }

  tokens.push({ type: 'EOF', value: '', position: pos });
  return tokens;
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
function classifyIdentifier(value: string, precedingTokens: Token[]): TokenType {
  const lastToken = precedingTokens.length > 0 ? precedingTokens[precedingTokens.length - 1] : null;
  if (lastToken?.type === 'OPERATOR') {
    return 'LITERAL';
  }
  return 'FIELD';
}
