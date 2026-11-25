import { ChevronDown, Download, FileCode, Plus, RefreshCw, Server } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router';
import type { Device } from '../../api/types';
import { iconSizes } from '../../constants/sizes';
import { Button } from '../../shared/Button';
import { ActionButton } from '../../ui/ActionButton';
import { exportDevicesAsCSV, exportDevicesAsJSON, exportDevicesAsYAML } from '../../utils/export';

interface DeviceListHeaderProps {
  deviceCount: number;
  filteredDevices: Device[];
  loading: boolean;
  onRefresh: () => void;
}

export function DeviceListHeader({
  deviceCount,
  filteredDevices,
  loading,
  onRefresh,
}: DeviceListHeaderProps) {
  const { t } = useTranslation('devices');
  const { t: tCommon } = useTranslation('common');
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
    <div className="flex flex-wrap items-center justify-between gap-3">
      <p className="text-sm text-text-muted">
        {deviceCount > 0 ? (
          <output className="figure">
            {tCommon('plurals.deviceCount', { count: deviceCount })}
          </output>
        ) : null}
      </p>
      <div className="flex flex-wrap items-center gap-2">
        <Button
          variant="secondary"
          icon={<Server className={iconSizes.md} />}
          onClick={() => navigate('/devices')}
          data-testid="view-running-state"
        >
          {t('list.viewRunningState')}
        </Button>
        <Button
          variant="secondary"
          icon={<RefreshCw className={iconSizes.md} />}
          onClick={onRefresh}
          disabled={loading}
        >
          {tCommon('buttons.refresh')}
        </Button>
        <Button
          variant="secondary"
          icon={<FileCode className={iconSizes.md} />}
          onClick={() => exportDevicesAsYAML(filteredDevices)}
          disabled={noDevices}
          title={t('list.downloadYamlTitle')}
        >
          {t('list.downloadYamlButton')}
        </Button>
        {/* Data-only exports — kept under a disclosure since JSON/CSV
            can't be re-imported into NIAC; useful for reporting / pipe
            into other tooling only. */}
        <div ref={menuRef} className="relative">
          <Button
            variant="ghost"
            icon={<Download className={iconSizes.md} />}
            onClick={() => setExportMenuOpen((v) => !v)}
            disabled={noDevices}
            title={t('list.dataExportTitle')}
          >
            {t('list.dataExportButton')}
            <ChevronDown className={iconSizes.sm} aria-hidden="true" />
          </Button>
          {exportMenuOpen && (
            <div className="absolute right-0 z-10 mt-1 w-44 rounded-lg border border-surface-border bg-surface-raised shadow-lg">
              <button
                type="button"
                onClick={() => {
                  exportDevicesAsJSON(filteredDevices);
                  setExportMenuOpen(false);
                }}
                className="block w-full px-3 py-row text-left text-sm text-text-primary hover:bg-surface-hover"
              >
                {t('list.exportJson')}
              </button>
              <button
                type="button"
                onClick={() => {
                  exportDevicesAsCSV(filteredDevices);
                  setExportMenuOpen(false);
                }}
                className="block w-full px-3 py-row text-left text-sm text-text-primary hover:bg-surface-hover"
              >
                {t('list.exportCsv')}
              </button>
            </div>
          )}
        </div>
        <ActionButton
          action="edit"
          variant="primary"
          icon={<Plus className={iconSizes.md} />}
          onClick={() => navigate('/device-config/new')}
          data-testid="device-add"
        >
          {t('list.states.addDevice')}
        </ActionButton>
      </div>
    </div>
  );
}
