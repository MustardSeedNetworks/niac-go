import { Activity, BellRing, Download, FileCog, FileUp, PlugZap, Settings } from 'lucide-react';
import { type FC, useCallback, useState } from 'react';
import {
  fetchConfig,
  fetchProtocolDebugLevels,
  fetchSimulationStatus,
  fetchTemplateContent,
  fetchUserConfigContent,
  startSimulation,
  stopSimulation,
  updateProtocolDebugLevels,
} from '../api/client';
import type { DebugLevel } from '../api/types';
import { StatBlock } from '../components/StatBlock';
import { POLL_INTERVALS } from '../constants/polling';
import { useApiResource } from '../hooks/useApiResource';
import { useUIStore } from '../stores/ui-store';
import { Button } from '../ui/Button';
import { Card, CardContent } from '../ui/Card';
import { Tag } from '../ui/Tag';
import { H2, SmallText } from '../ui/Typography';
import { fileToText } from '../utils/file';
import { formatBytes, formatTime, formatUptime, getErrorMessage } from '../utils/format';

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

  const { simulationSettings, openModal } = useUIStore();
  const [quickUploadFile, setQuickUploadFile] = useState<File | null>(null);
  const [starting, setStarting] = useState(false);
  const [stopping, setStopping] = useState(false);
  const [message, setMessage] = useState<{
    tone: 'success' | 'error';
    text: string;
  } | null>(null);

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

  const handleQuickUpload = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) {
      setQuickUploadFile(null);
      return;
    }

    const MaxSize = 10 * 1024 * 1024;
    if (file.size > MaxSize) {
      setMessage({
        tone: 'error',
        text: `File too large. Maximum size is ${formatBytes(MaxSize)}`,
      });
      e.target.value = '';
      return;
    }

    if (!file.name.match(/\.(yaml|yml)$/i)) {
      setMessage({
        tone: 'error',
        text: 'Please select a YAML file (.yaml or .yml)',
      });
      e.target.value = '';
      return;
    }

    setQuickUploadFile(file);
    setMessage(null);
  }, []);

  const openSettings = useCallback(() => {
    openModal('settings');
  }, [openModal]);

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
          <CardContent className="space-y-5">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <PlugZap className="h-6 w-6 text-violet-400" />
                <H2 className="mb-0">Start Simulation</H2>
              </div>
              <Button
                variant="ghost"
                size="sm"
                leftIcon={<Settings className="h-4 w-4" />}
                onClick={openSettings}
              >
                Configure
              </Button>
            </div>

            {/* Current Settings Display */}
            <div className="p-4 bg-gray-900/50 rounded-lg border border-white/5 space-y-3">
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-400">Interface:</span>
                <span className="text-sm text-white font-medium">
                  {simulationSettings.selectedInterface || (
                    <span className="text-gray-500 italic">Not selected</span>
                  )}
                </span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-400">Configuration:</span>
                <span className="text-sm text-white font-medium">
                  {quickUploadFile ? (
                    <>
                      {quickUploadFile.name}
                      <span className="text-gray-500 ml-1">(upload)</span>
                    </>
                  ) : simulationSettings.configName ? (
                    <>
                      {simulationSettings.configName}
                      <span className="text-gray-500 ml-1">
                        ({simulationSettings.configSource === 'template' ? 'template' : 'config'})
                      </span>
                    </>
                  ) : (
                    <span className="text-gray-500 italic">Not selected</span>
                  )}
                </span>
              </div>
            </div>

            {/* Quick Override Upload */}
            <div className="space-y-2">
              <span className="block text-sm text-gray-400">Quick Override (optional)</span>
              <div className="flex items-center gap-3">
                <label
                  htmlFor="quick-upload"
                  className="flex items-center gap-2 px-4 py-2 bg-gray-800 hover:bg-gray-700 border border-white/10 rounded-lg cursor-pointer transition-colors"
                >
                  <FileUp className="h-4 w-4 text-gray-400" />
                  <span className="text-sm text-white">
                    {quickUploadFile ? 'Change File' : 'Browse...'}
                  </span>
                </label>
                <input
                  id="quick-upload"
                  type="file"
                  accept=".yaml,.yml"
                  onChange={handleQuickUpload}
                  className="sr-only"
                />
                {quickUploadFile && (
                  <button
                    type="button"
                    onClick={() => setQuickUploadFile(null)}
                    className="text-sm text-red-400 hover:text-red-300"
                  >
                    Clear
                  </button>
                )}
              </div>
              <SmallText className="text-gray-500">
                Upload a config to use instead of the saved configuration.
              </SmallText>
            </div>

            {/* Messages */}
            {message && (
              <SmallText
                className={message.tone === 'success' ? 'text-emerald-300' : 'text-red-400'}
                role="alert"
                aria-live="polite"
              >
                {message.text}
              </SmallText>
            )}

            {/* Start Button */}
            <Button
              tone="violet"
              size="lg"
              disabled={!hasValidConfig || starting}
              onClick={handleStart}
              leftIcon={<Activity className="h-5 w-5" />}
            >
              {starting ? 'Starting...' : 'Start Simulation'}
            </Button>
          </CardContent>
        </Card>
      )}

      {/* Running Simulation Card */}
      {isDaemonMode && simStatus?.running && (
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
              <SmallText
                className={message.tone === 'success' ? 'text-emerald-300' : 'text-red-400'}
              >
                {message.text}
              </SmallText>
            )}

            <div className="flex flex-wrap gap-3">
              <Button
                variant="outline"
                disabled={stopping}
                onClick={handleStop}
                leftIcon={<Activity className="h-4 w-4" />}
              >
                {stopping ? 'Stopping...' : 'Stop Simulation'}
              </Button>
              <Button
                variant="ghost"
                leftIcon={<FileCog className="h-4 w-4" />}
                onClick={() => {
                  window.location.href = '/devices';
                }}
              >
                View Devices
              </Button>
              <Button
                variant="ghost"
                leftIcon={<Download className="h-4 w-4" />}
                onClick={async () => {
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
                }}
                title="Download the running config as YAML (the equivalent of niac config dump)."
              >
                Download YAML
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      <GlobalDebugLevelCard />
    </div>
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
