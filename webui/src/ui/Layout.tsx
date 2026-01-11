import { type FC, type ReactNode, createElement } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import type { LucideIcon } from 'lucide-react';

export interface NavItem {
  path: string;
  label: string;
  icon?: LucideIcon | ReactNode;
  badge?: string;
}

interface PageShellProps {
  children: ReactNode;
  className?: string;
}

export const PageShell: FC<PageShellProps> = ({ children, className = '' }) => {
  return (
    <div className={`min-h-screen bg-gradient-to-br from-gray-950 via-gray-900 to-gray-950 p-6 ${className}`}>
      <div className="mx-auto max-w-7xl">
        {children}
      </div>
    </div>
  );
};

interface PrimaryNavProps {
  items: NavItem[];
  currentPath?: string;
  onNavigate?: (path: string) => void;
  logo?: ReactNode;
  className?: string;
}

export const PrimaryNav: FC<PrimaryNavProps> = ({
  items,
  currentPath: externalPath,
  onNavigate: externalNavigate,
  logo,
  className = '',
}) => {
  const location = useLocation();
  const navigate = useNavigate();

  // Use external values if provided, otherwise use react-router
  const currentPath = externalPath ?? location.pathname;
  const handleNavigate = externalNavigate ?? ((path: string) => navigate(path));

  return (
    <nav className={`flex items-center gap-2 ${className}`}>
      {logo && <div className="mr-4">{logo}</div>}
      {items.map((item) => {
        const isActive = currentPath === item.path ||
          (item.path !== '/' && currentPath.startsWith(item.path));

        // Handle icon - could be a LucideIcon component or a ReactElement
        const iconElement = item.icon
          ? typeof item.icon === 'function'
            ? createElement(item.icon as LucideIcon, { className: 'h-4 w-4 flex-shrink-0' })
            : item.icon
          : null;

        return (
          <button
            key={item.path}
            onClick={() => handleNavigate(item.path)}
            className={`flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-all duration-200 ${
              isActive
                ? 'bg-violet-600/20 text-violet-300 border border-violet-500/30'
                : 'text-gray-400 hover:text-white hover:bg-white/5'
            }`}
          >
            {iconElement}
            <span className="whitespace-nowrap">{item.label}</span>
            {item.badge && (
              <span className="ml-1 px-1.5 py-0.5 text-xs rounded bg-violet-500/20 text-violet-300">
                {item.badge}
              </span>
            )}
          </button>
        );
      })}
    </nav>
  );
};

interface PageHeaderProps {
  title: string;
  description?: string;
  icon?: LucideIcon;
  actions?: ReactNode;
  className?: string;
}

export const PageHeader: FC<PageHeaderProps> = ({
  title,
  description,
  icon,
  actions,
  className = '',
}) => {
  return (
    <div className={`flex flex-wrap items-start justify-between gap-4 mb-6 ${className}`}>
      <div className="flex items-center gap-3">
        {icon && createElement(icon, { className: 'h-8 w-8 text-violet-400' })}
        <div>
          <h1 className="text-2xl font-bold text-white">{title}</h1>
          {description && (
            <p className="text-sm text-gray-400 mt-1">{description}</p>
          )}
        </div>
      </div>
      {actions && <div className="flex items-center gap-3">{actions}</div>}
    </div>
  );
};
