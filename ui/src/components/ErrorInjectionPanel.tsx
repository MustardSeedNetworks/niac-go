import { valibotResolver } from '@hookform/resolvers/valibot';
import { useEffect, useState } from 'react';
import { type SubmitHandler, useForm } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { useSearchParams } from 'react-router';
import { validFaultAddress } from '../api/behavior-fault-types';
import { clearAllErrors, clearError, injectError } from '../api/client';
import type { ErrorType } from '../api/types';
import { useAppState } from '../contexts/AppContext';
import { type ErrorInjectionFormFields, ErrorInjectionSchema } from '../schemas/forms';
import { Alert } from '../shared/Alert';
import { Card } from '../shared/Card';
import { ConfirmDialog } from '../shared/ConfirmDialog';
import { DataTable, type DataTableColumn } from '../shared/DataTable';
import { FormField } from '../shared/FormField';
import { Input } from '../shared/Input';
import { Select } from '../shared/Select';
import { ActionButton } from '../ui/ActionButton';
import { useDialogLabels } from '../ui/dialogLabels';
import { SmallText } from '../ui/Typography';
import { useTableLabels } from '../ui/tableLabels';
import { getErrorMessage } from '../utils/format';
import { TableSection } from './TableSection';

export function ErrorInjectionPanel() {
  const dialogLabels = useDialogLabels();
  const permission = useActionPermission('inject');
  const { t } = useTranslation('errors');
  const { t: tPages } = useTranslation('pages');
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

  const labels = useTableLabels(t('injection.activeErrorsTitle').toLowerCase());
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
        <StatusPill
          tone="warning"
          label={
            row.value.prefixBits !== undefined
              ? `/${row.value.prefixBits}`
              : (row.value.address ??
                (row.errorType === 'High Utilization'
                  ? `${row.value.value}%`
                  : `${row.value.value}/s`))
          }
        />
      ),
    },
    {
      key: 'actions',
      header: t('injection.tableHeaderActions'),
      cell: (row) => (
        <ActionButton
          action="inject"
          variant="ghost"
          size="sm"
          type="button"
          onClick={() => handleClearSpecific(row.deviceIp, row.iface, row.errorType)}
          disabled={busy}
          aria-label={t('injection.clearOneAriaLabel', {
            deviceIp: row.deviceIp,
            iface: row.iface,
          })}
        >
          {t('injection.clearOneButton')}
        </ActionButton>
      ),
    },
  ];

  return (
    <div className="stack-lg" data-testid="interface-fault-panel">
      {/* Injection Form */}
      <Card
        title={tPages('traffic.page.errorInjectionTitle')}
        subtitle={tPages('traffic.page.errorInjectionDescription')}
      >
        <div>
          <form onSubmit={handleSubmit(onInject)} className="stack-lg">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-comfortable">
              {/* Device Selector */}
              <FormField label={t('injection.deviceLabel')} id="error-device">
                <Select id="error-device" {...register('selectedDevice')}>
                  <option value="">{t('injection.deviceSelectPlaceholder')}</option>
                  {errorInfo?.targets?.map((target) => (
                    <option key={target.device} value={target.device}>
                      {target.device}
                      {target.address ? ` (${target.address})` : ''}
                    </option>
                  ))}
                </Select>
                {errors.selectedDevice ? (
                  <p className="text-xs text-status-error mt-tight">
                    {errors.selectedDevice.message}
                  </p>
                ) : null}
              </FormField>

              {/* Interface Selector — scoped to the selected device's own
                  interfaces (#897 p5f) so a typo can't silently no-op. */}
              <FormField label={t('injection.interfaceLabel')} id="error-interface">
                <Select
                  id="error-interface"
                  {...register('selectedInterface')}
                  disabled={!selectedTarget?.interfaces.length}
                >
                  <option value="">{t('injection.interfaceSelectPlaceholder')}</option>
                  {selectedTarget?.interfaces.map((iface) => (
                    <option key={iface} value={iface}>
                      {iface}
                    </option>
                  ))}
                </Select>
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
              </FormField>

              {/* Error Type Selector */}
              <FormField label={t('injection.errorTypeLabel')} id="error-type">
                <Select id="error-type" {...register('selectedErrorType')}>
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
                </Select>
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
              </FormField>

              {/* Value Slider */}
              {addressed ? (
                <FormField label={t('deviceFault.conflictAddress')}>
                  {(control) => (
                    <Input
                      {...control}
                      value={address}
                      onChange={(event) => setAddress(event.target.value)}
                    />
                  )}
                </FormField>
              ) : prefixed ? (
                <FormField
                  label={t('injection.prefixLabel')}
                  hint={t('injection.prefixHelper')}
                  error={prefix !== '' && !validPrefix ? t('injection.prefixInvalid') : undefined}
                >
                  {(control) => (
                    <Input
                      {...control}
                      data-testid="interface-fault-prefix"
                      type="number"
                      min={0}
                      max={32}
                      step={1}
                      value={prefix}
                      onChange={(event) => setPrefix(event.target.value)}
                      aria-invalid={prefix !== '' && !validPrefix}
                    />
                  )}
                </FormField>
              ) : (
                <FormField
                  label={t('injection.valueLabel', {
                    value:
                      selectedErrorType === 'High Utilization'
                        ? `${errorValue}%`
                        : `${errorValue}/s`,
                  })}
                  id="error-value"
                >
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
                </FormField>
              )}
            </div>

            {/* Message Display */}
            {visibleMessage && (
              <Alert tone={visibleMessage.type === 'success' ? 'success' : 'error'}>
                {visibleMessage.text}
              </Alert>
            )}

            {/* Action Buttons */}
            <div className="flex gap-default">
              <ActionButton
                variant="primary"
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
              </ActionButton>
              <ActionButton
                action="inject"
                variant="secondary"
                type="button"
                onClick={() => setShowClearAllConfirm(true)}
                disabled={busy}
              >
                {t('injection.clearAllButton')}
              </ActionButton>
            </div>
          </form>
        </div>
      </Card>

      {/* Active Errors Table */}
      {Object.keys(activeErrors).length > 0 && (
        <TableSection title={t('injection.activeErrorsTitle')}>
          <DataTable
            rows={activeRows}
            columns={activeErrorColumns}
            getRowKey={(row) => `${row.deviceIp}-${row.iface}-${row.errorType}`}
            caption={t('injection.activeErrorsTitle')}
            labels={labels}
            empty={null}
          />
        </TableSection>
      )}

      {/* Clear All Confirmation Modal */}
      <ConfirmDialog
        {...dialogLabels}
        open={showClearAllConfirm}
        onConfirm={handleClearAllConfirm}
        onCancel={() => setShowClearAllConfirm(false)}
        title={t('injection.clearAllConfirmTitle')}
        message={t('injection.clearAllConfirmMessage')}
        confirmLabel={t('injection.clearAllConfirmLabel')}
      />
    </div>
  );
}

import { useActionPermission } from '../contexts/ScopeContext';
import { StatusPill } from '../shared/StatusPill';
