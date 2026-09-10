import { valibotResolver } from '@hookform/resolvers/valibot';
import { type FC, useEffect, useState } from 'react';
import { type SubmitHandler, useForm } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { useSearchParams } from 'react-router';
import { validFaultAddress } from '../api/behavior-fault-types';
import { clearAllErrors, clearError, injectError } from '../api/client';
import type { ErrorType } from '../api/types';
import { useAppState } from '../contexts/AppContext';
import { type ErrorInjectionFormFields, ErrorInjectionSchema } from '../schemas/forms';
import { Button } from '../ui/Button';
import { Card, CardContent } from '../ui/Card';
import { ConfirmModal } from '../ui/ConfirmModal';
import { DataTable, type DataTableColumn } from '../ui/DataTable';
import { Input } from '../ui/Input';
import { Tag } from '../ui/Tag';
import { SmallText } from '../ui/Typography';
import { getErrorMessage } from '../utils/format';

export const ErrorInjectionPanel: FC = () => {
  const permission = useActionPermission('inject');
  const { t } = useTranslation('errors');
  const { data: errorInfo, error, refetch: refetchErrors } = useAppState('errorTypes');
  // Deep-link support: the Dashboard's Error Injection quick action links
  // here with ?errorType=<type> so a specific error type arrives
  // preselected instead of duplicating this form on the dashboard itself.
  const [searchParams] = useSearchParams();
  const presetErrorType = searchParams.get('errorType') ?? '';

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors, isSubmitting },
  } = useForm<ErrorInjectionFormFields>({
    resolver: valibotResolver(ErrorInjectionSchema),
    defaultValues: {
      selectedDevice: '',
      selectedInterface: '',
      selectedErrorType: presetErrorType,
      errorValue: 50,
    },
    mode: 'onBlur',
  });
  const selectedErrorType = watch('selectedErrorType');
  const selectedDevice = watch('selectedDevice');
  const selectedInterface = watch('selectedInterface');
  const [address, setAddress] = useState('');
  const [prefix, setPrefix] = useState('');
  const valueKind = errorInfo?.availableTypes.find(
    (type) => type.type === selectedErrorType,
  )?.valueKind;
  const addressed = valueKind === 'address';
  const prefixed = valueKind === 'prefix';
  const prefixBits = Number(prefix);
  const validPrefix =
    prefix.trim() !== '' && Number.isInteger(prefixBits) && prefixBits >= 0 && prefixBits <= 32;
  const selectedTarget = errorInfo?.targets?.find((target) => target.device === selectedDevice);
  const supported = selectedTarget?.errorTypes[selectedInterface] ?? [];

  // The previously selected interface may not exist on a newly selected
  // device — clear it rather than silently submitting a stale value.
  useEffect(() => {
    setValue('selectedInterface', '');
  }, [selectedDevice, setValue]);

  // The <select> options render once errorInfo resolves, after this form's
  // initial mount — so the uncontrolled defaultValues.selectedErrorType
  // above can land on a value with no matching <option> yet. Re-apply the
  // preselected type once the catalog (and the option it names) is loaded.
  useEffect(() => {
    if (
      presetErrorType &&
      errorInfo?.availableTypes?.some((errorType) => errorType.type === presetErrorType)
    ) {
      setValue('selectedErrorType', presetErrorType, { shouldValidate: true });
    }
  }, [presetErrorType, errorInfo, setValue]);
  const errorValue = watch('errorValue');

  const [clearingBusy, setClearingBusy] = useState(false);
  const [message, setMessage] = useState<{
    type: 'success' | 'error';
    text: string;
  } | null>(null);
  const [showClearAllConfirm, setShowClearAllConfirm] = useState(false);
  const visibleMessage = error ? { type: 'error', text: getErrorMessage(error) } : message;
  const activeErrors = errorInfo?.activeErrors ?? {};
  const activeEntries = Object.entries(activeErrors);

  // The wire shape is device -> interface -> errorType -> value; the table
  // wants one row per leaf.
  const activeRows = activeEntries.flatMap(([deviceIp, interfaces]) =>
    Object.entries(interfaces).flatMap(([iface, errorTypes]) =>
      Object.entries(errorTypes).map(([errorType, value]) => ({
        deviceIp,
        iface,
        errorType,
        value,
      })),
    ),
  );

  const onInject: SubmitHandler<ErrorInjectionFormFields> = async (values) => {
    if (
      permission.disabled ||
      !supported.includes(values.selectedErrorType) ||
      (addressed && !validFaultAddress(address)) ||
      (prefixed && !validPrefix)
    )
      return;
    setMessage(null);
    try {
      await injectError({
        device: values.selectedDevice,
        interface: values.selectedInterface,
        errorType: values.selectedErrorType,
        ...(addressed ? { address } : prefixed ? { prefixBits } : { value: values.errorValue }),
      });
      setMessage({ type: 'success', text: t('injection.injectSuccess') });
      refetchErrors();
    } catch (err: unknown) {
      setMessage({
        type: 'error',
        text: getErrorMessage(err) || t('injection.injectFailed'),
      });
    }
  };

  const handleClearAllConfirm = async () => {
    setShowClearAllConfirm(false);
    setClearingBusy(true);
    try {
      await clearAllErrors();
      setMessage({ type: 'success', text: t('injection.clearAllSuccess') });
      refetchErrors();
    } catch (err: unknown) {
      setMessage({
        type: 'error',
        text: getErrorMessage(err) || t('injection.clearAllFailed'),
      });
    } finally {
      setClearingBusy(false);
    }
  };

  const handleClearSpecific = async (device: string, iface: string, errorType: string) => {
    setClearingBusy(true);
    try {
      await clearError(device, iface, errorType);
      setMessage({
        type: 'success',
        text: t('injection.clearSpecificSuccess', { deviceIp: device, iface }),
      });
      refetchErrors();
    } catch (err: unknown) {
      setMessage({
        type: 'error',
        text: getErrorMessage(err) || t('injection.clearSpecificFailed'),
      });
    } finally {
      setClearingBusy(false);
    }
  };

  const busy = isSubmitting || clearingBusy;

  const activeErrorColumns: DataTableColumn<(typeof activeRows)[number]>[] = [
    {
      key: 'deviceIp',
      header: t('injection.tableHeaderDeviceIp'),
      cell: (row) => row.deviceIp,
    },
    { key: 'iface', header: t('injection.tableHeaderInterface'), cell: (row) => row.iface },
    { key: 'errorType', header: t('injection.tableHeaderErrorType'), cell: (row) => row.errorType },
    {
      key: 'value',
      header: t('injection.tableHeaderValue'),
      cell: (row) => (
        <Tag colorScheme="yellow">
          {row.value.prefixBits !== undefined
            ? `/${row.value.prefixBits}`
            : (row.value.address ??
              (row.errorType === 'High Utilization'
                ? `${row.value.value}%`
                : `${row.value.value}/s`))}
        </Tag>
      ),
    },
    {
      key: 'actions',
      header: t('injection.tableHeaderActions'),
      cell: (row) => (
        <Button
          type="button"
          size="xs"
          variant="ghost"
          tone="blue"
          onClick={() => handleClearSpecific(row.deviceIp, row.iface, row.errorType)}
          action="inject"
          disabled={busy}
          aria-label={t('injection.clearOneAriaLabel', {
            deviceIp: row.deviceIp,
            iface: row.iface,
          })}
        >
          {t('injection.clearOneButton')}
        </Button>
      ),
    },
  ];

  return (
    <div className="stack-lg" data-testid="interface-fault-panel">
      {/* Injection Form */}
      <Card>
        <CardContent>
          <form onSubmit={handleSubmit(onInject)} className="stack-lg">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-comfortable">
              {/* Device Selector */}
              <div>
                <label htmlFor="error-device" className="block text-sm font-medium mb-2">
                  {t('injection.deviceLabel')}
                </label>
                <select
                  id="error-device"
                  {...register('selectedDevice')}
                  className="w-full px-3 py-row bg-bg-elevated border border-border-default rounded-md focus:outline-none focus:ring-2 focus:ring-status-info"
                >
                  <option value="">{t('injection.deviceSelectPlaceholder')}</option>
                  {errorInfo?.targets?.map((target) => (
                    <option key={target.device} value={target.device}>
                      {target.device}
                      {target.address ? ` (${target.address})` : ''}
                    </option>
                  ))}
                </select>
                {errors.selectedDevice ? (
                  <p className="text-xs text-status-error mt-tight">
                    {errors.selectedDevice.message}
                  </p>
                ) : null}
              </div>

              {/* Interface Selector — scoped to the selected device's own
                  interfaces (#897 p5f) so a typo can't silently no-op. */}
              <div>
                <label htmlFor="error-interface" className="block text-sm font-medium mb-2">
                  {t('injection.interfaceLabel')}
                </label>
                <select
                  id="error-interface"
                  {...register('selectedInterface')}
                  disabled={!selectedTarget?.interfaces.length}
                  className="w-full px-3 py-row bg-bg-elevated border border-border-default rounded-md focus:outline-none focus:ring-2 focus:ring-status-info disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  <option value="">{t('injection.interfaceSelectPlaceholder')}</option>
                  {selectedTarget?.interfaces.map((iface) => (
                    <option key={iface} value={iface}>
                      {iface}
                    </option>
                  ))}
                </select>
                {!selectedDevice ? (
                  <SmallText className="text-text-muted mt-tight">
                    {t('injection.interfaceNoDeviceHint')}
                  </SmallText>
                ) : selectedTarget && selectedTarget.interfaces.length === 0 ? (
                  <SmallText className="text-text-muted mt-tight">
                    {t('injection.interfaceNoneHint')}
                  </SmallText>
                ) : null}
                {errors.selectedInterface ? (
                  <p className="text-xs text-status-error mt-tight">
                    {errors.selectedInterface.message}
                  </p>
                ) : null}
              </div>

              {/* Error Type Selector */}
              <div>
                <label htmlFor="error-type" className="block text-sm font-medium mb-2">
                  {t('injection.errorTypeLabel')}
                </label>
                <select
                  id="error-type"
                  {...register('selectedErrorType')}
                  className="w-full px-3 py-row bg-bg-elevated border border-border-default rounded-md focus:outline-none focus:ring-2 focus:ring-status-info"
                >
                  <option value="">{t('injection.errorTypeSelectPlaceholder')}</option>
                  {errorInfo?.availableTypes?.map((type: ErrorType) => (
                    <option
                      key={type.type}
                      value={type.type}
                      disabled={!!selectedInterface && !supported.includes(type.type)}
                    >
                      {type.type}
                    </option>
                  ))}
                </select>
                {selectedErrorType && errorInfo?.availableTypes && (
                  <SmallText className="text-text-muted mt-tight">
                    {
                      errorInfo.availableTypes.find(
                        (et: ErrorType) => et.type === selectedErrorType,
                      )?.description
                    }
                  </SmallText>
                )}
                {errors.selectedErrorType ? (
                  <p className="text-xs text-status-error mt-tight">
                    {errors.selectedErrorType.message}
                  </p>
                ) : null}
              </div>

              {/* Value Slider */}
              {addressed ? (
                <Input
                  label={t('deviceFault.conflictAddress')}
                  value={address}
                  onChange={(event) => setAddress(event.target.value)}
                />
              ) : prefixed ? (
                <Input
                  data-testid="interface-fault-prefix"
                  label={t('injection.prefixLabel')}
                  hint={t('injection.prefixHelper')}
                  error={prefix !== '' && !validPrefix ? t('injection.prefixInvalid') : undefined}
                  type="number"
                  min={0}
                  max={32}
                  step={1}
                  value={prefix}
                  onChange={(event) => setPrefix(event.target.value)}
                  aria-invalid={prefix !== '' && !validPrefix}
                />
              ) : (
                <div>
                  <label htmlFor="error-value" className="block text-sm font-medium mb-2">
                    {t('injection.valueLabel', {
                      value:
                        selectedErrorType === 'High Utilization'
                          ? `${errorValue}%`
                          : `${errorValue}/s`,
                    })}
                  </label>
                  <input
                    id="error-value"
                    type="range"
                    min="0"
                    max="100"
                    {...register('errorValue', { valueAsNumber: true })}
                    className="w-full h-2 bg-bg-elevated rounded-lg appearance-none cursor-pointer"
                  />
                  <SmallText className="text-text-muted">{t('injection.valueHelper')}</SmallText>
                  {errors.errorValue ? (
                    <p className="text-xs text-status-error mt-tight">
                      {errors.errorValue.message}
                    </p>
                  ) : null}
                </div>
              )}
            </div>

            {/* Message Display */}
            {visibleMessage && (
              <div
                role={visibleMessage.type === 'error' ? 'alert' : 'status'}
                className={`pad-sm rounded ${
                  visibleMessage.type === 'success'
                    ? 'bg-status-success/10 text-status-success border border-status-success/20'
                    : 'bg-status-error/10 text-status-error border border-status-error/20'
                }`}
              >
                {visibleMessage.text}
              </div>
            )}

            {/* Action Buttons */}
            <div className="flex gap-default">
              <Button
                action="inject"
                type="submit"
                data-testid="apply-interface-fault"
                disabled={
                  busy ||
                  !supported.includes(selectedErrorType) ||
                  (addressed && !validFaultAddress(address)) ||
                  (prefixed && !validPrefix)
                }
              >
                {isSubmitting ? t('injection.injectingButton') : t('injection.injectButton')}
              </Button>
              <Button
                type="button"
                onClick={() => setShowClearAllConfirm(true)}
                action="inject"
                disabled={busy}
                variant="secondary"
              >
                {t('injection.clearAllButton')}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      {/* Active Errors Table */}
      {Object.keys(activeErrors).length > 0 && (
        <Card>
          <CardContent>
            <h3 className="heading-3 mb-content">{t('injection.activeErrorsTitle')}</h3>
            <DataTable
              rows={activeRows}
              columns={activeErrorColumns}
              getRowKey={(row) => `${row.deviceIp}-${row.iface}-${row.errorType}`}
              emptyMessage={null}
            />
          </CardContent>
        </Card>
      )}

      {/* Clear All Confirmation Modal */}
      <ConfirmModal
        isOpen={showClearAllConfirm}
        onConfirm={handleClearAllConfirm}
        action="inject"
        onCancel={() => setShowClearAllConfirm(false)}
        title={t('injection.clearAllConfirmTitle')}
        message={t('injection.clearAllConfirmMessage')}
        confirmLabel={t('injection.clearAllConfirmLabel')}
        confirmTone="red"
      />
    </div>
  );
};

import { useActionPermission } from '../contexts/ScopeContext';
