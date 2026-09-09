import { useState } from 'react';
import { fetchHistoryPage } from '../../api/client';
import { POLL_INTERVALS } from '../../constants/polling';
import { useApiResource } from '../../hooks/useApiResource';

const PAGE_SIZE = 20;

export function useRunHistory() {
  const [cursors, setCursors] = useState<number[]>([]);
  const before = cursors.at(-1);
  const resource = useApiResource(
    async () => ({ before, records: await fetchHistoryPage(before) }),
    [before],
    {
      intervalMs: POLL_INTERVALS.slow,
    },
  );
  const current = resource.data?.before === before ? resource.data : null;
  const records = current?.records ?? [];
  const last = records.at(-1)?.id;
  const loading = !current && !resource.error;
  return {
    records,
    loading,
    error: resource.error,
    hasNewer: cursors.length > 0,
    hasOlder: records.length === PAGE_SIZE && last !== 1,
    newer: () => setCursors((value) => value.slice(0, -1)),
    older: () => {
      if (last) setCursors((value) => [...value, last]);
    },
  };
}
