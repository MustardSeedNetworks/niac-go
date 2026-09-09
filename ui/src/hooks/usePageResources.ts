import {
  fetchAlerts,
  fetchCaptureStatus,
  fetchConfig,
  fetchSegments,
  fetchTopology,
  fetchUsableInterfaces,
} from '../api/client';
import { fetchLibraryPcaps, fetchLibraryWalks, type LibraryFileEntry } from '../api/library-client';
import { POLL_INTERVALS } from '../constants/polling';
import { type ResourceOptions, useApiResource } from './useApiResource';

export const useConfigResource = () =>
  useApiResource(fetchConfig, ['config'], { intervalMs: POLL_INTERVALS.verySlow });
export const useAlertsResource = () =>
  useApiResource(fetchAlerts, ['alerts'], { intervalMs: POLL_INTERVALS.slow, errorToast: true });
export const useCaptureResource = () =>
  useApiResource(fetchCaptureStatus, ['capture'], { intervalMs: POLL_INTERVALS.fast });
export const useUsableInterfacesResource = () =>
  useApiResource(fetchUsableInterfaces, ['interfaces', 'usable']);

export const useLibraryResource = (
  kind: 'walks' | 'pcaps',
  options: ResourceOptions<LibraryFileEntry[]> = {},
) =>
  useApiResource(
    kind === 'walks' ? fetchLibraryWalks : fetchLibraryPcaps,
    ['library', kind],
    options,
  );

export const useTopologyResource = (sessionId: string | null) =>
  useApiResource(() => fetchTopology(sessionId ?? ''), ['topology', sessionId], {
    intervalMs: POLL_INTERVALS.slow,
    enabled: sessionId !== null,
  });

export const useSegmentsResource = (sessionId: string | null) =>
  useApiResource(() => fetchSegments(sessionId ?? ''), ['segments', sessionId], {
    intervalMs: POLL_INTERVALS.slow,
    enabled: sessionId !== null,
  });
