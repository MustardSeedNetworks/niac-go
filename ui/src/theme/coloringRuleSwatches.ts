/**
 * Packet-list colouring rule swatches. These stay hex rather than tokens: an
 * operator edits them in `<input type="color">`, which only takes #rrggbb,
 * and the rules persist to localStorage as written. Each pair carries its own
 * contrast (light text on a dark fill), so it reads the same in either theme.
 */
export interface RuleSwatch {
  foreground: string;
  background: string;
}

export const RULE_SWATCHES = {
  tcpSyn: { foreground: '#bbf7d0', background: '#14532d' },
  tcpRstFin: { foreground: '#fecaca', background: '#7f1d1d' },
  dns: { foreground: '#bfdbfe', background: '#1e3a5f' },
  arp: { foreground: '#fef08a', background: '#422006' },
  icmp: { foreground: '#a5f3fc', background: '#164e63' },
  httpError: { foreground: '#fed7aa', background: '#431407' },
  newRule: { foreground: '#ffffff', background: '#374151' },
} as const satisfies Record<string, RuleSwatch>;
