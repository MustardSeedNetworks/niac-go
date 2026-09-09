import '@fontsource-variable/manrope';
import '@fontsource-variable/inter';
import '@fontsource-variable/jetbrains-mono';
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { createBrowserRouter } from 'react-router';
import { RouterProvider } from 'react-router/dom';
import App from './App.tsx';
import { AuthGate } from './components/AuthGate';
import { ScopeProvider } from './contexts/ScopeContext';
import { initThemeFromStorage } from './hooks/useTheme';
import './i18n';
import './i18n/types';
import './index.css';

// Apply persisted/default theme before first paint to avoid a flash of
// the wrong palette. Defaults to dark when no preference is stored.
initThemeFromStorage();

const rootElement = document.getElementById('root');

if (!rootElement) {
  throw new Error('Root element not found');
}

const router = createBrowserRouter([
  {
    path: '*',
    element: (
      <AuthGate>
        <ScopeProvider>
          <App />
        </ScopeProvider>
      </AuthGate>
    ),
  },
]);

createRoot(rootElement).render(
  <StrictMode>
    <RouterProvider router={router} />
  </StrictMode>,
);
