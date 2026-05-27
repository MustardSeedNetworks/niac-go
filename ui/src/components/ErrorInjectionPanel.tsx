import { type FC, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  clearAllErrors,
  clearError,
  fetchDevices,
  fetchErrorTypes,
  injectError,
} from '../api/client';
import type { ErrorType } from '../api/types';
import { useApiResource } from '../hooks/useApiResource';
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

  const [selectedDevice, setSelectedDevice] = useState('');
  const [selectedInterface, setSelectedInterface] = useState('');
  const [selectedErrorType, setSelectedErrorType] = useState('');
  const [errorValue, setErrorValue] = useState(50);
  const [isSubmitting, setIsSubmitting] = useState(false);
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

  const handleInject = async () => {
    if (!(selectedDevice && selectedInterface && selectedErrorType)) {
      setMessage({
        type: 'error',
        text: t('injection.selectMissing'),
      });
      return;
    }

    setIsSubmitting(true);
    setMessage(null);

    try {
      await injectError({
        deviceIp: selectedDevice,
        interface: selectedInterface,
        errorType: selectedErrorType,
        value: errorValue,
      });
      setMessage({ type: 'success', text: t('injection.injectSuccess') });
      refetchErrors();
    } catch (err: unknown) {
      setMessage({
        type: 'error',
        text: getErrorMessage(err) || t('injection.injectFailed'),
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleClearAllConfirm = async () => {
    setShowClearAllConfirm(false);
    setIsSubmitting(true);
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
      setIsSubmitting(false);
    }
  };

  const handleClearSpecific = async (deviceIp: string, iface: string) => {
    setIsSubmitting(true);
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
      setIsSubmitting(false);
    }
  };

  return (
    <div className="space-y-4">
      {/* Injection Form */}
      <Card>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {/* Device Selector */}
            <div>
              <label htmlFor="error-device" className="block text-sm font-medium mb-2">
                {t('injection.deviceLabel')}
              </label>
              <select
                id="error-device"
                value={selectedDevice}
                onChange={(e) => setSelectedDevice(e.target.value)}
                className="w-full px-3 py-2 bg-bg-elevated border border-border-default rounded-md focus:outline-none focus:ring-2 focus:ring-status-info"
              >
                <option value="">{t('injection.deviceSelectPlaceholder')}</option>
                {devices?.map((dev) => (
                  <option key={dev.name} value={dev.ips?.[0]}>
                    {dev.name} ({dev.ips?.[0]})
                  </option>
                ))}
              </select>
            </div>

            {/* Interface Input */}
            <div>
              <label htmlFor="error-interface" className="block text-sm font-medium mb-2">
                {t('injection.interfaceLabel')}
              </label>
              <input
                id="error-interface"
                type="text"
                value={selectedInterface}
                onChange={(e) => setSelectedInterface(e.target.value)}
                placeholder={t('injection.interfacePlaceholder')}
                className="w-full px-3 py-2 bg-bg-elevated border border-border-default rounded-md focus:outline-none focus:ring-2 focus:ring-status-info"
              />
            </div>

            {/* Error Type Selector */}
            <div>
              <label htmlFor="error-type" className="block text-sm font-medium mb-2">
                {t('injection.errorTypeLabel')}
              </label>
              <select
                id="error-type"
                value={selectedErrorType}
                onChange={(e) => setSelectedErrorType(e.target.value)}
                className="w-full px-3 py-2 bg-bg-elevated border border-border-default rounded-md focus:outline-none focus:ring-2 focus:ring-status-info"
              >
                <option value="">{t('injection.errorTypeSelectPlaceholder')}</option>
                {errorInfo?.availableTypes?.map((type: ErrorType) => (
                  <option key={type.type} value={type.type}>
                    {type.type}
                  </option>
                ))}
              </select>
              {selectedErrorType && errorInfo?.availableTypes && (
                <SmallText className="text-text-muted mt-1">
                  {
                    errorInfo.availableTypes.find((et: ErrorType) => et.type === selectedErrorType)
                      ?.description
                  }
                </SmallText>
              )}
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
                value={errorValue}
                onChange={(e) => setErrorValue(Number.parseInt(e.target.value, 10))}
                className="w-full h-2 bg-bg-elevated rounded-lg appearance-none cursor-pointer"
              />
              <SmallText className="text-text-muted">{t('injection.valueHelper')}</SmallText>
            </div>
          </div>

          {/* Message Display */}
          {message && (
            <div
              className={`p-3 rounded ${
                message.type === 'success'
                  ? 'bg-status-success/10 text-status-success border border-status-success/20'
                  : 'bg-status-error/10 text-status-error border border-status-error/20'
              }`}
            >
              {message.text}
            </div>
          )}

          {/* Action Buttons */}
          <div className="flex gap-3">
            <Button onClick={handleInject} disabled={isSubmitting}>
              {isSubmitting ? t('injection.injectingButton') : t('injection.injectButton')}
            </Button>
            <Button
              onClick={() => setShowClearAllConfirm(true)}
              disabled={isSubmitting}
              variant="secondary"
            >
              {t('injection.clearAllButton')}
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Active Errors Table */}
      {Object.keys(activeErrors).length > 0 && (
        <Card>
          <CardContent>
            <h3 className="text-lg font-semibold mb-4">{t('injection.activeErrorsTitle')}</h3>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border-default">
                    <th className="text-left py-2 px-2">{t('injection.tableHeaderDeviceIp')}</th>
                    <th className="text-left py-2 px-2">{t('injection.tableHeaderInterface')}</th>
                    <th className="text-left py-2 px-2">{t('injection.tableHeaderErrorType')}</th>
                    <th className="text-left py-2 px-2">{t('injection.tableHeaderValue')}</th>
                    <th className="text-left py-2 px-2">{t('injection.tableHeaderActions')}</th>
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
                          <td className="py-2 px-2">{deviceIp}</td>
                          <td className="py-2 px-2">{iface}</td>
                          <td className="py-2 px-2">{errorType}</td>
                          <td className="py-2 px-2">
                            <Tag colorScheme="yellow">{value}%</Tag>
                          </td>
                          <td className="py-2 px-2">
                            <button
                              type="button"
                              onClick={() => handleClearSpecific(deviceIp, iface)}
                              disabled={isSubmitting}
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
