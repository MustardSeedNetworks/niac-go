// Copyright (c) 2025 Mustard Seed Networks. All rights reserved.

/**
 * Shared types for HelpDrawer components.
 */

import type { ReactNode } from 'react';

export type HelpTab = 'features' | 'glossary' | 'shortcuts';

export interface TabConfig {
  id: HelpTab;
  label: string;
  icon: ReactNode;
}

export interface Feature {
  title: string;
  description: string;
  path: string;
  badge?: string;
}

export interface GlossaryEntry {
  term: string;
  definition: string;
  category: 'protocol' | 'concept' | 'device';
}

export interface Shortcut {
  keys: string[];
  description: string;
  category: 'navigation' | 'actions' | 'general';
}
