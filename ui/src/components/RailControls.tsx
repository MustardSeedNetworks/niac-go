import { Moon, Sun } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import type { ConnectionState } from '../hooks/useConnectionStatus';
import { ConnectionStatus } from '../ui/ConnectionStatus';
import { Tooltip } from '../ui/Tooltip';
import { SessionSwitcher } from './SessionSwitcher';

interface RailControlsProps {
  collapsed: boolean;
  status: ConnectionState;
  isDark: boolean;
  toggleTheme: () => void;
}

export function RailControls({ collapsed, status, isDark, toggleTheme }: RailControlsProps) {
  const { t } = useTranslation('common');
  const label = t(isDark ? 'accessibility.switchToLightMode' : 'accessibility.switchToDarkMode');
  return (
    <div data-testid="rail-controls" className="mb-heading stack-xs">
      <div
        className={
          collapsed ? 'flex flex-col items-center' : 'flex items-center justify-between gap-compact'
        }
      >
        <ConnectionStatus status={status} compact={collapsed} />
        <Tooltip text={label}>
          <button
            type="button"
            onClick={toggleTheme}
            data-testid="theme-toggle"
            className="flex-center min-h-11 w-full max-w-11 rounded-lg border border-surface-border text-text-secondary hover:text-text-primary hover:bg-surface-hover focus-visible:outline-2 focus-visible:outline-brand-accent"
            aria-label={label}
          >
            {isDark ? (
              <Sun className="h-5 w-5" aria-hidden="true" />
            ) : (
              <Moon className="h-5 w-5" aria-hidden="true" />
            )}
          </button>
        </Tooltip>
      </div>
      {!collapsed && <SessionSwitcher />}
    </div>
  );
}
