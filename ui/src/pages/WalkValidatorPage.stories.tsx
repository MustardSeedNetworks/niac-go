/**
 * Stories for the walk validator page (/walk-validator).
 *
 * Migrated onto the shared page harness with the analyzer's (U6); the
 * validator-specific part is the two response payloads and the clicks that
 * render its per-line and per-file tables for axe.
 */
import type { Meta, StoryObj } from '@storybook/react-vite';
import { expect, userEvent, within } from 'storybook/test';
import type { WalkBatchValidationResponse, WalkValidationResponse } from '../api/types';
import {
  EMPTY_ROUTES,
  LOADED_ROUTES,
  pageMeta,
  settled,
  withFailure,
} from '../test/storybook/pageStory';
import { WalkValidatorPage } from './WalkValidatorPage';

const validation: WalkValidationResponse = {
  success: false,
  message: '2 issues found',
  result: {
    filename: 'cisco/c3900.walk',
    valid: false,
    totalLines: 480,
    validLines: 478,
    issues: [
      {
        line: 65,
        severity: 'error',
        message: 'malformed subnet mask',
        original: '.1.3.6.1.2.1.4.20.1.3.10.1.1.1 = IpAddress: 255.255.255.O',
        autoFix: true,
        oid: '.1.3.6.1.2.1.4.20.1.3.10.1.1.1',
      },
      {
        line: 212,
        severity: 'warning',
        message: 'value type not recognised, treated as OCTET STRING',
        original: '.1.3.6.1.2.1.1.9.1.3.4 = Opaque: 00 01',
        autoFix: false,
        oid: '.1.3.6.1.2.1.1.9.1.3.4',
      },
    ],
  },
};

const batch: WalkBatchValidationResponse = {
  success: false,
  message: '1 of 2 walk files has issues',
  totalFiles: 2,
  invalidFiles: 1,
  totalIssues: 2,
  results: {
    'cisco/c3900.walk': {
      filename: 'cisco/c3900.walk',
      valid: false,
      totalLines: 480,
      validLines: 478,
      issues: validation.result?.issues ?? [],
    },
    'juniper/ex4300.walk': {
      filename: 'juniper/ex4300.walk',
      valid: true,
      totalLines: 900,
      validLines: 900,
      issues: [],
    },
  },
};

const meta: Meta<typeof WalkValidatorPage> = {
  ...pageMeta('WalkValidatorPage', WalkValidatorPage),
  parameters: { route: '/walk-validator' },
};

export default meta;
type Story = StoryObj<typeof WalkValidatorPage>;

const validateRoutes = {
  '/api/v1/walk/validate': validation,
  '/api/v1/walk/validate-all': batch,
};

/** No walks in the library yet: nothing to validate. */
export const Empty: Story = { parameters: { api: EMPTY_ROUTES }, play: settled() };

/** After Validate: the per-line issues table. */
export const Loaded: Story = {
  parameters: { api: { ...LOADED_ROUTES, ...validateRoutes } },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByRole('button', { name: 'Validate' }));
    await expect(await canvas.findByText('malformed subnet mask')).toBeInTheDocument();
  },
};

/** After Validate all: the per-file batch results table. */
export const BatchResults: Story = {
  parameters: { api: { ...LOADED_ROUTES, ...validateRoutes } },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByRole('button', { name: /validate all/i }));
    await expect(await canvas.findByText('juniper/ex4300.walk')).toBeInTheDocument();
  },
};

/** The walk listing returns 500, so the picker cannot be populated. */
export const Error: Story = {
  parameters: { api: withFailure('/api/v1/library/walks') },
  play: settled(),
};
