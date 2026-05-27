import '@fontsource-variable/inter';
import '@fontsource-variable/jetbrains-mono';
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import App from './App.tsx';
import { LicenseProvider } from './contexts/LicenseContext';
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

createRoot(rootElement).render(
  <StrictMode>
    <BrowserRouter>
      <LicenseProvider>
        <App />
      </LicenseProvider>
    </BrowserRouter>
  </StrictMode>,
);
