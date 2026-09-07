/**
 * Stories for the walk validator page.
 *
 * Same reason as the analyzer's: the a11y gate had no page coverage, so
 * neither of this page's two tables (per-line issues, batch results) was
 * ever checked. The stories drive the page's own buttons so both tables
 * are rendered when axe runs.
 */
import type { Meta, StoryObj } from '@storybook/react-vite';
import { expect, userEvent, within } from 'storybook/test';
import type { WalkBatchValidationResponse, WalkValidationResponse } from '../api/types';
import { WalkValidatorPage } from './WalkValidatorPage';

const walks = [
  {
    name: 'cisco/c3900.walk',
    sizeBytes: 12_004,
    modifiedAt: '2026-09-01T10:00:00Z',
    source: 'starter',
    edited: false,
  },
];

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

/** Answers only the calls this page makes; anything else is a 404. */
const stubFetch = (input: RequestInfo | URL) => {
  const url = String(input instanceof Request ? input.url : input);
  const json = (body: unknown) =>
    Promise.resolve(
      new Response(JSON.stringify(body), { headers: { 'content-type': 'application/json' } }),
    );
  if (url.includes('/api/v1/library/walks')) return json(walks);
  if (url.includes('/api/v1/csrf-token')) return json({ token: 'story' });
  if (url.includes('/api/v1/walk/validate-all')) return json(batch);
  if (url.includes('/api/v1/walk/validate')) return json(validation);
  return Promise.resolve(new Response('not found', { status: 404 }));
};

/**
 * Pages render inside the shell's `<main>` (Sidebar.tsx). A story that
 * renders the page bare puts its card-section `<header>` elements outside
 * any sectioning content, where they map to `role="banner"` — so axe
 * reports duplicate banner landmarks that do not exist in the app. The
 * wrapper reproduces the real landmark context.
 */
const meta: Meta<typeof WalkValidatorPage> = {
  title: 'Pages/WalkValidatorPage',
  component: WalkValidatorPage,
  decorators: [
    (Story) => {
      globalThis.fetch = stubFetch as typeof fetch;
      return (
        <main>
          <Story />
        </main>
      );
    },
  ],
};

export default meta;
type Story = StoryObj<typeof WalkValidatorPage>;

/** Before a walk has been validated: the picker and the three actions. */
export const Empty: Story = {};

/** After Validate: the per-line issues table. */
export const WithIssues: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByRole('button', { name: 'Validate' }));
    await expect(await canvas.findByText('malformed subnet mask')).toBeInTheDocument();
  },
};

/** After Validate all: the per-file batch results table. */
export const BatchResults: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByRole('button', { name: /validate all/i }));
    await expect(await canvas.findByText('juniper/ex4300.walk')).toBeInTheDocument();
  },
};
