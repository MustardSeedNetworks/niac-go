import { Wrench } from 'lucide-react';
import { memo, type ReactNode, Suspense } from 'react';
import { Navigate, Route, Routes, useLocation } from 'react-router-dom';
import { ErrorBoundary, PageErrorBoundary } from './components/ErrorBoundary';
import { AppProvider, useAppState } from './contexts/AppContext';
import { useKeyboardShortcuts } from './hooks/useKeyboardShortcuts';
import { navGroups } from './navGroups';
import { DeviceEditorPageRef, type PageConfig, pages } from './pageRegistry';
import { Breadcrumbs } from './ui/Breadcrumbs';
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

function AppShell() {
  const { data: version } = useAppState('version');

  useKeyboardShortcuts();

  return (
    <SidebarLayout groups={navGroups} version={version?.version}>
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
                  label: 'New Device',
                  title: 'New Device',
                  description: 'Create a new network device configuration.',
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
                  label: 'Edit Device',
                  title: 'Edit Device',
                  description: 'Edit device configuration settings.',
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
