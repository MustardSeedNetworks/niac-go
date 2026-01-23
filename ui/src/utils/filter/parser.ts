import { tokenize } from './lexer';
import type { ASTNode, ComparisonOperator, Token } from './types';
import { BARE_PROTOCOLS } from './types';

export class ParseError extends Error {
  constructor(
    message: string,
    public position: number,
  ) {
    super(message);
    this.name = 'ParseError';
  }
}

/**
 * Parse a filter expression string into an AST.
 * Grammar (precedence low to high):
 *   expr     = or_expr
 *   or_expr  = and_expr ( '||' and_expr )*
 *   and_expr = not_expr ( '&&' not_expr )*
 *   not_expr = '!' not_expr | primary
 *   primary  = '(' expr ')' | comparison | field_exists
 */
export function parse(input: string): ASTNode {
  const tokens = tokenize(input);
  const parser = new Parser(tokens);
  const ast = parser.parseExpression();
  parser.expectEnd();
  return ast;
}

/**
 * Validate a filter expression. Returns null if valid, error message if invalid.
 */
export function validate(input: string): string | null {
  if (!input.trim()) {
    return null; // empty is valid (no filter)
  }
  try {
    parse(input);
    return null;
  } catch (err) {
    if (err instanceof ParseError) {
      return err.message;
    }
    return 'Invalid filter expression';
  }
}

class Parser {
  private pos = 0;

  constructor(private tokens: Token[]) {}

  parseExpression(): ASTNode {
    return this.parseOr();
  }

  private parseOr(): ASTNode {
    let left = this.parseAnd();
    while (this.peek().type === 'LOGICAL' && this.peek().value === '||') {
      this.advance();
      const right = this.parseAnd();
      left = { type: 'Logical', operator: '||', left, right };
    }
    return left;
  }

  private parseAnd(): ASTNode {
    let left = this.parseNot();
    while (this.peek().type === 'LOGICAL' && this.peek().value === '&&') {
      this.advance();
      const right = this.parseNot();
      left = { type: 'Logical', operator: '&&', left, right };
    }
    return left;
  }

  private parseNot(): ASTNode {
    if (this.peek().type === 'NOT') {
      this.advance();
      const operand = this.parseNot();
      return { type: 'Not', operand };
    }
    return this.parsePrimary();
  }

  private parsePrimary(): ASTNode {
    const token = this.peek();

    // Grouped expression
    if (token.type === 'LPAREN') {
      this.advance();
      const expr = this.parseExpression();
      if (this.peek().type !== 'RPAREN') {
        throw new ParseError('Expected closing parenthesis', this.peek().position);
      }
      this.advance();
      return { type: 'Group', expression: expr };
    }

    // Field name or bare protocol
    if (token.type === 'FIELD') {
      this.advance();
      const fieldName = token.value;

      // Check if next is an operator -> comparison
      if (this.peek().type === 'OPERATOR') {
        const operator = this.advance().value as ComparisonOperator;
        const valueToken = this.peek();

        if (valueToken.type !== 'LITERAL' && valueToken.type !== 'FIELD') {
          throw new ParseError(`Expected value after operator '${operator}'`, valueToken.position);
        }
        this.advance();

        const value = parseValue(valueToken.value);
        return { type: 'Comparison', field: fieldName, operator, value };
      }

      // Bare field name -> existence check or protocol match
      if (BARE_PROTOCOLS.includes(fieldName.toLowerCase() as (typeof BARE_PROTOCOLS)[number])) {
        return { type: 'FieldExists', field: fieldName.toLowerCase() };
      }

      return { type: 'FieldExists', field: fieldName };
    }

    throw new ParseError(
      `Unexpected token '${token.value || token.type}' at position ${token.position}`,
      token.position,
    );
  }

  private peek(): Token {
    return this.tokens[this.pos] ?? { type: 'EOF', value: '', position: -1 };
  }

  private advance(): Token {
    const token = this.tokens[this.pos];
    this.pos++;
    return token;
  }

  expectEnd(): void {
    if (this.peek().type !== 'EOF') {
      throw new ParseError(
        `Unexpected token '${this.peek().value}' at position ${this.peek().position}`,
        this.peek().position,
      );
    }
  }
}

function parseValue(raw: string): string | number {
  const num = Number(raw);
  if (!Number.isNaN(num) && raw.trim() !== '') {
    return num;
  }
  return raw;
}
