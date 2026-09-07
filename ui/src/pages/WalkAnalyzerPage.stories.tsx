/**
 * Stories for the walk analyzer page (/walk-analyzer).
 *
 * These were the first page stories in the tree (U2b) and hand-rolled their
 * own fetch stub and `<main>` wrapper. Both now come from the shared page
 * harness that U6 gave every route, so this file carries only what is
 * specific to the analyzer: the analysis payload and the click that puts the
 * migrated DataTables on screen before axe runs.
 */
import type { Meta, StoryObj } from '@storybook/react-vite';
import { expect, userEvent, within } from 'storybook/test';
import type { WalkAnalyzeResponse } from '../api/types';
import {
  EMPTY_ROUTES,
  LOADED_ROUTES,
  pageMeta,
  settled,
  withFailure,
} from '../test/storybook/pageStory';
import { WalkAnalyzerPage } from './WalkAnalyzerPage';

const analysis: WalkAnalyzeResponse = {
  success: true,
  result: {
    device: {
      sysname: 'core-sw-01',
      sysdescr: 'Cisco IOS Software, C9300',
      sysobjectid: '1.3.6.1.4.1.9.1.2494',
      syslocation: 'MDF rack 3',
    },
    interfaces: [
      {
        index: 1,
        name: 'GigabitEthernet1/0/1',
        description: 'uplink to dist-01',
        type: 'ethernetCsmacd',
        speed: 1_000_000_000,
        adminStatus: 'up',
        operStatus: 'up',
        macAddress: 'AA:BB:CC:00:00:01',
      },
      {
        index: 2,
        name: 'GigabitEthernet1/0/2',
        type: 'ethernetCsmacd',
        speed: 1_000_000_000,
        adminStatus: 'up',
        operStatus: 'down',
        macAddress: 'AA:BB:CC:00:00:02',
      },
    ],
    neighbors: [
      {
        localInterface: 'Gi1/0/1',
        remoteDevice: 'dist-01',
        remoteInterface: 'Te1/1/1',
        protocol: 'lldp',
      },
      {
        localInterface: 'Gi1/0/2',
        remoteDevice: 'ap-204',
        remoteInterface: 'eth0',
        protocol: 'cdp',
      },
    ],
    statistics: {
      totalInterfaces: 2,
      physicalInterfaces: 2,
      logicalInterfaces: 0,
      totalNeighbors: 2,
    },
  },
};

const meta: Meta<typeof WalkAnalyzerPage> = {
  ...pageMeta('WalkAnalyzerPage', WalkAnalyzerPage),
  parameters: { route: '/walk-analyzer' },
};

export default meta;
type Story = StoryObj<typeof WalkAnalyzerPage>;

const analyzeRoutes = { '/api/v1/walk/analyze': analysis };

/** No walks in the library yet: the picker has nothing to offer. */
export const Empty: Story = { parameters: { api: EMPTY_ROUTES }, play: settled() };

/** After Analyze: identity card plus the interfaces and neighbours tables. */
export const Loaded: Story = {
  parameters: { api: { ...LOADED_ROUTES, ...analyzeRoutes } },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId('walk-analyzer-analyze-button'));
    await expect(await canvas.findByText('GigabitEthernet1/0/1')).toBeInTheDocument();
    await expect(canvas.getByText('dist-01')).toBeInTheDocument();
  },
};

/** The walk listing returns 500, so the picker cannot be populated. */
export const Error: Story = {
  parameters: { api: withFailure('/api/v1/library/walks') },
  play: settled(),
};
