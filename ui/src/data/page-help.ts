import type { TFunction } from 'i18next';

/** A labelled term and its explanation — the shape of a "what does X do" list. */
export interface PageHelpTerm {
  term: string;
  description: string;
}

/** One renderable unit of a page's help. PageHelpBody switches on `kind`. */
export type PageHelpBlock =
  | { kind: 'paragraph'; text: string }
  | { kind: 'heading'; text: string }
  | { kind: 'terms'; items: PageHelpTerm[] }
  | { kind: 'tips'; items: string[] };

export const pageHelpRoutes = [
  '/',
  '/runtime',
  '/new-simulation',
  '/devices',
  '/segments',
  '/device-config',
  '/topology',
  '/alerts',
  '/traffic',
  '/debug',
  '/packets',
  '/config-diff',
  '/walk-validator',
  '/walk-analyzer',
  '/library/walks',
  '/library/pcaps',
];

export const getPageHelp = (t: TFunction<'help'>): Record<string, PageHelpBlock[]> => ({
  '/': [
    { kind: 'paragraph', text: t('pageHelp.dashboard.block0.text') },
    { kind: 'heading', text: t('pageHelp.dashboard.block1.text') },
    {
      kind: 'terms',
      items: [
        {
          term: t('pageHelp.dashboard.block2.item0.term'),
          description: t('pageHelp.dashboard.block2.item0.description'),
        },
        {
          term: t('pageHelp.dashboard.block2.item1.term'),
          description: t('pageHelp.dashboard.block2.item1.description'),
        },
        {
          term: t('pageHelp.dashboard.block2.item2.term'),
          description: t('pageHelp.dashboard.block2.item2.description'),
        },
        {
          term: t('pageHelp.dashboard.block2.item3.term'),
          description: t('pageHelp.dashboard.block2.item3.description'),
        },
      ],
    },
  ],
  '/runtime': [
    { kind: 'paragraph', text: t('pageHelp.runtime.block0.text') },
    { kind: 'heading', text: t('pageHelp.runtime.block1.text') },
    {
      kind: 'tips',
      items: [
        t('pageHelp.runtime.block2.item0'),
        t('pageHelp.runtime.block2.item1'),
        t('pageHelp.runtime.block2.item2'),
      ],
    },
  ],
  '/new-simulation': [
    { kind: 'paragraph', text: t('pageHelp.new_simulation.block0.text') },
    { kind: 'heading', text: t('pageHelp.new_simulation.block1.text') },
    { kind: 'paragraph', text: t('pageHelp.new_simulation.block2.text') },
  ],
  '/devices': [
    { kind: 'paragraph', text: t('pageHelp.devices.block0.text') },
    { kind: 'heading', text: t('pageHelp.devices.block1.text') },
    { kind: 'paragraph', text: t('pageHelp.devices.block2.text') },
  ],
  '/segments': [
    { kind: 'paragraph', text: t('pageHelp.segments.block0.text') },
    { kind: 'heading', text: t('pageHelp.segments.block1.text') },
    {
      kind: 'terms',
      items: [
        {
          term: t('pageHelp.segments.block2.item0.term'),
          description: t('pageHelp.segments.block2.item0.description'),
        },
        {
          term: t('pageHelp.segments.block2.item1.term'),
          description: t('pageHelp.segments.block2.item1.description'),
        },
      ],
    },
  ],
  '/device-config': [
    { kind: 'paragraph', text: t('pageHelp.device_config.block0.text') },
    { kind: 'heading', text: t('pageHelp.device_config.block1.text') },
    {
      kind: 'terms',
      items: [
        {
          term: t('pageHelp.device_config.block2.item0.term'),
          description: t('pageHelp.device_config.block2.item0.description'),
        },
        {
          term: t('pageHelp.device_config.block2.item1.term'),
          description: t('pageHelp.device_config.block2.item1.description'),
        },
        {
          term: t('pageHelp.device_config.block2.item2.term'),
          description: t('pageHelp.device_config.block2.item2.description'),
        },
      ],
    },
  ],
  '/topology': [
    { kind: 'paragraph', text: t('pageHelp.topology.block0.text') },
    { kind: 'heading', text: t('pageHelp.topology.block1.text') },
    { kind: 'paragraph', text: t('pageHelp.topology.block2.text') },
    { kind: 'heading', text: t('pageHelp.topology.block3.text') },
    { kind: 'paragraph', text: t('pageHelp.topology.block4.text') },
  ],
  '/alerts': [
    { kind: 'paragraph', text: t('pageHelp.alerts.block0.text') },
    { kind: 'heading', text: t('pageHelp.alerts.block1.text') },
    { kind: 'paragraph', text: t('pageHelp.alerts.block2.text') },
    { kind: 'heading', text: t('pageHelp.alerts.block3.text') },
    { kind: 'paragraph', text: t('pageHelp.alerts.block4.text') },
  ],
  '/traffic': [
    { kind: 'paragraph', text: t('pageHelp.traffic.block0.text') },
    { kind: 'heading', text: t('pageHelp.traffic.block1.text') },
    {
      kind: 'terms',
      items: [
        {
          term: t('pageHelp.traffic.block2.item0.term'),
          description: t('pageHelp.traffic.block2.item0.description'),
        },
        {
          term: t('pageHelp.traffic.block2.item1.term'),
          description: t('pageHelp.traffic.block2.item1.description'),
        },
        {
          term: t('pageHelp.traffic.block2.item2.term'),
          description: t('pageHelp.traffic.block2.item2.description'),
        },
      ],
    },
    { kind: 'paragraph', text: t('pageHelp.traffic.block3.text') },
  ],
  '/debug': [
    { kind: 'paragraph', text: t('pageHelp.debug.block0.text') },
    { kind: 'heading', text: t('pageHelp.debug.block1.text') },
    { kind: 'paragraph', text: t('pageHelp.debug.block2.text') },
  ],
  '/packets': [
    { kind: 'heading', text: t('pageHelp.packets.block0.text') },
    { kind: 'paragraph', text: t('pageHelp.packets.block1.text') },
    { kind: 'heading', text: t('pageHelp.packets.block2.text') },
    { kind: 'paragraph', text: t('pageHelp.packets.block3.text') },
  ],
  '/config-diff': [
    { kind: 'paragraph', text: t('pageHelp.config_diff.block0.text') },
    {
      kind: 'terms',
      items: [
        {
          term: t('pageHelp.config_diff.block1.item0.term'),
          description: t('pageHelp.config_diff.block1.item0.description'),
        },
        {
          term: t('pageHelp.config_diff.block1.item1.term'),
          description: t('pageHelp.config_diff.block1.item1.description'),
        },
      ],
    },
  ],
  '/walk-validator': [
    { kind: 'paragraph', text: t('pageHelp.walk_validator.block0.text') },
    { kind: 'heading', text: t('pageHelp.walk_validator.block1.text') },
    {
      kind: 'terms',
      items: [
        {
          term: t('pageHelp.walk_validator.block2.item0.term'),
          description: t('pageHelp.walk_validator.block2.item0.description'),
        },
        {
          term: t('pageHelp.walk_validator.block2.item1.term'),
          description: t('pageHelp.walk_validator.block2.item1.description'),
        },
        {
          term: t('pageHelp.walk_validator.block2.item2.term'),
          description: t('pageHelp.walk_validator.block2.item2.description'),
        },
      ],
    },
    { kind: 'heading', text: t('pageHelp.walk_validator.block3.text') },
    { kind: 'paragraph', text: t('pageHelp.walk_validator.block4.text') },
  ],
  '/walk-analyzer': [
    { kind: 'paragraph', text: t('pageHelp.walk_analyzer.block0.text') },
    { kind: 'heading', text: t('pageHelp.walk_analyzer.block1.text') },
    {
      kind: 'terms',
      items: [
        {
          term: t('pageHelp.walk_analyzer.block2.item0.term'),
          description: t('pageHelp.walk_analyzer.block2.item0.description'),
        },
        {
          term: t('pageHelp.walk_analyzer.block2.item1.term'),
          description: t('pageHelp.walk_analyzer.block2.item1.description'),
        },
        {
          term: t('pageHelp.walk_analyzer.block2.item2.term'),
          description: t('pageHelp.walk_analyzer.block2.item2.description'),
        },
      ],
    },
  ],
  '/library/walks': [
    { kind: 'paragraph', text: t('pageHelp.library_walks.block0.text') },
    { kind: 'paragraph', text: t('pageHelp.library_walks.block1.text') },
  ],
  '/library/pcaps': [
    { kind: 'paragraph', text: t('pageHelp.library_pcaps.block0.text') },
    { kind: 'paragraph', text: t('pageHelp.library_pcaps.block1.text') },
  ],
});
