import { type CSSProperties, useCallback, useMemo, useState } from 'react';
import type { PcapPacket } from '../api/types';
import type { Packet } from '../components/PacketList';
import { evaluate } from '../utils/filter/evaluator';
import { parse, validate } from '../utils/filter/parser';
import type { ASTNode } from '../utils/filter/types';
import {
  type ColoringRule,
  getDefaultRules,
  loadColoringRules,
  saveColoringRules,
} from '../utils/coloring-rules';

interface CompiledRule {
  rule: ColoringRule;
  ast: ASTNode | null;
}

/**
 * Hook for managing coloring rules and evaluating them against packets.
 *
 * Returns:
 * - rules: The current set of coloring rules
 * - setRules: Update rules (persists to localStorage)
 * - getRowStyle: Given a packet, returns CSS styles for the matching rule
 * - resetRules: Reset to default rules
 */
export function useColoringRules() {
  const [rules, setRulesState] = useState<ColoringRule[]>(loadColoringRules);

  // Compile rules into ASTs for fast evaluation
  const compiledRules = useMemo<CompiledRule[]>(() => {
    return rules
      .filter((r) => r.enabled)
      .map((rule) => {
        const error = validate(rule.filter);
        if (error !== null) {
          return { rule, ast: null };
        }
        try {
          return { rule, ast: parse(rule.filter) };
        } catch {
          return { rule, ast: null };
        }
      })
      .filter((cr): cr is CompiledRule & { ast: ASTNode } => cr.ast !== null);
  }, [rules]);

  // Persist rules on change
  const setRules = useCallback((newRules: ColoringRule[]) => {
    setRulesState(newRules);
    saveColoringRules(newRules);
  }, []);

  // Reset to defaults
  const resetRules = useCallback(() => {
    const defaults = getDefaultRules();
    setRulesState(defaults);
    saveColoringRules(defaults);
  }, []);

  // Evaluate packet against rules (first matching rule wins)
  const getRowStyle = useCallback(
    (packet: Packet | PcapPacket): CSSProperties | undefined => {
      for (const { rule, ast } of compiledRules) {
        try {
          if (evaluate(ast, packet)) {
            return {
              color: rule.foreground,
              backgroundColor: rule.background,
            };
          }
        } catch {
          // Skip invalid rules
        }
      }
      return undefined;
    },
    [compiledRules],
  );

  return { rules, setRules, getRowStyle, resetRules };
}
