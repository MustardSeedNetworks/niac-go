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
import {
  ChevronLeft,
  ChevronRight,
  HelpCircle,
  type LucideIcon,
  Menu,
  Settings,
  X,
} from 'lucide-react';
import { createElement, type FC, type ReactNode, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useLocation, useNavigate } from 'react-router';
import { iconSizes } from '../constants/sizes';
import { prefetchRoute } from '../utils/prefetch';
import { safeGetItem, safeSetItem } from '../utils/storage';
import { MsnMark } from './MsnMark';

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

interface SidebarLayoutProps {
  groups: SidebarNavGroup[];
  version?: string;
  children: ReactNode;
  /**
   * Drawer callbacks — all optional. Pass only the ones your product uses;
   * the corresponding footer button only renders when its callback is provided.
   * niac uses help/settings. Add more here if a new product needs another drawer.
   */
  onOpenHelp?: () => void;
  onOpenSettings?: () => void;
  topBar?: ReactNode;
}

const STORAGE_KEY = 'niac-sidebar-collapsed';

interface NavItemButtonProps {
  item: SidebarNavItem;
  active: boolean;
  collapsed: boolean;
  onNavigate: (path: string) => void;
}

function badgeClass(badge: string): string {
  if (badge === 'New') return 'bg-status-success/20 text-status-success';
  if (badge === 'Beta') return 'bg-status-warning/20 text-status-warning';
  return 'bg-brand-primary/20 text-brand-accent';
}

const NavItemButton: FC<NavItemButtonProps> = ({ item, active, collapsed, onNavigate }) => (
  <button
    type="button"
    onClick={() => onNavigate(item.path)}
    onMouseEnter={() => prefetchRoute(item.path)}
    // Keyed by route so a spec names the destination rather than the label,
    // which is translated copy and changes without the navigation changing.
    data-testid={`nav-item-${item.path === '/' ? 'root' : item.path.replace(/^\//, '')}`}
    aria-current={active ? 'page' : undefined}
    /* 44px minimum target, 11px radius, and a 3px left bar for the active
       route. The bar carries the state rather than a gradient fill: a filled
       row competes with status colour, and the rail is chrome. */
    className={`group relative flex items-center gap-default w-full min-h-11 px-3 py-2.5 rounded-[11px] text-sm font-medium transition-all duration-200 ${
      active
        ? 'bg-[color-mix(in_oklab,var(--color-brand-primary)_16%,transparent)] text-text-primary'
        : 'text-text-muted hover:text-text-primary hover:bg-surface-hover'
    }`}
    title={collapsed ? item.label : undefined}
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
        <span className="flex-1 text-left truncate">{item.label}</span>
        {item.badge ? (
          <span className={`px-1.5 py-0.5 text-xs rounded font-medium ${badgeClass(item.badge)}`}>
            {item.badge}
          </span>
        ) : null}
      </>
    ) : null}
  </button>
);

interface FooterIconButtonProps {
  collapsed: boolean;
  onClick: () => void;
  icon: LucideIcon;
  label: string;
  title: string;
  'data-testid'?: string;
}

const FooterIconButton: FC<FooterIconButtonProps> = ({
  collapsed,
  onClick,
  icon,
  label,
  title,
  'data-testid': dataTestId,
}) => (
  <button
    type="button"
    onClick={onClick}
    data-testid={dataTestId}
    className={`${collapsed ? 'w-full' : 'flex-1'} flex items-center ${
      collapsed ? 'justify-center' : 'gap-compact'
    } px-3 py-row rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover transition-colors text-sm font-medium`}
    title={title}
    aria-label={title}
  >
    {createElement(icon, { className: `${iconSizes.md} flex-shrink-0` })}
    {!collapsed ? <span>{label}</span> : null}
  </button>
);

interface SidebarHeaderProps {
  collapsed: boolean;
  onCollapse: () => void;
}

const SidebarHeader: FC<SidebarHeaderProps> = ({ collapsed, onCollapse }) => {
  const { t } = useTranslation();
  return (
    <div
      className={`flex items-center ${
        collapsed ? 'justify-center' : 'justify-between'
      } px-3 py-4 border-b border-surface-border`}
    >
      <div className={`flex items-center gap-compact ${collapsed ? 'justify-center' : ''}`}>
        <div className="relative flex-shrink-0">
          <div className="h-9 w-9 rounded-[11px] bg-brand-primary flex-center">
            <span className="figure text-sm font-extrabold tracking-tight text-on-brand">NI</span>
          </div>
          <div className="absolute -top-0.5 -right-0.5 h-2.5 w-2.5 rounded-full bg-status-success border-2 border-surface-raised" />
        </div>
        {!collapsed ? (
          <span className="font-display font-bold text-lg text-text-primary tracking-tight">
            {t('app.title')}
          </span>
        ) : null}
      </div>
      {!collapsed ? (
        <button
          type="button"
          onClick={onCollapse}
          className="p-1.5 rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover transition-colors lg:flex hidden"
          title="Collapse sidebar"
          aria-label="Collapse sidebar"
        >
          <ChevronLeft className={iconSizes.md} />
        </button>
      ) : null}
    </div>
  );
};

interface SidebarFooterProps {
  collapsed: boolean;
  version?: string;
  onOpenHelp?: () => void;
  onOpenSettings?: () => void;
  onExpand: () => void;
}

const SidebarFooter: FC<SidebarFooterProps> = ({
  collapsed,
  version,
  onOpenHelp,
  onOpenSettings,
  onExpand,
}) => {
  const { t } = useTranslation();
  return (
    <div className={`px-3 py-4 border-t border-surface-border ${collapsed ? 'text-center' : ''}`}>
      <div className={`${collapsed ? 'stack-sm' : 'flex items-center gap-compact'} mb-heading`}>
        {onOpenHelp ? (
          <FooterIconButton
            collapsed={collapsed}
            onClick={onOpenHelp}
            icon={HelpCircle}
            label={t('footer.help')}
            title={t('footer.openHelp')}
            data-testid="sidebar-help-button"
          />
        ) : null}
        {onOpenSettings ? (
          <FooterIconButton
            collapsed={collapsed}
            onClick={onOpenSettings}
            icon={Settings}
            label={t('footer.settings')}
            title={t('footer.openSettings')}
            data-testid="sidebar-settings-button"
          />
        ) : null}
      </div>

      {version ? (
        <div className={`text-xs font-mono text-text-muted ${collapsed ? '' : 'flex-between'}`}>
          {!collapsed ? <span>{t('footer.version')}</span> : null}
          <span>{version}</span>
        </div>
      ) : null}
      {/* Whose tool this is, under what it is. Quiet by design: the product
          mark at the top of the rail is the one that has to be recognised. */}
      <MsnMark collapsed={collapsed} className="mt-3" />
      {collapsed ? (
        <button
          type="button"
          onClick={onExpand}
          className="mt-inline p-1.5 rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover transition-colors"
          title={t('footer.expandSidebar')}
          aria-label={t('footer.expandSidebar')}
        >
          <ChevronRight className={iconSizes.md} />
        </button>
      ) : null}
    </div>
  );
};

interface SidebarBodyProps {
  groups: SidebarNavGroup[];
  collapsed: boolean;
  version?: string;
  onCollapse: () => void;
  onExpand: () => void;
  onNavigate: (path: string) => void;
  isActive: (path: string) => boolean;
  onOpenHelp?: () => void;
  onOpenSettings?: () => void;
}

const SidebarBody: FC<SidebarBodyProps> = ({
  groups,
  collapsed,
  version,
  onCollapse,
  onExpand,
  onNavigate,
  isActive,
  onOpenHelp,
  onOpenSettings,
}) => {
  const { t } = useTranslation();
  // group.label is either a plain display string ("Account") or an
  // i18n key ("common:sections.modules"). t() returns the translation
  // if the key resolves; otherwise the defaultValue (label itself).
  const translateLabel = (label: string): string =>
    label ? t(label, { defaultValue: label }) : '';
  return (
    <>
      <SidebarHeader collapsed={collapsed} onCollapse={onCollapse} />
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
      <SidebarFooter
        collapsed={collapsed}
        version={version}
        onOpenHelp={onOpenHelp}
        onOpenSettings={onOpenSettings}
        onExpand={onExpand}
      />
    </>
  );
};

interface MobileTopBarProps {
  mobileOpen: boolean;
  toggleMobile: () => void;
}

const MobileTopBar: FC<MobileTopBarProps> = ({ mobileOpen, toggleMobile }) => {
  const { t } = useTranslation();
  return (
    <header className="lg:hidden fixed top-0 left-0 right-0 z-50 flex-between px-4 py-row-lg bg-surface-raised/95 backdrop-blur-xl border-b border-surface-border">
      <div className="flex items-center gap-compact">
        <div className="h-8 w-8 rounded-[11px] bg-brand-primary flex-center">
          <span className="figure text-xs font-extrabold tracking-tight text-on-brand">NI</span>
        </div>
        <span className="font-display font-bold text-text-primary">{t('app.title')}</span>
      </div>
      <button
        type="button"
        onClick={toggleMobile}
        data-testid="mobile-menu-toggle"
        className="pad-xs rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover transition-colors"
        title={mobileOpen ? 'Close menu' : 'Open menu'}
        aria-label={mobileOpen ? 'Close menu' : 'Open menu'}
      >
        {mobileOpen ? <X className={iconSizes.lg} /> : <Menu className={iconSizes.lg} />}
      </button>
    </header>
  );
};

export const SidebarLayout: FC<SidebarLayoutProps> = ({
  groups,
  version,
  children,
  onOpenHelp,
  onOpenSettings,
  topBar,
}) => {
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
  useEffect(() => {
    setMobileOpen(false);
  }, [location.pathname]);

  const isActive = (path: string) =>
    location.pathname === path || (path !== '/' && location.pathname.startsWith(path));

  // Both asides below stay in the DOM regardless of viewport: the responsive
  // classes toggle display, not mount. So every sidebar testid exists twice,
  // and a test has to say which surface it means — each aside carries
  // data-testid="sidebar-mobile" / "sidebar-desktop" and specs scope through
  // it (see e2e/support/sidebar.ts).
  //
  // Previously the mobile copy emitted no testids at all, to keep unscoped
  // getByTestId calls resolving to one element. That kept the desktop tests
  // simple and left the mobile navigation undrivable — nothing could open it,
  // so no test could reach any mobile layout (#1320).
  const body = () => (
    <SidebarBody
      groups={groups}
      collapsed={collapsed}
      version={version}
      onCollapse={() => setCollapsed(true)}
      onExpand={() => setCollapsed(false)}
      onNavigate={(p) => navigate(p)}
      isActive={isActive}
      onOpenHelp={onOpenHelp}
      onOpenSettings={onOpenSettings}
    />
  );

  return (
    <div className="min-h-screen text-text-primary bg-gradient-to-br from-surface-base via-surface-raised to-surface-deep">
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:fixed focus:top-2 focus:left-2 focus:z-[100] focus:px-4 focus:py-row focus:rounded-lg focus:bg-brand-primary focus:text-on-brand focus:outline-none"
      >
        {t('accessibility.skipToMainContent')}
      </a>

      <MobileTopBar mobileOpen={mobileOpen} toggleMobile={() => setMobileOpen(!mobileOpen)} />

      {mobileOpen ? (
        <button
          type="button"
          className="lg:hidden fixed inset-0 z-40 bg-scrim/60 backdrop-blur-sm"
          onClick={() => setMobileOpen(false)}
          aria-label="Close menu"
        />
      ) : null}

      <aside
        data-testid="sidebar-mobile"
        className={`lg:hidden fixed top-0 left-0 z-50 h-full w-72 bg-surface-raised/95 backdrop-blur-xl border-r border-surface-border transform transition-transform duration-300 ease-in-out ${
          mobileOpen ? 'translate-x-0' : '-translate-x-full'
        }`}
      >
        <div className="flex flex-col h-full">{body()}</div>
      </aside>

      <aside
        data-testid="sidebar-desktop"
        className={`hidden lg:flex fixed top-0 left-0 z-40 h-full flex-col bg-gradient-to-b from-rail-from to-rail-to backdrop-blur-xl border-r border-hairline transition-all duration-300 ease-in-out ${
          collapsed ? 'w-16' : 'w-[252px]'
        }`}
      >
        {body()}
      </aside>

      <main
        id="main-content"
        className={`transition-all duration-300 ease-in-out pt-16 lg:pt-0 ${
          collapsed ? 'lg:pl-16' : 'lg:pl-64'
        }`}
      >
        {topBar}
        <div className="pad sm:pad-lg lg:pad-xl">{children}</div>
      </main>
    </div>
  );
};
