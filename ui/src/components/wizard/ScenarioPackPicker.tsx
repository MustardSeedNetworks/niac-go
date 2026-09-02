import { Boxes } from 'lucide-react';
import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import { fetchScenarioPacks, type ScenarioGenerateRequest } from '../../api/scenario-client';
import { iconSizes } from '../../constants/sizes';
import { useApiResource } from '../../hooks/useApiResource';
import { Button } from '../../ui/Button';
import { SmallText } from '../../ui/Typography';

interface ScenarioPackPickerProps {
  request: ScenarioGenerateRequest;
  onChange: (request: ScenarioGenerateRequest) => void;
}

const ScenarioPackPickerContent: FC<ScenarioPackPickerProps> = ({ request, onChange }) => {
  const { t } = useTranslation('pages');
  const { t: tCommon } = useTranslation('common');
  const { data: packs, loading, error, refetch } = useApiResource(fetchScenarioPacks, []);
  const selected = packs?.find((pack) => JSON.stringify(pack.request) === JSON.stringify(request));

  const renderPacks = (purpose: 'presentation' | 'stress') =>
    packs
      ?.filter((pack) => pack.mapPurpose === purpose)
      .map((pack) => (
        <button
          key={pack.id}
          type="button"
          data-testid={`scenario-pack-${pack.id}`}
          aria-pressed={selected?.id === pack.id}
          className={`min-h-11 rounded-lg border p-4 text-left transition-colors focus:outline-none focus:ring-2 focus:ring-brand-primary ${
            selected?.id === pack.id
              ? 'border-brand-accent bg-brand-primary/15'
              : 'border-surface-border bg-bg-base/50 hover:bg-surface-hover'
          }`}
          onClick={() => onChange(pack.request)}
        >
          <span className="block font-medium text-text-primary">
            {t(`newSimWizard.fleet.packMetadata.${pack.id}.name`, { defaultValue: pack.name })}
          </span>
          <SmallText className="mt-tight block text-text-muted">
            {t(`newSimWizard.fleet.packMetadata.${pack.id}.description`, {
              defaultValue: pack.description,
            })}
          </SmallText>
          <SmallText className="mt-tight block text-brand-accent">
            {t('newSimWizard.fleet.packSummary', {
              devices: pack.manifest.deviceCount,
              links: pack.manifest.linkCount,
              version: pack.version,
            })}
          </SmallText>
        </button>
      ));

  return (
    <div className="stack-sm">
      <div className="flex items-start gap-default">
        <Boxes className={`${iconSizes.lg} text-brand-accent`} />
        <div>
          <p className="text-sm font-medium text-text-primary">
            {t('newSimWizard.fleet.packsTitle')}
          </p>
          <SmallText className="text-text-muted">{t('newSimWizard.fleet.packsHelp')}</SmallText>
        </div>
      </div>
      {loading && (
        <SmallText className="text-text-muted">{t('newSimWizard.fleet.packsLoading')}</SmallText>
      )}
      {error && (
        <div className="flex items-center gap-default" role="alert">
          <SmallText className="text-status-error">{t('newSimWizard.fleet.packsError')}</SmallText>
          <Button size="sm" variant="outline" onClick={() => void refetch()}>
            {tCommon('buttons.retry')}
          </Button>
        </div>
      )}
      {packs && (
        <div className="stack-default">
          <section className="stack-sm" aria-labelledby="presentation-map-packs">
            <p id="presentation-map-packs" className="text-sm font-medium text-text-primary">
              {t('newSimWizard.fleet.presentationPacksTitle')}
            </p>
            <div className="grid gap-default md:grid-cols-2 xl:grid-cols-3">
              {renderPacks('presentation')}
            </div>
          </section>
          <section className="stack-sm" aria-labelledby="stress-map-packs">
            <div>
              <p id="stress-map-packs" className="text-sm font-medium text-text-primary">
                {t('newSimWizard.fleet.stressPacksTitle')}
              </p>
              <SmallText className="text-text-muted">
                {t('newSimWizard.fleet.stressPacksHelp')}
              </SmallText>
            </div>
            <div className="grid gap-default md:grid-cols-2 xl:grid-cols-3">
              {renderPacks('stress')}
            </div>
          </section>
        </div>
      )}
    </div>
  );
};

export const ScenarioPackPicker: FC<ScenarioPackPickerProps> = (props) => (
  <ScenarioPackPickerContent {...props} />
);
