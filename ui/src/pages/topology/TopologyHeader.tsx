/**
 * Topology Header Component
 *
 * Header bar for the network topology visualization page with
 * refresh controls, zoom controls, and layout options.
 */

import { LayoutGrid, Maximize2, RefreshCw, ZoomIn, ZoomOut } from 'lucide-react';
import type { FC } from 'react';
import { Button } from '../../ui/Button';

interface TopologyHeaderProps {
  title?: string;
  deviceCount: number;
  linkCount: number;
  isRefreshing: boolean;
  onRefresh: () => void;
  onZoomIn?: () => void;
  onZoomOut?: () => void;
  onFitView?: () => void;
  onAutoLayout?: () => void;
}

/**
 * TopologyHeader displays the page title, device/link counts, and
 * provides controls for refreshing data and manipulating the view.
 */
export const TopologyHeader: FC<TopologyHeaderProps> = ({
  title = 'Network Topology',
  deviceCount,
  linkCount,
  isRefreshing,
  onRefresh,
  onZoomIn,
  onZoomOut,
  onFitView,
  onAutoLayout,
}) => {
  return (
    <div className="flex items-center justify-between p-4 border-b border-white/10 bg-bg-surface/80 backdrop-blur-sm">
      <div className="flex items-center gap-4">
        <h1 className="text-xl font-semibold text-text-primary">{title}</h1>
        <div className="flex items-center gap-3 text-sm text-text-muted">
          <span>{deviceCount} devices</span>
          <span className="text-text-disabled">|</span>
          <span>{linkCount} links</span>
        </div>
      </div>

      <div className="flex items-center gap-2">
        {/* Zoom controls */}
        {onZoomOut && (
          <Button
            variant="ghost"
            size="sm"
            onClick={onZoomOut}
            title="Zoom out"
            aria-label="Zoom out"
          >
            <ZoomOut className="w-4 h-4" />
          </Button>
        )}
        {onZoomIn && (
          <Button variant="ghost" size="sm" onClick={onZoomIn} title="Zoom in" aria-label="Zoom in">
            <ZoomIn className="w-4 h-4" />
          </Button>
        )}
        {onFitView && (
          <Button
            variant="ghost"
            size="sm"
            onClick={onFitView}
            title="Fit to view"
            aria-label="Fit to view"
          >
            <Maximize2 className="w-4 h-4" />
          </Button>
        )}

        <div className="w-px h-6 bg-white/10" />

        {/* Layout control */}
        {onAutoLayout && (
          <Button
            variant="ghost"
            size="sm"
            onClick={onAutoLayout}
            title="Auto-layout"
            aria-label="Auto-layout"
          >
            <LayoutGrid className="w-4 h-4" />
          </Button>
        )}

        {/* Refresh */}
        <Button
          variant="outline"
          size="sm"
          onClick={onRefresh}
          disabled={isRefreshing}
          className="gap-2"
        >
          <RefreshCw className={`w-4 h-4 ${isRefreshing ? 'animate-spin' : ''}`} />
          Refresh
        </Button>
      </div>
    </div>
  );
};
