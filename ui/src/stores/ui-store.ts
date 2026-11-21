import { create } from 'zustand';
import { devtools, persist } from 'zustand/middleware';
import { immer } from 'zustand/middleware/immer';

/**
 * UI Store State
 *
 * Manages global UI state like sidebar, modals, and preferences.
 */
export interface UIStoreState {
  // Sidebar
  sidebarOpen: boolean;
  sidebarCollapsed: boolean;

  // Modals
  activeModal: ModalType | null;
  modalData: Record<string, unknown>;

  // Debug Console
  debugConsoleOpen: boolean;
  debugConsoleHeight: number;

  // Preferences (theme lives in useTheme — see note on the persist config below)
  compactMode: boolean;

  // Simulation settings
  simulationSettings: SimulationSettings;

  // Actions
  setSidebarOpen: (open: boolean) => void;
  toggleSidebar: () => void;
  setSidebarCollapsed: (collapsed: boolean) => void;
  toggleSidebarCollapsed: () => void;

  openModal: (modal: ModalType, data?: Record<string, unknown>) => void;
  closeModal: () => void;

  setDebugConsoleOpen: (open: boolean) => void;
  toggleDebugConsole: () => void;
  setDebugConsoleHeight: (height: number) => void;

  setCompactMode: (compact: boolean) => void;

  // Simulation settings actions
  setSimulationSettings: (settings: Partial<SimulationSettings>) => void;
  resetSimulationSettings: () => void;

  reset: () => void;
}

export type ModalType =
  | 'device-create'
  | 'device-edit'
  | 'device-delete'
  | 'device-clone'
  | 'config-import'
  | 'config-export'
  | 'simulation-start'
  | 'simulation-stop'
  | 'replay-start'
  | 'error-injection'
  | 'settings'
  | 'about';

// Simulation settings for Settings-driven workflow
export type ConfigSource = 'userConfig' | 'upload';

export interface SimulationSettings {
  selectedInterface: string;
  configSource: ConfigSource;
  configName: string;
}

const DEFAULT_DEBUG_CONSOLE_HEIGHT = 300;

const DEFAULT_SIMULATION_SETTINGS: SimulationSettings = {
  selectedInterface: '',
  configSource: 'userConfig',
  configName: '',
};

export const useUIStore = create<UIStoreState>()(
  devtools(
    persist(
      immer((set) => ({
        // Initial state
        sidebarOpen: true,
        sidebarCollapsed: false,
        activeModal: null,
        modalData: {},
        debugConsoleOpen: false,
        debugConsoleHeight: DEFAULT_DEBUG_CONSOLE_HEIGHT,
        compactMode: false,
        simulationSettings: DEFAULT_SIMULATION_SETTINGS,

        // Sidebar actions
        setSidebarOpen: (open) =>
          set((state) => {
            state.sidebarOpen = open;
          }),

        toggleSidebar: () =>
          set((state) => {
            state.sidebarOpen = !state.sidebarOpen;
          }),

        setSidebarCollapsed: (collapsed) =>
          set((state) => {
            state.sidebarCollapsed = collapsed;
          }),

        toggleSidebarCollapsed: () =>
          set((state) => {
            state.sidebarCollapsed = !state.sidebarCollapsed;
          }),

        // Modal actions
        openModal: (modal, data = {}) =>
          set((state) => {
            state.activeModal = modal;
            state.modalData = data;
          }),

        closeModal: () =>
          set((state) => {
            state.activeModal = null;
            state.modalData = {};
          }),

        // Debug console actions
        setDebugConsoleOpen: (open) =>
          set((state) => {
            state.debugConsoleOpen = open;
          }),

        toggleDebugConsole: () =>
          set((state) => {
            state.debugConsoleOpen = !state.debugConsoleOpen;
          }),

        setDebugConsoleHeight: (height) =>
          set((state) => {
            state.debugConsoleHeight = Math.max(150, Math.min(600, height));
          }),

        // Preference actions
        setCompactMode: (compact) =>
          set((state) => {
            state.compactMode = compact;
          }),

        // Simulation settings actions
        setSimulationSettings: (settings) =>
          set((state) => {
            state.simulationSettings = { ...state.simulationSettings, ...settings };
          }),

        resetSimulationSettings: () =>
          set((state) => {
            state.simulationSettings = DEFAULT_SIMULATION_SETTINGS;
          }),

        // Reset
        reset: () =>
          set((state) => {
            state.activeModal = null;
            state.modalData = {};
            state.debugConsoleOpen = false;
          }),
      })),
      {
        name: 'niac-ui-store',
        // Theme is intentionally NOT persisted here — useTheme owns it under
        // the 'niac-theme' key. Two owners silently diverged (was a bug).
        partialize: (state) => ({
          sidebarCollapsed: state.sidebarCollapsed,
          debugConsoleHeight: state.debugConsoleHeight,
          compactMode: state.compactMode,
          simulationSettings: state.simulationSettings,
        }),
      },
    ),
    { name: 'UIStore' },
  ),
);
