import type { LibraryNetwork } from '../../api/types';

/**
 * Shared types + helpers for the ConfigPicker family of components.
 *
 * Keeping the union shape (ConfigItem) and the persisted-pref helpers in
 * one .ts module means the list, card and row subcomponents can all
 * import from a single declarative source.
 */

export type ViewMode = 'grid' | 'list';

export const VIEW_PREF_KEY = 'niac.configs.viewMode';
export const FAVORITES_STORAGE_KEY = 'niac.configs.favorites';

/**
 * ConfigItem is one row in the unified Configs list: a library network
 * or a one-shot local upload.
 */
export type ConfigItem =
  | {
      kind: 'saved';
      key: string;
      name: string;
      description: string;
      deviceCount: number;
      config: LibraryNetwork;
    }
  | {
      kind: 'local';
      key: string;
      name: string;
      description: string;
      deviceCount: number;
      file: File;
    };

/**
 * Selection is what the parent Simulation page tracks. Only one of
 * source values is active at a time. Library networks have no
 * filesystem path exposed to the frontend (the daemon confines them
 * behind an os.Root by name — see internal/library/list.go), so the
 * daemon always loads a picked network's content via
 * fetchLibraryNetworkContent(name) and sends it inline.
 */
export interface Selection {
  source: 'userConfig' | 'upload' | null;
  name: string;
}

export interface ConfigPickerProps {
  /** The currently selected config. */
  selection: Selection;
  /** Called when the user picks a saved (user) network from the library. */
  onSelectUserConfig: (config: LibraryNetwork) => void;
  /** Called when the user uploads a file (or clears it). */
  onUpload: (file: File | null) => void;
  /** The current upload file, if any. */
  uploadFile: File | null;
}

/**
 * readPref / writePref persist UI state (view mode, source filter)
 * in localStorage. SSR-safe — they no-op when window is undefined.
 */
export function readPref<T extends string>(key: string, fallback: T): T {
  if (typeof window === 'undefined') return fallback;
  const saved = window.localStorage.getItem(key);
  return (saved as T) || fallback;
}

export function writePref(key: string, value: string) {
  if (typeof window !== 'undefined') {
    window.localStorage.setItem(key, value);
  }
}
