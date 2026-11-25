import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { executeDeviceAction } from '../api/client';
import { useAppState } from '../contexts/AppContext';
import { useActionPermission } from '../contexts/ScopeContext';
import { Alert } from '../shared/Alert';
import { Card } from '../shared/Card';
import { FormField } from '../shared/FormField';
import { Select } from '../shared/Select';
import { ActionButton } from '../ui/ActionButton';
import { SmallText } from '../ui/Typography';
import { getErrorMessage } from '../utils/format';

/**
 * Operations an operator runs against a device once — a reboot, a spanning-tree
 * topology change. They belong beside the fault panels because reaching for
 * "make this device reboot" is the same job as reaching for "make DNS fail";
 * until the daemon advertised them here the only way to run one was to author a
 * behaviour timeline and wait for it to fire.
 */
export function DeviceActionPanel() {
  const { t } = useTranslation('errors');
  const permission = useActionPermission('inject');
  const { data, error, loading, refetch } = useAppState('errorTypes');
  const [device, setDevice] = useState('');
  const [action, setAction] = useState('');
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState('');
  const [done, setDone] = useState('');
  const target = data?.actionTargets?.find((entry) => entry.device === device);
  const actions =
    data?.availableActions?.filter((entry) => target?.actions.includes(entry.type)) ?? [];
  const selected = actions.find((entry) => entry.type === action);
  const disabled = permission.disabled || busy || loading || Boolean(error);

  async function run() {
    if (disabled || !selected) return;
    setBusy(true);
    setFailure('');
    setDone('');
    try {
      await executeDeviceAction(device, action);
      setDone(t('deviceAction.executed', { device, action }));
      refetch();
    } catch (cause) {
      setFailure(getErrorMessage(cause));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Card title={t('deviceAction.title')} data-testid="device-action-panel">
      <div className="stack">
        <FormField label={t('deviceAction.target')}>
          {(control) => (
            <Select
              {...control}
              value={device}
              options={[
                { value: '', label: t('deviceAction.selectTarget') },
                ...(data?.actionTargets ?? []).map((entry) => ({
                  value: entry.device,
                  label: entry.device,
                })),
              ]}
              onValueChange={(next) => {
                setDevice(next);
                setAction('');
                setDone('');
              }}
            />
          )}
        </FormField>
        <FormField label={t('deviceAction.action')}>
          {(control) => (
            <Select
              {...control}
              value={action}
              options={[
                { value: '', label: t('deviceAction.selectAction') },
                ...actions.map((entry) => ({ value: entry.type, label: entry.type })),
              ]}
              onValueChange={(next) => {
                setAction(next);
                setDone('');
              }}
            />
          )}
        </FormField>
        {selected && <SmallText>{selected.description}</SmallText>}
        {(failure || error) && <Alert tone="error">{failure || getErrorMessage(error)}</Alert>}
        {done && <SmallText>{done}</SmallText>}
        <ActionButton
          action="inject"
          variant="secondary"
          data-testid="run-device-action"
          disabled={disabled || !selected}
          onClick={() => void run()}
        >
          {t('deviceAction.run')}
        </ActionButton>
      </div>
    </Card>
  );
}
