import { Network } from 'lucide-react';
import { type FC, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { fetchUsableInterfaces } from '../../api/client';
import type { NetworkInterface, Template, UserConfig } from '../../api/types';
import { iconSizes } from '../../constants/sizes';
import { Card, CardContent } from '../../ui/Card';
import { SmallText } from '../../ui/Typography';
import { ConfigPicker } from '../simulation/ConfigPicker';
import type { WizardState } from './wizard-types';

interface TemplateStepProps {
  state: WizardState;
  onSelectTemplate: (template: Template) => void;
  onSelectUserConfig: (config: UserConfig) => void;
  onUpload: (file: File | null) => void;
  onSelectEmpty: () => void;
  onInterfaceChange: (iface: string) => void;
}

/**
 * Step 1 — pick a starting config (built-in template, saved config, local
 * upload, or blank) and the network interface the simulation will run on.
 * Composes the same `ConfigPicker` used on the Simulation page; the
 * interface picker mirrors `RuntimeControlPage`'s inline `<select>` since
 * the daemon requires an interface up front (there's no "load config
 * without starting" endpoint to defer it to a later step).
 */
export const TemplateStep: FC<TemplateStepProps> = ({
  state,
  onSelectTemplate,
  onSelectUserConfig,
  onUpload,
  onSelectEmpty,
  onInterfaceChange,
}) => {
  const { t } = useTranslation('pages');
  const [interfaces, setInterfaces] = useState<NetworkInterface[]>([]);
  const [loading, setLoading] = useState(true);

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
    source: state.uploadFile ? ('upload' as const) : state.source === 'empty' ? null : state.source,
    name: state.uploadFile
      ? state.uploadFile.name
      : (state.template?.name ?? state.userConfig?.name ?? ''),
    path: state.userConfig?.path,
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
              onClick={onSelectEmpty}
              className={`rounded border px-3 py-row text-xs font-medium ${
                state.source === 'empty'
                  ? 'border-brand-accent bg-brand-primary/20 text-brand-accent'
                  : 'border-surface-border bg-bg-surface/60 text-text-primary hover:bg-surface-hover'
              }`}
            >
              {t('newSimWizard.template.startEmpty')}
            </button>
          </div>
          <SmallText className="text-text-muted">{t('newSimWizard.template.help')}</SmallText>
        </CardContent>
      </Card>

      <ConfigPicker
        selection={selection}
        onSelectTemplate={onSelectTemplate}
        onSelectUserConfig={onSelectUserConfig}
        onUpload={onUpload}
        uploadFile={state.uploadFile}
      />
    </div>
  );
};
