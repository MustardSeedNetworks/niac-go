import { create } from 'zustand';
import { devtools, persist } from 'zustand/middleware';
import { immer } from 'zustand/middleware/immer';
import type { Device, DeviceType } from '../types';

/**
 * Device Store State
 *
 * Manages device list and selection state.
 */
export interface DeviceStoreState {
  // State
  devices: Device[];
  selectedDeviceId: string | null;
  deviceFilter: DeviceFilter;
  isLoading: boolean;
  error: string | null;

  // Computed (selectors)
  selectedDevice: () => Device | undefined;
  filteredDevices: () => Device[];
  deviceCount: () => number;

  // Actions
  setDevices: (devices: Device[]) => void;
  addDevice: (device: Device) => void;
  updateDevice: (hostname: string, updates: Partial<Device>) => void;
  removeDevice: (hostname: string) => void;
  selectDevice: (hostname: string | null) => void;
  setFilter: (filter: Partial<DeviceFilter>) => void;
  clearFilter: () => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
  reset: () => void;
}

export interface DeviceFilter {
  search: string;
  type: DeviceType | 'all';
  hasSnmp: boolean | null;
  hasLldp: boolean | null;
}

const defaultFilter: DeviceFilter = {
  search: '',
  type: 'all',
  hasSnmp: null,
  hasLldp: null,
};

// Helper functions for filtering to reduce complexity
const matchesSearch = (device: Device, search: string): boolean => {
  if (!search) return true;
  const searchLower = search.toLowerCase();
  return (
    device.hostname.toLowerCase().includes(searchLower) ||
    Boolean(device.ip?.toLowerCase().includes(searchLower)) ||
    device.mac.toLowerCase().includes(searchLower)
  );
};

const matchesFilter = (device: Device, filter: DeviceFilter): boolean => {
  if (!matchesSearch(device, filter.search)) return false;
  if (filter.type !== 'all' && device.type !== filter.type) return false;
  if (filter.hasSnmp !== null && Boolean(device.snmpAgent?.community) !== filter.hasSnmp)
    return false;
  if (filter.hasLldp !== null && Boolean(device.lldp?.enabled) !== filter.hasLldp) return false;
  return true;
};

export const useDeviceStore = create<DeviceStoreState>()(
  devtools(
    persist(
      immer((set, get) => ({
        // Initial state
        devices: [],
        selectedDeviceId: null,
        deviceFilter: { ...defaultFilter },
        isLoading: false,
        error: null,

        // Computed selectors
        selectedDevice: () => {
          const { devices, selectedDeviceId } = get();
          return devices.find((d) => d.hostname === selectedDeviceId);
        },

        filteredDevices: () => {
          const { devices, deviceFilter } = get();
          return devices.filter((device) => matchesFilter(device, deviceFilter));
        },

        deviceCount: () => get().devices.length,

        // Actions
        setDevices: (devices) =>
          set((state) => {
            state.devices = devices;
            state.error = null;
          }),

        addDevice: (device) =>
          set((state) => {
            state.devices.push(device);
          }),

        updateDevice: (hostname, updates) =>
          set((state) => {
            const index = state.devices.findIndex((d) => d.hostname === hostname);
            if (index !== -1) {
              state.devices[index] = { ...state.devices[index], ...updates };
            }
          }),

        removeDevice: (hostname) =>
          set((state) => {
            state.devices = state.devices.filter((d) => d.hostname !== hostname);
            if (state.selectedDeviceId === hostname) {
              state.selectedDeviceId = null;
            }
          }),

        selectDevice: (hostname) =>
          set((state) => {
            state.selectedDeviceId = hostname;
          }),

        setFilter: (filter) =>
          set((state) => {
            state.deviceFilter = { ...state.deviceFilter, ...filter };
          }),

        clearFilter: () =>
          set((state) => {
            state.deviceFilter = { ...defaultFilter };
          }),

        setLoading: (loading) =>
          set((state) => {
            state.isLoading = loading;
          }),

        setError: (error) =>
          set((state) => {
            state.error = error;
          }),

        reset: () =>
          set((state) => {
            state.devices = [];
            state.selectedDeviceId = null;
            state.deviceFilter = { ...defaultFilter };
            state.isLoading = false;
            state.error = null;
          }),
      })),
      {
        name: 'niac-device-store',
        partialize: (state) => ({
          selectedDeviceId: state.selectedDeviceId,
          deviceFilter: state.deviceFilter,
        }),
      },
    ),
    { name: 'DeviceStore' },
  ),
);
