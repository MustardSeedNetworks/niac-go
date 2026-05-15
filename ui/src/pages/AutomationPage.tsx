import { BellRing } from 'lucide-react';
import { type FC, useEffect, useState } from 'react';
import { fetchAlerts, fetchStats, updateAlerts } from '../api/client';
import type { AlertConfig } from '../api/types';
import { iconSizes } from '../constants/sizes';
import { useApiResource } from '../hooks/useApiResource';
import { Button } from '../ui/Button';
import { Card, CardContent } from '../ui/Card';
import { H2, P, SmallText } from '../ui/Typography';
import { getErrorMessage } from '../utils/format';

/**
 * Alerts page — configure packet-threshold + webhook alerting for the
 * running daemon. Wires GET/PUT /api/v1/alerts. Updates take effect
 * immediately; no daemon restart required.
 */
export const AutomationPage: FC = () => {
  const { data: stats } = useApiResource(fetchStats, []);
  const errorCount = stats?.stack.errors ?? 0;

  return (
    <div className="space-y-6">
      <AlertConfigCard recentErrors={errorCount} />
    </div>
  );
};

/**
 * Alert Config Card - Configure alert thresholds and webhooks
 */
const AlertConfigCard: FC<{ recentErrors: number }> = ({ recentErrors }) => {
  const { data, loading, error } = useApiResource(fetchAlerts, [], {
    intervalMs: 15000,
  });
  const [threshold, setThreshold] = useState('');
  const [webhook, setWebhook] = useState('');
  const [dirty, setDirty] = useState(false);
  const [saving, setSaving] = useState(false);
  const [status, setStatus] = useState<{
    tone: 'success' | 'error';
    text: string;
  } | null>(null);

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
    setStatus(null);
    try {
      const payload: AlertConfig = {
        packetsThreshold: threshold ? Number(threshold) : 0,
        webhookUrl: webhook.trim(),
      };
      await updateAlerts(payload);
      setDirty(false);
      setStatus({ tone: 'success', text: 'Alert configuration saved' });
    } catch (err) {
      setStatus({ tone: 'error', text: getErrorMessage(err) });
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
    setStatus(null);
  };

  return (
    <Card className="border-white/5 bg-gray-900/70">
      <CardContent className="space-y-4">
        <H2 className="flex items-center gap-2">
          <BellRing className={`${iconSizes.lg} text-orange-300`} />
          Alert policy
        </H2>
        <P className="text-gray-300">
          The daemon fires a webhook when total packet count crosses the threshold. Updates take
          effect immediately — no CLI restart required. Leave the threshold blank or zero to disable
          packet alerts entirely. The webhook destination is also gated by the daemon's{' '}
          <code>--webhook-allowed-host</code> allowlist when set (see{' '}
          <a
            href="https://github.com/krisarmstrong/niac-go/blob/main/SECURITY.md"
            className="text-violet-300 underline"
          >
            SECURITY.md
          </a>
          ).
        </P>
        <SmallText className="text-gray-500">
          Recent errors counter: <strong className="text-gray-300">{recentErrors}</strong>
        </SmallText>
        {loading && <SmallText className="text-gray-400">Loading alert configuration…</SmallText>}
        {error && (
          <SmallText className="text-red-400">Unable to load alerts: {error.message}</SmallText>
        )}
        {data && (
          <>
            <div className="grid gap-4 md:grid-cols-2">
              <div>
                <SmallText className="text-gray-400">Packet threshold</SmallText>
                <input
                  className="mt-1 w-full rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white focus:border-violet-400 focus:outline-none"
                  type="number"
                  min="0"
                  placeholder="100000"
                  value={threshold}
                  title="Total packet count that triggers the alert. 0 or blank disables the alert."
                  onChange={(event) => {
                    setThreshold(event.target.value);
                    setDirty(true);
                    setStatus(null);
                  }}
                />
              </div>
              <div>
                <SmallText className="text-gray-400">Webhook URL</SmallText>
                <input
                  className="mt-1 w-full rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white focus:border-violet-400 focus:outline-none"
                  placeholder="https://hooks.example.com/niac"
                  value={webhook}
                  title="POST'd JSON when the threshold trips. Must be http(s) and not point at a private/loopback/link-local IP. The daemon's --webhook-allowed-host flag further locks this down."
                  onChange={(event) => {
                    setWebhook(event.target.value);
                    setDirty(true);
                    setStatus(null);
                  }}
                />
              </div>
            </div>
            {status && (
              <SmallText
                className={status.tone === 'success' ? 'text-emerald-300' : 'text-red-400'}
              >
                {status.text}
              </SmallText>
            )}
            <div className="flex flex-wrap gap-3">
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
