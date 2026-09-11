import { type FC, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  fetchAttachmentPolicies,
  fetchSimulationAttachments,
  preflightSimulation,
} from '../../api/client';
import { type ApiErrorDetail, isApiError } from '../../api/errors';
import type {
  AttachmentMode,
  AttachmentPolicy,
  SimulationPreflightReport,
  SimulationPreflightRequest,
} from '../../api/fabric-types';
import type { SimulationRequest } from '../../api/types';
import { ApiErrorMessage } from '../../ui/ApiErrorMessage';
import { Button } from '../../ui/Button';
import { Card, CardContent } from '../../ui/Card';
import { H2, SmallText } from '../../ui/Typography';
import { bindingOptions } from './attachment-options';

interface PreflightStepProps {
  request: SimulationRequest;
  onStart: (request: SimulationPreflightRequest) => void;
  starting?: boolean;
}

const ACCESS_VLAN_DEFAULT = 200;

const inputClass =
  'rounded border border-surface-border bg-bg-elevated px-3 py-row text-sm text-text-primary';

export const PreflightStep: FC<PreflightStepProps> = ({ request, onStart, starting = false }) => {
  const { t } = useTranslation('pages');
  const [attachment, setAttachment] = useState('');
  const [mode, setMode] = useState<AttachmentMode>('access');
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

  // The binding was three guesses before AP-0: a free-text attachment
  // defaulting to `tester` that no generated pack answers to, plus a mode and a
  // VLAN the operator's policies were never asked about. Both halves are read
  // from the daemon here instead.
  const [policies, setPolicies] = useState<AttachmentPolicy[] | null>(null);
  const [attachmentNames, setAttachmentNames] = useState<string[] | null>(null);
  const [routed, setRouted] = useState(true);
  const [discovering, setDiscovering] = useState(true);

  const { interface: interfaceName, configData, configPath, templateName } = request;

  useEffect(() => {
    let current = true;
    fetchAttachmentPolicies()
      .then((response) => {
        if (current) setPolicies(response.policies ?? []);
      })
      .catch(() => {
        if (current) setPolicies([]);
      });
    return () => {
      current = false;
    };
  }, []);

  useEffect(() => {
    let current = true;
    setDiscovering(true);
    fetchSimulationAttachments({ interface: interfaceName, configData, configPath, templateName })
      .then((response) => {
        if (!current) return;
        setRouted(response.routed);
        setAttachmentNames(response.attachments ?? []);
        setAttachment(response.attachments?.[0] ?? '');
      })
      .catch(() => {
        // The configuration could not be read -- preflight will say why. Fall
        // back to the typed field so the screen stays usable meanwhile.
        if (!current) return;
        setAttachmentNames(null);
        setRouted(true);
      })
      .finally(() => {
        if (current) setDiscovering(false);
      });
    return () => {
      current = false;
    };
  }, [interfaceName, configData, configPath, templateName]);

  const options = useMemo(
    () => bindingOptions(policies ?? [], interfaceName),
    [policies, interfaceName],
  );
  const approvedVlans = options.vlansByMode[mode];
  const hasPolicy = options.modes.length > 0;
  // Until both answers are in, the binding fields would show defaults the
  // daemon may refuse -- the very guessing AP-0 removes. Wait instead.
  const resolving = policies === null || discovering;

  // Snap the binding onto the operator's approvals as soon as they arrive, so
  // the fields never sit on a combination the daemon would refuse.
  useEffect(() => {
    const [fallback] = options.modes;
    if (fallback === undefined) return;
    setMode((current) => (options.modes.includes(current) ? current : fallback));
  }, [options]);

  useEffect(() => {
    const [fallback] = approvedVlans;
    if (fallback === undefined) return;
    setAccessVlan((current) => (approvedVlans.includes(current) ? current : fallback));
  }, [approvedVlans]);

  // Each label is looked up by a literal key: the extraction gate reads t()
  // calls statically, and a template-literal key silently drops the string
  // from every catalog.
  const modeLabel = (available: AttachmentMode): string => {
    switch (available) {
      case 'access':
        return t('newSimWizard.preflight.accessMode');
      case 'trunk':
        return t('newSimWizard.preflight.trunkMode');
      case 'direct':
        return t('newSimWizard.preflight.directMode');
    }
  };

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

  const selectMode = (next: AttachmentMode) => {
    setMode(next);
    const [fallback] = options.vlansByMode[next];
    if (fallback !== undefined && !options.vlansByMode[next].includes(accessVlan)) {
      selectVlan(fallback, next);
    }
    invalidate();
  };

  const selectVlan = (vlan: number, forMode: AttachmentMode = mode) => {
    setAccessVlan(vlan);
    if (forMode === 'trunk') setSessionId(`scenario-${vlan}`);
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

  const vlanIsApproved = approvedVlans.length > 0;

  return (
    <Card className="border-surface-border bg-bg-surface/70">
      <CardContent className="stack-lg">
        <div>
          <H2>{t('newSimWizard.preflight.title')}</H2>
          <SmallText>{t('newSimWizard.preflight.help')}</SmallText>
        </div>
        {!hasPolicy && !resolving && (
          <SmallText className="text-status-warning" data-testid="wizard-no-policy-notice">
            {t('newSimWizard.preflight.noPolicyNotice', { interface: interfaceName })}
          </SmallText>
        )}
        {resolving && (
          <SmallText data-testid="wizard-binding-loading">
            {t('newSimWizard.preflight.resolvingBinding')}
          </SmallText>
        )}
        {/* A flat scenario binds no attachment, so there is nothing to pick. */}
        {!resolving && routed && (
          <label
            htmlFor="preflight-attachment"
            className="grid gap-compact text-xs text-text-muted"
          >
            {t('newSimWizard.preflight.attachmentLabel')}
            {attachmentNames === null ? (
              <input
                id="preflight-attachment"
                data-testid="wizard-attachment-name"
                value={attachment}
                onChange={(event) => {
                  setAttachment(event.target.value);
                  invalidate();
                }}
                className={inputClass}
              />
            ) : (
              <select
                id="preflight-attachment"
                data-testid="wizard-attachment-name"
                value={attachment}
                onChange={(event) => {
                  setAttachment(event.target.value);
                  invalidate();
                }}
                className={inputClass}
              >
                {attachmentNames.map((name) => (
                  <option key={name} value={name}>
                    {name}
                  </option>
                ))}
              </select>
            )}
          </label>
        )}
        {!resolving && (
          <label className="grid gap-compact text-xs text-text-muted">
            {t('newSimWizard.preflight.modeLabel')}
            <select
              data-testid="wizard-attachment-mode"
              value={mode}
              onChange={(event) => selectMode(event.target.value as AttachmentMode)}
              className={inputClass}
            >
              {(hasPolicy
                ? options.modes
                : (['access', 'trunk', 'direct'] as AttachmentMode[])
              ).map((available) => (
                <option key={available} value={available}>
                  {modeLabel(available)}
                </option>
              ))}
            </select>
          </label>
        )}
        {!resolving && mode === 'trunk' && (
          <label className="grid gap-compact text-xs text-text-muted">
            {t('newSimWizard.preflight.sessionIdLabel')}
            <input
              data-testid="wizard-session-id"
              value={sessionId}
              onChange={(event) => {
                setSessionId(event.target.value);
                invalidate();
              }}
              className={inputClass}
            />
          </label>
        )}
        {!resolving && (mode === 'access' || mode === 'trunk') && (
          <label htmlFor="preflight-vlan" className="grid gap-compact text-xs text-text-muted">
            {vlanIsApproved
              ? t('newSimWizard.preflight.approvedVlanLabel')
              : t('newSimWizard.preflight.vlanLabel')}
            {vlanIsApproved ? (
              <select
                id="preflight-vlan"
                data-testid="wizard-access-vlan"
                value={accessVlan}
                onChange={(event) => {
                  selectVlan(Number(event.target.value));
                  invalidate();
                }}
                className={inputClass}
              >
                {approvedVlans.map((vlan) => (
                  <option key={vlan} value={vlan}>
                    {vlan}
                  </option>
                ))}
              </select>
            ) : (
              <input
                type="number"
                min={1}
                max={4094}
                id="preflight-vlan"
                data-testid="wizard-access-vlan"
                value={Number.isNaN(accessVlan) ? '' : accessVlan}
                onChange={(event) => {
                  selectVlan(event.target.valueAsNumber);
                  invalidate();
                }}
                className={inputClass}
              />
            )}
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
            action="start"
            loading={checking}
            disabled={
              discovering ||
              (routed && !attachment) ||
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
            action="start"
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
