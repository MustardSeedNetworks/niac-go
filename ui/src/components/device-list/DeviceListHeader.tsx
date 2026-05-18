import { ChevronDown, Download, FileCode, Plus, RefreshCw, Server } from 'lucide-react';
import { type FC, useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import type { Device } from '../../api/types';
import { iconSizes } from '../../constants/sizes';
import { Button } from '../../ui/Button';
import { H2, P } from '../../ui/Typography';
import { exportDevicesAsCSV, exportDevicesAsJSON, exportDevicesAsYAML } from '../../utils/export';

interface DeviceListHeaderProps {
  deviceCount: number;
  filteredDevices: Device[];
  loading: boolean;
  onRefresh: () => void;
}

export const DeviceListHeader: FC<DeviceListHeaderProps> = ({
  deviceCount,
  filteredDevices,
  loading,
  onRefresh,
}) => {
  const navigate = useNavigate();
  const [exportMenuOpen, setExportMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  const noDevices = filteredDevices.length === 0;

  // Close the data-export menu when clicking outside.
  useEffect(() => {
    if (!exportMenuOpen) return;
    const onClick = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setExportMenuOpen(false);
      }
    };
    document.addEventListener('mousedown', onClick);
    return () => document.removeEventListener('mousedown', onClick);
  }, [exportMenuOpen]);

  return (
    <div className="flex flex-wrap items-center justify-between gap-4">
      <div>
        <H2 className="flex items-center gap-2">
          <Server className={`${iconSizes.lg} text-brand-accent`} />
          Device Configuration
        </H2>
        <P className="text-text-muted mt-1">
          Manage network device configurations for simulation.
          {deviceCount > 0 && (
            <output className="ml-2 text-brand-accent">{deviceCount} devices</output>
          )}
        </P>
      </div>
      <div className="flex gap-2">
        <Button
          variant="outline"
          leftIcon={<RefreshCw className={iconSizes.md} />}
          onClick={onRefresh}
          disabled={loading}
        >
          Refresh
        </Button>
        <Button
          variant="outline"
          leftIcon={<FileCode className={iconSizes.md} />}
          onClick={() => exportDevicesAsYAML(filteredDevices)}
          disabled={noDevices}
          title="Download a NIAC-loadable YAML config of the filtered devices"
        >
          Download YAML
        </Button>
        {/* Data-only exports — kept under a disclosure since JSON/CSV
            can't be re-imported into NIAC; useful for reporting / pipe
            into other tooling only. */}
        <div ref={menuRef} className="relative">
          <Button
            variant="ghost"
            leftIcon={<Download className={iconSizes.md} />}
            rightIcon={<ChevronDown className={iconSizes.sm} />}
            onClick={() => setExportMenuOpen((v) => !v)}
            disabled={noDevices}
            title="Export device data for reporting (not re-importable)"
          >
            Data
          </Button>
          {exportMenuOpen && (
            <div className="absolute right-0 z-10 mt-1 w-44 rounded-lg border border-surface-border bg-bg-surface shadow-lg">
              <button
                type="button"
                onClick={() => {
                  exportDevicesAsJSON(filteredDevices);
                  setExportMenuOpen(false);
                }}
                className="block w-full px-3 py-2 text-left text-sm text-text-primary hover:bg-surface-hover"
              >
                JSON (data only)
              </button>
              <button
                type="button"
                onClick={() => {
                  exportDevicesAsCSV(filteredDevices);
                  setExportMenuOpen(false);
                }}
                className="block w-full px-3 py-2 text-left text-sm text-text-primary hover:bg-surface-hover"
              >
                CSV (data only)
              </button>
            </div>
          )}
        </div>
        <Button
          tone="violet"
          leftIcon={<Plus className={iconSizes.md} />}
          onClick={() => navigate('/device-config/new')}
        >
          Add Device
        </Button>
      </div>
    </div>
  );
};
