/**
 * Custom Device Node Component for React Flow topology visualization
 */

import { Handle, Position } from '@xyflow/react';
import { type FC, memo } from 'react';
import { useTranslation } from 'react-i18next';
import { getTopologyDeviceColor, getTopologyDeviceIcon } from '../../constants/device-types';
import type { DeviceNodeData } from './types';

interface DeviceNodeProps {
  data: DeviceNodeData;
  selected?: boolean;
}

/**
 * DeviceNode renders a network device as a symbol with its name beneath it.
 *
 * It used to be a 280x180 card carrying the name, the type, the first IP and
 * up to three protocol chips. At any zoom that fits a real scenario the cards
 * were the diagram — the shape of the network was the thing you could not see
 * (#2056). The symbol carries the device type, the label carries identity, and
 * everything that left the canvas is still on the node's own tooltip and
 * accessible name, one hover away rather than one click.
 */
export const DeviceNode: FC<DeviceNodeProps> = memo(({ data, selected }) => {
  const { t } = useTranslation('pages');
  const deviceType = data.type ?? 'unknown';
  const Symbol = getTopologyDeviceIcon(deviceType);
  const color = getTopologyDeviceColor(deviceType);

  // The node shows a symbol and a name, so IPs and protocols are invisible
  // without this. Surfaced through both title (native hover tooltip) and
  // aria-label (screen readers).
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
      className="group relative flex flex-col items-center gap-tight bg-transparent"
      // Sized to NODE_WIDTH in layout.ts, which is what dagre spaces on. The
      // label is allowed two lines beneath a fixed-height symbol, so every
      // node occupies the same box whatever its name.
      style={{ width: '112px' }}
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

      {/* The symbol is the node. The plate behind it keeps a light glyph
          legible over a dark canvas and gives selection somewhere to land
          that is not the glyph's own silhouette. */}
      <div
        className={`
          flex-center rounded-2xl border-2 transition-all duration-200
          group-hover:shadow-lg group-hover:shadow-scrim/30
          ${selected ? 'ring-2 ring-brand-primary ring-offset-2 ring-offset-surface-base' : ''}
        `}
        style={{
          width: '64px',
          height: '64px',
          color,
          borderColor: selected ? color : 'var(--color-border-muted)',
          backgroundColor: `color-mix(in srgb, ${color} 14%, var(--color-bg-elevated))`,
        }}
      >
        <Symbol className="w-8 h-8" />
      </div>

      <div
        data-testid="topology-device-label"
        className="w-full text-center text-xs font-medium leading-tight text-text-primary line-clamp-2 break-words"
      >
        {data.label}
      </div>
    </button>
  );
});

DeviceNode.displayName = 'DeviceNode';
