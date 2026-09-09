import type { Meta, StoryObj } from '@storybook/react-vite';
import { expect, within } from 'storybook/test';
import { ApiError, NetworkError } from '../../api/errors';
import { WizardStatusNotice } from './WizardStatusNotice';

const meta: Meta<typeof WizardStatusNotice> = {
  title: 'Wizard/StatusNotice',
  component: WizardStatusNotice,
  decorators: [
    (Story) => (
      <main>
        <Story />
      </main>
    ),
  ],
};
export default meta;
type Story = StoryObj<typeof WizardStatusNotice>;

export const Loading: Story = {
  args: { loading: true, error: null },
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement).getByRole('status')).toHaveTextContent(
      'Checking simulation availability',
    );
  },
};
export const RequestFailed: Story = {
  args: { loading: false, error: new ApiError('Status request returned HTTP 403.', 403) },
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement).getByRole('alert')).toHaveTextContent('HTTP 403');
  },
};
export const DaemonUnavailable: Story = {
  args: { loading: false, error: new NetworkError() },
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement).getByRole('alert')).toHaveTextContent('Cannot reach NIAC');
  },
};
export const WrongMode: Story = {
  args: { loading: false, error: new ApiError('Not implemented', 501) },
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement).getByRole('alert')).toHaveTextContent('niac daemon');
  },
};
