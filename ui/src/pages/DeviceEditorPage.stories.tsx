/**
 * Stories for DeviceEditorPage (/device-config/new and /device-config/:hostname).
 *
 * The editor is routed from App.tsx rather than from pageRegistry's static
 * table, but it is lazily imported there like every other page — which is how
 * the U6 gate found it. Its two routes are one component in two states, so
 * Empty is the new-device form and Loaded is an authored device, reached
 * through a matched Route so useParams sees the hostname.
 */
import type { Meta, StoryObj } from '@storybook/react-vite';
import type { DeviceEditorSchema } from '../api/client';
import type { DeviceDetailResponse } from '../api/types';
import { LOADED_ROUTES, pageMeta, settled, withFailure } from '../test/storybook/pageStory';
import { DeviceEditorPage } from './DeviceEditorPage';

const HOSTNAME = 'core-sw-01';
const DEVICE_PATH = `/api/v1/config/devices/${HOSTNAME}`;

const device: DeviceDetailResponse = {
  hostname: HOSTNAME,
  mac: 'AA:BB:CC:00:00:01',
  ips: ['10.0.0.2'],
  type: 'switch',
  vlan: 10,
  interfaces: ['GigabitEthernet1/0/1', 'GigabitEthernet1/0/2'],
  rawYaml: `hostname: ${HOSTNAME}\nmac: AA:BB:CC:00:00:01\ntype: switch\n`,
};

const schema: DeviceEditorSchema = {
  type: 'switch',
  label: 'Switch',
  visibleSections: ['identity', 'interfaces', 'snmp', 'lldp'],
};

const editorRoutes = {
  ...LOADED_ROUTES,
  '/api/v1/device-schemas/switch': schema,
  [DEVICE_PATH]: device,
};

const meta: Meta<typeof DeviceEditorPage> = {
  ...pageMeta('DeviceEditorPage', DeviceEditorPage),
  parameters: { route: '/device-config/new', routePattern: '/device-config/:hostname' },
};

export default meta;
type Story = StoryObj<typeof DeviceEditorPage>;

/** A new device: the blank authoring form. */
export const Empty: Story = { parameters: { api: editorRoutes }, play: settled() };

/** An authored device loaded for editing. */
export const Loaded: Story = {
  parameters: { api: editorRoutes, route: `/device-config/${HOSTNAME}` },
  play: settled(),
};

/** The device read returns 500 while its hostname is in the URL. */
export const Error: Story = {
  parameters: {
    api: { ...editorRoutes, ...withFailure(DEVICE_PATH) },
    route: `/device-config/${HOSTNAME}`,
  },
  play: settled(),
};
