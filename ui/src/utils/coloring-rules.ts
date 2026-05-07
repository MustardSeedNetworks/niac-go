import { safeGetItem, safeSetItem } from './storage';

const STORAGE_KEY = 'niac-coloring-rules';

/** A single coloring rule with a filter expression and colors. */
export interface ColoringRule {
  id: string;
  name: string;
  filter: string;
  foreground: string;
  background: string;
  enabled: boolean;
}

/** Default coloring rules mimicking Wireshark conventions. */
const DEFAULT_RULES: ColoringRule[] = [
  {
    id: 'rule-tcp-syn',
    name: 'TCP SYN',
    filter: 'tcp.flags.syn',
    foreground: '#bbf7d0',
    background: '#14532d',
    enabled: true,
  },
  {
    id: 'rule-tcp-rst',
    name: 'TCP RST/FIN',
    filter: 'tcp.flags.rst || tcp.flags.fin',
    foreground: '#fecaca',
    background: '#7f1d1d',
    enabled: true,
  },
  {
    id: 'rule-dns',
    name: 'DNS',
    filter: 'dns',
    foreground: '#bfdbfe',
    background: '#1e3a5f',
    enabled: true,
  },
  {
    id: 'rule-arp',
    name: 'ARP',
    filter: 'arp',
    foreground: '#fef08a',
    background: '#422006',
    enabled: true,
  },
  {
    id: 'rule-icmp',
    name: 'ICMP',
    filter: 'icmp',
    foreground: '#a5f3fc',
    background: '#164e63',
    enabled: true,
  },
  {
    id: 'rule-http-error',
    name: 'HTTP Errors',
    filter: 'protocol == "HTTP" && frame.len > 0',
    foreground: '#fed7aa',
    background: '#431407',
    enabled: false,
  },
];

/** Load coloring rules from localStorage or return defaults. */
export function loadColoringRules(): ColoringRule[] {
  const stored = safeGetItem(STORAGE_KEY);
  if (stored) {
    try {
      const parsed = JSON.parse(stored) as ColoringRule[];
      if (Array.isArray(parsed) && parsed.length > 0) {
        return parsed;
      }
    } catch {
      // Ignore parse errors, return defaults
    }
  }
  return [...DEFAULT_RULES];
}

/** Save coloring rules to localStorage. */
export function saveColoringRules(rules: ColoringRule[]): void {
  safeSetItem(STORAGE_KEY, JSON.stringify(rules));
}

/** Reset coloring rules to defaults. */
export function getDefaultRules(): ColoringRule[] {
  return [...DEFAULT_RULES];
}

/** Generate a unique rule ID. */
export function generateRuleId(): string {
  return `rule-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;
}
