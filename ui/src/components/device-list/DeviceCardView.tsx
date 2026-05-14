import { Copy, Edit3, Network, Trash2 } from 'lucide-react';
import type { FC } from 'react';
import {
  deviceTypeColors,
  deviceTypeIcons,
  getDeviceColorClasses,
} from '../../constants/device-types';
import { useDeviceList } from '../../contexts/DeviceListContext';
import { Card, CardContent } from '../../ui/Card';
import { Tag } from '../../ui/Tag';

export const DeviceCardView: FC = () => {
  const {
    devices,
    selectedDevices,
    onSelectDevice,
    onEdit,
    onClone,
    onDelete,
    getDeviceProtocols,
  } = useDeviceList();
  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
      {devices.map((device) => {
        // Devices in the wild use type values like "ap" or "access-point" that
        // aren't in the DeviceType union. Fall back to 'unknown' so the lookup
        // can never be undefined and crash the page.
        const safeType = device.type && device.type in deviceTypeIcons ? device.type : 'unknown';
        const DeviceIcon = deviceTypeIcons[safeType];
        const typeColor = deviceTypeColors[safeType];
        const colorClasses = getDeviceColorClasses(typeColor);
        const deviceProtocols = getDeviceProtocols(device);

        return (
          <div key={device.hostname} className="rounded-2xl">
            <Card
              className={`group border-white/5 bg-gray-900/70 hover:border-violet-500/30 transition-all h-full ${
                selectedDevices.has(device.hostname)
                  ? 'ring-2 ring-violet-500/50 border-violet-500/30'
                  : ''
              }`}
            >
              <CardContent className="p-4 space-y-3">
                {/* Header with checkbox and type icon */}
                <div className="flex items-start justify-between">
                  <label className="flex items-center gap-3">
                    <input
                      type="checkbox"
                      checked={selectedDevices.has(device.hostname)}
                      onChange={() => onSelectDevice(device.hostname)}
                      className="h-4 w-4 rounded border-gray-600 bg-gray-800 text-violet-500 focus:ring-violet-500 cursor-pointer"
                    />
                    <div className={`p-2 rounded-lg ${colorClasses.bg}`}>
                      <DeviceIcon className={`h-5 w-5 ${colorClasses.text}`} />
                    </div>
                  </label>
                  <Tag colorScheme={typeColor} className="text-xs capitalize">
                    {device.type?.replace('_', ' ') || 'unknown'}
                  </Tag>
                </div>

                <button
                  type="button"
                  onClick={() => onEdit(device.hostname)}
                  className="w-full text-left rounded-lg focus:outline-none focus:ring-2 focus:ring-violet-500 focus:ring-offset-2 focus:ring-offset-gray-900"
                  aria-label={`Edit device ${device.hostname}`}
                >
                  {/* Device name and IP */}
                  <div>
                    <h3 className="font-semibold text-white group-hover:text-violet-300 transition-colors truncate">
                      {device.hostname}
                    </h3>
                    <div className="flex items-center gap-1 mt-1">
                      <Network className="h-3.5 w-3.5 text-gray-500" />
                      <span className="text-sm text-gray-400 font-mono">
                        {device.ip || device.ips?.[0] || 'No IP'}
                      </span>
                      {device.ips && device.ips.length > 1 && (
                        <span className="text-xs text-gray-500">+{device.ips.length - 1}</span>
                      )}
                    </div>
                  </div>

                  {/* MAC Address */}
                  <div className="text-xs text-gray-500 font-mono truncate">{device.mac}</div>

                  {/* Protocols */}
                  <div className="flex flex-wrap gap-1">
                    {deviceProtocols.length > 0 ? (
                      deviceProtocols.slice(0, 3).map((proto) => (
                        <Tag key={proto} colorScheme="gray" className="text-xs">
                          {proto}
                        </Tag>
                      ))
                    ) : (
                      <span className="text-gray-500 text-xs">No protocols</span>
                    )}
                    {deviceProtocols.length > 3 && (
                      <Tag colorScheme="gray" className="text-xs">
                        +{deviceProtocols.length - 3}
                      </Tag>
                    )}
                  </div>
                </button>

                {/* Actions */}
                <div className="flex justify-end gap-1 pt-2 border-t border-white/5">
                  <button
                    type="button"
                    onClick={() => onEdit(device.hostname)}
                    className="p-2 text-gray-400 hover:text-violet-300 hover:bg-white/10 rounded-lg transition-colors"
                    title="Edit device"
                  >
                    <Edit3 className="h-4 w-4" />
                  </button>
                  <button
                    type="button"
                    onClick={() => onClone(device.hostname)}
                    className="p-2 text-gray-400 hover:text-blue-300 hover:bg-white/10 rounded-lg transition-colors"
                    title="Clone device"
                  >
                    <Copy className="h-4 w-4" />
                  </button>
                  <button
                    type="button"
                    onClick={() => onDelete(device.hostname)}
                    className="p-2 text-gray-400 hover:text-red-400 hover:bg-white/10 rounded-lg transition-colors"
                    title="Delete device"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              </CardContent>
            </Card>
          </div>
        );
      })}
    </div>
  );
};
