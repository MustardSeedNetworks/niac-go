import { type FC, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { preflightSimulation } from '../../api/client';
import { type ApiErrorDetail, isApiError } from '../../api/errors';
import type { SimulationPreflightReport, SimulationPreflightRequest } from '../../api/fabric-types';
import type { SimulationRequest } from '../../api/types';
import { ApiErrorMessage } from '../../ui/ApiErrorMessage';
import { Button } from '../../ui/Button';
import { Card, CardContent } from '../../ui/Card';
import { H2, SmallText } from '../../ui/Typography';

interface PreflightStepProps {
  request: SimulationRequest;
  onStart: (request: SimulationPreflightRequest) => void;
  starting?: boolean;
}

const ACCESS_VLAN_DEFAULT = 200;

export const PreflightStep: FC<PreflightStepProps> = ({ request, onStart, starting = false }) => {
  const { t } = useTranslation('pages');
  const [attachment, setAttachment] = useState('tester');
  const [mode, setMode] = useState<'direct' | 'access' | 'trunk'>('access');
  const [accessVlan, setAccessVlan] = useState(ACCESS_VLAN_DEFAULT);
  const [sessionId, setSessionId] = useState(
    request.sessionId ?? `scenario-${ACCESS_VLAN_DEFAULT}`,
  );
  const [report, setReport] = useState<SimulationPreflightReport | null>(null);
  const [approvedPayload, setApprovedPayload] = useState<SimulationPreflightRequest | null>(null);
  const [error, setError] = useState('');
  // The server enumerates validation failures one per offending field (#1461);
  // keeping only err.message threw that diagnosis away on the one screen whose
  // job is to show it (#1472).
  const [errorDetails, setErrorDetails] = useState<readonly ApiErrorDetail[]>([]);
  const [checking, setChecking] = useState(false);
  const requestSequence = useRef(0);

  const payload: SimulationPreflightRequest = {
    ...request,
    attachment,
    attachmentMode: mode,
    ...(mode === 'trunk' ? { sessionId } : {}),
    ...(mode === 'access' || mode === 'trunk' ? { accessVlan } : {}),
  };

  const invalidate = () => {
    requestSequence.current += 1;
    setReport(null);
    setApprovedPayload(null);
    setError('');
    setChecking(false);
  };

  const check = async () => {
    const sequence = requestSequence.current + 1;
    requestSequence.current = sequence;
    const checkedPayload = payload;
    setChecking(true);
    setError('');
    setErrorDetails([]);
    try {
      const result = await preflightSimulation(checkedPayload);
      if (requestSequence.current !== sequence) return;
      setReport(result);
      setApprovedPayload(result.safe ? checkedPayload : null);
    } catch (err) {
      if (requestSequence.current !== sequence) return;
      setReport(null);
      setApprovedPayload(null);
      setError(err instanceof Error ? err.message : t('newSimWizard.preflight.requestError'));
      setErrorDetails(isApiError(err) ? err.details : []);
    } finally {
      if (requestSequence.current === sequence) setChecking(false);
    }
  };

  return (
    <Card className="border-surface-border bg-bg-surface/70">
      <CardContent className="stack-lg">
        <div>
          <H2>{t('newSimWizard.preflight.title')}</H2>
          <SmallText>{t('newSimWizard.preflight.help')}</SmallText>
        </div>
        <label className="grid gap-compact text-xs text-text-muted">
          {t('newSimWizard.preflight.attachmentLabel')}
          <input
            data-testid="wizard-attachment-name"
            value={attachment}
            onChange={(event) => {
              setAttachment(event.target.value);
              invalidate();
            }}
            className="rounded border border-surface-border bg-bg-elevated px-3 py-row text-sm text-text-primary"
          />
        </label>
        <label className="grid gap-compact text-xs text-text-muted">
          {t('newSimWizard.preflight.modeLabel')}
          <select
            data-testid="wizard-attachment-mode"
            value={mode}
            onChange={(event) => {
              setMode(event.target.value as 'direct' | 'access' | 'trunk');
              invalidate();
            }}
            className="rounded border border-surface-border bg-bg-elevated px-3 py-row text-sm text-text-primary"
          >
            <option value="access">{t('newSimWizard.preflight.accessMode')}</option>
            <option value="trunk">{t('newSimWizard.preflight.trunkMode')}</option>
            <option value="direct">{t('newSimWizard.preflight.directMode')}</option>
          </select>
        </label>
        {mode === 'trunk' && (
          <label className="grid gap-compact text-xs text-text-muted">
            {t('newSimWizard.preflight.sessionIdLabel')}
            <input
              data-testid="wizard-session-id"
              value={sessionId}
              onChange={(event) => {
                setSessionId(event.target.value);
                invalidate();
              }}
              className="rounded border border-surface-border bg-bg-elevated px-3 py-row text-sm text-text-primary"
            />
          </label>
        )}
        {(mode === 'access' || mode === 'trunk') && (
          <label className="grid gap-compact text-xs text-text-muted">
            {t('newSimWizard.preflight.vlanLabel')}
            <input
              type="number"
              min={1}
              max={4094}
              data-testid="wizard-access-vlan"
              value={Number.isNaN(accessVlan) ? '' : accessVlan}
              onChange={(event) => {
                setAccessVlan(event.target.valueAsNumber);
                if (mode === 'trunk') setSessionId(`scenario-${event.target.valueAsNumber}`);
                invalidate();
              }}
              className="rounded border border-surface-border bg-bg-elevated px-3 py-row text-sm text-text-primary"
            />
          </label>
        )}
        {error && <ApiErrorMessage message={error} details={errorDetails} />}
        {report?.safe && (
          <div
            className="rounded-lg border border-status-success/40 bg-status-success/10 pad-sm"
            role="status"
          >
            <SmallText className="text-status-success">
              {t('newSimWizard.preflight.safe', { count: report.topology.networks?.length ?? 0 })}
            </SmallText>
            <SmallText className="mt-tight block text-text-secondary">
              {t('newSimWizard.preflight.physicalVlanSummary', {
                vlan: report.topology.binding.physicalVlan ?? t('newSimWizard.preflight.untagged'),
              })}
            </SmallText>
          </div>
        )}
        {report && !report.safe && (
          <div
            className="rounded-lg border-2 border-status-error bg-status-error/15 pad-default"
            role="alert"
          >
            <p className="font-semibold text-status-error">
              {t('newSimWizard.preflight.unsafeTitle')}
            </p>
            <ul className="mt-tight list-disc pl-5 text-sm text-status-error">
              {(report.diagnostics ?? []).map((diagnostic) => (
                <li key={`${diagnostic.code}-${diagnostic.field}`}>{diagnostic.message}</li>
              ))}
            </ul>
          </div>
        )}
        <div className="flex gap-default">
          <Button
            variant="outline"
            data-testid="wizard-preflight-check"
            onClick={() => void check()}
            loading={checking}
            disabled={
              !attachment ||
              (mode === 'trunk' && !/^[a-z0-9](?:[a-z0-9-]{0,38}[a-z0-9])?$/.test(sessionId)) ||
              ((mode === 'access' || mode === 'trunk') &&
                (!Number.isInteger(accessVlan) || accessVlan < 1 || accessVlan > 4094))
            }
          >
            {t('newSimWizard.preflight.checkLabel')}
          </Button>
          <Button
            tone="violet"
            data-testid="wizard-preflight-start"
            onClick={() => approvedPayload && onStart(approvedPayload)}
            loading={starting}
            disabled={!approvedPayload || starting}
          >
            {t('newSimWizard.preflight.startLabel')}
          </Button>
        </div>
      </CardContent>
    </Card>
  );
};
