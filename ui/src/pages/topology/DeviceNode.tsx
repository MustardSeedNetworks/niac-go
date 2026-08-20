/**
 * Custom Device Node Component for React Flow topology visualization
 */

import { Handle, Position } from '@xyflow/react';
import { type FC, memo } from 'react';
import { useTranslation } from 'react-i18next';
import {
  topologyDeviceColors as deviceColors,
  getTopologyDeviceIcon,
} from '../../constants/device-types';
import type { DeviceNodeData } from './types';

interface DeviceNodeProps {
  data: DeviceNodeData;
  selected?: boolean;
}

/**
 * DeviceNode renders a network device in the topology graph.
 * Displays device icon, name, type, IPs, and protocols.
 */
export const DeviceNode: FC<DeviceNodeProps> = memo(({ data, selected }) => {
  const { t } = useTranslation('pages');
  const deviceType = (data.type as string)?.toLowerCase() || 'unknown';
  const Icon = getTopologyDeviceIcon(deviceType);
  const color = deviceColors[deviceType] || deviceColors.unknown;

  // The card truncates the device name and shows `+N` overflow for IPs and
  // protocols, so the full data is invisible without a tooltip. Build a
  // multi-line description and surface it through both title (native hover
  // tooltip) and aria-label (screen readers).
  const tooltipLines: string[] = [
    t('topology.deviceNode.tooltipSummary', { label: data.label, type: data.type }),
  ];
  if (data.ips && data.ips.length > 0) {
    tooltipLines.push(t('topology.deviceNode.tooltipIps', { ips: data.ips.join(', ') }));
  }
  if (data.protocols && data.protocols.length > 0) {
    tooltipLines.push(
      t('topology.deviceNode.tooltipProtocols', { protocols: data.protocols.join(', ') }),
    );
  }
  const tooltip = tooltipLines.join('\n');

  return (
    <button
      type="button"
      data-testid="topology-device-node"
      title={tooltip}
      aria-label={tooltip}
      className={`
        relative px-4 py-row-lg rounded-xl border-2 transition-all duration-200 text-left
        ${selected ? 'ring-2 ring-brand-primary ring-offset-2 ring-offset-surface-base' : ''}
        hover:shadow-lg hover:shadow-scrim/30
      `}
      style={{
        backgroundColor: 'var(--color-bg-elevated)',
        borderColor: selected ? color : 'var(--color-border-muted)',
        // Wide enough that 16-char device names (niac-core-sw-01) fit
        // without "..." truncation, capped so packed layouts still
        // breathe. Layout step width below is tuned to this.
        minWidth: '200px',
        maxWidth: '260px',
      }}
      onClick={() => data.onClick?.(data.label)}
    >
      {/* ReactFlow edge anchors. Without these handles the canvas
          renders nodes fine but every edge silently fails to draw —
          there's no spot for the line to attach to.
          Default handles only (left=target, right=source) so edges
          always route left-to-right deterministically. Extra
          top/bottom handles confused ReactFlow's auto-routing into
          drawing edges out the side of the card. */}
      <Handle
        type="target"
        position={Position.Left}
        className="!w-2 !h-2 !bg-brand-accent !border-0"
      />
      <Handle
        type="source"
        position={Position.Right}
        className="!w-2 !h-2 !bg-brand-accent !border-0"
      />

      {/* Icon and name */}
      <div className="flex items-center gap-default">
        <div
          className="pad-xs rounded-lg"
          style={{
            backgroundColor: `color-mix(in srgb, ${color} 20%, transparent)`,
          }}
        >
          <Icon className="w-5 h-5" />
        </div>
        <div className="min-w-0 flex-1">
          <div
            data-testid="topology-device-label"
            className="font-semibold text-text-primary text-sm truncate"
          >
            {data.label}
          </div>
          <div className="text-xs text-text-muted capitalize">{data.type}</div>
        </div>
      </div>

      {/* IPs */}
      {data.ips && data.ips.length > 0 && (
        <div className="mt-inline pt-2 border-t border-surface-border">
          <div className="text-xs font-mono text-text-muted truncate">
            {data.ips[0]}
            {data.ips.length > 1 && (
              <span className="text-text-muted"> +{data.ips.length - 1}</span>
            )}
          </div>
        </div>
      )}

      {/* Protocols */}
      {data.protocols && data.protocols.length > 0 && (
        <div className="mt-inline flex flex-wrap gap-tight">
          {data.protocols.slice(0, 3).map((proto) => (
            <span
              key={proto}
              className="px-1.5 py-0.5 text-[10px] rounded-md bg-surface-hover text-text-secondary"
            >
              {proto}
            </span>
          ))}
          {data.protocols.length > 3 && (
            <span className="px-1.5 py-0.5 text-[10px] rounded-md bg-surface-hover text-text-muted">
              +{data.protocols.length - 3}
            </span>
          )}
        </div>
      )}
    </button>
  );
});

DeviceNode.displayName = 'DeviceNode';
