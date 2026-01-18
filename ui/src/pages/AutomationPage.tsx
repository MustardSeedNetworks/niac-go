import { BellRing, Workflow } from 'lucide-react';
import { type FC, useEffect, useState } from 'react';
import { fetchAlerts, fetchStats, updateAlerts } from '../api/client';
import type { AlertConfig } from '../api/types';
import { useApiResource } from '../hooks/useApiResource';
import { Button } from '../ui/Button';
import { Card, CardContent } from '../ui/Card';
import { H2, P, SmallText } from '../ui/Typography';
import { getErrorMessage } from '../utils/format';

/**
 * Automation Page - Automation & Alerts
 *
 * Configure alert thresholds, webhook targets, and future workflow automations.
 */
export const AutomationPage: FC = () => {
  const { data: stats } = useApiResource(fetchStats, []);

  return (
    <div className="space-y-6">
      <Card className="border-white/5 bg-gray-900/70">
        <CardContent className="space-y-4">
          <H2 className="mb-0 flex items-center gap-2">
            <Workflow className="h-5 w-5 text-yellow-300" />
            Automation roadmap
          </H2>
          <P>
            Define alert thresholds, webhook routes, and (soon) runnable workflow automations. Use
            the CLI flags today, then manage them graphically here as the 2.0 UI matures. Current
            alert counter: {stats?.stack.errors ?? 0}.
          </P>
          <ul className="space-y-3 text-sm text-gray-300">
            <li className="rounded-lg border border-white/5 bg-gray-950/50 p-3">
              Webhooks inherit settings from the `--alert-webhook` flag and can be overridden per
              run.
            </li>
            <li className="rounded-lg border border-white/5 bg-gray-950/50 p-3">
              Packet thresholds mirror CLI/TUI options so headless and web control stay aligned.
            </li>
            <li className="rounded-lg border border-white/5 bg-gray-950/50 p-3">
              Future: orchestrate multi-run scenarios and publish signed run reports.
            </li>
          </ul>
        </CardContent>
      </Card>
      <AlertConfigCard />
    </div>
  );
};

/**
 * Alert Config Card - Configure alert thresholds and webhooks
 */
const AlertConfigCard: FC = () => {
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
        <H2 className="mb-0 flex items-center gap-2">
          <BellRing className="h-5 w-5 text-orange-300" />
          Alert policy
        </H2>
        <P className="text-gray-300">
          Updates take effect immediately—no CLI restart required. Leave the threshold blank or zero
          to disable packet alerts entirely.
        </P>
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
              <Button tone="violet" disabled={!dirty || saving} onClick={commit}>
                {saving ? 'Saving…' : 'Save alerts'}
              </Button>
              <Button variant="outline" disabled={!dirty || saving} onClick={reset}>
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
