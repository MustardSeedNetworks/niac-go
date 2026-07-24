import { Activity, BellRing, Network, PlugZap } from 'lucide-react';
import { type FC, useCallback, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { fetchUsableInterfaces, startSimulation, stopSimulation } from '../api/client';
import { fetchLibraryNetworkContent } from '../api/library-client';
import type { LibraryNetwork, NetworkInterface, SimulationRequest, Template } from '../api/types';
import { ConfigPicker } from '../components/simulation/ConfigPicker';
import { PreflightStep } from '../components/wizard/PreflightStep';
import { iconSizes } from '../constants/sizes';
import { useErrorToast } from '../hooks/useErrorToast';
import { useSimulationStatus } from '../hooks/useSimulationStatus';
import { useUIStore } from '../stores/ui-store';
import { Button } from '../ui/Button';
import { Card, CardContent } from '../ui/Card';
import { ConfirmModal } from '../ui/ConfirmModal';
import { H2, SmallText } from '../ui/Typography';
import { fileToText } from '../utils/file';
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
  const { data: simStatus, refetch: refetchSimStatus } = useSimulationStatus();

  const { simulationSettings, setSimulationSettings } = useUIStore();
  const [interfaces, setInterfaces] = useState<NetworkInterface[]>([]);
  const [interfacesLoading, setInterfacesLoading] = useState(true);
  const [quickUploadFile, setQuickUploadFile] = useState<File | null>(null);
  const [preparedRequest, setPreparedRequest] = useState<{
    payload: SimulationRequest;
    sequence: number;
  } | null>(null);
  const [preparing, setPreparing] = useState(false);
  const [starting, setStarting] = useState(false);
  const [stopping, setStopping] = useState(false);
  const [showStopConfirm, setShowStopConfirm] = useState(false);
  // Success-only: failures are surfaced as toasts (see showError below), not
  // a page-level banner.
  const [successMessage, setSuccessMessage] = useState<string | null>(null);
  const showError = useErrorToast();
  const requestSequence = useRef(0);

  const invalidatePreparedRequest = useCallback(() => {
    requestSequence.current += 1;
    setPreparedRequest(null);
    setPreparing(false);
  }, []);

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
      invalidatePreparedRequest();
    },
    [invalidatePreparedRequest, setSimulationSettings],
  );

  const handleSelectTemplate = useCallback(
    (template: Template) => {
      setSimulationSettings({
        configSource: 'template',
        configName: template.name,
      });
      setQuickUploadFile(null);
      invalidatePreparedRequest();
    },
    [invalidatePreparedRequest, setSimulationSettings],
  );

  const handleSelectUserConfig = useCallback(
    (config: LibraryNetwork) => {
      setSimulationSettings({
        configSource: 'userConfig',
        configName: config.name,
      });
      setQuickUploadFile(null);
      invalidatePreparedRequest();
    },
    [invalidatePreparedRequest, setSimulationSettings],
  );

  const handleUpload = useCallback(
    (file: File | null) => {
      setQuickUploadFile(file);
      invalidatePreparedRequest();
      if (file) {
        // Picking an upload clears any previously-selected template / config so
        // the start handler doesn't get confused about which source wins.
        setSimulationSettings({
          configSource: 'upload',
          configName: '',
        });
        setSuccessMessage(null);
      }
    },
    [invalidatePreparedRequest, setSimulationSettings],
  );

  const isDaemonMode = simStatus !== null;
  const hasValidConfig =
    simulationSettings.selectedInterface && (simulationSettings.configName || quickUploadFile);

  const handlePrepare = useCallback(async () => {
    if (!simulationSettings.selectedInterface) {
      showError(new Error(t('runtime.errorNoInterface')));
      return;
    }

    if (!simulationSettings.configName && !quickUploadFile) {
      showError(new Error(t('runtime.errorNoConfig')));
      return;
    }

    const sequence = requestSequence.current + 1;
    requestSequence.current = sequence;
    setPreparing(true);
    setSuccessMessage(null);

    try {
      let configData: string | undefined;
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
      // Handle a saved library network — the library confines entries
      // behind an os.Root keyed by name (internal/library/list.go), so
      // there's no filesystem path to hand the daemon; fetch the content
      // and send it inline instead.
      else if (simulationSettings.configSource === 'userConfig') {
        const userConfigContent = await fetchLibraryNetworkContent(simulationSettings.configName);
        configData = userConfigContent.content;
      }

      if (requestSequence.current !== sequence) return;
      setPreparedRequest({
        payload: {
          interface: simulationSettings.selectedInterface,
          configData: configData,
          templateName: templateName,
        },
        sequence,
      });
    } catch (err) {
      if (requestSequence.current !== sequence) return;
      showError(err);
    } finally {
      if (requestSequence.current === sequence) setPreparing(false);
    }
  }, [simulationSettings, quickUploadFile, showError, t]);

  const handleStart = useCallback(
    async (request: SimulationRequest) => {
      setStarting(true);
      setSuccessMessage(null);
      try {
        await startSimulation(request);
        setSuccessMessage(t('runtime.startSuccess'));
        refetchSimStatus();
        setQuickUploadFile(null);
        invalidatePreparedRequest();
      } catch (err) {
        showError(err);
      } finally {
        setStarting(false);
      }
    },
    [invalidatePreparedRequest, refetchSimStatus, showError, t],
  );

  const handleStopClick = useCallback(() => {
    setShowStopConfirm(true);
  }, []);

  const handleStopConfirmed = useCallback(async () => {
    setShowStopConfirm(false);
    setStopping(true);
    setSuccessMessage(null);

    try {
      await stopSimulation();
      setSuccessMessage(t('runtime.stopSuccess'));
      refetchSimStatus();
    } catch (err) {
      showError(err);
    } finally {
      setStopping(false);
    }
  }, [showError, t]);

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
                  {t('runtime.daemonModeInstructions')}
                </SmallText>
                <code className="mt-inline block rounded bg-scrim/40 pad-sm font-mono text-sm text-status-warning">
                  niac daemon
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
                    {t('runtime.interfaceLabel')}
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
                      title={t('runtime.interfaceTitle')}
                      className="w-full rounded border border-surface-border bg-bg-elevated py-row pl-10 pr-3 text-sm text-text-primary focus:border-brand-accent focus:outline-none disabled:cursor-not-allowed disabled:opacity-50"
                    >
                      {interfacesLoading && (
                        <option value="">{t('runtime.interfaceLoadingOption')}</option>
                      )}
                      {!interfacesLoading && interfaces.length === 0 && (
                        <option value="">{t('runtime.noUsableInterfaces')}</option>
                      )}
                      {!interfacesLoading && interfaces.length > 0 && (
                        <option value="">{t('runtime.interfaceSelectPrompt')}</option>
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
                  disabled={!hasValidConfig || preparing || starting}
                  onClick={handlePrepare}
                  leftIcon={<Activity className={iconSizes.md} />}
                  // Pulse the button when everything's picked so it's the obvious next click.
                  className={
                    hasValidConfig && !preparing && !starting
                      ? 'animate-pulse shadow-lg shadow-brand-primary/30'
                      : undefined
                  }
                  title={
                    !simulationSettings.selectedInterface
                      ? t('runtime.startTitlePickInterface')
                      : !simulationSettings.configName && !quickUploadFile
                        ? t('runtime.startTitlePickConfig')
                        : t('runtime.startTitleReady')
                  }
                >
                  {preparing
                    ? t('runtime.preparingConnectionLabel')
                    : t('runtime.reviewConnectionLabel')}
                </Button>
              </div>
            </div>

            {/* Picked-config status pill */}
            <div className="flex-between rounded border border-surface-border bg-bg-surface/40 px-3 py-row text-xs">
              <span className="text-text-muted">{t('runtime.configurationLabel')}</span>
              {simulationSettings.configName || quickUploadFile ? (
                <SmallText className="text-status-success">
                  {quickUploadFile
                    ? `${quickUploadFile.name} ${t('runtime.configUploadSuffix')}`
                    : `${simulationSettings.configName} (${simulationSettings.configSource === 'template' ? t('runtime.configSourceTemplate') : t('runtime.configSourceConfig')})`}
                </SmallText>
              ) : (
                <SmallText className="italic text-text-muted">
                  {t('runtime.pickConfigHint')}
                </SmallText>
              )}
            </div>

            <ConfigPicker
              selection={{
                source: quickUploadFile ? 'upload' : (simulationSettings.configSource ?? null),
                name: quickUploadFile ? quickUploadFile.name : simulationSettings.configName,
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

            {preparedRequest && (
              <PreflightStep
                key={preparedRequest.sequence}
                request={preparedRequest.payload}
                onStart={(request) => void handleStart(request)}
                starting={starting}
              />
            )}

            {successMessage && (
              <SmallText className="text-status-success" role="status" aria-live="polite">
                {successMessage}
              </SmallText>
            )}
          </CardContent>
        </Card>
      )}

      {isDaemonMode && simStatus?.running && (
        <RunningSimulationCard
          simStatus={simStatus}
          stopping={stopping}
          onStop={handleStopClick}
          message={successMessage}
        />
      )}

      <AdvancedSection />

      <ConfirmModal
        isOpen={showStopConfirm}
        onConfirm={handleStopConfirmed}
        onCancel={() => setShowStopConfirm(false)}
        title={t('runtime.stopConfirmTitle')}
        message={t('runtime.stopConfirmMessage')}
        confirmLabel={t('runtime.stopConfirmLabel')}
      />
    </div>
  );
};
