import type { FC } from 'react';
import { memo } from 'react';
import {
  topologyDeviceColors as deviceColors,
  topologyDeviceIcons as deviceIcons,
} from '../../constants/device-types';
import type { DeviceNodeData } from './topologyTypes';

// ============================================================================
// Custom Device Node Component
// ============================================================================

export const TopologyNode: FC<{ data: DeviceNodeData; selected?: boolean }> = memo(
  ({ data, selected }) => {
    const deviceType = (data.type as string)?.toLowerCase() || 'unknown';
    const Icon = deviceIcons[deviceType] || deviceIcons.unknown;
    const color = deviceColors[deviceType] || deviceColors.unknown;

    const statusColor = {
      online: 'bg-green-500',
      offline: 'bg-gray-500',
      warning: 'bg-yellow-500',
    }[data.status || 'online'];

    return (
      <button
        type="button"
        className={`
        relative px-4 py-3 rounded-xl border-2 transition-all duration-200 text-left
        ${selected ? 'ring-2 ring-brand-500 ring-offset-2 ring-offset-gray-900' : ''}
        hover:shadow-lg hover:shadow-black/30
      `}
        style={{
          backgroundColor: 'var(--color-bg-elevated)',
          borderColor: selected ? color : 'var(--color-border-muted)',
          minWidth: '140px',
        }}
        onClick={() => data.onClick?.(data.label)}
      >
        {/* Status indicator */}
        <div
          className={`absolute -top-1 -right-1 w-3 h-3 rounded-full ${statusColor} border-2 border-gray-900`}
        />

        {/* Icon and name */}
        <div className="flex items-center gap-3">
          <div
            className="p-2 rounded-lg"
            style={{
              backgroundColor: `color-mix(in srgb, ${color} 20%, transparent)`,
            }}
          >
            <Icon className="w-5 h-5" />
          </div>
          <div className="min-w-0 flex-1">
            <div className="font-semibold text-white text-sm truncate">{data.label}</div>
            <div className="text-xs text-gray-400 capitalize">{data.type}</div>
          </div>
        </div>

        {/* IPs */}
        {data.ips && data.ips.length > 0 && (
          <div className="mt-2 pt-2 border-t border-white/10">
            <div className="text-xs font-mono text-gray-400 truncate">
              {data.ips[0]}
              {data.ips.length > 1 && (
                <span className="text-gray-500"> +{data.ips.length - 1}</span>
              )}
            </div>
          </div>
        )}

        {/* Protocols */}
        {data.protocols && data.protocols.length > 0 && (
          <div className="mt-2 flex flex-wrap gap-1">
            {data.protocols.slice(0, 3).map((proto) => (
              <span
                key={proto}
                className="px-1.5 py-0.5 text-[10px] rounded-md bg-white/10 text-gray-300"
              >
                {proto}
              </span>
            ))}
            {data.protocols.length > 3 && (
              <span className="px-1.5 py-0.5 text-[10px] rounded-md bg-white/5 text-gray-500">
                +{data.protocols.length - 3}
              </span>
            )}
          </div>
        )}
      </button>
    );
  },
);

TopologyNode.displayName = 'TopologyNode';
