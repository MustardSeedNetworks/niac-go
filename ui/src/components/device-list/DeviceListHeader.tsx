import { Download, Plus, RefreshCw, Server } from 'lucide-react';
import type { FC } from 'react';
import { useNavigate } from 'react-router-dom';
import type { Device } from '../../api/types';
import { iconSizes } from '../../constants/sizes';
import { Button } from '../../ui/Button';
import { H2, P } from '../../ui/Typography';
import { exportDevicesAsCSV, exportDevicesAsJSON } from '../../utils/export';

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

  return (
    <div className="flex flex-wrap items-center justify-between gap-4">
      <div>
        <H2 className="flex items-center gap-2">
          <Server className={`${iconSizes.lg} text-violet-300`} />
          Device Configuration
        </H2>
        <P className="text-gray-400 mt-1">
          Manage network device configurations for simulation.
          {deviceCount > 0 && (
            <output className="ml-2 text-violet-300">{deviceCount} devices</output>
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
          leftIcon={<Download className={iconSizes.md} />}
          onClick={() => exportDevicesAsJSON(filteredDevices)}
          disabled={filteredDevices.length === 0}
        >
          JSON
        </Button>
        <Button
          variant="outline"
          leftIcon={<Download className={iconSizes.md} />}
          onClick={() => exportDevicesAsCSV(filteredDevices)}
          disabled={filteredDevices.length === 0}
        >
          CSV
        </Button>
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
