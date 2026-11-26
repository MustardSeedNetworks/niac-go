/**
 * Sidebar layout shell — persistent collapsible left navigation.
 *
 * Shared shell pattern — kept visually and behaviorally consistent across
 * seed / stem / niac by convention; each repo owns this file independently
 * (no master, no sync). All colors/spacing reference theme tokens;
 * per-product brand identity comes from each repo's index.css token values.
 *
 * Drawer triggers (help, settings, history) call up to the host App
 * via callback props so the actual drawer components stay mounted at
 * AppShell level alongside the existing test/state plumbing.
 */
import { type LucideIcon, Menu, X } from 'lucide-react';
import { createElement, type ReactNode, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useLocation, useNavigate } from 'react-router';
import msnLogo from '../assets/msn-logo.svg';
import { iconSizes } from '../constants/sizes';
import { accessibleNameIncluding } from '../lib/accessibleName';
import { RailFooter, type RailFooterProps } from '../shared/RailFooter';
import { Tooltip } from '../shared/Tooltip';
import { prefetchRoute } from '../utils/prefetch';
import { safeGetItem, safeSetItem } from '../utils/storage';
import { ProductMark } from './ProductMark';

export interface SidebarNavItem {
  path: string;
  label: string;
  icon: LucideIcon;
  badge?: string;
}

export interface SidebarNavGroup {
  label: string;
  items: SidebarNavItem[];
}

/** What the shared RailFooter needs from the app; the layout adds the collapse state. */
export type SidebarFooterProps = Omit<
  RailFooterProps,
  'collapsed' | 'variant' | 'collapse' | 'mark'
>;

interface SidebarLayoutProps {
  groups: SidebarNavGroup[];
  footer: SidebarFooterProps;
  children: ReactNode;
}

const STORAGE_KEY = 'niac-sidebar-collapsed';

interface NavItemButtonProps {
  item: SidebarNavItem;
  active: boolean;
  collapsed: boolean;
  onNavigate: (path: string) => void;
}

function badgeClass(badge: string): string {
  if (badge === 'New') return 'bg-status-success/20 text-status-success-strong';
  if (badge === 'Beta') return 'bg-status-warning/20 text-status-warning-strong';
  return 'bg-brand-primary/20 text-brand-primary-strong';
}

function NavItemButton({ item, active, collapsed, onNavigate }: NavItemButtonProps) {
  return (
    <Tooltip text={collapsed ? item.label : undefined}>
      <button
        type="button"
        onClick={() => onNavigate(item.path)}
        onMouseEnter={() => prefetchRoute(item.path)}
        // Keyed by route so a spec names the destination rather than the label,
        // which is translated copy and changes without the navigation changing.
        data-testid={`nav-item-${item.path === '/' ? 'root' : item.path.replace(/^\//, '')}`}
        aria-current={active ? 'page' : undefined}
        aria-label={accessibleNameIncluding(
          item.label,
          [item.label, !collapsed ? item.badge : undefined].filter(Boolean).join(' '),
        )}
        /* 44px minimum target, 11px radius, and a 3px left bar for the active
       route. The bar carries the state rather than a gradient fill: a filled
       row competes with status colour, and the rail is chrome. */
        className={`group relative flex items-center gap-default w-full min-h-11 px-3 py-2.5 rounded-[11px] text-sm font-medium transition-all duration-200 ${
          active
            ? 'bg-[color-mix(in_oklab,var(--color-brand-primary)_16%,transparent)] text-text-primary'
            : 'text-text-muted hover:text-text-primary hover:bg-surface-hover'
        }`}
      >
        {active ? (
          <span
            aria-hidden="true"
            className="absolute inset-y-1 left-0 w-[3px] rounded-full bg-brand-primary"
          />
        ) : null}
        {createElement(item.icon, {
          className: `${iconSizes.lg} flex-shrink-0 ${
            active ? 'text-brand-primary' : 'text-text-muted group-hover:text-text-secondary'
          }`,
        })}
        {!collapsed ? (
          <>
            <span className="flex-1 text-left truncate">
              {item.label}
              {/* A flex gap is not a word boundary. axe reads text nodes, so
                  "DNS" + "new" would glue into one token the name cannot
                  contain. */}
              {item.badge ? ' ' : null}
            </span>
            {item.badge ? (
              <span
                className={`px-1.5 py-0.5 text-xs rounded font-medium ${badgeClass(item.badge)}`}
              >
                {item.badge}
              </span>
            ) : null}
          </>
        ) : null}
      </button>
    </Tooltip>
  );
}

function SidebarHeader({ collapsed }: { collapsed: boolean }) {
  const { t } = useTranslation();
  return (
    <div
      className={`flex items-center ${
        collapsed ? 'justify-center' : ''
      } px-3 py-4 border-b border-surface-border`}
    >
      <div className="flex items-center gap-compact">
        <ProductMark />
        {!collapsed ? (
          <span className="font-display font-bold text-lg text-text-primary tracking-tight">
            {t('app.title')}
          </span>
        ) : null}
      </div>
    </div>
  );
}

interface SidebarBodyProps {
  groups: SidebarNavGroup[];
  collapsed: boolean;
  variant: 'rail' | 'drawer';
  footer: SidebarFooterProps;
  onToggleCollapse: () => void;
  onNavigate: (path: string) => void;
  isActive: (path: string) => boolean;
}

function SidebarBody({
  groups,
  collapsed,
  variant,
  footer,
  onToggleCollapse,
  onNavigate,
  isActive,
}: SidebarBodyProps) {
  const { t } = useTranslation();
  // group.label is either a plain display string ("Account") or an
  // i18n key ("common:sections.modules"). t() returns the translation
  // if the key resolves; otherwise the defaultValue (label itself).
  const translateLabel = (label: string): string =>
    label ? t(label, { defaultValue: label }) : '';
  return (
    <>
      <SidebarHeader collapsed={collapsed} />
      <nav className="flex-1 overflow-y-auto py-4 px-cell stack-xl">
        {groups.map((group, groupIndex) => (
          <div key={group.label || `nav-group-${String(groupIndex)}`}>
            {!collapsed && group.label ? (
              <h3 className="px-3 mb-2 text-xs font-semibold text-text-muted uppercase tracking-wider">
                {translateLabel(group.label)}
              </h3>
            ) : null}
            {collapsed ? <div className="h-px bg-surface-border mx-2 mb-2" /> : null}
            <div className="stack-xs">
              {group.items.map((item) => (
                <NavItemButton
                  key={item.path}
                  item={item}
                  active={isActive(item.path)}
                  collapsed={collapsed}
                  onNavigate={onNavigate}
                />
              ))}
            </div>
          </div>
        ))}
      </nav>
      <RailFooter
        {...footer}
        collapsed={collapsed}
        variant={variant}
        mark={{
          src: msnLogo,
          /* allow-hardcoded: a company name is a proper noun, not copy. */
          name: 'Mustard Seed Networks',
        }}
        collapse={{
          collapseLabel: t('accessibility.collapseSidebar'),
          expandLabel: t('accessibility.expandSidebar'),
          onToggle: onToggleCollapse,
        }}
      />
    </>
  );
}

interface MobileTopBarProps {
  mobileOpen: boolean;
  toggleMobile: () => void;
}

function MobileTopBar({ mobileOpen, toggleMobile }: MobileTopBarProps) {
  const { t } = useTranslation();
  return (
    <header className="lg:hidden fixed top-0 left-0 right-0 z-50 flex-between px-4 py-row-lg bg-surface-raised/95 backdrop-blur-xl border-b border-surface-border">
      <div className="flex items-center gap-compact">
        <ProductMark size="sm" />
        <span className="font-display font-bold text-text-primary">{t('app.title')}</span>
      </div>
      <Tooltip text={mobileOpen ? t('accessibility.closeMenu') : t('accessibility.openMenu')}>
        <button
          type="button"
          onClick={toggleMobile}
          data-testid="mobile-menu-toggle"
          className="pad-xs rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover transition-colors"
          aria-label={mobileOpen ? t('accessibility.closeMenu') : t('accessibility.openMenu')}
        >
          {mobileOpen ? <X className={iconSizes.lg} /> : <Menu className={iconSizes.lg} />}
        </button>
      </Tooltip>
    </header>
  );
}

export function SidebarLayout({ groups, footer, children }: SidebarLayoutProps) {
  const { t } = useTranslation();
  const location = useLocation();
  const navigate = useNavigate();
  const [collapsed, setCollapsed] = useState(() => safeGetItem(STORAGE_KEY) === 'true');
  const [mobileOpen, setMobileOpen] = useState(false);

  useEffect(() => {
    safeSetItem(STORAGE_KEY, String(collapsed));
  }, [collapsed]);

  // Close the drawer when the route changes. The dependency array was empty,
  // so this ran once on mount — where mobileOpen is already false — and the
  // drawer therefore never closed on navigation at all. On a phone, tapping a
  // nav item changed the route and left the drawer covering the page it had
  // just navigated to, with the toggle the only way out.
  //
  // Invisible on desktop, where the drawer is display:none at lg+, which is why
  // it survived until something drove a real phone viewport (#1320).
  // Backstop for a navigation this component did not start — browser back and
  // forward. A click-driven navigation has already closed the drawer, so this
  // sets an unchanged value and React bails out of the re-render.
  useEffect(() => {
    setMobileOpen(false);
  }, [location.pathname]);

  const isActive = (path: string) =>
    location.pathname === path || (path !== '/' && location.pathname.startsWith(path));

  // The desktop rail stays in the DOM at every viewport (the responsive
  // classes toggle display, not mount), and the open drawer renders the same
  // body, so while it is open every sidebar testid exists twice. A test has to
  // say which surface it means — each aside carries
  // data-testid="sidebar-mobile" / "sidebar-desktop" and specs scope through
  // it (see e2e/support/sidebar.ts).
  //
  // Previously the mobile copy emitted no testids at all, to keep unscoped
  // getByTestId calls resolving to one element. That kept the desktop tests
  // simple and left the mobile navigation undrivable — nothing could open it,
  // so no test could reach any mobile layout (#1320).
  const body = (railCollapsed: boolean, variant: 'rail' | 'drawer') => (
    <SidebarBody
      groups={groups}
      collapsed={railCollapsed}
      variant={variant}
      footer={footer}
      onToggleCollapse={() => setCollapsed(!collapsed)}
      onNavigate={(p) => {
        // Close with the click that caused the navigation, not from an
        // observer that fires whenever React gets to it. The effect above
        // still closes the drawer for history navigation, but by then this
        // has already set the same value, so it cannot surprise anything
        // that looked at the drawer in between (#2083).
        setMobileOpen(false);
        navigate(p);
      }}
      isActive={isActive}
    />
  );

  return (
    <div className="min-h-screen text-text-primary bg-gradient-to-br from-surface-base via-surface-raised to-surface-deep">
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:fixed focus:top-2 focus:left-2 focus:z-[100] focus:px-4 focus:py-2 focus:rounded-lg focus:bg-brand-primary focus:text-on-brand focus:outline-none"
      >
        {t('accessibility.skipToMainContent')}
      </a>

      <MobileTopBar mobileOpen={mobileOpen} toggleMobile={() => setMobileOpen(!mobileOpen)} />

      {mobileOpen ? (
        <button
          type="button"
          className="lg:hidden fixed inset-0 z-40 bg-scrim/60 backdrop-blur-sm"
          onClick={() => setMobileOpen(false)}
          aria-label={t('accessibility.closeMenu')}
        />
      ) : null}

      {/* The drawer mounts its rail on open, as seed's does: parked in the tree
          it kept every control and a second product mark in a menu nobody
          could see (#2284). The empty panel stays for the slide, `invisible`
          when closed: parked at x -288..0 it still painted a box the fleet's
          390px gate reads as content leaving the viewport. */}
      <aside
        data-testid="sidebar-mobile"
        className={`lg:hidden fixed top-0 left-0 z-50 h-full w-72 bg-surface-raised/95 backdrop-blur-xl border-r border-surface-border transform transition-[transform,visibility] duration-300 ease-in-out ${
          mobileOpen ? 'translate-x-0' : '-translate-x-full invisible'
        }`}
      >
        {mobileOpen ? <div className="flex flex-col h-full">{body(false, 'drawer')}</div> : null}
      </aside>

      <aside
        data-testid="sidebar-desktop"
        /* 224px. The width and main's left offset below are one Tailwind step
           (w-56 / pl-56) so they cannot drift — they were 252px against 256px
           before the density pass (UI-NIAC-19). */
        className={`hidden lg:flex fixed top-0 left-0 z-40 h-full flex-col bg-gradient-to-b from-rail-from to-rail-to backdrop-blur-xl border-r border-hairline transition-all duration-300 ease-in-out ${
          collapsed ? 'w-16' : 'w-56'
        }`}
      >
        {body(collapsed, 'rail')}
      </aside>

      <main
        id="main-content"
        className={`transition-all duration-300 ease-in-out pt-16 lg:pt-0 ${
          collapsed ? 'lg:pl-16' : 'lg:pl-56'
        }`}
      >
        <div className="pad">{children}</div>
      </main>
    </div>
  );
}
