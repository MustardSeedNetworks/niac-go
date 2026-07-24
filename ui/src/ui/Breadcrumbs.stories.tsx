/**
 * Breadcrumbs primitive stories (Wave 5 / niac-W5-2c).
 *
 * Breadcrumbs reads from the React Router location; wrap stories
 * in MemoryRouter to inject a synthetic path.
 */
import type { Meta, StoryObj } from '@storybook/react-vite';
import { MemoryRouter } from 'react-router';
import { Breadcrumbs } from './Breadcrumbs';

const meta: Meta<typeof Breadcrumbs> = {
  title: 'UI/Breadcrumbs',
  component: Breadcrumbs,
  parameters: { layout: 'padded' },
  decorators: [
    (Story, ctx) => {
      const path = (ctx.parameters.path as string | undefined) ?? '/';
      return (
        <MemoryRouter initialEntries={[path]}>
          <div className="p-4">
            <Story />
          </div>
        </MemoryRouter>
      );
    },
  ],
};
export default meta;

type Story = StoryObj<typeof Breadcrumbs>;

export const Root: Story = {
  parameters: { path: '/', docs: { description: { story: 'At /, Breadcrumbs renders nothing.' } } },
};

export const SingleSegment: Story = { parameters: { path: '/devices' } };

export const NestedTwoSegments: Story = { parameters: { path: '/devices/router-01' } };

export const DeepPath: Story = {
  parameters: { path: '/devices/router-01/protocols/dhcp' },
};
