import { Activity, BellRing, Network, PlugZap } from 'lucide-react';
import { type FC, useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  fetchSimulationStatus,
  fetchUsableInterfaces,
  fetchUserConfigContent,
  startSimulation,
  stopSimulation,
} from '../api/client';
import type { NetworkInterface, Template, UserConfig } from '../api/types';
import { ConfigPicker } from '../components/simulation/ConfigPicker';
import { POLL_INTERVALS } from '../constants/polling';
import { iconSizes } from '../constants/sizes';
import { useApiResource } from '../hooks/useApiResource';
import { useUIStore } from '../stores/ui-store';
import { Button } from '../ui/Button';
import { Card, CardContent } from '../ui/Card';
import { H2, SmallText } from '../ui/Typography';
import { fileToText } from '../utils/file';
import { getErrorMessage } from '../utils/format';
import { AdvancedSection } from './runtime/AdvancedSection';
import { RunningSimulationCard } from './runtime/RunningSimulationCard';
import { SelectedNetworkPreview } from './runtime/SelectedNetworkPreview';

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
  const { t } = useTranslation('pages');
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
      let templateName: string | undefined;

      // Handle quick upload file
      if (quickUploadFile) {
        configData = await fileToText(quickUploadFile);
      }
      // Handle template — pass the template name so the daemon loads
      // the YAML directly from disk. This preserves the template's own
      // directory as the include_path base, which matters for vendor
      // templates that reference walk files via relative paths.
      // Fetching the content and sending it inline used to trip the
      // walk-file path-traversal guard for templates like
      // vendors/paloalto-firewall.yaml.
      else if (simulationSettings.configSource === 'template') {
        templateName = simulationSettings.configName;
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
        templateName: templateName,
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
    <div className="stack-xl">
      {/* Daemon Mode Warning */}
      {!isDaemonMode && (
        <Card className="border-status-warning/30 bg-status-warning/20">
          <CardContent className="stack">
            <div className="flex items-start gap-default">
              <BellRing className={`mt-tight ${iconSizes.lg} text-status-warning`} />
              <div>
                <p className="font-semibold text-status-warning">
                  {t('runtime.daemonModeWarning')}
                </p>
                <SmallText className="text-status-warning/90">
                  To use simulation controls, start NIAC in daemon mode:
                </SmallText>
                <code className="mt-inline block rounded bg-scrim/40 pad-sm font-mono text-sm text-status-warning">
                  niac daemon --listen :8080 --token yourtoken
                </code>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Start Simulation Card */}
      {isDaemonMode && !simStatus?.running && (
        <Card className="border-surface-border bg-gradient-to-br from-brand-primary/30 to-bg-surface/70">
          <CardContent className="stack-lg">
            {/* Single-row action bar: interface + start. The interface
                dropdown takes the natural width of its content; Start sits
                immediately next to it so the action is obvious. */}
            <div className="flex flex-wrap items-end gap-default">
              <div className="flex items-center gap-default">
                <PlugZap className={`${iconSizes.xl} text-brand-accent`} />
                <H2>{t('runtime.startSimulationTitle')}</H2>
              </div>
              <div className="ml-auto flex flex-wrap items-end gap-default">
                <div className="min-w-[14rem]">
                  <label htmlFor="rc-interface" className="block text-xs text-text-muted">
                    Network interface
                  </label>
                  <div className="relative">
                    <Network
                      className={`absolute left-3 top-1/2 -translate-y-1/2 ${iconSizes.md} text-text-muted`}
                    />
                    <select
                      id="rc-interface"
                      value={simulationSettings.selectedInterface}
                      onChange={handleInterfaceChange}
                      disabled={interfacesLoading || interfaces.length === 0}
                      title="Pick the host interface the daemon should bind to. Loopback (lo0/lo) is safest for local testing."
                      className="w-full rounded border border-surface-border bg-bg-elevated py-row pl-10 pr-3 text-sm text-text-primary focus:border-brand-accent focus:outline-none disabled:cursor-not-allowed disabled:opacity-50"
                    >
                      {interfacesLoading && <option value="">Loading…</option>}
                      {!interfacesLoading && interfaces.length === 0 && (
                        <option value="">{t('runtime.noUsableInterfaces')}</option>
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
                  size="md"
                  disabled={!hasValidConfig || starting}
                  onClick={handleStart}
                  leftIcon={<Activity className={iconSizes.md} />}
                  // Pulse the button when everything's picked so it's the obvious next click.
                  className={
                    hasValidConfig && !starting
                      ? 'animate-pulse shadow-lg shadow-brand-primary/30'
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
            <div className="flex-between rounded border border-surface-border bg-bg-surface/40 px-3 py-row text-xs">
              <span className="text-text-muted">Configuration:</span>
              {simulationSettings.configName || quickUploadFile ? (
                <SmallText className="text-status-success">
                  {quickUploadFile
                    ? `${quickUploadFile.name} (upload)`
                    : `${simulationSettings.configName} (${simulationSettings.configSource === 'template' ? 'template' : 'config'})`}
                </SmallText>
              ) : (
                <SmallText className="italic text-text-muted">
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

            {(simulationSettings.configSource || quickUploadFile) && (
              <SelectedNetworkPreview
                source={quickUploadFile ? 'upload' : (simulationSettings.configSource ?? null)}
                name={quickUploadFile ? quickUploadFile.name : simulationSettings.configName}
                uploadFile={quickUploadFile}
              />
            )}

            {message && (
              <SmallText
                className={message.tone === 'success' ? 'text-status-success' : 'text-status-error'}
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
