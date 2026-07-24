import { Wrench } from 'lucide-react';
import { memo, type ReactNode, Suspense, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Navigate, Route, Routes, useLocation } from 'react-router-dom';
import { ErrorBoundary, PageErrorBoundary } from './components/ErrorBoundary';
import { HeaderBar } from './components/HeaderBar';
import { HelpDrawer } from './components/HelpDrawer';
import { SettingsDrawer } from './components/SettingsDrawer';
import { ReadOnlyView } from './components/ui/ReadOnlyView';
import { AppProvider, useAppState } from './contexts/AppContext';
import { useKeyboardShortcuts } from './hooks/useKeyboardShortcuts';
import { useNavGroups } from './navGroups';
import { DeviceEditorPageRef, type PageConfig, usePages } from './pageRegistry';
import { Breadcrumbs } from './ui/Breadcrumbs';
import { PageHeader } from './ui/PageHeader';
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
  const { t } = useTranslation('pages');
  const { data: version } = useAppState('version');
  const navGroups = useNavGroups();
  const pages = usePages();
  // Drawers used to live inside Sidebar.tsx; moved here as part of Phase 1
  // so the canonical Sidebar (synced from stem) stays drawer-agnostic.
  const [helpOpen, setHelpOpen] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);

  useKeyboardShortcuts(() => setHelpOpen(true));

  return (
    <SidebarLayout
      groups={navGroups}
      version={version?.version}
      topBar={<HeaderBar />}
      onOpenHelp={() => setHelpOpen(true)}
      onOpenSettings={() => setSettingsOpen(true)}
    >
      <ToastContainer />
      {/* #762: a read-only-scoped token sees every page read-only via
          one wrap. ReadOnlyView's HTML <fieldset disabled> propagates
          `disabled` to every descendant button/input/select/textarea
          so individual pages don't need to gate their own controls.
          Read-write and admin tokens see no chrome change. */}
      <ReadOnlyView>
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
            <Route
              path="/pcap-analyzer"
              element={<Navigate to="/packets?view=files" replace={true} />}
            />
            <Route path="*" element={<Navigate to="/" replace={true} />} />
          </Routes>
        </Suspense>
      </ReadOnlyView>
      <SettingsDrawer
        isOpen={settingsOpen}
        onClose={() => setSettingsOpen(false)}
        version={version?.version}
      />
      <HelpDrawer isOpen={helpOpen} onClose={() => setHelpOpen(false)} />
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
        <section className="stack-xl">
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
