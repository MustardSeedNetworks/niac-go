/**
 * Topology Legend Component
 *
 * Displays a legend showing device types and link speeds
 * used in the network topology visualization.
 */

import { Eye, EyeOff, Network } from 'lucide-react';
import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import {
  topologyDeviceColors as deviceColors,
  topologyDeviceIcons as deviceIcons,
} from '../../constants/device-types';

interface TopologyLegendProps {
  show: boolean;
  onToggle: () => void;
}

/**
 * TopologyLegend shows device type icons/colors and link speed colors.
 * Can be collapsed to save space on the canvas.
 */
export const TopologyLegend: FC<TopologyLegendProps> = ({ show, onToggle }) => {
  const { t } = useTranslation('pages');
  if (!show) {
    return (
      <button
        type="button"
        onClick={onToggle}
        className="flex items-center gap-2 px-3 py-2 rounded-lg bg-bg-elevated/90 border border-surface-border text-sm text-text-secondary hover:bg-bg-elevated/90 transition-colors"
      >
        <Eye className="w-4 h-4" />
        Show Legend
      </button>
    );
  }

  return (
    <div className="bg-bg-elevated/95 backdrop-blur-sm border border-surface-border rounded-xl p-4 min-w-[200px]">
      <div className="flex items-center justify-between mb-3">
        <span className="text-sm font-semibold text-text-primary">Legend</span>
        <button
          type="button"
          onClick={onToggle}
          className="text-text-muted hover:text-text-primary"
        >
          <EyeOff className="w-4 h-4" />
        </button>
      </div>

      <div className="space-y-4">
        {/* Device Types */}
        <div>
          <div className="text-xs text-text-muted uppercase tracking-wide mb-2">
            {t('topology.legend.deviceTypesHeading')}
          </div>
          <div className="space-y-1.5">
            {[
              { type: 'router', label: 'Router' },
              { type: 'switch', label: 'Switch' },
              { type: 'firewall', label: 'Firewall' },
              { type: 'server', label: 'Server' },
              { type: 'workstation', label: 'Workstation' },
              { type: 'access-point', label: 'Access Point' },
            ].map(({ type, label }) => {
              const Icon = deviceIcons[type] || Network;
              const color = deviceColors[type];
              return (
                <div key={type} className="flex items-center gap-2">
                  <div className="w-4 h-4 flex items-center justify-center">
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

        {/* Link Speeds */}
        <div>
          <div className="text-xs text-text-muted uppercase tracking-wide mb-2">
            {t('topology.legend.linkSpeedsHeading')}
          </div>
          <div className="space-y-1.5">
            {[
              { label: '100M', color: 'var(--color-link-100m)' },
              { label: '1G', color: 'var(--color-link-1g)' },
              { label: '10G', color: 'var(--color-link-10g)' },
              { label: 'Trunk/LAG', color: 'var(--color-link-trunk)' },
            ].map(({ label, color }) => (
              <div key={label} className="flex items-center gap-2">
                <div className="w-6 h-0.5 rounded" style={{ backgroundColor: color }} />
                <span className="text-xs text-text-secondary">{label}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
};
