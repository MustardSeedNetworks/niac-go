import type { LucideIcon } from 'lucide-react';
import { ChevronLeft, ChevronRight, HelpCircle, Menu, Network, Settings, X } from 'lucide-react';
import { createElement, type FC, type ReactNode, useEffect, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { HelpDrawer } from '../components/HelpDrawer';
import { SettingsDrawer } from '../components/SettingsDrawer';
import { iconSizes } from '../constants/sizes';
import { prefetchRoute } from '../utils/prefetch';
import { safeGetItem, safeSetItem } from '../utils/storage';
import { ConnectionStatus } from './ConnectionStatus';

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

interface SidebarProps {
  groups: SidebarNavGroup[];
  version?: string;
  children: ReactNode;
}

// Store collapsed state in localStorage
const STORAGE_KEY = 'niac-sidebar-collapsed';

export const SidebarLayout: FC<SidebarProps> = ({ groups, version, children }) => {
  const location = useLocation();
  const navigate = useNavigate();
  const [collapsed, setCollapsed] = useState(() => {
    return safeGetItem(STORAGE_KEY) === 'true';
  });
  const [mobileOpen, setMobileOpen] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [helpOpen, setHelpOpen] = useState(false);

  // Save collapsed state
  useEffect(() => {
    safeSetItem(STORAGE_KEY, String(collapsed));
  }, [collapsed]);

  // Close mobile menu on navigation
  useEffect(() => {
    setMobileOpen(false);
  }, []);

  const isActive = (path: string) =>
    location.pathname === path || (path !== '/' && location.pathname.startsWith(path));

  const renderNavItem = (item: SidebarNavItem) => {
    const active = isActive(item.path);
    return (
      <button
        type="button"
        key={item.path}
        onClick={() => navigate(item.path)}
        onMouseEnter={() => prefetchRoute(item.path)}
        className={`
          group flex items-center gap-3 w-full px-3 py-2.5 rounded-lg text-sm font-medium
          transition-all duration-200
          ${
            active
              ? 'bg-gradient-to-r from-brand-600/30 to-brand-500/20 text-text-primary shadow-[inset_0_1px_0_rgba(255,255,255,0.1)]'
              : 'text-text-muted hover:text-text-primary hover:bg-white/5'
          }
        `}
        title={collapsed ? item.label : undefined}
      >
        {createElement(item.icon, {
          className: `${iconSizes.lg} flex-shrink-0 ${active ? 'text-brand-400' : 'text-text-muted group-hover:text-text-secondary'}`,
        })}
        {!collapsed && (
          <>
            <span className="flex-1 text-left truncate">{item.label}</span>
            {item.badge && (
              <span
                className={`px-1.5 py-0.5 text-xs rounded font-medium ${
                  item.badge === 'New'
                    ? 'bg-status-success/20 text-status-success'
                    : item.badge === 'Beta'
                      ? 'bg-status-warning/20 text-status-warning'
                      : 'bg-brand-500/20 text-brand-300'
                }`}
              >
                {item.badge}
              </span>
            )}
          </>
        )}
        {collapsed && item.badge && (
          <span className="absolute left-12 px-1.5 py-0.5 text-xs rounded font-medium bg-brand-500/20 text-brand-300">
            {item.badge}
          </span>
        )}
      </button>
    );
  };

  const sidebarContent = (
    <>
      {/* Logo */}
      <div
        className={`flex items-center ${collapsed ? 'justify-center' : 'justify-between'} px-3 py-4 border-b border-white/10`}
      >
        <div className={`flex items-center gap-2 ${collapsed ? 'justify-center' : ''}`}>
          <div className="relative flex-shrink-0">
            <div className="h-9 w-9 rounded-lg bg-gradient-to-br from-brand-500 to-brand-700 flex items-center justify-center shadow-lg shadow-brand-500/30">
              <Network className={`${iconSizes.lg} text-text-primary`} />
            </div>
            <div className="absolute -top-0.5 -right-0.5 h-2.5 w-2.5 rounded-full bg-status-success border-2 border-border-muted" />
          </div>
          {!collapsed && (
            <span className="font-display font-bold text-lg text-text-primary tracking-tight">
              NIAC
            </span>
          )}
        </div>
        {!collapsed && (
          <button
            type="button"
            onClick={() => setCollapsed(true)}
            className="p-1.5 rounded-lg text-text-muted hover:text-text-primary hover:bg-white/10 transition-colors lg:flex hidden"
            title="Collapse the sidebar to icons-only mode to give the main content more horizontal space"
            aria-label="Collapse sidebar"
          >
            <ChevronLeft className={iconSizes.md} />
          </button>
        )}
      </div>

      {/* Navigation */}
      <nav className="flex-1 overflow-y-auto py-4 px-2 space-y-6">
        {groups.map((group) => (
          <div key={group.label}>
            {!collapsed && (
              <h3 className="px-3 mb-2 text-xs font-semibold text-text-muted uppercase tracking-wider">
                {group.label}
              </h3>
            )}
            {collapsed && <div className="h-px bg-white/10 mx-2 mb-2" />}
            <div className="space-y-1">{group.items.map(renderNavItem)}</div>
          </div>
        ))}
      </nav>

      {/* Footer */}
      <div className={`px-3 py-4 border-t border-white/10 ${collapsed ? 'text-center' : ''}`}>
        {/* Action Buttons */}
        <div className={`${collapsed ? 'space-y-2' : 'flex items-center gap-2'} mb-3`}>
          <button
            type="button"
            onClick={() => setHelpOpen(true)}
            className={`
              ${collapsed ? 'w-full' : 'flex-1'}
              flex items-center ${collapsed ? 'justify-center' : 'gap-2'} px-3 py-2 rounded-lg
              text-text-muted hover:text-text-primary hover:bg-white/10 transition-colors
              text-sm font-medium
            `}
            title="Open the help dialog with keyboard shortcuts, documentation links, and supported features"
            aria-label="Open help"
          >
            <HelpCircle className={`${iconSizes.md} flex-shrink-0`} />
            {!collapsed && <span>Help</span>}
          </button>
          <button
            type="button"
            onClick={() => setSettingsOpen(true)}
            className={`
              ${collapsed ? 'w-full' : 'flex-1'}
              flex items-center ${collapsed ? 'justify-center' : 'gap-2'} px-3 py-2 rounded-lg
              text-text-muted hover:text-text-primary hover:bg-white/10 transition-colors
              text-sm font-medium
            `}
            title="Open the settings modal to change theme, default capture interface, and other application preferences"
            aria-label="Open settings"
          >
            <Settings className={`${iconSizes.md} flex-shrink-0`} />
            {!collapsed && <span>Settings</span>}
          </button>
        </div>

        {/* Connection Status */}
        {!collapsed && <ConnectionStatus />}

        {/* Version Info */}
        {version && (
          <div
            className={`text-xs font-mono text-text-muted ${collapsed ? '' : 'flex items-center justify-between'}`}
          >
            {!collapsed && <span>Version</span>}
            <span className={collapsed ? '' : 'text-text-muted'}>{version}</span>
          </div>
        )}
        {collapsed && (
          <button
            type="button"
            onClick={() => setCollapsed(false)}
            className="mt-2 p-1.5 rounded-lg text-text-muted hover:text-text-primary hover:bg-white/10 transition-colors"
            title="Expand the sidebar to show navigation labels alongside icons"
            aria-label="Expand sidebar"
          >
            <ChevronRight className={iconSizes.md} />
          </button>
        )}
      </div>
    </>
  );

  return (
    <div className="min-h-screen bg-gradient-to-br from-bg-base via-bg-surface to-gray-950">
      {/* Skip to main content link for accessibility */}
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:fixed focus:top-2 focus:left-2 focus:z-[100] focus:px-4 focus:py-2 focus:rounded-lg focus:bg-brand-600 focus:text-text-primary focus:outline-none"
      >
        Skip to main content
      </a>

      {/* Mobile header */}
      <header className="lg:hidden fixed top-0 left-0 right-0 z-50 flex items-center justify-between px-4 py-3 bg-bg-surface/95 backdrop-blur-xl border-b border-white/10">
        <div className="flex items-center gap-2">
          <div className="h-8 w-8 rounded-lg bg-gradient-to-br from-brand-500 to-brand-700 flex items-center justify-center">
            <Network className={`${iconSizes.md} text-text-primary`} />
          </div>
          <span className="font-display font-bold text-text-primary">NIAC</span>
        </div>
        <button
          type="button"
          onClick={() => setMobileOpen(!mobileOpen)}
          className="p-2 rounded-lg text-text-muted hover:text-text-primary hover:bg-white/10 transition-colors"
          title={
            mobileOpen
              ? 'Close the navigation drawer'
              : 'Open the navigation drawer to switch pages'
          }
          aria-label={mobileOpen ? 'Close menu' : 'Open menu'}
        >
          {mobileOpen ? <X className={iconSizes.lg} /> : <Menu className={iconSizes.lg} />}
        </button>
      </header>

      {/* Mobile sidebar overlay */}
      {mobileOpen && (
        <button
          type="button"
          className="lg:hidden fixed inset-0 z-40 bg-black/60 backdrop-blur-sm"
          onClick={() => setMobileOpen(false)}
          title="Tap outside to close the navigation drawer"
          aria-label="Close menu"
        />
      )}

      {/* Mobile sidebar */}
      <aside
        className={`
          lg:hidden fixed top-0 left-0 z-50 h-full w-72 bg-bg-surface/95 backdrop-blur-xl border-r border-white/10
          transform transition-transform duration-300 ease-in-out
          ${mobileOpen ? 'translate-x-0' : '-translate-x-full'}
        `}
      >
        <div className="flex flex-col h-full">{sidebarContent}</div>
      </aside>

      {/* Desktop sidebar */}
      <aside
        className={`
          hidden lg:flex fixed top-0 left-0 z-40 h-full flex-col
          bg-bg-surface/80 backdrop-blur-xl border-r border-white/10
          transition-all duration-300 ease-in-out
          ${collapsed ? 'w-16' : 'w-64'}
        `}
      >
        {sidebarContent}
      </aside>

      {/* Main content */}
      <main
        id="main-content"
        className={`
          transition-all duration-300 ease-in-out
          pt-16 lg:pt-0
          ${collapsed ? 'lg:pl-16' : 'lg:pl-64'}
        `}
      >
        <div className="p-4 sm:p-6 lg:p-8">{children}</div>
      </main>

      {/* Drawers */}
      <SettingsDrawer
        isOpen={settingsOpen}
        onClose={() => setSettingsOpen(false)}
        version={version}
      />
      <HelpDrawer isOpen={helpOpen} onClose={() => setHelpOpen(false)} />
    </div>
  );
};
