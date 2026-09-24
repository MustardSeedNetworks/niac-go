import { Network } from 'lucide-react';
import { type FC, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { fetchUsableInterfaces } from '../../api/client';
import type { ScenarioPack } from '../../api/scenario-client';
import type { LibraryNetwork, NetworkInterface, Template } from '../../api/types';
import { iconSizes } from '../../constants/sizes';
import { Card, CardContent } from '../../ui/Card';
import { SmallText } from '../../ui/Typography';
import { ConfigPicker } from '../simulation/ConfigPicker';
import { FleetGeneratorCard } from './FleetGeneratorCard';
import type { WizardState } from './wizard-types';

interface TemplateStepProps {
  state: WizardState;
  onSelectTemplate: (template: Template) => void;
  onSelectUserConfig: (config: LibraryNetwork) => void;
  onUpload: (file: File | null) => void;
  onSelectEmpty: () => void;
  onSelectFleet: () => void;
  onSelectPack: (pack: ScenarioPack) => void;
  onFleetChange: (request: WizardState['fleetRequest']) => void;
  onInterfaceChange: (iface: string) => void;
}

/**
 * Step 1 — pick a starting config (built-in template, saved config, local
 * upload, or blank) and the network interface the simulation will use after
 * authoring and preflight. The selected content is copied into a revisioned
 * draft; this step never changes the daemon's active configuration.
 */
export const TemplateStep: FC<TemplateStepProps> = ({
  state,
  onSelectTemplate,
  onSelectUserConfig,
  onUpload,
  onSelectEmpty,
  onSelectFleet,
  onSelectPack,
  onFleetChange,
  onInterfaceChange,
}) => {
  const { t } = useTranslation('pages');
  const [interfaces, setInterfaces] = useState<NetworkInterface[]>([]);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState<'start' | 'library'>('start');
  const startTab = useRef<HTMLButtonElement>(null);
  const libraryTab = useRef<HTMLButtonElement>(null);
  const returnToStart = () => {
    setTab('start');
    startTab.current?.focus();
  };
  const openLibrary = () => {
    setTab('library');
    libraryTab.current?.focus();
  };

  useEffect(() => {
    let cancelled = false;
    fetchUsableInterfaces()
      .then((resp) => {
        if (!cancelled) setInterfaces(resp.interfaces);
      })
      .catch(() => {
        // Best-effort — an empty list still lets the daemon-mode warning
        // upstream explain what's missing.
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const selection = {
    source:
      state.uploadFile || state.source === 'upload'
        ? ('upload' as const)
        : state.source === 'template' || state.source === 'userConfig'
          ? state.source
          : null,
    name: state.uploadFile
      ? state.uploadFile.name
      : (state.template?.name ?? state.userConfig?.name ?? ''),
  };

  return (
    <div className="stack-lg">
      <Card className="border-surface-border bg-bg-surface/70">
        <CardContent className="stack">
          <div className="flex items-center gap-default">
            <Network className={`${iconSizes.lg} text-brand-accent`} />
            <div className="min-w-[14rem] flex-1">
              <label htmlFor="wizard-interface" className="block text-xs text-text-muted">
                {t('newSimWizard.template.interfaceLabel')}
              </label>
              <select
                id="wizard-interface"
                data-testid="wizard-interface-select"
                value={state.selectedInterface}
                onChange={(e) => onInterfaceChange(e.target.value)}
                disabled={loading || interfaces.length === 0}
                className="w-full rounded border border-surface-border bg-bg-elevated py-row px-3 text-sm text-text-primary focus:border-brand-accent focus:outline-none disabled:cursor-not-allowed disabled:opacity-50"
              >
                {loading && <option value="">{t('runtime.interfaceLoadingOption')}</option>}
                {!loading && interfaces.length === 0 && (
                  <option value="">{t('runtime.noUsableInterfaces')}</option>
                )}
                {!loading && interfaces.length > 0 && (
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
            <button
              type="button"
              data-testid="wizard-start-empty"
              data-wizard-source="empty"
              aria-pressed={state.source === 'empty'}
              onClick={onSelectEmpty}
              className={`rounded border px-3 py-row text-xs font-medium ${
                state.source === 'empty'
                  ? 'border-brand-accent bg-brand-primary/20 text-brand-primary-strong'
                  : 'border-surface-border bg-bg-surface/60 text-text-primary hover:bg-surface-hover'
              }`}
            >
              {t('newSimWizard.template.startEmpty')}
            </button>
          </div>
          <SmallText className="text-text-muted">{t('newSimWizard.template.help')}</SmallText>
        </CardContent>
      </Card>

      <div
        role="tablist"
        aria-label={t('newSimWizard.template.startingPoint')}
        className="flex gap-compact"
      >
        {(['start', 'library'] as const).map((value) => (
          <button
            key={value}
            ref={value === 'start' ? startTab : libraryTab}
            type="button"
            id={`wizard-source-tab-${value}`}
            data-testid={`wizard-source-tab-${value}`}
            role="tab"
            aria-selected={tab === value}
            aria-controls="wizard-source-panel"
            tabIndex={tab === value ? 0 : -1}
            onClick={() => setTab(value)}
            onKeyDown={(event) => {
              if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return;
              event.preventDefault();
              const next =
                event.key === 'Home'
                  ? 'start'
                  : event.key === 'End'
                    ? 'library'
                    : tab === 'start'
                      ? 'library'
                      : 'start';
              if (next === 'start') returnToStart();
              else openLibrary();
            }}
            className={`min-h-11 rounded border px-4 py-row text-sm focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-primary ${tab === value ? 'border-brand-accent text-brand-primary-strong bg-brand-primary/10' : 'border-surface-border text-text-primary'}`}
          >
            {value === 'start'
              ? t('newSimWizard.template.startingPoint')
              : t('newSimWizard.template.browseLibrary')}
          </button>
        ))}
      </div>
      <div
        id="wizard-source-panel"
        role="tabpanel"
        aria-labelledby={`wizard-source-tab-${tab}`}
        data-testid="wizard-source-panel"
        className="stack-lg"
      >
        {tab === 'start' ? (
          <>
            {selection.source && (
              <button
                type="button"
                data-wizard-source="library"
                aria-pressed="true"
                onClick={openLibrary}
                data-testid="wizard-selected-library"
                className="rounded border border-brand-accent bg-brand-primary/10 p-4 text-left text-text-primary focus-visible:outline-2 focus-visible:outline-brand-primary"
              >
                {t('newSimWizard.template.librarySelection', { name: selection.name })}
              </button>
            )}
            <FleetGeneratorCard
              request={state.fleetRequest}
              selected={state.source === 'generated' && state.fleetPackId === null}
              selectedPackId={state.source === 'generated' ? state.fleetPackId : null}
              onChange={onFleetChange}
              onSelect={onSelectFleet}
              onSelectPack={onSelectPack}
            />
          </>
        ) : (
          <ConfigPicker
            selection={selection}
            onSelectTemplate={(template) => {
              onSelectTemplate(template);
              returnToStart();
            }}
            onSelectUserConfig={(config) => {
              onSelectUserConfig(config);
              returnToStart();
            }}
            onUpload={(file) => {
              onUpload(file);
              if (file) returnToStart();
            }}
            uploadFile={state.uploadFile}
            filterByDeviceFamily
          />
        )}
      </div>
    </div>
  );
};
