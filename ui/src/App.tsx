import { Layers, Wrench } from 'lucide-react';
import { memo, type ReactNode, type RefObject, Suspense, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Route, Routes, useLocation } from 'react-router';
import { disconnectRuntimeAPI } from './api/requestCore';
import { AppPageHeader } from './components/AppPageHeader';
import { ErrorBoundary, PageErrorBoundary } from './components/ErrorBoundary';
import { HelpDrawer } from './components/HelpDrawer';
import { NotFound } from './components/NotFound';
import { SettingsDrawer } from './components/SettingsDrawer';
import { AppProvider, useAppContext } from './contexts/AppContext';
import { pageHelpRoutes } from './data/page-help';
import { useBuildVersion } from './hooks/useBuildVersion';
import { useConnectionStatus } from './hooks/useConnectionStatus';
import { useDocumentTitle } from './hooks/useDocumentTitle';
import { useFocusOnRouteChange } from './hooks/useFocusOnRouteChange';
import { useKeyboardShortcuts } from './hooks/useKeyboardShortcuts';
import { useTheme } from './hooks/useTheme';
import { useNavGroups } from './navGroups';
import { DeviceEditorPageRef, type PageConfig, usePages } from './pageRegistry';
import { PageLoader } from './ui/PageLoader';
import { type SidebarFooterProps, SidebarLayout } from './ui/Sidebar';
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
  const status = useConnectionStatus();
  const { isDark, toggleTheme } = useTheme();
  const { t: tCommon } = useTranslation('common');
  const { sessionId, setSessionId, sessions } = useAppContext();
  const { t } = useTranslation('pages');
  const { version } = useBuildVersion();
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
  const { pathname } = useLocation();

  // `checking` is no verdict yet, so the rail shows no status until the first
  // probe answers.
  const railStatus: SidebarFooterProps['status'] =
    status === 'connected'
      ? {
          state: 'connected',
          label: tCommon('connection.railConnected'),
          detail: tCommon('connection.railConnectedDetail'),
        }
      : status === 'disconnected'
        ? {
            state: 'disconnected',
            label: tCommon('connection.railDisconnected'),
            detail: tCommon('connection.railDisconnectedDetail'),
          }
        : undefined;

  const footer: SidebarFooterProps = {
    status: railStatus,
    productSlot: {
      label: tCommon('session.label'),
      value: sessionId ?? tCommon('session.none'),
      icon: <Layers />,
      options: sessions.flatMap(({ sessionId: id }) => (id ? [{ value: id, label: id }] : [])),
      selected: sessionId ?? '',
      onSelect: setSessionId,
    },
    help: { label: tCommon('accessibility.openHelp'), onClick: () => setHelpOpen(true) },
    settings: {
      label: tCommon('accessibility.openSettings'),
      onClick: () => setSettingsOpen(true),
    },
    theme: {
      label: tCommon(isDark ? 'accessibility.switchToLightMode' : 'accessibility.switchToDarkMode'),
      onClick: toggleTheme,
      isDark,
    },
    signOut: { label: tCommon('auth.disconnect'), onClick: disconnectRuntimeAPI },
    version: { value: version, label: tCommon('footer.version') },
  };

  useDocumentTitle();
  useKeyboardShortcuts(() => setHelpOpen(true));
  useFocusOnRouteChange(pageTitleRef);

  return (
    <SidebarLayout groups={navGroups} footer={footer}>
      {/* Keyed on the route so a suspending navigation mounts a NEW boundary.
          Every page is lazy(), and React Router updates in a transition: a
          transition that suspends against an EXISTING boundary keeps the whole
          previous tree on screen with no fallback, while the router has already
          pushed the new URL synchronously. The rail, the header and the body
          then describe a page the operator has navigated away from, for as long
          as the chunk takes -- which reads as a tap that did nothing, and left
          the mobile shell spec clicking a layout that was about to move
          (#2151). A newly mounted boundary shows its fallback instead, so the
          shell commits at the new location and only the page body waits. */}
      <Suspense key={pathname} fallback={<PageLoader />}>
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
                  eyebrow: t('deviceEditor.eyebrow'),
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
                  eyebrow: t('deviceEditor.eyebrow'),
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
          <Route path="*" element={<NotFound />} />
        </Routes>
      </Suspense>
      <SettingsDrawer
        connectionStatus={status}
        isOpen={settingsOpen}
        onClose={() => setSettingsOpen(false)}
        version={version}
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
    const documented = pageHelpRoutes.includes(page.path);
    return (
      <PageErrorBoundary key={location.pathname}>
        <section className="stack-xl" data-testid="page-frame">
          <AppPageHeader
            titleRef={titleRef}
            kicker={page.eyebrow}
            title={page.title}
            subtitle={page.description}
            onHelp={documented ? () => onOpenHelp(page.path) : undefined}
          />
          {children}
        </section>
      </PageErrorBoundary>
    );
  },
);

PageWithErrorBoundary.displayName = 'PageWithErrorBoundary';
