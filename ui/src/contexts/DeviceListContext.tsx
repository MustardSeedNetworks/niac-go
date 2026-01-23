import { createContext, type FC, type ReactNode, useContext } from 'react';
import type { Device } from '../api/types';

interface DeviceListContextValue {
  devices: Device[];
  selectedDevices: Set<string>;
  onSelectDevice: (hostname: string) => void;
  onSelectAll: () => void;
  onEdit: (hostname: string) => void;
  onClone: (hostname: string) => void;
  onDelete: (hostname: string) => void;
  getDeviceProtocols: (device: Device) => string[];
}

const DeviceListContext = createContext<DeviceListContextValue | null>(null);

export function useDeviceList(): DeviceListContextValue {
  const context = useContext(DeviceListContext);
  if (!context) {
    throw new Error('useDeviceList must be used within a DeviceListProvider');
  }
  return context;
}

interface DeviceListProviderProps {
  devices: Device[];
  selectedDevices: Set<string>;
  onSelectDevice: (hostname: string) => void;
  onSelectAll: () => void;
  onEdit: (hostname: string) => void;
  onClone: (hostname: string) => void;
  onDelete: (hostname: string) => void;
  getDeviceProtocols: (device: Device) => string[];
  children: ReactNode;
}

export const DeviceListProvider: FC<DeviceListProviderProps> = ({
  devices,
  selectedDevices,
  onSelectDevice,
  onSelectAll,
  onEdit,
  onClone,
  onDelete,
  getDeviceProtocols,
  children,
}) => {
  const value: DeviceListContextValue = {
    devices,
    selectedDevices,
    onSelectDevice,
    onSelectAll,
    onEdit,
    onClone,
    onDelete,
    getDeviceProtocols,
  };

  return <DeviceListContext.Provider value={value}>{children}</DeviceListContext.Provider>;
};
