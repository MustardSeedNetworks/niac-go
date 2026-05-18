import { Building2, FileCode, Globe, Router, Server, Shield, Wifi } from 'lucide-react';
import type { FC } from 'react';
import type { Template, UserConfig } from '../../api/types';

/**
 * Shared types + helpers for the ConfigPicker family of components.
 *
 * Keeping the union shape (ConfigItem), the persisted-pref helpers, and
 * the type→icon / type→tint tables in one .ts module means the list,
 * card, row, and chip subcomponents can all import from a single
 * declarative source.
 */

export type ViewMode = 'grid' | 'list';

export const VIEW_PREF_KEY = 'niac.configs.viewMode';
export const FAVORITES_STORAGE_KEY = 'niac.configs.favorites';

export const TEMPLATE_TYPE_ICON: Record<Template['type'], FC<{ className?: string }>> = {
  basic: Globe,
  router: Router,
  switch: FileCode,
  'access-point': Wifi,
  server: Server,
  firewall: Shield,
  complete: Building2,
  custom: FileCode,
};

export const TEMPLATE_TYPE_TINT: Record<Template['type'], string> = {
  basic: 'bg-status-info/15 text-status-info border-status-info/30',
  router: 'bg-status-warning/15 text-status-warning border-status-warning/30',
  switch: 'bg-status-success/15 text-status-success border-status-success/30',
  'access-point': 'bg-brand-primary/15 text-brand-accent border-brand-accent/30',
  server: 'bg-status-info/15 text-status-info border-status-info/30',
  firewall: 'bg-status-error/15 text-status-error border-status-error/30',
  complete: 'bg-status-warning/15 text-status-warning border-status-warning/30',
  custom: 'bg-pink-500/15 text-pink-200 border-pink-400/30',
};

/**
 * ConfigItem is one row in the unified Configs list, regardless of
 * whether the underlying source is a built-in template, a saved user
 * config, or a one-shot local upload.
 */
export type ConfigItem =
  | {
      kind: 'builtin';
      key: string;
      name: string;
      description: string;
      deviceCount: number;
      template: Template;
    }
  | {
      kind: 'saved';
      key: string;
      name: string;
      description: string;
      deviceCount: number;
      config: UserConfig;
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
 * source values is active at a time; the source string picks which
 * of name/path is meaningful.
 */
export interface Selection {
  source: 'template' | 'userConfig' | 'upload' | null;
  name: string;
  path?: string;
}

export interface ConfigPickerProps {
  /** The currently selected config. */
  selection: Selection;
  /** Called when the user picks a built-in template. */
  onSelectTemplate: (template: Template) => void;
  /** Called when the user picks a saved (user) config. */
  onSelectUserConfig: (config: UserConfig) => void;
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
