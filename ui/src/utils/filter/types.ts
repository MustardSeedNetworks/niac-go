/** Token types produced by the lexer */
export type TokenType =
  | 'FIELD'
  | 'LITERAL'
  | 'OPERATOR'
  | 'LOGICAL'
  | 'NOT'
  | 'LPAREN'
  | 'RPAREN'
  | 'EOF';

export interface Token {
  type: TokenType;
  value: string;
  position: number;
}

/** AST node types for the filter expression parser */
export type ASTNode =
  | FieldExistsNode
  | ComparisonNode
  | LogicalNode
  | NotNode
  | GroupNode
  | LiteralNode
  | FieldAccessNode;

export interface FieldExistsNode {
  type: 'FieldExists';
  field: string;
}

export interface ComparisonNode {
  type: 'Comparison';
  field: string;
  operator: ComparisonOperator;
  value: string | number;
}

export interface LogicalNode {
  type: 'Logical';
  operator: '&&' | '||';
  left: ASTNode;
  right: ASTNode;
}

export interface NotNode {
  type: 'Not';
  operand: ASTNode;
}

export interface GroupNode {
  type: 'Group';
  expression: ASTNode;
}

export interface LiteralNode {
  type: 'Literal';
  value: string | number;
}

export interface FieldAccessNode {
  type: 'FieldAccess';
  field: string;
}

export type ComparisonOperator = '==' | '!=' | '>' | '<' | '>=' | '<=' | 'contains';

/** Known filter fields with their descriptions */
export const FILTER_FIELDS: Record<string, string> = {
  'ip.src': 'Source IP address',
  'ip.dst': 'Destination IP address',
  'tcp.port': 'TCP port (source or destination)',
  'udp.port': 'UDP port (source or destination)',
  'tcp.srcport': 'TCP source port',
  'tcp.dstport': 'TCP destination port',
  'udp.srcport': 'UDP source port',
  'udp.dstport': 'UDP destination port',
  'tcp.flags.syn': 'TCP SYN flag',
  'tcp.flags.ack': 'TCP ACK flag',
  'tcp.flags.fin': 'TCP FIN flag',
  'tcp.flags.rst': 'TCP RST flag',
  'tcp.flags.psh': 'TCP PSH flag',
  'eth.src': 'Source MAC address',
  'eth.dst': 'Destination MAC address',
  protocol: 'Protocol name',
  'frame.len': 'Frame length in bytes',
};

/** Bare protocol names that act as existence checks */
export const BARE_PROTOCOLS = [
  'tcp',
  'udp',
  'icmp',
  'arp',
  'dns',
  'http',
  'https',
  'dhcp',
  'ssh',
  'tls',
  'snmp',
  'lldp',
  'cdp',
  'stp',
] as const;
