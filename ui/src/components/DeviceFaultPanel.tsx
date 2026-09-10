import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { DeviceFaultPayload } from '../api/api-response-types';
import { validFaultAddress } from '../api/behavior-fault-types';
import { clearError, injectError } from '../api/client';
import { useAppState } from '../contexts/AppContext';
import { useActionPermission } from '../contexts/ScopeContext';
import { Button } from '../ui/Button';
import { Card, CardContent } from '../ui/Card';
import { Input, Select } from '../ui/Input';
import { H3, SmallText } from '../ui/Typography';
import { getErrorMessage } from '../utils/format';

export function DeviceFaultPanel() {
  const { t } = useTranslation('errors');
  const permission = useActionPermission('inject');
  const { data, error, loading, refetch } = useAppState('errorTypes');
  const [device, setDevice] = useState('');
  const [faultType, setFaultType] = useState('');
  const [value, setValue] = useState(1);
  const [address, setAddress] = useState('');
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState('');
  const target = data?.deviceTargets?.find((entry) => entry.device === device);
  const types =
    data?.availableDeviceTypes?.filter((entry) => target?.errorTypes.includes(entry.type)) ?? [];
  const fault = types.find((entry) => entry.type === faultType);
  const submittedValue = fault?.valueKind === 'toggle' ? 1 : value;
  const disabled = permission.disabled || busy || loading || Boolean(error);
  const valid = Boolean(
    fault &&
      (fault.valueKind === 'address'
        ? validFaultAddress(address)
        : Number.isInteger(submittedValue) &&
          submittedValue >= 1 &&
          submittedValue <= fault.maxValue),
  );
  const active = Object.entries(data?.activeDeviceErrors ?? {}).flatMap(([name, faults]) =>
    Object.entries(faults).map(([type, amount]) => ({ device: name, type, amount })),
  );

  async function apply() {
    if (disabled || !valid) return;
    setBusy(true);
    setFailure('');
    try {
      const target = { device, interface: '', errorType: faultType };
      await injectError(
        fault?.valueKind === 'address'
          ? { ...target, address }
          : { ...target, value: submittedValue },
      );
      refetch();
    } catch (cause) {
      setFailure(getErrorMessage(cause));
    } finally {
      setBusy(false);
    }
  }

  async function clear(name: string, type: string) {
    if (disabled) return;
    setBusy(true);
    setFailure('');
    try {
      await clearError(name, '', type);
      refetch();
    } catch (cause) {
      setFailure(getErrorMessage(cause));
    } finally {
      setBusy(false);
    }
  }

  function displayValue(type: string, payload: DeviceFaultPayload) {
    if (payload.address !== undefined) return payload.address;
    const amount = payload.value;
    const kind = data?.availableDeviceTypes?.find((entry) => entry.type === type)?.valueKind;
    if (kind === 'toggle') return amount > 0 ? t('deviceFault.armed') : t('deviceFault.clear');
    if (kind === 'percent') return `${amount}%`;
    if (kind === 'milliseconds') return `${amount} ms`;
    return String(amount);
  }

  return (
    <Card data-testid="device-fault-panel">
      <CardContent className="stack">
        <H3>{t('deviceFault.title')}</H3>
        <Select
          label={t('deviceFault.target')}
          value={device}
          options={[
            { value: '', label: t('deviceFault.selectTarget') },
            ...(data?.deviceTargets ?? []).map((entry) => ({
              value: entry.device,
              label: entry.device,
            })),
          ]}
          onChange={(next) => {
            setDevice(next);
            setFaultType('');
            setValue(1);
            setAddress('');
          }}
        />
        <Select
          label={t('deviceFault.type')}
          value={faultType}
          options={[
            { value: '', label: t('deviceFault.selectType') },
            ...types.map((entry) => ({ value: entry.type, label: entry.type })),
          ]}
          onChange={(next) => {
            setFaultType(next);
            setValue(1);
            setAddress('');
          }}
        />
        {fault && <SmallText>{fault.description}</SmallText>}
        {fault?.valueKind === 'address' && (
          <Input
            label={t('deviceFault.conflictAddress')}
            value={address}
            onChange={(event) => setAddress(event.target.value)}
          />
        )}
        {fault && fault.valueKind !== 'toggle' && fault.valueKind !== 'address' && (
          <Input
            type="number"
            label={
              fault.valueKind === 'percent'
                ? t('deviceFault.percent')
                : t('deviceFault.milliseconds')
            }
            min={1}
            max={fault.maxValue}
            step={1}
            value={value}
            onChange={(event) => setValue(Number(event.target.value))}
          />
        )}
        {(failure || error) && <p role="alert">{failure || getErrorMessage(error)}</p>}
        <Button
          action="inject"
          variant="secondary"
          data-testid="apply-device-fault"
          disabled={disabled || !valid}
          onClick={() => void apply()}
        >
          {t('deviceFault.apply')}
        </Button>
        <ul className="stack-sm">
          {active.map((entry) => (
            <li
              key={`${entry.device}:${entry.type}`}
              className="flex flex-wrap items-center gap-default"
            >
              <span>{entry.device}</span>
              <span>{entry.type}</span>
              <span>{displayValue(entry.type, entry.amount)}</span>
              <Button
                action="inject"
                variant="secondary"
                disabled={disabled}
                onClick={() => void clear(entry.device, entry.type)}
                aria-label={t('deviceFault.clearOne', { device: entry.device, type: entry.type })}
              >
                {t('deviceFault.clear')}
              </Button>
            </li>
          ))}
        </ul>
      </CardContent>
    </Card>
  );
}
