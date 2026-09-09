import { useState } from 'react';
import { fetchHistoryPage } from '../api/client';
import { POLL_INTERVALS } from '../constants/polling';
import { useAppState } from '../contexts/AppContext';
import { useApiResource } from './useApiResource';

const PAGE_SIZE = 20;

export function useRunHistory() {
  const [cursors, setCursors] = useState<number[]>([]);
  const before = cursors.at(-1);
  const latest = useAppState('history');
  const olderPage = useApiResource(() => fetchHistoryPage(before), ['history', before ?? null], {
    intervalMs: POLL_INTERVALS.slow,
    enabled: before !== undefined,
  });
  const { data, error, loading } = before === undefined ? latest : olderPage;
  const records = data ?? [];
  const last = records.at(-1)?.id;
  return {
    records,
    loading,
    error,
    hasNewer: cursors.length > 0,
    hasOlder: records.length === PAGE_SIZE && last !== 1,
    newer: () => {
      if (cursors.length === 1) void latest.refetch();
      setCursors((value) => value.slice(0, -1));
    },
    older: () => {
      if (last) setCursors((value) => [...value, last]);
    },
  };
}
