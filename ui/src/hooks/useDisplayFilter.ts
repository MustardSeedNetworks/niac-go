import { useMemo } from 'react';
import type { Packet } from '../components/PacketList';
import type { PcapPacket } from '../api/types';
import { evaluate, parse, validate } from '../utils/filter';
import type { ASTNode } from '../utils/filter';

interface UseDisplayFilterResult<T> {
  filteredPackets: T[];
  isValid: boolean;
  error: string | null;
}

/**
 * Hook that parses a filter expression and evaluates it against a packet list.
 * Returns the filtered packets and validation state.
 */
export function useDisplayFilter<T extends Packet | PcapPacket>(
  packets: T[],
  filterExpression: string,
): UseDisplayFilterResult<T> {
  const { ast, isValid, error } = useMemo(() => {
    const trimmed = filterExpression.trim();
    if (!trimmed) {
      return { ast: null, isValid: true, error: null };
    }
    const validationError = validate(trimmed);
    if (validationError) {
      return { ast: null, isValid: false, error: validationError };
    }
    try {
      const parsed = parse(trimmed);
      return { ast: parsed, isValid: true, error: null };
    } catch (err) {
      return {
        ast: null,
        isValid: false,
        error: err instanceof Error ? err.message : 'Parse error',
      };
    }
  }, [filterExpression]);

  const filteredPackets = useMemo(() => {
    if (!ast) {
      return packets;
    }
    return packets.filter((packet) => evaluate(ast as ASTNode, packet));
  }, [packets, ast]);

  return { filteredPackets, isValid, error };
}
