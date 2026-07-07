import { useAppState } from '../contexts/AppContext';

/**
 * useSimulationStatus — the single shared poll of `GET /api/v1/simulation`.
 *
 * Before this hook, DashboardPage, RuntimeControlPage, and
 * PacketInspectorPage each ran their own independent
 * `useApiResource(fetchSimulationStatus, …)` poll at `POLL_INTERVALS.fast`
 * against the same endpoint — three timers, three in-flight requests,
 * for state that only changes when a simulation starts or stops.
 *
 * The actual fetch + interval now lives once in `AppProvider`
 * (see contexts/AppContext.tsx); this hook is a thin, readable alias over
 * `useAppState('simStatus')` so call sites read as
 * `const { data: simStatus } = useSimulationStatus();` instead of the more
 * generic `useAppState('simStatus')`. HeaderBar's status chip, the
 * Dashboard status banner, RuntimeControlPage's start/stop controls, and
 * PacketInspectorPage's active-interface detection all consume this one
 * poller.
 */
export const useSimulationStatus = () => useAppState('simStatus');
