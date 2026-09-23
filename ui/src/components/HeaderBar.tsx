/**
 * HeaderBar — NIAC's slim three-slot header.
 *
 * Per the canonical convention (stem/ui/SHELL.md): per-product, not synced.
 * NIAC has no profile system and no interface picker, so the right slot
 * carries only the theme toggle. The left slot carries the scenario
 * switcher: a daemon runs several scenarios at once and every runtime read
 * is scoped to one, so which one this browser reads belongs in the chrome
 * rather than on a page the operator has to navigate to. Settings + Help
 * live in the sidebar footer; there is no logout (NIAC is single-user).
 *
 *   ┌──────────────────────────────────────────────────────────────┐
 *   │ [logo] NIAC  [ConnectionStatus][SessionSwitcher] … [theme]   │
 *   └──────────────────────────────────────────────────────────────┘
 */
import { Moon, Network, Sun } from 'lucide-react';
import type { FC, ReactElement } from 'react';
import { useTranslation } from 'react-i18next';
import type { ConnectionState } from '../hooks/useConnectionStatus';
import { ConnectionStatus } from '../ui/ConnectionStatus';
import { Tooltip } from '../ui/Tooltip';
import { SessionSwitcher } from './SessionSwitcher';

interface HeaderBarProps {
  status: ConnectionState;
  isDark: boolean;
  toggleTheme: () => void;
}

export const HeaderBar: FC<HeaderBarProps> = ({ status, isDark, toggleTheme }): ReactElement => {
  const { t } = useTranslation('common');
  const themeToggleLabel = isDark
    ? t('accessibility.switchToLightMode')
    : t('accessibility.switchToDarkMode');

  return (
    <>
      {/* Left slot: logo + product name + connection and scenario state */}
      <div className="flex items-center gap-default min-w-0">
        <div className="h-8 w-8 rounded-lg bg-gradient-to-br from-brand-primary to-brand-accent flex-center shrink-0">
          <Network className="h-5 w-5 text-text-inverse" aria-hidden="true" />
        </div>
        <span className="font-display font-bold text-text-primary truncate">NIAC</span>
        <ConnectionStatus status={status} />
        <SessionSwitcher />
      </div>

      {/* Right slot: theme toggle only (no profiles / interfaces in NIAC).
       * Settings + Help live in the sidebar footer; see stem/ui/SHELL.md. */}
      <div className="flex items-center gap-tight">
        <Tooltip text={themeToggleLabel}>
          <button
            type="button"
            onClick={toggleTheme}
            className="pad-xs rounded-lg text-text-secondary hover:text-text-primary hover:bg-surface-hover"
            aria-label={themeToggleLabel}
          >
            {isDark ? (
              <Sun className="h-5 w-5" aria-hidden="true" />
            ) : (
              <Moon className="h-5 w-5" aria-hidden="true" />
            )}
          </button>
        </Tooltip>
      </div>
    </>
  );
};
