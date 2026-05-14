import { Copy, Edit3, Trash2 } from 'lucide-react';
import type { FC } from 'react';
import { deviceTypeColors, deviceTypeIcons } from '../../constants/device-types';
import { iconSizes } from '../../constants/sizes';
import { useDeviceList } from '../../contexts/DeviceListContext';
import { Card, CardContent } from '../../ui/Card';
import { Tag } from '../../ui/Tag';

export const DeviceTableView: FC = () => {
  const {
    devices,
    selectedDevices,
    onSelectDevice,
    onSelectAll,
    onEdit,
    onClone,
    onDelete,
    getDeviceProtocols,
  } = useDeviceList();
  return (
    <Card className="border-white/5 bg-gray-900/70">
      <CardContent className="p-0">
        {/* Table header */}
        <div>
          <div className="flex items-center gap-4 border-b border-white/10 px-4 py-3 bg-gray-950/40">
            <input
              type="checkbox"
              checked={selectedDevices.size === devices.length && devices.length > 0}
              onChange={onSelectAll}
              className="h-4 w-4 rounded border-gray-600 bg-gray-800 text-violet-500 focus:ring-violet-500"
              aria-label="Select all devices"
            />
            <div className="flex-1 grid grid-cols-12 gap-4 text-sm font-medium text-gray-400">
              <div className="col-span-3">Hostname</div>
              <div className="col-span-2">Type</div>
              <div className="col-span-2">IP Address</div>
              <div className="col-span-3">Protocols</div>
              <div className="col-span-2 text-right">Actions</div>
            </div>
          </div>
        </div>

        {/* Device rows */}
        <div className="divide-y divide-white/5">
          {devices.map((device) => {
            // Defensive: wild device types like "ap" / "access-point" aren't
            // in the DeviceType union; fall back to 'unknown' so the lookup
            // can't be undefined.
            const safeType =
              device.type && device.type in deviceTypeIcons ? device.type : 'unknown';
            const DeviceIcon = deviceTypeIcons[safeType];
            const typeColor = deviceTypeColors[safeType];
            const deviceProtocols = getDeviceProtocols(device);

            return (
              <div
                key={device.hostname}
                className={`flex items-center gap-4 px-4 py-3 hover:bg-white/5 transition-colors ${
                  selectedDevices.has(device.hostname) ? 'bg-violet-500/10' : ''
                }`}
              >
                <input
                  type="checkbox"
                  checked={selectedDevices.has(device.hostname)}
                  onChange={() => onSelectDevice(device.hostname)}
                  className="h-4 w-4 rounded border-gray-600 bg-gray-800 text-violet-500 focus:ring-violet-500"
                  aria-label={`Select ${device.hostname}`}
                />
                <div className="flex-1 grid grid-cols-12 gap-4 items-center">
                  {/* Hostname */}
                  <div className="col-span-3 flex items-center gap-2">
                    <DeviceIcon className={`${iconSizes.md} text-gray-400`} />
                    <button
                      type="button"
                      onClick={() => onEdit(device.hostname)}
                      className="text-white hover:text-violet-300 font-medium truncate"
                    >
                      {device.hostname}
                    </button>
                  </div>

                  {/* Type */}
                  <div className="col-span-2">
                    <Tag colorScheme={typeColor} className="text-xs capitalize">
                      {device.type?.replace('_', ' ') || 'unknown'}
                    </Tag>
                  </div>

                  {/* IP */}
                  <div className="col-span-2">
                    <span className="text-gray-300 text-sm font-mono">
                      {device.ip || device.ips?.[0] || '—'}
                    </span>
                    {device.ips && device.ips.length > 1 && (
                      <span className="ml-1 text-gray-500 text-xs">+{device.ips.length - 1}</span>
                    )}
                  </div>

                  {/* Protocols */}
                  <div className="col-span-3 flex flex-wrap gap-1">
                    {deviceProtocols.length > 0 ? (
                      deviceProtocols.slice(0, 4).map((proto) => (
                        <Tag key={proto} colorScheme="gray" className="text-xs">
                          {proto}
                        </Tag>
                      ))
                    ) : (
                      <span className="text-gray-500 text-sm">No protocols</span>
                    )}
                    {deviceProtocols.length > 4 && (
                      <Tag colorScheme="gray" className="text-xs">
                        +{deviceProtocols.length - 4}
                      </Tag>
                    )}
                  </div>

                  {/* Actions */}
                  <div className="col-span-2 flex justify-end gap-1">
                    <button
                      type="button"
                      onClick={() => onEdit(device.hostname)}
                      className="p-2 text-gray-400 hover:text-violet-300 hover:bg-white/5 rounded-lg transition-colors"
                      title="Edit device"
                    >
                      <Edit3 className={iconSizes.md} />
                    </button>
                    <button
                      type="button"
                      onClick={() => onClone(device.hostname)}
                      className="p-2 text-gray-400 hover:text-blue-300 hover:bg-white/5 rounded-lg transition-colors"
                      title="Clone device"
                    >
                      <Copy className={iconSizes.md} />
                    </button>
                    <button
                      type="button"
                      onClick={() => onDelete(device.hostname)}
                      className="p-2 text-gray-400 hover:text-red-400 hover:bg-white/5 rounded-lg transition-colors"
                      title="Delete device"
                    >
                      <Trash2 className={iconSizes.md} />
                    </button>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </CardContent>
    </Card>
  );
};
