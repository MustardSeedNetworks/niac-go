import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import App from './App.tsx';
import { initThemeFromStorage } from './hooks/useTheme';
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
      <App />
    </BrowserRouter>
  </StrictMode>,
);
