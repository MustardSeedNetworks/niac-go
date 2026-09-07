/**
 * Stories for the walk analyzer page.
 *
 * The a11y gate only ever saw atoms — no page had a story, so the two
 * walk pages' tables were never checked (U2b). Both stories drive the
 * page through its own Analyze button so the migrated DataTables are on
 * screen when axe runs, rather than asserting against an empty shell.
 */
import type { Meta, StoryObj } from '@storybook/react-vite';
import { expect, userEvent, within } from 'storybook/test';
import type { WalkAnalyzeResponse } from '../api/types';
import { WalkAnalyzerPage } from './WalkAnalyzerPage';

const walks = [
  {
    name: 'cisco/c9300.walk',
    sizeBytes: 84_213,
    modifiedAt: '2026-09-01T10:00:00Z',
    source: 'starter',
    edited: false,
  },
];

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

/** Answers only the calls this page makes; anything else is a 404. */
const stubFetch = (analyzeBody: unknown) => (input: RequestInfo | URL) => {
  const url = String(input instanceof Request ? input.url : input);
  const json = (body: unknown) =>
    Promise.resolve(
      new Response(JSON.stringify(body), { headers: { 'content-type': 'application/json' } }),
    );
  if (url.includes('/api/v1/library/walks')) return json(walks);
  if (url.includes('/api/v1/csrf-token')) return json({ token: 'story' });
  if (url.includes('/api/v1/walk/analyze')) return json(analyzeBody);
  return Promise.resolve(new Response('not found', { status: 404 }));
};

/**
 * Pages render inside the shell's `<main>` (Sidebar.tsx). A story that
 * renders the page bare puts its card-section `<header>` elements outside
 * any sectioning content, where they map to `role="banner"` — so axe
 * reports duplicate banner landmarks that do not exist in the app. The
 * wrapper reproduces the real landmark context.
 */
const meta: Meta<typeof WalkAnalyzerPage> = {
  title: 'Pages/WalkAnalyzerPage',
  component: WalkAnalyzerPage,
  decorators: [
    (Story) => {
      globalThis.fetch = stubFetch(analysis) as typeof fetch;
      return (
        <main>
          <Story />
        </main>
      );
    },
  ],
};

export default meta;
type Story = StoryObj<typeof WalkAnalyzerPage>;

/** Before a walk has been analysed: the picker and the Analyze button only. */
export const Empty: Story = {};

/** After Analyze: identity card plus the interfaces and neighbours tables. */
export const Analyzed: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId('walk-analyzer-analyze-button'));
    await expect(await canvas.findByText('GigabitEthernet1/0/1')).toBeInTheDocument();
    await expect(canvas.getByText('dist-01')).toBeInTheDocument();
  },
};
