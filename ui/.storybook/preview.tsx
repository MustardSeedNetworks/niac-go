import type { Preview } from '@storybook/react-vite';
import { Suspense } from 'react';
import { I18nextProvider } from 'react-i18next';
import i18n from '../src/i18n';
import '../src/index.css';

const preview: Preview = {
  parameters: {
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/i,
      },
    },

    a11y: {
      // 'todo' - show a11y violations in the test UI only
      // 'error' - fail CI on a11y violations
      // 'off' - skip a11y checks entirely
      test: 'error',
    },

    backgrounds: {
      default: 'light',
      values: [
        { name: 'light', value: '#ffffff' },
        { name: 'dark', value: '#1a1a2e' },
        { name: 'surface', value: '#f8fafc' },
      ],
    },
  },

  decorators: [
    // Stories rendered without this see an uninitialised i18next: every
    // useTranslation() call warns NO_I18NEXT_INSTANCE and renders nothing, so a
    // component's real copy — including the accessible names the a11y addon
    // evaluates — is simply absent. The interaction suite was passing against
    // unlabelled UI. seed's preview has wrapped stories this way from the start.
    (Story) => (
      <I18nextProvider i18n={i18n}>
        <Suspense fallback={<div className="pad-sm text-text-muted">Loading…</div>}>
          <div className="font-sans antialiased">
            <Story />
          </div>
        </Suspense>
      </I18nextProvider>
    ),
  ],

  tags: ['autodocs'],
};

export default preview;
