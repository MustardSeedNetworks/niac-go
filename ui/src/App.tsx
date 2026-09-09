import { Wrench } from 'lucide-react';
import { memo, type ReactNode, type RefObject, Suspense, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Navigate, Route, Routes, useLocation } from 'react-router';
import { ErrorBoundary, PageErrorBoundary } from './components/ErrorBoundary';
import { HeaderBar } from './components/HeaderBar';
import { HelpDrawer } from './components/HelpDrawer';
import { SettingsDrawer } from './components/SettingsDrawer';
import { AppProvider, useAppState } from './contexts/AppContext';
import { pageHelp } from './data/page-help';
import { useFocusOnRouteChange } from './hooks/useFocusOnRouteChange';
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
  // Route the (?) was clicked on, cleared on close so the next click re-applies.
  const [helpPath, setHelpPath] = useState<string | undefined>(undefined);
  const [settingsOpen, setSettingsOpen] = useState(false);
  // Held here rather than inside PageWithErrorBoundary: that subtree is keyed
  // on the pathname, so anything living in it is remounted by the very
  // navigation it would have to remember.
  const pageTitleRef = useRef<HTMLHeadingElement>(null);

  useKeyboardShortcuts(() => setHelpOpen(true));
  useFocusOnRouteChange(pageTitleRef);

  return (
    <SidebarLayout
      groups={navGroups}
      version={version?.version}
      topBar={<HeaderBar />}
      onOpenHelp={() => setHelpOpen(true)}
      onOpenSettings={() => setSettingsOpen(true)}
    >
      <ToastContainer />
      <Suspense fallback={<PageLoader />}>
        <Routes>
          {pages.map((page) => (
            <Route
              key={page.path}
              path={page.path}
              element={
                <PageWithErrorBoundary page={page} onOpenHelp={setHelpPath} titleRef={pageTitleRef}>
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
                onOpenHelp={setHelpPath}
                titleRef={pageTitleRef}
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
                onOpenHelp={setHelpPath}
                titleRef={pageTitleRef}
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
          <Route path="*" element={<Navigate to="/" replace={true} />} />
        </Routes>
      </Suspense>
      <SettingsDrawer
        isOpen={settingsOpen}
        onClose={() => setSettingsOpen(false)}
        version={version?.version}
      />
      <HelpDrawer
        isOpen={helpOpen || helpPath !== undefined}
        onClose={() => {
          setHelpOpen(false);
          setHelpPath(undefined);
        }}
        section={helpPath}
      />
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
  ({
    page,
    onOpenHelp,
    titleRef,
    children,
  }: {
    page: PageConfig;
    onOpenHelp: (path: string) => void;
    titleRef: RefObject<HTMLHeadingElement | null>;
    children: ReactNode;
  }) => {
    const location = useLocation();
    // Dynamic routes (the per-device editor) carry no page help; they get no
    // (?) rather than one that opens on someone else's content.
    const documented = page.path in pageHelp;
    return (
      <PageErrorBoundary key={location.pathname}>
        <section className="stack-xl">
          <Breadcrumbs />
          <PageHeader
            titleRef={titleRef}
            icon={page.icon}
            eyebrow={page.eyebrow}
            title={page.title}
            description={page.description}
            onHelp={documented ? () => onOpenHelp(page.path) : undefined}
          />
          {children}
        </section>
      </PageErrorBoundary>
    );
  },
);

PageWithErrorBoundary.displayName = 'PageWithErrorBoundary';
