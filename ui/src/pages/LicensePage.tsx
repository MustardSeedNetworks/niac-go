/**
 * LicensePage — surfaces the active license tier, the features it unlocks,
 * and how to activate or upgrade.
 *
 * NIAC validates licenses offline (no phone-home): activation, trial, and
 * deactivation are local `niac license` CLI operations that write the license
 * file the daemon reads. This page is therefore a read-only view of
 * GET /api/v1/license (via useLicense) plus the commands to run — it gives the
 * CLI's `niac license status` a home in the UI and makes the Free/Pro tier
 * visible instead of invisible.
 */
import { Check, KeyRound, RefreshCw } from 'lucide-react';
import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import { iconSizes } from '../constants/sizes';
import { useLicense } from '../contexts/LicenseContext';
import { Alert } from '../ui/Alert';
import { Button } from '../ui/Button';
import { Card, CardContent, CardHeader, CardRow } from '../ui/Card';

export const LicensePage: FC = () => {
  const { t } = useTranslation('pages');
  const { status, loading, error, refresh } = useLicense();

  // Offline license operations — the command strings are kept verbatim.
  const commands = [
    { label: t('license.cmd.status'), cmd: 'niac license status' },
    { label: t('license.cmd.activate'), cmd: 'niac license activate -k <KEY>' },
    { label: t('license.cmd.trial'), cmd: 'niac license trial' },
    { label: t('license.cmd.deactivate'), cmd: 'niac license deactivate' },
  ];

  if (loading) {
    return (
      <Card>
        <CardContent>
          <p className="text-text-secondary">{t('license.loading')}</p>
        </CardContent>
      </Card>
    );
  }

  if (error || !status) {
    return (
      <Alert status="error">
        <div className="flex items-center justify-between gap-comfortable">
          <span>{error ?? t('license.loadError')}</span>
          <Button variant="outline" onClick={() => void refresh()}>
            <RefreshCw className={iconSizes.md} />
            {t('license.retry')}
          </Button>
        </div>
      </Alert>
    );
  }

  const tierStatus = status.isActivated ? 'success' : 'warning';

  return (
    <div className="stack-xl">
      {!status.licenseEnforced && <Alert status="info">{t('license.devMode')}</Alert>}

      <Card>
        <CardHeader>
          <div className="flex items-center gap-compact">
            <KeyRound className={`${iconSizes.lg} text-text-secondary`} />
            <h2 className="text-lg font-semibold text-text-primary">{t('license.currentTier')}</h2>
          </div>
        </CardHeader>
        <CardContent>
          <CardRow label={t('license.tier')} value={status.tier} status={tierStatus} />
          <CardRow
            label={t('license.activation')}
            value={status.isActivated ? t('license.activated') : t('license.notActivated')}
            status={tierStatus}
          />
          {status.isTrialMode && (
            <CardRow
              label={t('license.trial')}
              value={t('license.trialDaysRemaining', { days: status.trialDaysRemaining })}
              status="info"
            />
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <h2 className="text-lg font-semibold text-text-primary">{t('license.features')}</h2>
        </CardHeader>
        <CardContent>
          {status.features.length === 0 ? (
            <p className="text-text-secondary">{t('license.noFeatures')}</p>
          ) : (
            <ul className="grid gap-compact sm:grid-cols-2">
              {status.features.map((feature) => (
                <li
                  key={feature}
                  className="flex items-center gap-compact text-sm text-text-secondary"
                >
                  <Check className={`${iconSizes.md} text-status-success`} />
                  <code className="text-text-primary">{feature}</code>
                </li>
              ))}
            </ul>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <h2 className="text-lg font-semibold text-text-primary">{t('license.manage')}</h2>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col gap-comfortable">
            <p className="text-sm text-text-secondary">{t('license.manageHint')}</p>
            <div className="flex flex-col gap-compact">
              {commands.map(({ label, cmd }) => (
                <div key={cmd} className="flex items-center gap-comfortable">
                  <span className="w-24 shrink-0 text-xs uppercase tracking-wide text-text-muted">
                    {label}
                  </span>
                  <code className="flex-1 rounded bg-surface-raised px-cell py-compact text-sm text-text-primary">
                    {cmd}
                  </code>
                </div>
              ))}
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
};

export default LicensePage;
