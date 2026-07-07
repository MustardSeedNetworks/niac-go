import { valibotResolver } from '@hookform/resolvers/valibot';
import { type FC, useEffect, useState } from 'react';
import { type SubmitHandler, useForm } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { useSearchParams } from 'react-router-dom';
import {
  clearAllErrors,
  clearError,
  fetchDevices,
  fetchErrorTypes,
  injectError,
} from '../api/client';
import type { ErrorType } from '../api/types';
import { useApiResource } from '../hooks/useApiResource';
import { type ErrorInjectionFormFields, ErrorInjectionSchema } from '../schemas/forms';
import { Button } from '../ui/Button';
import { Card, CardContent } from '../ui/Card';
import { ConfirmModal } from '../ui/ConfirmModal';
import { Tag } from '../ui/Tag';
import { SmallText } from '../ui/Typography';
import { getErrorMessage } from '../utils/format';

export const ErrorInjectionPanel: FC = () => {
  const { t } = useTranslation('errors');
  const { data: devices } = useApiResource(fetchDevices, []);
  const { data: errorInfo, refetch: refetchErrors } = useApiResource(fetchErrorTypes, [], {
    intervalMs: 5000,
  });
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
  const activeErrors = errorInfo?.activeErrors ?? {};
  const activeEntries = Object.entries(activeErrors) as [
    string,
    Record<string, Record<string, number>>,
  ][];

  const onInject: SubmitHandler<ErrorInjectionFormFields> = async (values) => {
    setMessage(null);
    try {
      await injectError({
        deviceIp: values.selectedDevice,
        interface: values.selectedInterface,
        errorType: values.selectedErrorType,
        value: values.errorValue,
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

  const handleClearSpecific = async (deviceIp: string, iface: string) => {
    setClearingBusy(true);
    try {
      await clearError(deviceIp, iface);
      setMessage({
        type: 'success',
        text: t('injection.clearSpecificSuccess', { deviceIp, iface }),
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

  return (
    <div className="stack-lg">
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
                  {devices?.map((dev) => (
                    <option key={dev.name} value={dev.ips?.[0]}>
                      {dev.name} ({dev.ips?.[0]})
                    </option>
                  ))}
                </select>
                {errors.selectedDevice ? (
                  <p className="text-xs text-status-error mt-tight">
                    {errors.selectedDevice.message}
                  </p>
                ) : null}
              </div>

              {/* Interface Input */}
              <div>
                <label htmlFor="error-interface" className="block text-sm font-medium mb-2">
                  {t('injection.interfaceLabel')}
                </label>
                <input
                  id="error-interface"
                  type="text"
                  {...register('selectedInterface')}
                  placeholder={t('injection.interfacePlaceholder')}
                  className="w-full px-3 py-row bg-bg-elevated border border-border-default rounded-md focus:outline-none focus:ring-2 focus:ring-status-info"
                />
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
                    <option key={type.type} value={type.type}>
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
              <div>
                <label htmlFor="error-value" className="block text-sm font-medium mb-2">
                  {t('injection.valueLabel', { value: errorValue })}
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
                  <p className="text-xs text-status-error mt-tight">{errors.errorValue.message}</p>
                ) : null}
              </div>
            </div>

            {/* Message Display */}
            {message && (
              <div
                className={`pad-sm rounded ${
                  message.type === 'success'
                    ? 'bg-status-success/10 text-status-success border border-status-success/20'
                    : 'bg-status-error/10 text-status-error border border-status-error/20'
                }`}
              >
                {message.text}
              </div>
            )}

            {/* Action Buttons */}
            <div className="flex gap-default">
              <Button type="submit" disabled={busy}>
                {isSubmitting ? t('injection.injectingButton') : t('injection.injectButton')}
              </Button>
              <Button
                type="button"
                onClick={() => setShowClearAllConfirm(true)}
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
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border-default">
                    <th className="text-left py-row px-cell">
                      {t('injection.tableHeaderDeviceIp')}
                    </th>
                    <th className="text-left py-row px-cell">
                      {t('injection.tableHeaderInterface')}
                    </th>
                    <th className="text-left py-row px-cell">
                      {t('injection.tableHeaderErrorType')}
                    </th>
                    <th className="text-left py-row px-cell">{t('injection.tableHeaderValue')}</th>
                    <th className="text-left py-row px-cell">
                      {t('injection.tableHeaderActions')}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {activeEntries.map(([deviceIp, interfaces]) =>
                    Object.entries(interfaces).map(([iface, errorTypes]) =>
                      Object.entries(errorTypes).map(([errorType, value]) => (
                        <tr
                          key={`${deviceIp}-${iface}-${errorType}`}
                          className="border-b border-border-default"
                        >
                          <td className="py-row px-cell">{deviceIp}</td>
                          <td className="py-row px-cell">{iface}</td>
                          <td className="py-row px-cell">{errorType}</td>
                          <td className="py-row px-cell">
                            <Tag colorScheme="yellow">{value}%</Tag>
                          </td>
                          <td className="py-row px-cell">
                            <button
                              type="button"
                              onClick={() => handleClearSpecific(deviceIp, iface)}
                              disabled={busy}
                              className="text-status-info hover:text-status-info text-sm"
                              aria-label={t('injection.clearOneAriaLabel', { deviceIp, iface })}
                            >
                              {t('injection.clearOneButton')}
                            </button>
                          </td>
                        </tr>
                      )),
                    ),
                  )}
                </tbody>
              </table>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Clear All Confirmation Modal */}
      <ConfirmModal
        isOpen={showClearAllConfirm}
        onConfirm={handleClearAllConfirm}
        onCancel={() => setShowClearAllConfirm(false)}
        title={t('injection.clearAllConfirmTitle')}
        message={t('injection.clearAllConfirmMessage')}
        confirmLabel={t('injection.clearAllConfirmLabel')}
        confirmTone="red"
      />
    </div>
  );
};
