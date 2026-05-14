import { Activity, BellRing, Download, FileCog, Network, PlugZap } from 'lucide-react';
import { type FC, useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  fetchConfig,
  fetchProtocolDebugLevels,
  fetchSimulationStatus,
  fetchTemplateContent,
  fetchUsableInterfaces,
  fetchUserConfigContent,
  startSimulation,
  stopSimulation,
  updateProtocolDebugLevels,
} from '../api/client';
import type { DebugLevel, NetworkInterface, Template, UserConfig } from '../api/types';
import { StatBlock } from '../components/StatBlock';
import { ConfigPicker } from '../components/simulation/ConfigPicker';
import { POLL_INTERVALS } from '../constants/polling';
import { useApiResource } from '../hooks/useApiResource';
import { useUIStore } from '../stores/ui-store';
import { Button } from '../ui/Button';
import { Card, CardContent } from '../ui/Card';
import { Tag } from '../ui/Tag';
import { H2, SmallText } from '../ui/Typography';
import { fileToText } from '../utils/file';
import { formatTime, formatUptime, getErrorMessage } from '../utils/format';

/**
 * Simulation Control Page (formerly RuntimeControlPage)
 *
 * Simplified interface for starting/stopping simulations.
 * Configuration is managed via Settings drawer.
 *
 * Features:
 * - Status display when simulation is running
 * - Start/Stop controls using settings from UI store
 * - Quick override with file upload
 * - Link to Settings for configuration management
 */
export const RuntimeControlPage: FC = () => {
  const [refetchTrigger, setRefetchTrigger] = useState(0);
  const { data: simStatus } = useApiResource(fetchSimulationStatus, [refetchTrigger], {
    intervalMs: POLL_INTERVALS.fast,
  });

  const { simulationSettings, setSimulationSettings } = useUIStore();
  const [interfaces, setInterfaces] = useState<NetworkInterface[]>([]);
  const [interfacesLoading, setInterfacesLoading] = useState(true);
  const [quickUploadFile, setQuickUploadFile] = useState<File | null>(null);
  const [starting, setStarting] = useState(false);
  const [stopping, setStopping] = useState(false);
  const [message, setMessage] = useState<{
    tone: 'success' | 'error';
    text: string;
  } | null>(null);

  // Hydrate the interface dropdown so the user can pick an interface inline
  // instead of having to open the Settings drawer.
  useEffect(() => {
    let cancelled = false;
    fetchUsableInterfaces()
      .then((resp) => {
        if (cancelled) return;
        setInterfaces(resp.interfaces);
      })
      .catch(() => {
        // Ignored — interface list is best-effort. The Configure button still
        // works, and an empty list shows the "no interfaces" state below.
      })
      .finally(() => {
        if (!cancelled) setInterfacesLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const handleInterfaceChange = useCallback(
    (e: React.ChangeEvent<HTMLSelectElement>) => {
      setSimulationSettings({ selectedInterface: e.target.value });
    },
    [setSimulationSettings],
  );

  const handleSelectTemplate = useCallback(
    (template: Template) => {
      setSimulationSettings({
        configSource: 'template',
        configName: template.name,
        configPath: undefined,
      });
      setQuickUploadFile(null);
    },
    [setSimulationSettings],
  );

  const handleSelectUserConfig = useCallback(
    (config: UserConfig) => {
      setSimulationSettings({
        configSource: 'userConfig',
        configName: config.name,
        configPath: config.path,
      });
      setQuickUploadFile(null);
    },
    [setSimulationSettings],
  );

  const handleUpload = useCallback(
    (file: File | null) => {
      setQuickUploadFile(file);
      if (file) {
        // Picking an upload clears any previously-selected template / config so
        // the start handler doesn't get confused about which source wins.
        setSimulationSettings({
          configSource: 'upload',
          configName: '',
          configPath: undefined,
        });
        setMessage(null);
      }
    },
    [setSimulationSettings],
  );

  const isDaemonMode = simStatus !== null;
  const hasValidConfig =
    simulationSettings.selectedInterface && (simulationSettings.configName || quickUploadFile);

  const handleStart = useCallback(async () => {
    if (!simulationSettings.selectedInterface) {
      setMessage({
        tone: 'error',
        text: 'Please select an interface in Settings',
      });
      return;
    }

    if (!simulationSettings.configName && !quickUploadFile) {
      setMessage({
        tone: 'error',
        text: 'Please select a configuration in Settings or upload a config file',
      });
      return;
    }

    setStarting(true);
    setMessage(null);

    try {
      let configData: string | undefined;
      let configPath: string | undefined;

      // Handle quick upload file
      if (quickUploadFile) {
        configData = await fileToText(quickUploadFile);
      }
      // Handle template - need to apply it first to get a config path
      else if (simulationSettings.configSource === 'template') {
        // Fetch template content and send as configData
        const templateContent = await fetchTemplateContent(simulationSettings.configName);
        configData = templateContent.content;
      }
      // Handle user config - use the stored path
      else if (simulationSettings.configSource === 'userConfig') {
        if (simulationSettings.configPath) {
          configPath = simulationSettings.configPath;
        } else {
          // Fetch user config content and send as configData
          const userConfigContent = await fetchUserConfigContent(simulationSettings.configName);
          configData = userConfigContent.content;
        }
      }

      await startSimulation({
        interface: simulationSettings.selectedInterface,
        configPath: configPath,
        configData: configData,
      });

      setMessage({ tone: 'success', text: 'Simulation started successfully!' });
      setRefetchTrigger((t) => t + 1);
      setQuickUploadFile(null);
    } catch (err) {
      setMessage({ tone: 'error', text: getErrorMessage(err) });
    } finally {
      setStarting(false);
    }
  }, [simulationSettings, quickUploadFile]);

  const handleStop = useCallback(async () => {
    if (
      !window.confirm(
        'Are you sure you want to stop the simulation? This will interrupt the current run.',
      )
    ) {
      return;
    }

    setStopping(true);
    setMessage(null);

    try {
      await stopSimulation();
      setMessage({ tone: 'success', text: 'Simulation stopped' });
      setRefetchTrigger((t) => t + 1);
    } catch (err) {
      setMessage({ tone: 'error', text: getErrorMessage(err) });
    } finally {
      setStopping(false);
    }
  }, []);

  return (
    <div className="space-y-6">
      {/* Daemon Mode Warning */}
      {!isDaemonMode && (
        <Card className="border-yellow-500/30 bg-yellow-900/20">
          <CardContent className="space-y-3">
            <div className="flex items-start gap-3">
              <BellRing className="mt-1 h-5 w-5 text-yellow-400" />
              <div>
                <p className="font-semibold text-yellow-200">Daemon Mode Not Detected</p>
                <SmallText className="text-yellow-300/90">
                  To use simulation controls, start NIAC in daemon mode:
                </SmallText>
                <code className="mt-2 block rounded bg-black/40 p-3 font-mono text-sm text-yellow-100">
                  niac daemon --listen :8080 --token yourtoken
                </code>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Start Simulation Card */}
      {isDaemonMode && !simStatus?.running && (
        <Card className="border-white/5 bg-gradient-to-br from-violet-900/30 to-gray-900/70">
          <CardContent className="space-y-4">
            {/* Single-row action bar: interface + start. The interface
                dropdown takes the natural width of its content; Start sits
                immediately next to it so the action is obvious. */}
            <div className="flex flex-wrap items-end gap-3">
              <div className="flex items-center gap-3">
                <PlugZap className="h-6 w-6 text-violet-400" />
                <H2 className="mb-0">Start Simulation</H2>
              </div>
              <div className="ml-auto flex flex-wrap items-end gap-3">
                <div className="min-w-[14rem]">
                  <label htmlFor="rc-interface" className="block text-xs text-gray-400">
                    Network interface
                  </label>
                  <div className="relative">
                    <Network className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-500" />
                    <select
                      id="rc-interface"
                      value={simulationSettings.selectedInterface}
                      onChange={handleInterfaceChange}
                      disabled={interfacesLoading || interfaces.length === 0}
                      title="Pick the host interface the daemon should bind to. Loopback (lo0/lo) is safest for local testing."
                      className="w-full rounded border border-white/10 bg-gray-800 py-2 pl-10 pr-3 text-sm text-white focus:border-violet-400 focus:outline-none disabled:cursor-not-allowed disabled:opacity-50"
                    >
                      {interfacesLoading && <option value="">Loading…</option>}
                      {!interfacesLoading && interfaces.length === 0 && (
                        <option value="">No usable interfaces</option>
                      )}
                      {!interfacesLoading && interfaces.length > 0 && (
                        <option value="">Select interface…</option>
                      )}
                      {interfaces.map((iface) => (
                        <option key={iface.name} value={iface.name}>
                          {iface.name}
                          {iface.addresses.length > 0 ? ` (${iface.addresses[0]})` : ''}
                        </option>
                      ))}
                    </select>
                  </div>
                </div>
                <Button
                  tone="violet"
                  size="lg"
                  disabled={!hasValidConfig || starting}
                  onClick={handleStart}
                  leftIcon={<Activity className="h-5 w-5" />}
                  // Pulse the button when everything's picked so it's the obvious next click.
                  className={
                    hasValidConfig && !starting
                      ? 'animate-pulse shadow-lg shadow-violet-500/30'
                      : undefined
                  }
                  title={
                    !simulationSettings.selectedInterface
                      ? 'Pick a network interface first'
                      : !simulationSettings.configName && !quickUploadFile
                        ? 'Pick a config below first'
                        : 'Start the simulation on the selected interface with the picked config'
                  }
                >
                  {starting ? 'Starting…' : 'Start Simulation'}
                </Button>
              </div>
            </div>

            {/* Picked-config status pill */}
            <div className="flex items-center justify-between rounded border border-white/5 bg-gray-900/40 px-3 py-2 text-xs">
              <span className="text-gray-400">Configuration:</span>
              {simulationSettings.configName || quickUploadFile ? (
                <SmallText className="text-emerald-300">
                  {quickUploadFile
                    ? `${quickUploadFile.name} (upload)`
                    : `${simulationSettings.configName} (${simulationSettings.configSource === 'template' ? 'template' : 'config'})`}
                </SmallText>
              ) : (
                <SmallText className="italic text-gray-500">
                  Pick one below — Start unlocks once you do
                </SmallText>
              )}
            </div>

            <ConfigPicker
              selection={{
                source: quickUploadFile ? 'upload' : (simulationSettings.configSource ?? null),
                name: quickUploadFile ? quickUploadFile.name : simulationSettings.configName,
                path: simulationSettings.configPath,
              }}
              onSelectTemplate={handleSelectTemplate}
              onSelectUserConfig={handleSelectUserConfig}
              onUpload={handleUpload}
              uploadFile={quickUploadFile}
            />

            {message && (
              <SmallText
                className={message.tone === 'success' ? 'text-emerald-300' : 'text-red-400'}
                role="alert"
                aria-live="polite"
              >
                {message.text}
              </SmallText>
            )}
          </CardContent>
        </Card>
      )}

      {isDaemonMode && simStatus?.running && (
        <RunningSimulationCard
          simStatus={simStatus}
          stopping={stopping}
          onStop={handleStop}
          message={message}
        />
      )}

      <AdvancedSection />
    </div>
  );
};

interface RunningSimulationCardProps {
  simStatus: {
    interface?: string;
    configName?: string;
    configPath?: string;
    deviceCount: number;
    uptimeSeconds: number;
    startedAt?: string;
  };
  stopping: boolean;
  onStop: () => void;
  message: { tone: 'success' | 'error'; text: string } | null;
}

/**
 * RunningSimulationCard — the green "simulation is live" card. Extracted
 * from the main render to keep RuntimeControlPage's cognitive complexity
 * under the project gate.
 */
const RunningSimulationCard: FC<RunningSimulationCardProps> = ({
  simStatus,
  stopping,
  onStop,
  message,
}) => {
  const navigate = useNavigate();

  const handleDownload = async () => {
    try {
      const doc = await fetchConfig();
      const blob = new Blob([doc.content], { type: 'application/x-yaml' });
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = doc.filename || 'niac-config.yaml';
      link.click();
      URL.revokeObjectURL(url);
    } catch (err) {
      console.error('Failed to download config:', err);
    }
  };

  return (
    <Card className="border-green-500/30 bg-gradient-to-br from-green-900/30 to-gray-900/70">
      <CardContent className="space-y-5">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="h-3 w-3 animate-pulse rounded-full bg-green-400" />
            <H2 className="mb-0">Simulation Running</H2>
          </div>
          <Tag colorScheme="green">ACTIVE</Tag>
        </div>

        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          <StatBlock
            label="Interface"
            value={simStatus.interface || '—'}
            helper="Network interface"
          />
          <StatBlock
            label="Config"
            value={simStatus.configName || '—'}
            helper={simStatus.configPath || 'Configuration file'}
          />
          <StatBlock
            label="Devices"
            value={simStatus.deviceCount.toString()}
            helper="Simulated devices"
          />
          <StatBlock
            label="Uptime"
            value={formatUptime(simStatus.uptimeSeconds)}
            helper="Time running"
          />
          <StatBlock
            label="Started"
            value={simStatus.startedAt ? formatTime(simStatus.startedAt) : '—'}
            helper="Start time"
          />
        </div>

        {message && (
          <SmallText className={message.tone === 'success' ? 'text-emerald-300' : 'text-red-400'}>
            {message.text}
          </SmallText>
        )}

        <div className="flex flex-wrap gap-3">
          <Button
            variant="outline"
            disabled={stopping}
            onClick={onStop}
            leftIcon={<Activity className="h-4 w-4" />}
          >
            {stopping ? 'Stopping…' : 'Stop Simulation'}
          </Button>
          <Button
            variant="ghost"
            leftIcon={<FileCog className="h-4 w-4" />}
            onClick={() => navigate('/devices')}
          >
            View Devices
          </Button>
          <Button
            variant="ghost"
            leftIcon={<Download className="h-4 w-4" />}
            onClick={handleDownload}
            title="Download the running config as YAML (the equivalent of niac config dump)."
          >
            Download YAML
          </Button>
        </div>
      </CardContent>
    </Card>
  );
};

/**
 * AdvancedSection — collapsible "Show advanced" container for power-user
 * knobs (currently just the global debug-level selector). Hidden by
 * default so the page reads as a clean Start/Stop flow for the 90% case.
 */
const AdvancedSection: FC = () => {
  const [open, setOpen] = useState(false);
  return (
    <details
      className="rounded border border-white/10 bg-gray-950/40"
      open={open}
      onToggle={(e) => setOpen(e.currentTarget.open)}
    >
      <summary className="flex cursor-pointer items-center gap-2 px-3 py-2 text-sm text-gray-300 hover:text-white">
        <span className="text-gray-500">{open ? '▾' : '▸'}</span>
        <span>Advanced</span>
        <SmallText className="text-gray-500">(global protocol debug level)</SmallText>
      </summary>
      <div className="border-t border-white/10 p-3">
        <GlobalDebugLevelCard />
      </div>
    </details>
  );
};

/**
 * GlobalDebugLevelCard — applies the same DebugLevel to every protocol in the
 * running stack via PUT /api/v1/debug/levels. CLI parity for the
 * --debug/--verbose/--quiet family of flags. The Protocol Debug page is
 * still the canonical place for per-protocol fine-tuning; this is the 90%
 * case for users who just want "loud" or "quiet" globally.
 */
const DEBUG_LEVELS: DebugLevel[] = ['OFF', 'ERROR', 'WARN', 'INFO', 'DEBUG', 'TRACE'];

const GlobalDebugLevelCard: FC = () => {
  const { data, refetch } = useApiResource(fetchProtocolDebugLevels, [], {
    intervalMs: 0,
  });
  const [pending, setPending] = useState<DebugLevel | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const current: DebugLevel = pending ?? data?.defaultLevel ?? 'INFO';

  const apply = async (level: DebugLevel) => {
    if (!data) return;
    setBusy(true);
    setError(null);
    try {
      await updateProtocolDebugLevels({
        protocols: data.protocols.map((p) => ({ protocol: p.protocol, level })),
      });
      setPending(null);
      await refetch();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <Card className="border-white/5 bg-gray-900/70">
      <CardContent className="space-y-3">
        <H2 className="mb-0 flex items-center gap-2 text-lg">
          <Activity className="h-5 w-5 text-violet-300" />
          Debug level
        </H2>
        <SmallText className="text-gray-400">
          Sets every protocol to the same level. Use Protocol Debug for per-protocol tuning.
        </SmallText>
        <div className="flex flex-wrap items-center gap-3">
          <select
            value={current}
            onChange={(e) => setPending(e.target.value as DebugLevel)}
            disabled={busy || !data}
            className="rounded border border-white/10 bg-gray-950/60 px-3 py-1.5 text-sm text-gray-100 focus:border-violet-400 focus:outline-none disabled:opacity-50"
            aria-label="Global debug level"
            title="Applies to every protocol in the running stack. OFF silences everything; TRACE is the loudest."
          >
            {DEBUG_LEVELS.map((lvl) => (
              <option key={lvl} value={lvl}>
                {lvl}
              </option>
            ))}
          </select>
          <Button
            tone="violet"
            disabled={busy || pending === null || !data}
            onClick={() => void apply(current)}
            title="PUT /api/v1/debug/levels with every protocol set to the chosen level."
          >
            {busy ? 'Applying…' : 'Apply'}
          </Button>
          {error && (
            <SmallText className="text-red-300" role="alert">
              {error}
            </SmallText>
          )}
        </div>
      </CardContent>
    </Card>
  );
};

export default RuntimeControlPage;
