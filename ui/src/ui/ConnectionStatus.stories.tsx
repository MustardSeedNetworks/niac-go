/**
 * ConnectionStatus primitive stories (Wave 5 / niac-W5-2c).
 *
 * The component polls /api/v1/version. We mock the fetch globally
 * per story so each variant deterministically lands on a status
 * (connected / disconnected / checking) without hitting a real
 * backend.
 */
import type { Meta, StoryObj } from '@storybook/react-vite';
import { ConnectionStatus } from './ConnectionStatus';

type FetchInput = Parameters<typeof fetch>[0];
type FetchInit = Parameters<typeof fetch>[1];

function mockVersionFetch(behavior: 'ok' | 'fail' | 'hang'): () => void {
  const original = window.fetch;
  window.fetch = (input: FetchInput, init?: FetchInit) => {
    const url =
      typeof input === 'string' ? input : input instanceof URL ? input.toString() : input.url;
    if (url.includes('/api/v1/version')) {
      if (behavior === 'hang') {
        return new Promise(() => {
          /* never resolves — exercises the 'checking' branch */
        });
      }
      if (behavior === 'ok') {
        return Promise.resolve(new Response('{}', { status: 200 }));
      }
      return Promise.resolve(new Response('', { status: 503 }));
    }
    return original(input, init);
  };
  return () => {
    window.fetch = original;
  };
}

const meta: Meta<typeof ConnectionStatus> = {
  title: 'UI/ConnectionStatus',
  component: ConnectionStatus,
  parameters: { layout: 'centered' },
  decorators: [
    (Story, ctx) => {
      const behavior = (ctx.parameters.fetchBehavior as 'ok' | 'fail' | 'hang') ?? 'ok';
      const restore = mockVersionFetch(behavior);
      // restore on unmount via React effect inside Story wrapper
      // (storybook's decorator unmounts the wrapper, which collects the closure)
      setTimeout(restore, 60_000);
      return <Story />;
    },
  ],
};
export default meta;

type Story = StoryObj<typeof ConnectionStatus>;

export const Connected: Story = {
  parameters: { fetchBehavior: 'ok' },
};

export const Disconnected: Story = {
  parameters: { fetchBehavior: 'fail' },
};

export const Checking: Story = {
  parameters: {
    fetchBehavior: 'hang',
    docs: {
      description: { story: 'Fetch never resolves — exercises the initial checking state.' },
    },
  },
};
