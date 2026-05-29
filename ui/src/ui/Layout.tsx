import type { LucideIcon } from 'lucide-react';
import { ChevronDown, ChevronLeft, ChevronRight, HelpCircle, X } from 'lucide-react';
import {
  createElement,
  type FC,
  type ReactNode,
  useCallback,
  useEffect,
  useRef,
  useState,
} from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { iconSizes } from '../constants/sizes';

export interface NavItem {
  path: string;
  label: string;
  icon?: LucideIcon | ReactNode;
  badge?: string;
}

export interface NavGroup {
  label: string;
  items: NavItem[];
}

interface PageShellProps {
  children: ReactNode;
  className?: string;
}

export const PageShell: FC<PageShellProps> = ({ children, className = '' }) => (
  <div
    className={`min-h-screen bg-gradient-to-br from-bg-base via-bg-surface to-surface-deep ${className}`}
  >
    <div className="mx-auto max-w-7xl px-4 sm:px-6 py-6">{children}</div>
  </div>
);

/**
 * PrimaryNav Component
 *
 * Top-level horizontal navigation bar with grouped nav items,
 * active-state highlighting, optional logo, and version display.
 * Supports both flat item lists and grouped navigation sections.
 */
interface PrimaryNavProps {
  items: NavItem[];
  groups?: NavGroup[];
  currentPath?: string;
  onNavigate?: (path: string) => void;
  logo?: ReactNode;
  version?: string;
  className?: string;
}

export const PrimaryNav: FC<PrimaryNavProps> = ({
  items,
  groups,
  currentPath: externalPath,
  onNavigate: externalNavigate,
  logo,
  version,
  className = '',
}) => {
  const location = useLocation();
  const navigate = useNavigate();
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const [showLeftScroll, setShowLeftScroll] = useState(false);
  const [showRightScroll, setShowRightScroll] = useState(false);
  const [openGroup, setOpenGroup] = useState<string | null>(null);

  const currentPath = externalPath ?? location.pathname;
  const handleNavigate = externalNavigate ?? ((path: string) => navigate(path));

  // Check scroll position
  const checkScroll = useCallback(() => {
    const container = scrollContainerRef.current;
    if (container) {
      setShowLeftScroll(container.scrollLeft > 0);
      setShowRightScroll(container.scrollLeft < container.scrollWidth - container.clientWidth - 1);
    }
  }, []);

  useEffect(() => {
    checkScroll();
    window.addEventListener('resize', checkScroll);
    return () => window.removeEventListener('resize', checkScroll);
  }, [checkScroll]);

  const scroll = (direction: 'left' | 'right') => {
    const container = scrollContainerRef.current;
    if (container) {
      const scrollAmount = 200;
      container.scrollBy({
        left: direction === 'left' ? -scrollAmount : scrollAmount,
        behavior: 'smooth',
      });
      setTimeout(checkScroll, 300);
    }
  };

  const renderNavItem = (item: NavItem, isGrouped = false) => {
    const isActive =
      currentPath === item.path || (item.path !== '/' && currentPath.startsWith(item.path));

    const iconElement = item.icon
      ? typeof item.icon === 'function'
        ? createElement(item.icon as LucideIcon, {
            className: 'h-4 w-4 flex-shrink-0',
          })
        : item.icon
      : null;

    return (
      <button
        type="button"
        key={item.path}
        onClick={() => {
          handleNavigate(item.path);
          setOpenGroup(null);
        }}
        className={`flex items-center gap-compact px-3 py-row rounded-lg text-sm font-medium transition-all duration-200 whitespace-nowrap ${
          isActive
            ? 'bg-gradient-to-r from-brand-primary/30 to-brand-primary/20 text-text-primary border-l-2 border-brand-primary shadow-edge-highlight'
            : 'text-text-muted hover:text-text-primary hover:bg-surface-hover'
        } ${isGrouped ? 'w-full justify-start' : ''}`}
      >
        {iconElement}
        <span>{item.label}</span>
        {item.badge && (
          <span
            className={`ml-tight px-1.5 py-0.5 text-xs rounded font-medium ${
              item.badge === 'New'
                ? 'bg-status-success/20 text-status-success'
                : item.badge === 'Beta'
                  ? 'bg-status-warning/20 text-status-warning'
                  : 'bg-brand-primary/20 text-brand-accent'
            }`}
          >
            {item.badge}
          </span>
        )}
      </button>
    );
  };

  // Use groups if provided, otherwise use flat items
  const navItems = groups ? groups.flatMap((g) => g.items) : items;

  return (
    <nav className={`relative flex items-center gap-compact mb-section ${className}`}>
      {/* Logo */}
      {logo && <div className="flex-shrink-0 mr-4">{logo}</div>}

      {/* Scroll left button */}
      {showLeftScroll && (
        <button
          type="button"
          onClick={() => scroll('left')}
          className="flex-shrink-0 p-1.5 rounded-lg bg-bg-elevated/80 text-text-muted hover:text-text-primary hover:bg-bg-elevated transition-colors"
          aria-label="Scroll left"
        >
          <ChevronLeft className={iconSizes.md} />
        </button>
      )}

      {/* Navigation items */}
      <div
        ref={scrollContainerRef}
        className="flex items-center gap-tight overflow-x-auto scrollbar-hide scroll-smooth flex-1"
        onScroll={checkScroll}
      >
        {groups
          ? // Grouped navigation
            groups.map((group, groupIndex) => (
              <div key={group.label} className="relative flex items-center">
                {groupIndex > 0 && <div className="h-6 w-px bg-surface-hover mx-2" />}
                <div className="relative">
                  <button
                    type="button"
                    onClick={() => setOpenGroup(openGroup === group.label ? null : group.label)}
                    className="flex items-center gap-tight px-3 py-row text-sm font-medium text-text-muted hover:text-text-primary transition-colors"
                  >
                    <span>{group.label}</span>
                    <ChevronDown
                      className={`h-3 w-3 transition-transform ${openGroup === group.label ? 'rotate-180' : ''}`}
                    />
                  </button>
                  {openGroup === group.label && (
                    <div className="absolute top-full left-0 mt-tight py-compact min-w-[180px] rounded-lg bg-bg-surface/95 backdrop-blur-xl border border-surface-border shadow-xl z-50 animate-slide-down">
                      {group.items.map((item) => (
                        <div key={item.path} className="px-1">
                          {renderNavItem(item, true)}
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            ))
          : // Flat navigation
            navItems.map((item) => renderNavItem(item))}
      </div>

      {/* Scroll right button */}
      {showRightScroll && (
        <button
          type="button"
          onClick={() => scroll('right')}
          className="flex-shrink-0 p-1.5 rounded-lg bg-bg-elevated/80 text-text-muted hover:text-text-primary hover:bg-bg-elevated transition-colors"
          aria-label="Scroll right"
        >
          <ChevronRight className={iconSizes.md} />
        </button>
      )}

      {/* Version badge */}
      {version && (
        <div className="flex-shrink-0 ml-inline px-cell py-compact text-xs font-mono text-text-muted bg-bg-elevated/50 rounded">
          {version}
        </div>
      )}
    </nav>
  );
};

/**
 * PageHeader Component
 *
 * Renders a consistent page title area with optional icon, description,
 * action buttons, and breadcrumb navigation. Used at the top of every page
 * via the PageTemplate wrapper.
 */
interface PageHeaderProps {
  title: string;
  description?: string;
  icon?: LucideIcon;
  actions?: ReactNode;
  breadcrumbs?: { label: string; href?: string }[];
  /**
   * Rich help content shown in a side-panel when the user clicks the (?) icon.
   * Pass any ReactNode (paragraphs, lists, links). Omit to hide the button.
   */
  help?: ReactNode;
  className?: string;
}

export const PageHeader: FC<PageHeaderProps> = ({
  title,
  description,
  icon,
  actions,
  breadcrumbs,
  help,
  className = '',
}) => {
  const [helpOpen, setHelpOpen] = useState(false);
  return (
    <div className={`mb-section animate-fade-in ${className}`}>
      {breadcrumbs && breadcrumbs.length > 0 && (
        <Breadcrumb items={breadcrumbs} className="mb-heading" />
      )}
      <div className="flex flex-wrap items-start justify-between gap-comfortable">
        <div className="flex items-center gap-default">
          {icon && createElement(icon, { className: 'h-8 w-8 text-brand-accent' })}
          <div>
            <h1 className="heading-1 text-text-primary font-display">{title}</h1>
            {description && (
              <p className="text-sm text-text-muted mt-tight max-w-2xl">{description}</p>
            )}
          </div>
        </div>
        <div className="flex items-center gap-default">
          {actions}
          {help && (
            <button
              type="button"
              onClick={() => setHelpOpen(true)}
              aria-label={`Open help for ${title}`}
              title={`What is ${title}?`}
              className="rounded-full p-1.5 text-text-muted hover:bg-surface-hover hover:text-text-primary"
            >
              <HelpCircle className={iconSizes.lg} />
            </button>
          )}
        </div>
      </div>
      {help && helpOpen && (
        <HelpPanel title={title} onClose={() => setHelpOpen(false)}>
          {help}
        </HelpPanel>
      )}
    </div>
  );
};

/**
 * HelpPanel — fixed side panel rendered from PageHeader's (?) button.
 * Closes on overlay click or X. Content is opaque ReactNode so each page
 * can ship its own help with formatting / links / inline code.
 */
const HelpPanel: FC<{ title: string; children: ReactNode; onClose: () => void }> = ({
  title,
  children,
  onClose,
}) => {
  // Close on Escape (covers keyboard users; the overlay button below covers
  // pointer + screen-reader users).
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose]);

  return (
    <div
      className="fixed inset-0 z-50 flex justify-end bg-scrim/40 backdrop-blur-sm"
      role="dialog"
      aria-modal="true"
      aria-label={`Help: ${title}`}
    >
      {/* Click-outside dismiss, keyboard-accessible. */}
      <button
        type="button"
        aria-label="Close help (overlay)"
        className="absolute inset-0 cursor-default"
        onClick={onClose}
      />
      <aside className="relative h-full w-full max-w-md overflow-y-auto bg-bg-surface pad-lg shadow-2xl">
        <div className="mb-content flex-between">
          <h2 className="heading-3 text-text-primary">{title}</h2>
          <button
            type="button"
            onClick={onClose}
            aria-label="Close help"
            className="rounded p-1 text-text-muted hover:bg-surface-hover hover:text-text-primary"
          >
            <X className="h-5 w-5" />
          </button>
        </div>
        <div className="prose prose-invert prose-sm max-w-none text-text-primary">{children}</div>
      </aside>
    </div>
  );
};

// Status indicator component
interface StatusIndicatorProps {
  status: 'online' | 'offline' | 'warning' | 'error' | 'pending';
  label?: string;
  pulse?: boolean;
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}

const statusColors = {
  online: 'bg-status-success',
  offline: 'bg-bg-muted',
  warning: 'bg-status-warning',
  error: 'bg-status-error',
  pending: 'bg-status-info',
};

const statusLabels = {
  online: 'Online',
  offline: 'Offline',
  warning: 'Warning',
  error: 'Error',
  pending: 'Pending',
};

export const StatusIndicator: FC<StatusIndicatorProps> = ({
  status,
  label,
  pulse = false,
  size = 'md',
  className = '',
}) => {
  const sizeClasses = {
    sm: 'h-2 w-2',
    md: 'h-2.5 w-2.5',
    lg: 'h-3 w-3',
  };

  return (
    <div className={`flex items-center gap-compact ${className}`}>
      <span
        className={`relative inline-flex ${sizeClasses[size]} rounded-full ${statusColors[status]}`}
      >
        {pulse && (status === 'online' || status === 'pending') && (
          <span
            className={`absolute inset-0 rounded-full ${statusColors[status]} animate-ping opacity-75`}
          />
        )}
      </span>
      {(label || label === undefined) && (
        <span className="text-sm text-text-secondary">{label ?? statusLabels[status]}</span>
      )}
    </div>
  );
};

// Breadcrumb component
interface BreadcrumbItem {
  label: string;
  href?: string;
}

interface BreadcrumbProps {
  items: BreadcrumbItem[];
  className?: string;
}

export const Breadcrumb: FC<BreadcrumbProps> = ({ items, className = '' }) => (
  <nav className={`flex items-center gap-tight text-sm ${className}`} aria-label="Breadcrumb">
    {items.map((item, index) => (
      <div key={item.label} className="flex items-center gap-tight">
        {index > 0 && <ChevronRight className={`${iconSizes.md} text-text-disabled`} />}
        {item.href ? (
          <Link
            to={item.href}
            className="text-text-muted hover:text-text-primary transition-colors"
          >
            {item.label}
          </Link>
        ) : (
          <span className="text-text-secondary font-medium">{item.label}</span>
        )}
      </div>
    ))}
  </nav>
);
