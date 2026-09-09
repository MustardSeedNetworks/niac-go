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
  const {
    data: olderRecords,
    error: olderError,
    loading: olderLoading,
  } = useApiResource(() => fetchHistoryPage(before), ['history', before ?? null], {
    intervalMs: POLL_INTERVALS.slow,
    enabled: before !== undefined,
  });
  const isLatest = before === undefined;
  const records = (isLatest ? latest.data : olderRecords) ?? [];
  const last = records.at(-1)?.id;
  return {
    records,
    loading: isLatest ? latest.loading : olderLoading,
    error: isLatest ? latest.error : olderError,
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
