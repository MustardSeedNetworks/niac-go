import { BellRing } from 'lucide-react';
import { type FC, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { fetchAlerts, fetchStats, updateAlerts } from '../api/client';
import type { AlertConfig } from '../api/types';
import { iconSizes } from '../constants/sizes';
import { useApiResource } from '../hooks/useApiResource';
import { useErrorToast } from '../hooks/useErrorToast';
import { Button } from '../ui/Button';
import { Card, CardContent } from '../ui/Card';
import { H2, P, SmallText } from '../ui/Typography';

/**
 * Alerts page — configure packet-threshold + webhook alerting for the
 * running daemon. Wires GET/PUT /api/v1/alerts. Updates take effect
 * immediately; no daemon restart required.
 */
export const AutomationPage: FC = () => {
  const { data: stats } = useApiResource(fetchStats, []);
  const errorCount = stats?.stack.errors ?? 0;

  return (
    <div className="stack-xl">
      <AlertConfigCard recentErrors={errorCount} />
    </div>
  );
};

/**
 * Alert Config Card - Configure alert thresholds and webhooks
 */
const AlertConfigCard: FC<{ recentErrors: number }> = ({ recentErrors }) => {
  const { t } = useTranslation('pages');
  const { data, loading, error } = useApiResource(fetchAlerts, [], {
    intervalMs: 15000,
    errorToast: true,
  });
  const [threshold, setThreshold] = useState('');
  const [webhook, setWebhook] = useState('');
  const [dirty, setDirty] = useState(false);
  const [saving, setSaving] = useState(false);
  // Success-only: save failures are surfaced as toasts, not a page banner.
  const [savedMessage, setSavedMessage] = useState<string | null>(null);
  const showError = useErrorToast();

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
      setDirty(false);
      setSavedMessage('Alert configuration saved');
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
    <Card className="border-surface-border bg-bg-surface/70">
      <CardContent className="stack-lg">
        <H2 className="flex items-center gap-compact">
          <BellRing className={`${iconSizes.lg} text-status-warning`} />
          Alert policy
        </H2>
        <P className="text-text-secondary">
          The daemon fires a webhook when total packet count crosses the threshold. Updates take
          effect immediately — no CLI restart required. Leave the threshold blank or zero to disable
          packet alerts entirely. The webhook destination is also gated by the daemon's{' '}
          <code>--webhook-allowed-host</code> allowlist when set (see{' '}
          <a
            href="https://github.com/krisarmstrong/niac-go/blob/main/SECURITY.md"
            className="text-brand-accent underline"
          >
            SECURITY.md
          </a>
          ).
        </P>
        <SmallText className="text-text-muted">
          Recent errors counter: <strong className="text-text-secondary">{recentErrors}</strong>
        </SmallText>
        {loading && <SmallText className="text-text-muted">Loading alert configuration…</SmallText>}
        {error && (
          <SmallText className="text-status-error">
            Unable to load alerts: {error.message}
          </SmallText>
        )}
        {data && (
          <>
            <div className="grid gap-comfortable md:grid-cols-2">
              <div>
                <SmallText className="text-text-muted">
                  {t('automation.page.packetThresholdLabel')}
                </SmallText>
                <input
                  className="mt-tight w-full rounded-lg border border-surface-border bg-bg-base/60 pad-xs text-sm text-text-primary focus:border-brand-accent focus:outline-none"
                  type="number"
                  min="0"
                  placeholder="100000"
                  value={threshold}
                  title="Total packet count that triggers the alert. 0 or blank disables the alert."
                  onChange={(event) => {
                    setThreshold(event.target.value);
                    setDirty(true);
                    setSavedMessage(null);
                  }}
                />
              </div>
              <div>
                <SmallText className="text-text-muted">
                  {t('automation.page.webhookUrlLabel')}
                </SmallText>
                <input
                  className="mt-tight w-full rounded-lg border border-surface-border bg-bg-base/60 pad-xs text-sm text-text-primary focus:border-brand-accent focus:outline-none"
                  placeholder="https://hooks.example.com/niac"
                  value={webhook}
                  title="POST'd JSON when the threshold trips. Must be http(s) and not point at a private/loopback/link-local IP. The daemon's --webhook-allowed-host flag further locks this down."
                  onChange={(event) => {
                    setWebhook(event.target.value);
                    setDirty(true);
                    setSavedMessage(null);
                  }}
                />
              </div>
            </div>
            {savedMessage && <SmallText className="text-status-success">{savedMessage}</SmallText>}
            <div className="flex flex-wrap gap-default">
              <Button
                tone="violet"
                disabled={!dirty || saving}
                onClick={commit}
                title="Save alert config to the running daemon. Takes effect immediately — no restart required."
              >
                {saving ? 'Saving…' : 'Save alerts'}
              </Button>
              <Button
                variant="outline"
                disabled={!dirty || saving}
                onClick={reset}
                title="Discard unsaved changes and reload the saved values."
              >
                Reset
              </Button>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
};

export default AutomationPage;
