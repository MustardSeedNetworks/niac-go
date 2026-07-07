/**
 * Topology Legend Component
 *
 * Documents every visual encoding the topology graph actually renders:
 * device type icons/colors, node status dots, edge speed colors, the
 * discovered-vs-declared line style, utilization color/width tiers,
 * arrowhead directionality, and the neighborhood-focus dimming
 * behavior. Kept in sync with the rendering logic in DeviceNode.tsx,
 * TrunkEdge.tsx, and layout.ts — when one of those changes a visual
 * encoding, this legend needs the matching update.
 */

import { Eye, EyeOff, Network } from 'lucide-react';
import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import {
  topologyDeviceColors as deviceColors,
  topologyDeviceIcons as deviceIcons,
} from '../../constants/device-types';
import { UTILIZATION_CRITICAL_COLOR, UTILIZATION_HIGH_COLOR } from './layout';

interface TopologyLegendProps {
  show: boolean;
  onToggle: () => void;
}

const DEVICE_TYPES = [
  { type: 'router', label: 'Router' },
  { type: 'switch', label: 'Switch' },
  { type: 'firewall', label: 'Firewall' },
  { type: 'server', label: 'Server' },
  { type: 'workstation', label: 'Workstation' },
  { type: 'access-point', label: 'Access Point' },
];

// Matches DeviceNode.tsx's statusColor map (bg-status-success /
// bg-bg-muted / bg-status-warning) — kept as CSS custom properties so
// the legend swatches track the same theme tokens as the live dots.
const STATUS_DOTS = [
  { status: 'online', swatchClass: 'bg-status-success', labelKey: 'statusOnline' },
  { status: 'offline', swatchClass: 'bg-bg-muted', labelKey: 'statusOffline' },
  { status: 'warning', swatchClass: 'bg-status-warning', labelKey: 'statusWarning' },
] as const;

// Matches layout.ts's linkSpeedColors / getEdgeColor.
const LINK_SPEEDS = [
  { label: '100M', color: 'var(--color-link-100m)' },
  { label: '1G', color: 'var(--color-link-1g)' },
  { label: '10G', color: 'var(--color-link-10g)' },
  { label: 'Trunk/LAG', color: 'var(--color-link-trunk)' },
];

// Matches layout.ts's utilizationStyle buckets (25-59 % keeps the
// base color; only the amber/red tiers are visually distinct enough
// to document).
const UTILIZATION_TIERS = [
  { color: UTILIZATION_HIGH_COLOR, labelKey: 'utilizationHigh' },
  { color: UTILIZATION_CRITICAL_COLOR, labelKey: 'utilizationCritical' },
] as const;

/**
 * TopologyLegend documents device types, node status, link speeds,
 * and the discovered/declared + utilization + direction edge
 * encodings. Can be collapsed to save space on the canvas.
 */
export const TopologyLegend: FC<TopologyLegendProps> = ({ show, onToggle }) => {
  const { t } = useTranslation('pages');
  if (!show) {
    return (
      <button
        type="button"
        onClick={onToggle}
        className="flex items-center gap-compact px-3 py-row rounded-lg bg-bg-elevated/90 border border-surface-border text-sm text-text-secondary hover:bg-bg-elevated/90 transition-colors"
      >
        <Eye className="w-4 h-4" />
        Show Legend
      </button>
    );
  }

  return (
    <div className="bg-bg-elevated/95 backdrop-blur-sm border border-surface-border rounded-xl pad min-w-[200px]">
      <div className="flex-between mb-heading">
        <span className="text-sm font-semibold text-text-primary">Legend</span>
        <button
          type="button"
          onClick={onToggle}
          className="text-text-muted hover:text-text-primary"
        >
          <EyeOff className="w-4 h-4" />
        </button>
      </div>

      <div className="stack-lg">
        {/* Nodes */}
        <div>
          <div className="text-[11px] font-semibold text-text-primary uppercase tracking-wide mb-2">
            {t('topology.legend.nodesHeading')}
          </div>
          <div className="stack-sm">
            {/* Device Types */}
            <div>
              <div className="text-xs text-text-muted uppercase tracking-wide mb-2">
                {t('topology.legend.deviceTypesHeading')}
              </div>
              <div className="space-y-1.5">
                {DEVICE_TYPES.map(({ type, label }) => {
                  const Icon = deviceIcons[type] || Network;
                  const color = deviceColors[type];
                  return (
                    <div key={type} className="flex items-center gap-compact">
                      <div className="w-4 h-4 flex-center">
                        <Icon className="w-4 h-4 text-current" />
                      </div>
                      <span className="text-xs text-text-secondary">{label}</span>
                      <div
                        className="w-2 h-2 rounded-full ml-auto"
                        style={{ backgroundColor: color }}
                      />
                    </div>
                  );
                })}
              </div>
            </div>

            {/* Status dot colors */}
            <div>
              <div className="text-xs text-text-muted uppercase tracking-wide mb-2">
                {t('topology.legend.statusHeading')}
              </div>
              <div className="space-y-1.5">
                {STATUS_DOTS.map(({ status, swatchClass, labelKey }) => (
                  <div key={status} className="flex items-center gap-compact">
                    <div className={`w-2.5 h-2.5 rounded-full ${swatchClass}`} />
                    <span className="text-xs text-text-secondary">
                      {t(`topology.legend.${labelKey}`)}
                    </span>
                  </div>
                ))}
              </div>
            </div>

            {/* Focus / dimming hint */}
            <p className="text-xs text-text-muted">{t('topology.legend.focusHint')}</p>
          </div>
        </div>

        {/* Edges */}
        <div>
          <div className="text-[11px] font-semibold text-text-primary uppercase tracking-wide mb-2">
            {t('topology.legend.edgesHeading')}
          </div>
          <div className="stack-sm">
            {/* Link Speeds */}
            <div>
              <div className="text-xs text-text-muted uppercase tracking-wide mb-2">
                {t('topology.legend.linkSpeedsHeading')}
              </div>
              <div className="space-y-1.5">
                {LINK_SPEEDS.map(({ label, color }) => (
                  <div key={label} className="flex items-center gap-compact">
                    <div className="w-6 h-0.5 rounded" style={{ backgroundColor: color }} />
                    <span className="text-xs text-text-secondary">{label}</span>
                  </div>
                ))}
              </div>
            </div>

            {/* Utilization color/width tiers */}
            <div>
              <div className="text-xs text-text-muted uppercase tracking-wide mb-2">
                {t('topology.legend.utilizationHeading')}
              </div>
              <div className="space-y-1.5">
                {UTILIZATION_TIERS.map(({ color, labelKey }) => (
                  <div key={labelKey} className="flex items-center gap-compact">
                    <div className="w-6 h-1 rounded" style={{ backgroundColor: color }} />
                    <span className="text-xs text-text-secondary">
                      {t(`topology.legend.${labelKey}`)}
                    </span>
                  </div>
                ))}
              </div>
            </div>

            {/* Line style: discovered (dashed) vs declared (solid) */}
            <div>
              <div className="text-xs text-text-muted uppercase tracking-wide mb-2">
                {t('topology.legend.lineStyleHeading')}
              </div>
              <div className="space-y-1.5">
                <div className="flex items-center gap-compact">
                  <div className="w-6 h-0.5 rounded bg-text-secondary" />
                  <span className="text-xs text-text-secondary">
                    {t('topology.legend.lineStyleDeclared')}
                  </span>
                </div>
                <div className="flex items-center gap-compact">
                  <svg
                    className="w-6 h-0.5 text-text-secondary"
                    aria-hidden="true"
                    role="presentation"
                    viewBox="0 0 24 2"
                  >
                    <line
                      x1="0"
                      y1="1"
                      x2="24"
                      y2="1"
                      stroke="currentColor"
                      strokeWidth="2"
                      strokeDasharray="4 3"
                    />
                  </svg>
                  <span className="text-xs text-text-secondary">
                    {t('topology.legend.lineStyleDiscovered')}
                  </span>
                </div>
              </div>
            </div>

            {/* Direction: single vs double arrowheads */}
            <div>
              <div className="text-xs text-text-muted uppercase tracking-wide mb-2">
                {t('topology.legend.directionHeading')}
              </div>
              <div className="space-y-1.5">
                <div className="flex items-center gap-compact">
                  <span aria-hidden="true" className="text-xs text-text-secondary w-6">
                    &rarr;
                  </span>
                  <span className="text-xs text-text-secondary">
                    {t('topology.legend.directionSingle')}
                  </span>
                </div>
                <div className="flex items-center gap-compact">
                  <span aria-hidden="true" className="text-xs text-text-secondary w-6">
                    &harr;
                  </span>
                  <span className="text-xs text-text-secondary">
                    {t('topology.legend.directionDouble')}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
