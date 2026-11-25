import { type ReactNode, useEffect, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';
import { updateAlerts } from '../api/client';
import type { AlertConfig } from '../api/types';
import { useAppState } from '../contexts/AppContext';
import { useAlertsResource } from '../hooks/usePageResources';
import { useResourceError } from '../hooks/useResourceError';
import { useToast } from '../hooks/useToast';
import { Alert } from '../shared/Alert';
import { Button } from '../shared/Button';
import { Card } from '../shared/Card';
import { FormField } from '../shared/FormField';
import { Input } from '../shared/Input';
import { Skeleton } from '../shared/Skeleton';
import { ActionButton } from '../ui/ActionButton';
import { P, SmallText } from '../ui/Typography';

/**
 * Alerts page — configure packet-threshold + webhook alerting for the
 * running daemon. Wires GET/PUT /api/v1/alerts. Updates take effect
 * immediately; no daemon restart required.
 */
export function AlertsPage() {
  const { t } = useTranslation('pages');
  // The error count only decorates the alert card's helper text, so a failed
  // stats poll is a toast rather than a blocked page -- but it is not nothing:
  // silently showing zero recent errors reads as "all clear".
  const { data: stats, error } = useAppState('stats');
  useResourceError(error, t('alerts.statsFailed'));
  const errorCount = stats?.stack.errors ?? 0;

  return (
    <div className="stack-xl">
      <AlertConfigCard recentErrors={errorCount} />
    </div>
  );
}

/**
 * Alert Config Card - Configure alert thresholds and webhooks
 */
function AlertConfigCard({ recentErrors }: { recentErrors: number }) {
  const { t } = useTranslation('pages');
  const { data, loading, error, refetch } = useAlertsResource();
  const [threshold, setThreshold] = useState('');
  const [webhook, setWebhook] = useState('');
  const [dirty, setDirty] = useState(false);
  const [saving, setSaving] = useState(false);
  // Success-only: save failures are surfaced as toasts, not a page banner.
  const [savedMessage, setSavedMessage] = useState<string | null>(null);
  const { showError } = useToast();

  useEffect(() => {
    if (data && !dirty) {
      setThreshold(data.packetsThreshold ? String(data.packetsThreshold) : '');
      setWebhook(data.webhookUrl ?? '');
    }
  }, [data, dirty]);

  const commit = async () => {
    if (!dirty || saving) {
      return;
    }
    setSaving(true);
    setSavedMessage(null);
    try {
      const payload: AlertConfig = {
        packetsThreshold: threshold ? Number(threshold) : 0,
        webhookUrl: webhook.trim(),
      };
      await updateAlerts(payload);
      await refetch();
      setDirty(false);
      setSavedMessage(t('alerts.page.saved'));
    } catch (err) {
      showError(err);
    } finally {
      setSaving(false);
    }
  };

  const reset = () => {
    if (!data) {
      return;
    }
    setThreshold(data.packetsThreshold ? String(data.packetsThreshold) : '');
    setWebhook(data.webhookUrl ?? '');
    setDirty(false);
    setSavedMessage(null);
  };

  return (
    <Card title={t('alerts.alertPolicy')}>
      <div className="stack-lg">
        <P className="text-text-secondary">
          <Trans
            i18nKey="alerts.page.intro"
            ns="pages"
            components={{
              code: <code />,
              policy: <SecurityPolicyLink />,
            }}
          />
        </P>
        <SmallText className="text-text-muted">
          {t('alerts.recentErrors')} <strong className="text-text-secondary">{recentErrors}</strong>
        </SmallText>
        {loading && (
          <div role="status" aria-label={t('alerts.loadingAlerts')}>
            <Skeleton className="h-10 w-full" />
          </div>
        )}
        {error && <Alert tone="error">{t('alerts.loadError', { message: error.message })}</Alert>}
        {data && (
          <>
            <div className="grid gap-4 md:grid-cols-2">
              <FormField
                label={t('alerts.page.packetThresholdLabel')}
                hint={t('alerts.page.packetThresholdHelp')}
              >
                {(control) => (
                  <Input
                    {...control}
                    data-testid="alert-packet-threshold"
                    type="number"
                    min="0"
                    placeholder="100000"
                    value={threshold}
                    onChange={(event) => {
                      setThreshold(event.target.value);
                      setDirty(true);
                      setSavedMessage(null);
                    }}
                  />
                )}
              </FormField>
              <FormField
                label={t('alerts.page.webhookUrlLabel')}
                hint={t('alerts.page.webhookUrlHelp')}
              >
                {(control) => (
                  <Input
                    {...control}
                    placeholder="https://hooks.example.com/niac"
                    value={webhook}
                    onChange={(event) => {
                      setWebhook(event.target.value);
                      setDirty(true);
                      setSavedMessage(null);
                    }}
                  />
                )}
              </FormField>
            </div>
            {savedMessage && <Alert tone="success">{savedMessage}</Alert>}
            <div className="flex flex-wrap gap-2">
              <ActionButton
                action="edit"
                variant="primary"
                disabled={!dirty || saving}
                onClick={commit}
                title={t('alerts.page.saveHelp')}
              >
                {saving ? t('alerts.page.savingButton') : t('alerts.page.saveButton')}
              </ActionButton>
              <Button
                variant="secondary"
                disabled={!dirty || saving}
                onClick={reset}
                title={t('alerts.page.resetHelp')}
              >
                {t('alerts.page.resetButton')}
              </Button>
            </div>
          </>
        )}
      </div>
    </Card>
  );
}

// Trans fills the link text from the locale string.
function SecurityPolicyLink({ children }: { children?: ReactNode }) {
  return (
    <a
      href="https://github.com/krisarmstrong/niac-go/blob/main/SECURITY.md"
      className="text-brand-accent underline"
    >
      {children}
    </a>
  );
}

export default AlertsPage;
