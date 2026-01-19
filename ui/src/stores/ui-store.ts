// Copyright (c) 2025 Mustard Seed Networks. All rights reserved.

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

  // Notifications
  notifications: Notification[];

  // Debug Console
  debugConsoleOpen: boolean;
  debugConsoleHeight: number;

  // Preferences
  theme: 'dark' | 'light' | 'system';
  compactMode: boolean;

  // Actions
  setSidebarOpen: (open: boolean) => void;
  toggleSidebar: () => void;
  setSidebarCollapsed: (collapsed: boolean) => void;
  toggleSidebarCollapsed: () => void;

  openModal: (modal: ModalType, data?: Record<string, unknown>) => void;
  closeModal: () => void;

  addNotification: (notification: Omit<Notification, 'id' | 'timestamp'>) => void;
  removeNotification: (id: string) => void;
  clearNotifications: () => void;

  setDebugConsoleOpen: (open: boolean) => void;
  toggleDebugConsole: () => void;
  setDebugConsoleHeight: (height: number) => void;

  setTheme: (theme: 'dark' | 'light' | 'system') => void;
  setCompactMode: (compact: boolean) => void;

  reset: () => void;
}

export type ModalType =
  | 'device-create'
  | 'device-edit'
  | 'device-delete'
  | 'device-clone'
  | 'config-import'
  | 'config-export'
  | 'template-select'
  | 'template-upload'
  | 'simulation-start'
  | 'simulation-stop'
  | 'replay-start'
  | 'error-injection'
  | 'settings'
  | 'about';

export interface Notification {
  id: string;
  type: 'success' | 'error' | 'warning' | 'info';
  title: string;
  message?: string;
  timestamp: number;
  duration?: number;
}

const DEFAULT_DEBUG_CONSOLE_HEIGHT = 300;

export const useUIStore = create<UIStoreState>()(
  devtools(
    persist(
      immer((set) => ({
        // Initial state
        sidebarOpen: true,
        sidebarCollapsed: false,
        activeModal: null,
        modalData: {},
        notifications: [],
        debugConsoleOpen: false,
        debugConsoleHeight: DEFAULT_DEBUG_CONSOLE_HEIGHT,
        theme: 'dark',
        compactMode: false,

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

        // Notification actions
        addNotification: (notification) =>
          set((state) => {
            const id = `notif-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`;
            state.notifications.push({
              ...notification,
              id,
              timestamp: Date.now(),
            });
          }),

        removeNotification: (id) =>
          set((state) => {
            state.notifications = state.notifications.filter((n) => n.id !== id);
          }),

        clearNotifications: () =>
          set((state) => {
            state.notifications = [];
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
        setTheme: (theme) =>
          set((state) => {
            state.theme = theme;
          }),

        setCompactMode: (compact) =>
          set((state) => {
            state.compactMode = compact;
          }),

        // Reset
        reset: () =>
          set((state) => {
            state.activeModal = null;
            state.modalData = {};
            state.notifications = [];
            state.debugConsoleOpen = false;
          }),
      })),
      {
        name: 'niac-ui-store',
        partialize: (state) => ({
          sidebarCollapsed: state.sidebarCollapsed,
          debugConsoleHeight: state.debugConsoleHeight,
          theme: state.theme,
          compactMode: state.compactMode,
        }),
      },
    ),
    { name: 'UIStore' },
  ),
);
