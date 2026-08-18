/**
 * Shared types for HelpDrawer components.
 *
 * Public types live in `/src/data/help-content.ts`. We re-export the ones
 * the UI consumes to keep imports stable for the rest of the app.
 */

import type { ReactNode } from 'react';

export type {
  Category,
  ConfigField,
  Example,
  FAQEntry,
  Feature,
  GlossaryEntry,
  HelpItem,
  Metric,
  Parameter,
  Shortcut,
  TestHelp,
} from '../../data/help-content';

export type HelpTab =
  | 'overview'
  | 'pages'
  | 'devices'
  | 'protocols'
  | 'commands'
  | 'glossary'
  | 'shortcuts'
  | 'faq';

export interface TabConfig {
  id: HelpTab;
  label: string;
  icon: ReactNode;
}
