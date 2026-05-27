import { Moon, Sun, Wrench } from 'lucide-react';
import { memo, type ReactElement, type ReactNode, Suspense } from 'react';
import { useTranslation } from 'react-i18next';
import { Navigate, Route, Routes, useLocation } from 'react-router-dom';
import { ErrorBoundary, PageErrorBoundary } from './components/ErrorBoundary';
import { AppProvider, useAppState } from './contexts/AppContext';
import { useKeyboardShortcuts } from './hooks/useKeyboardShortcuts';
import { useTheme } from './hooks/useTheme';
import { useNavGroups } from './navGroups';
import { DeviceEditorPageRef, type PageConfig, usePages } from './pageRegistry';
import { Breadcrumbs } from './ui/Breadcrumbs';
import { ConnectionStatus } from './ui/ConnectionStatus';
import { PageHeader } from './ui/Layout';
import { PageLoader } from './ui/PageLoader';
import { SidebarLayout } from './ui/Sidebar';
import { ToastContainer } from './ui/ToastContainer';
import './App.css';

/**
 * App is the root of the React tree. Splits out into:
 *
 *   ErrorBoundary  — top-level crash catch + reload button
 *     AppProvider  — global state (version, stats, history, etc.)
 *       AppShell   — sidebar chrome + routed page below
 *
 * The route table itself lives in pageRegistry.tsx so adding a new page
 * is a single-file edit. Sidebar groups live in navGroups.ts.
 */
export default function App() {
  return (
    <ErrorBoundary>
      <AppProvider>
        <AppShell />
      </AppProvider>
    </ErrorBoundary>
  );
}

/**
 * TopBar renders the sticky upper-right control cluster: connection
 * indicator + theme toggle. Mirrors stem's UI shell so cross-product
 * navigation has consistent affordances.
 */
function TopBar(): ReactElement {
  const { t } = useTranslation('common');
  const { isDark, toggleTheme } = useTheme();
  const themeToggleLabel = isDark
    ? t('accessibility.switchToLightMode')
    : t('accessibility.switchToDarkMode');
  return (
    <>
      {/* Left side reserved for future breadcrumbs / page-level chrome. */}
      <div className="flex items-center" />
      <div className="flex items-center gap-2">
        <ConnectionStatus />
        <button
          type="button"
          onClick={toggleTheme}
          className="p-2 rounded-lg text-text-secondary hover:text-text-primary hover:bg-surface-hover"
          title={themeToggleLabel}
          aria-label={themeToggleLabel}
        >
          {isDark ? (
            <Sun className="h-5 w-5" aria-hidden="true" />
          ) : (
            <Moon className="h-5 w-5" aria-hidden="true" />
          )}
        </button>
      </div>
    </>
  );
}

function AppShell() {
  const { t } = useTranslation('pages');
  const { data: version } = useAppState('version');
  const navGroups = useNavGroups();
  const pages = usePages();

  useKeyboardShortcuts();

  return (
    <SidebarLayout groups={navGroups} version={version?.version} topBar={<TopBar />}>
      <ToastContainer />
      <Suspense fallback={<PageLoader />}>
        <Routes>
          {pages.map((page) => (
            <Route
              key={page.path}
              path={page.path}
              element={
                <PageWithErrorBoundary page={page}>
                  <page.component />
                </PageWithErrorBoundary>
              }
            />
          ))}
          {/* Dynamic routes for the device editor — both reuse the
              same lazy-loaded component as the Device Library, but
              wear different page-header metadata. */}
          <Route
            path="/device-config/new"
            element={
              <PageWithErrorBoundary
                page={{
                  path: '/device-config/new',
                  label: t('deviceEditor.newLabel'),
                  title: t('deviceEditor.newTitle'),
                  description: t('deviceEditor.newDescription'),
                  icon: Wrench,
                  component: DeviceEditorPageRef,
                }}
              >
                <DeviceEditorPageRef />
              </PageWithErrorBoundary>
            }
          />
          <Route
            path="/device-config/:hostname"
            element={
              <PageWithErrorBoundary
                page={{
                  path: '/device-config/:hostname',
                  label: t('deviceEditor.editLabel'),
                  title: t('deviceEditor.editTitle'),
                  description: t('deviceEditor.editDescription'),
                  icon: Wrench,
                  component: DeviceEditorPageRef,
                }}
              >
                <DeviceEditorPageRef />
              </PageWithErrorBoundary>
            }
          />
          {/* Back-compat for folded-in pages — bookmarks and copied URLs
              continue to work after pages moved into their host sections. */}
          <Route path="/templates" element={<Navigate to="/runtime" replace={true} />} />
          <Route path="/neighbors" element={<Navigate to="/topology" replace={true} />} />
          <Route path="/analysis" element={<Navigate to="/traffic" replace={true} />} />
          <Route path="/pcap-analyzer" element={<Navigate to="/packets" replace={true} />} />
          <Route path="*" element={<Navigate to="/" replace={true} />} />
        </Routes>
      </Suspense>
    </SidebarLayout>
  );
}

/**
 * PageWithErrorBoundary wraps every routed page in a PageErrorBoundary
 * keyed on the current pathname. The key is critical — without it,
 * navigating away from a crashed page kept the boundary's error state,
 * so the next page rendered the previous page's failure UI.
 */
const PageWithErrorBoundary = memo(
  ({ page, children }: { page: PageConfig; children: ReactNode }) => {
    const location = useLocation();
    return (
      <PageErrorBoundary key={location.pathname}>
        <section className="space-y-6">
          <Breadcrumbs />
          <PageHeader
            icon={page.icon}
            title={page.title}
            description={page.description}
            help={page.help}
          />
          {children}
        </section>
      </PageErrorBoundary>
    );
  },
);

PageWithErrorBoundary.displayName = 'PageWithErrorBoundary';
