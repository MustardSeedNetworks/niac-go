import { type FC, type KeyboardEvent } from 'react';
import { Network, Edit3, Copy, Trash2 } from 'lucide-react';
import { Card, CardContent, Tag } from '../../ui';
import { deviceTypeIcons, deviceTypeColors, getDeviceColorClasses } from '../../constants/deviceTypes';
import type { DeviceViewProps } from './types';

export const DeviceCardView: FC<DeviceViewProps> = ({
  devices,
  selectedDevices,
  onSelectDevice,
  onEdit,
  onClone,
  onDelete,
  getDeviceProtocols,
}) => {
  const handleCardKeyDown = (e: KeyboardEvent, hostname: string) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      onEdit(hostname);
    }
  };

  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
      {devices.map((device) => {
        const DeviceIcon = deviceTypeIcons[device.type ?? 'unknown'];
        const typeColor = deviceTypeColors[device.type ?? 'unknown'];
        const colorClasses = getDeviceColorClasses(typeColor);
        const deviceProtocols = getDeviceProtocols(device);

        return (
          <div
            key={device.hostname}
            role="button"
            tabIndex={0}
            aria-label={`Edit device ${device.hostname}`}
            onClick={() => onEdit(device.hostname)}
            onKeyDown={(e) => handleCardKeyDown(e, device.hostname)}
            className="cursor-pointer group focus:outline-none focus:ring-2 focus:ring-violet-500 focus:ring-offset-2 focus:ring-offset-gray-900 rounded-2xl"
          >
            <Card
              className={`border-white/5 bg-gray-900/70 hover:border-violet-500/30 transition-all h-full ${
                selectedDevices.has(device.hostname) ? 'ring-2 ring-violet-500/50 border-violet-500/30' : ''
              }`}
            >
              <CardContent className="p-4 space-y-3">
                {/* Header with checkbox and type icon */}
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-3">
                    <div
                      onClick={(e) => {
                        e.stopPropagation();
                        onSelectDevice(device.hostname);
                      }}
                      className="flex items-center"
                    >
                      <input
                        type="checkbox"
                        checked={selectedDevices.has(device.hostname)}
                        onChange={() => {}}
                        className="h-4 w-4 rounded border-gray-600 bg-gray-800 text-violet-500 focus:ring-violet-500 cursor-pointer"
                      />
                    </div>
                    <div className={`p-2 rounded-lg ${colorClasses.bg}`}>
                      <DeviceIcon className={`h-5 w-5 ${colorClasses.text}`} />
                    </div>
                  </div>
                  <Tag colorScheme={typeColor} className="text-xs capitalize">
                    {device.type?.replace('_', ' ') || 'unknown'}
                  </Tag>
                </div>

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
                      <span className="text-xs text-gray-500">
                        +{device.ips.length - 1}
                      </span>
                    )}
                  </div>
                </div>

                {/* MAC Address */}
                <div className="text-xs text-gray-500 font-mono truncate">
                  {device.mac}
                </div>

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

                {/* Actions */}
                <div className="flex justify-end gap-1 pt-2 border-t border-white/5">
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      onEdit(device.hostname);
                    }}
                    className="p-2 text-gray-400 hover:text-violet-300 hover:bg-white/10 rounded-lg transition-colors"
                    title="Edit device"
                  >
                    <Edit3 className="h-4 w-4" />
                  </button>
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      onClone(device.hostname);
                    }}
                    className="p-2 text-gray-400 hover:text-blue-300 hover:bg-white/10 rounded-lg transition-colors"
                    title="Clone device"
                  >
                    <Copy className="h-4 w-4" />
                  </button>
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      onDelete(device.hostname);
                    }}
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
