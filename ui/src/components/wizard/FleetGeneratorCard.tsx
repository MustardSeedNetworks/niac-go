import { Network } from 'lucide-react';
import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import {
  isScenarioRequestValid,
  type ScenarioCounts,
  type ScenarioGenerateRequest,
} from '../../api/scenario-client';
import { iconSizes } from '../../constants/sizes';
import { Button } from '../../ui/Button';
import { Card, CardContent } from '../../ui/Card';
import { SmallText } from '../../ui/Typography';

interface FleetGeneratorCardProps {
  request: ScenarioGenerateRequest;
  selected: boolean;
  onChange: (request: ScenarioGenerateRequest) => void;
  onSelect: () => void;
}

const countFields: Array<keyof ScenarioCounts> = [
  'siteWanRouters',
  'distributionSwitches',
  'accessSwitches',
  'serverSwitches',
  'accessPointsPerAccess',
  'workstationsPerAccess',
  'wirelessControllers',
];

const inputClass =
  'min-h-11 w-full rounded border border-surface-border bg-bg-elevated px-3 py-row text-sm text-text-primary focus:border-brand-accent focus:outline-none focus:ring-2 focus:ring-brand-primary/60';

const countLimits: Record<keyof ScenarioCounts, { min: number; max: number; step?: number }> = {
  siteWanRouters: { min: 1, max: 2 },
  firewalls: { min: 1, max: 2 },
  coreSwitches: { min: 1, max: 2 },
  distributionSwitches: { min: 2, max: 8, step: 2 },
  accessSwitches: { min: 1, max: 20 },
  serverSwitches: { min: 1, max: 8 },
  accessPointsPerAccess: { min: 0, max: 9 },
  workstationsPerAccess: { min: 0, max: 39 },
  wirelessControllers: { min: 0, max: 8 },
};

export const FleetGeneratorCard: FC<FleetGeneratorCardProps> = ({
  request,
  selected,
  onChange,
  onSelect,
}) => {
  const { t } = useTranslation('pages');
  const updateCount = (field: keyof ScenarioCounts, value: number) => {
    const counts = { ...request.counts, [field]: value };
    if (field === 'siteWanRouters') {
      counts.firewalls = value;
      counts.coreSwitches = value;
    }
    if (field === 'accessSwitches') {
      counts.accessPointsPerAccess = Math.min(
        counts.accessPointsPerAccess,
        Math.floor(154 / value),
      );
      counts.workstationsPerAccess = Math.min(counts.workstationsPerAccess, Math.floor(79 / value));
    }
    onChange({ ...request, counts });
  };
  const updateSite = (
    index: number,
    field: 'code' | 'octet' | 'location',
    value: string | number,
  ) =>
    onChange({
      ...request,
      sites: request.sites.map((site, siteIndex) =>
        siteIndex === index ? { ...site, [field]: value } : site,
      ),
    });

  return (
    <Card className={selected ? 'border-brand-accent bg-brand-primary/10' : undefined}>
      <CardContent className="stack-lg">
        <div className="flex flex-wrap items-start justify-between gap-default">
          <div className="flex items-start gap-default">
            <Network className={`${iconSizes.lg} text-brand-accent`} />
            <div>
              <div className="font-medium text-text-primary">{t('newSimWizard.fleet.title')}</div>
              <SmallText className="text-text-muted">{t('newSimWizard.fleet.help')}</SmallText>
            </div>
          </div>
          <Button
            type="button"
            tone="violet"
            className="min-h-11"
            data-testid="wizard-select-fleet"
            disabled={!isScenarioRequestValid(request)}
            onClick={onSelect}
          >
            {selected ? t('newSimWizard.fleet.selected') : t('newSimWizard.fleet.select')}
          </Button>
        </div>

        <div className="grid gap-default md:grid-cols-3">
          {(['domain', 'snmpCommunity', 'attachmentName'] as const).map((field) => (
            <label key={field} className="stack-xs text-xs text-text-muted">
              {t(`newSimWizard.fleet.${field}`)}
              <input
                data-testid={`fleet-${field}`}
                className={inputClass}
                value={request[field]}
                maxLength={field === 'domain' ? 237 : field === 'snmpCommunity' ? 255 : 64}
                onChange={(event) => onChange({ ...request, [field]: event.target.value })}
              />
            </label>
          ))}
        </div>

        <div className="stack-sm">
          <div className="flex items-center justify-between gap-default">
            <span className="text-sm font-medium text-text-primary">
              {t('newSimWizard.fleet.sites')}
            </span>
            <div className="flex gap-sm">
              <Button
                type="button"
                variant="outline"
                className="min-h-11"
                disabled={request.sites.length === 1}
                onClick={() => onChange({ ...request, sites: request.sites.slice(0, -1) })}
              >
                {t('newSimWizard.fleet.removeSite')}
              </Button>
              <Button
                type="button"
                variant="outline"
                className="min-h-11"
                disabled={request.sites.length === 4}
                onClick={() => {
                  const next = request.sites.length + 1;
                  onChange({
                    ...request,
                    sites: [
                      ...request.sites,
                      { code: `SITE${next}`, octet: 239 + next, location: `Site ${next}` },
                    ],
                  });
                }}
              >
                {t('newSimWizard.fleet.addSite')}
              </Button>
            </div>
          </div>
          {request.sites.map((site, index) => (
            <div key={index} className="grid gap-default md:grid-cols-[1fr_8rem_2fr]">
              <label className="stack-xs text-xs text-text-muted">
                {t('newSimWizard.fleet.siteCode')}
                <input
                  className={inputClass}
                  value={site.code}
                  onChange={(event) => updateSite(index, 'code', event.target.value.toUpperCase())}
                />
              </label>
              <label className="stack-xs text-xs text-text-muted">
                {t('newSimWizard.fleet.siteOctet')}
                <input
                  className={inputClass}
                  type="number"
                  min={1}
                  max={253}
                  value={site.octet}
                  onChange={(event) => updateSite(index, 'octet', Number(event.target.value))}
                />
              </label>
              <label className="stack-xs text-xs text-text-muted">
                {t('newSimWizard.fleet.siteLocation')}
                <input
                  className={inputClass}
                  value={site.location}
                  onChange={(event) => updateSite(index, 'location', event.target.value)}
                />
              </label>
            </div>
          ))}
        </div>

        <div className="grid gap-default sm:grid-cols-2 lg:grid-cols-3">
          {countFields.map((field) => (
            <label key={field} className="stack-xs text-xs text-text-muted">
              {t(`newSimWizard.fleet.counts.${field}`)}
              <input
                className={inputClass}
                type="number"
                min={countLimits[field].min}
                max={
                  field === 'accessPointsPerAccess'
                    ? Math.min(9, Math.floor(154 / request.counts.accessSwitches))
                    : field === 'workstationsPerAccess'
                      ? Math.min(39, Math.floor(79 / request.counts.accessSwitches))
                      : countLimits[field].max
                }
                step={countLimits[field].step}
                value={request.counts[field]}
                onChange={(event) => updateCount(field, Number(event.target.value))}
              />
            </label>
          ))}
        </div>
        {!isScenarioRequestValid(request) && (
          <SmallText role="alert" className="text-status-error">
            {t('newSimWizard.fleet.invalid')}
          </SmallText>
        )}
      </CardContent>
    </Card>
  );
};
