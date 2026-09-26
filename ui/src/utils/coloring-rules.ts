import { RULE_SWATCHES } from '../theme/coloringRuleSwatches';
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
    ...RULE_SWATCHES.tcpSyn,
    enabled: true,
  },
  {
    id: 'rule-tcp-rst',
    name: 'TCP RST/FIN',
    filter: 'tcp.flags.rst || tcp.flags.fin',
    ...RULE_SWATCHES.tcpRstFin,
    enabled: true,
  },
  {
    id: 'rule-dns',
    name: 'DNS',
    filter: 'dns',
    ...RULE_SWATCHES.dns,
    enabled: true,
  },
  {
    id: 'rule-arp',
    name: 'ARP',
    filter: 'arp',
    ...RULE_SWATCHES.arp,
    enabled: true,
  },
  {
    id: 'rule-icmp',
    name: 'ICMP',
    filter: 'icmp',
    ...RULE_SWATCHES.icmp,
    enabled: true,
  },
  {
    id: 'rule-http-error',
    name: 'HTTP Errors',
    filter: 'protocol == "HTTP" && frame.len > 0',
    ...RULE_SWATCHES.httpError,
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
  return `rule-${Date.now()}-${Math.random().toString(36).substring(2, 7)}`;
}
