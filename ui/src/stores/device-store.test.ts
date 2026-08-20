import { act } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { Device } from '../types';

// Mock localStorage for zustand persist middleware
const localStorageMock = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: vi.fn((key: string) => store[key] ?? null),
    setItem: vi.fn((key: string, value: string) => {
      store[key] = value;
    }),
    removeItem: vi.fn((key: string) => {
      delete store[key];
    }),
    clear: vi.fn(() => {
      store = {};
    }),
    get length() {
      return Object.keys(store).length;
    },
    key: vi.fn((index: number) => Object.keys(store)[index] ?? null),
  };
})();

// Set up localStorage mock before importing the store
Object.defineProperty(globalThis, 'localStorage', {
  value: localStorageMock,
  writable: true,
});

// Import store after localStorage mock is set up
const { useDeviceStore } = await import('./device-store');

const mockDevice = (overrides: Partial<Device> = {}): Device => ({
  hostname: 'router-1',
  mac: '00:11:22:33:44:55',
  type: 'router',
  ip: '192.168.1.1',
  ...overrides,
});

describe('useDeviceStore', () => {
  afterEach(() => {
    act(() => {
      useDeviceStore.getState().reset();
    });
  });

  describe('setDevices', () => {
    it('sets the device list and clears errors', () => {
      const devices = [mockDevice(), mockDevice({ hostname: 'switch-1', type: 'switch' })];

      act(() => {
        useDeviceStore.getState().setError('old error');
        useDeviceStore.getState().setDevices(devices);
      });

      const state = useDeviceStore.getState();
      expect(state.devices).toHaveLength(2);
      expect(state.error).toBeNull();
    });
  });

  describe('addDevice', () => {
    it('appends a device to the list', () => {
      act(() => {
        useDeviceStore.getState().setDevices([mockDevice()]);
        useDeviceStore.getState().addDevice(mockDevice({ hostname: 'switch-1' }));
      });

      expect(useDeviceStore.getState().devices).toHaveLength(2);
    });
  });

  describe('updateDevice', () => {
    it('updates an existing device by hostname', () => {
      act(() => {
        useDeviceStore.getState().setDevices([mockDevice()]);
        useDeviceStore.getState().updateDevice('router-1', { ip: '10.0.0.1' });
      });

      expect(useDeviceStore.getState().devices[0]?.ip).toBe('10.0.0.1');
    });

    it('does nothing for non-existent hostname', () => {
      act(() => {
        useDeviceStore.getState().setDevices([mockDevice()]);
        useDeviceStore.getState().updateDevice('nonexistent', { ip: '10.0.0.1' });
      });

      expect(useDeviceStore.getState().devices[0]?.ip).toBe('192.168.1.1');
    });
  });

  describe('removeDevice', () => {
    it('removes a device by hostname', () => {
      act(() => {
        useDeviceStore.getState().setDevices([mockDevice(), mockDevice({ hostname: 'switch-1' })]);
        useDeviceStore.getState().removeDevice('router-1');
      });

      const state = useDeviceStore.getState();
      expect(state.devices).toHaveLength(1);
      expect(state.devices[0]?.hostname).toBe('switch-1');
    });

    it('clears selection if removed device was selected', () => {
      act(() => {
        useDeviceStore.getState().setDevices([mockDevice()]);
        useDeviceStore.getState().selectDevice('router-1');
        useDeviceStore.getState().removeDevice('router-1');
      });

      expect(useDeviceStore.getState().selectedDeviceId).toBeNull();
    });
  });

  describe('selectDevice', () => {
    it('sets the selected device ID', () => {
      act(() => {
        useDeviceStore.getState().selectDevice('router-1');
      });

      expect(useDeviceStore.getState().selectedDeviceId).toBe('router-1');
    });

    it('clears selection with null', () => {
      act(() => {
        useDeviceStore.getState().selectDevice('router-1');
        useDeviceStore.getState().selectDevice(null);
      });

      expect(useDeviceStore.getState().selectedDeviceId).toBeNull();
    });
  });

  describe('selectedDevice (computed)', () => {
    it('returns the selected device', () => {
      act(() => {
        useDeviceStore.getState().setDevices([mockDevice()]);
        useDeviceStore.getState().selectDevice('router-1');
      });

      const selected = useDeviceStore.getState().selectedDevice();
      expect(selected?.hostname).toBe('router-1');
    });

    it('returns undefined when nothing is selected', () => {
      act(() => {
        useDeviceStore.getState().setDevices([mockDevice()]);
      });

      expect(useDeviceStore.getState().selectedDevice()).toBeUndefined();
    });
  });

  describe('filteredDevices (computed)', () => {
    it('filters by search term on hostname', () => {
      act(() => {
        useDeviceStore
          .getState()
          .setDevices([mockDevice(), mockDevice({ hostname: 'switch-1', type: 'switch' })]);
        useDeviceStore.getState().setFilter({ search: 'router' });
      });

      const filtered = useDeviceStore.getState().filteredDevices();
      expect(filtered).toHaveLength(1);
      expect(filtered[0]?.hostname).toBe('router-1');
    });

    it('filters by device type', () => {
      act(() => {
        useDeviceStore
          .getState()
          .setDevices([mockDevice(), mockDevice({ hostname: 'switch-1', type: 'switch' })]);
        useDeviceStore.getState().setFilter({ type: 'switch' });
      });

      const filtered = useDeviceStore.getState().filteredDevices();
      expect(filtered).toHaveLength(1);
      expect(filtered[0]?.type).toBe('switch');
    });

    it('returns all devices when filter is "all"', () => {
      act(() => {
        useDeviceStore
          .getState()
          .setDevices([mockDevice(), mockDevice({ hostname: 'switch-1', type: 'switch' })]);
      });

      expect(useDeviceStore.getState().filteredDevices()).toHaveLength(2);
    });
  });

  describe('deviceCount (computed)', () => {
    it('returns the number of devices', () => {
      act(() => {
        useDeviceStore.getState().setDevices([mockDevice(), mockDevice({ hostname: 'switch-1' })]);
      });

      expect(useDeviceStore.getState().deviceCount()).toBe(2);
    });
  });

  describe('reset', () => {
    it('resets all state to defaults', () => {
      act(() => {
        useDeviceStore.getState().setDevices([mockDevice()]);
        useDeviceStore.getState().selectDevice('router-1');
        useDeviceStore.getState().setFilter({ search: 'test' });
        useDeviceStore.getState().setLoading(true);
        useDeviceStore.getState().setError('error');
        useDeviceStore.getState().reset();
      });

      const state = useDeviceStore.getState();
      expect(state.devices).toHaveLength(0);
      expect(state.selectedDeviceId).toBeNull();
      expect(state.deviceFilter.search).toBe('');
      expect(state.isLoading).toBe(false);
      expect(state.error).toBeNull();
    });
  });
});
