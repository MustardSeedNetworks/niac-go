import { type FC, useState } from 'react';
import {
  Card,
  CardContent,
  Button,
  Tag,
  SmallText,
} from '@krisarmstrong/web-foundation';
import { useApiResource } from '../hooks/useApiResource';
import {
  fetchDevices,
  fetchErrorTypes,
  injectError,
  clearError,
  clearAllErrors,
} from '../api/client';

export const ErrorInjectionPanel: FC = () => {
  const { data: devices } = useApiResource(fetchDevices, []);
  const { data: errorInfo, refetch: refetchErrors } = useApiResource(fetchErrorTypes, [], { intervalMs: 5000 });

  const [selectedDevice, setSelectedDevice] = useState('');
  const [selectedInterface, setSelectedInterface] = useState('');
  const [selectedErrorType, setSelectedErrorType] = useState('');
  const [errorValue, setErrorValue] = useState(50);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const handleInject = async () => {
    if (!selectedDevice || !selectedInterface || !selectedErrorType) {
      setMessage({ type: 'error', text: 'Please select device, interface, and error type' });
      return;
    }

    setIsSubmitting(true);
    setMessage(null);

    try {
      await injectError({
        device_ip: selectedDevice,
        interface: selectedInterface,
        error_type: selectedErrorType,
        value: errorValue,
      });
      setMessage({ type: 'success', text: 'Error injected successfully' });
      refetchErrors();
    } catch (err: any) {
      setMessage({ type: 'error', text: err.message || 'Failed to inject error' });
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleClearAll = async () => {
    if (!confirm('Clear all injected errors?')) return;

    setIsSubmitting(true);
    try {
      await clearAllErrors();
      setMessage({ type: 'success', text: 'All errors cleared' });
      refetchErrors();
    } catch (err: any) {
      setMessage({ type: 'error', text: err.message || 'Failed to clear errors' });
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleClearSpecific = async (deviceIP: string, iface: string) => {
    setIsSubmitting(true);
    try {
      await clearError(deviceIP, iface);
      setMessage({ type: 'success', text: `Cleared error on ${deviceIP} ${iface}` });
      refetchErrors();
    } catch (err: any) {
      setMessage({ type: 'error', text: err.message || 'Failed to clear error' });
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
              <label className="block text-sm font-medium mb-2">Device</label>
              <select
                value={selectedDevice}
                onChange={(e) => setSelectedDevice(e.target.value)}
                className="w-full px-3 py-2 bg-gray-800 border border-gray-700 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="">Select device...</option>
                {devices?.map((dev) => (
                  <option key={dev.name} value={dev.ips?.[0]}>
                    {dev.name} ({dev.ips?.[0]})
                  </option>
                ))}
              </select>
            </div>

            {/* Interface Input */}
            <div>
              <label className="block text-sm font-medium mb-2">Interface</label>
              <input
                type="text"
                value={selectedInterface}
                onChange={(e) => setSelectedInterface(e.target.value)}
                placeholder="e.g., eth0, GigabitEthernet0/1"
                className="w-full px-3 py-2 bg-gray-800 border border-gray-700 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>

            {/* Error Type Selector */}
            <div>
              <label className="block text-sm font-medium mb-2">Error Type</label>
              <select
                value={selectedErrorType}
                onChange={(e) => setSelectedErrorType(e.target.value)}
                className="w-full px-3 py-2 bg-gray-800 border border-gray-700 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="">Select error type...</option>
                {errorInfo?.available_types?.map((type: any) => (
                  <option key={type.type} value={type.type}>
                    {type.type}
                  </option>
                ))}
              </select>
              {selectedErrorType && errorInfo?.available_types && (
                <SmallText className="text-gray-400 mt-1">
                  {errorInfo.available_types.find((t: any) => t.type === selectedErrorType)?.description}
                </SmallText>
              )}
            </div>

            {/* Value Slider */}
            <div>
              <label className="block text-sm font-medium mb-2">
                Value: {errorValue}%
              </label>
              <input
                type="range"
                min="0"
                max="100"
                value={errorValue}
                onChange={(e) => setErrorValue(parseInt(e.target.value))}
                className="w-full h-2 bg-gray-700 rounded-lg appearance-none cursor-pointer"
              />
              <SmallText className="text-gray-400">
                0 = No errors, 100 = Maximum errors
              </SmallText>
            </div>
          </div>

          {/* Message Display */}
          {message && (
            <div className={`p-3 rounded ${
              message.type === 'success'
                ? 'bg-green-500/10 text-green-400 border border-green-500/20'
                : 'bg-red-500/10 text-red-400 border border-red-500/20'
            }`}>
              {message.text}
            </div>
          )}

          {/* Action Buttons */}
          <div className="flex gap-3">
            <Button
              onClick={handleInject}
              disabled={isSubmitting}
            >
              {isSubmitting ? 'Injecting...' : 'Inject Error'}
            </Button>
            <Button
              onClick={handleClearAll}
              disabled={isSubmitting}
              variant="secondary"
            >
              Clear All Errors
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Active Errors Table */}
      {errorInfo?.active_errors && Object.keys(errorInfo.active_errors).length > 0 && (
        <Card>
          <CardContent>
            <h3 className="text-lg font-semibold mb-4">Active Errors</h3>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-gray-700">
                    <th className="text-left py-2 px-2">Device IP</th>
                    <th className="text-left py-2 px-2">Interface</th>
                    <th className="text-left py-2 px-2">Error Type</th>
                    <th className="text-left py-2 px-2">Value</th>
                    <th className="text-left py-2 px-2">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {Object.entries(errorInfo.active_errors).map(([deviceIP, interfaces]: [string, any]) =>
                    Object.entries(interfaces).map(([iface, errorTypes]: [string, any]) =>
                      Object.entries(errorTypes).map(([errorType, value]: [string, any]) => (
                        <tr key={`${deviceIP}-${iface}-${errorType}`} className="border-b border-gray-800">
                          <td className="py-2 px-2">{deviceIP}</td>
                          <td className="py-2 px-2">{iface}</td>
                          <td className="py-2 px-2">{errorType}</td>
                          <td className="py-2 px-2">
                            <Tag colorScheme="yellow">{value}%</Tag>
                          </td>
                          <td className="py-2 px-2">
                            <button
                              onClick={() => handleClearSpecific(deviceIP, iface)}
                              disabled={isSubmitting}
                              className="text-blue-400 hover:text-blue-300 text-sm"
                            >
                              Clear
                            </button>
                          </td>
                        </tr>
                      ))
                    )
                  )}
                </tbody>
              </table>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
};
