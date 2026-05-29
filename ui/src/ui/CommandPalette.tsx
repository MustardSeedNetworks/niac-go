/**
 * CommandPalette — keyboard-first command/navigation palette (cmdk).
 *
 * Mount at the AppShell level. Opens on Cmd+K (macOS) / Ctrl+K (others).
 * Populates with:
 *   - All sidebar nav entries (jump to page)
 *   - Common actions (Open Settings, Open Help, Toggle Theme)
 */
import { Command } from 'cmdk';
import { HelpCircle, Moon, Search, Settings as SettingsIcon, Sun } from 'lucide-react';
import { type FC, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import type { SidebarNavGroup } from './Sidebar';

export interface CommandPaletteAction {
  id: string;
  label: string;
  hint?: string;
  icon?: typeof SettingsIcon;
  perform: () => void;
}

export interface CommandPaletteProps {
  groups: SidebarNavGroup[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
  extraActions?: CommandPaletteAction[];
  onOpenSettings?: () => void;
  onOpenHelp?: () => void;
  onToggleTheme?: () => void;
  isDark?: boolean;
}

export const CommandPalette: FC<CommandPaletteProps> = ({
  groups,
  open,
  onOpenChange,
  extraActions = [],
  onOpenSettings,
  onOpenHelp,
  onToggleTheme,
  isDark,
}) => {
  const { t } = useTranslation('common');
  const navigate = useNavigate();
  const [value, setValue] = useState('');

  useEffect(() => {
    const handler = (e: KeyboardEvent): void => {
      if ((e.metaKey || e.ctrlKey) && (e.key === 'k' || e.key === 'K')) {
        e.preventDefault();
        onOpenChange(!open);
      }
    };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, [open, onOpenChange]);

  useEffect(() => {
    if (!open) {
      setValue('');
    }
  }, [open]);

  const handleSelect = (perform: () => void): void => {
    perform();
    onOpenChange(false);
  };

  return (
    <Command.Dialog
      open={open}
      onOpenChange={onOpenChange}
      label={t('commandPalette.dialogLabel')}
      className="fixed inset-0 z-[60] flex items-start justify-center pt-[10vh]"
      shouldFilter={true}
    >
      <button
        type="button"
        className="absolute inset-0 bg-scrim/70 backdrop-blur-sm"
        onClick={() => onOpenChange(false)}
        aria-label={t('commandPalette.closeAriaLabel')}
      />
      <div className="relative mx-4 w-full max-w-xl rounded-2xl border border-surface-border bg-bg-surface/95 shadow-2xl">
        <div className="flex items-center gap-compact border-b border-surface-border px-4 py-row-lg">
          <Search className="h-4 w-4 text-text-muted" aria-hidden="true" />
          <Command.Input
            autoFocus={true}
            value={value}
            onValueChange={setValue}
            placeholder={t('commandPalette.inputPlaceholder')}
            className="flex-1 bg-transparent text-sm text-text-primary placeholder:text-text-muted focus:outline-none"
          />
          <kbd className="hidden sm:inline-flex items-center rounded border border-surface-border px-1.5 py-0.5 text-[11px] text-text-muted">
            {t('commandPalette.escKey')}
          </kbd>
        </div>
        <Command.List className="max-h-[60vh] overflow-y-auto px-cell py-row text-sm">
          <Command.Empty className="px-3 py-6 text-center text-text-muted">
            {t('commandPalette.noMatches')}
          </Command.Empty>

          {groups.map((group) => (
            <Command.Group
              key={group.label}
              heading={group.label}
              className="px-1 py-compact text-xs uppercase tracking-wider text-text-muted"
            >
              {group.items.map((item) => {
                const Icon = item.icon;
                return (
                  <Command.Item
                    key={item.path}
                    value={`${group.label} ${item.label} ${item.path}`}
                    onSelect={() => handleSelect(() => navigate(item.path))}
                    className="flex cursor-pointer items-center gap-default rounded-md px-3 py-row text-text-primary aria-selected:bg-surface-hover"
                  >
                    <Icon className="h-4 w-4 text-text-muted" aria-hidden="true" />
                    <span className="flex-1">{item.label}</span>
                    <span className="text-xs text-text-muted">{item.path}</span>
                  </Command.Item>
                );
              })}
            </Command.Group>
          ))}

          <Command.Group
            heading={t('commandPalette.actionsHeading')}
            className="px-1 py-compact text-xs uppercase tracking-wider text-text-muted"
          >
            {onOpenSettings && (
              <Command.Item
                value="open settings"
                onSelect={() => handleSelect(onOpenSettings)}
                className="flex cursor-pointer items-center gap-default rounded-md px-3 py-row text-text-primary aria-selected:bg-surface-hover"
              >
                <SettingsIcon className="h-4 w-4 text-text-muted" aria-hidden="true" />
                <span>{t('commandPalette.openSettings')}</span>
              </Command.Item>
            )}
            {onOpenHelp && (
              <Command.Item
                value="open help"
                onSelect={() => handleSelect(onOpenHelp)}
                className="flex cursor-pointer items-center gap-default rounded-md px-3 py-row text-text-primary aria-selected:bg-surface-hover"
              >
                <HelpCircle className="h-4 w-4 text-text-muted" aria-hidden="true" />
                <span>{t('commandPalette.openHelp')}</span>
              </Command.Item>
            )}
            {onToggleTheme && (
              <Command.Item
                value="toggle theme dark light mode"
                onSelect={() => handleSelect(onToggleTheme)}
                className="flex cursor-pointer items-center gap-default rounded-md px-3 py-row text-text-primary aria-selected:bg-surface-hover"
              >
                {isDark ? (
                  <Sun className="h-4 w-4 text-text-muted" aria-hidden="true" />
                ) : (
                  <Moon className="h-4 w-4 text-text-muted" aria-hidden="true" />
                )}
                <span>
                  {isDark
                    ? t('accessibility.switchToLightMode')
                    : t('accessibility.switchToDarkMode')}
                </span>
              </Command.Item>
            )}
            {extraActions.map((action) => {
              const Icon = action.icon;
              return (
                <Command.Item
                  key={action.id}
                  value={`${action.label} ${action.hint ?? ''}`}
                  onSelect={() => handleSelect(action.perform)}
                  className="flex cursor-pointer items-center gap-default rounded-md px-3 py-row text-text-primary aria-selected:bg-surface-hover"
                >
                  {Icon ? (
                    <Icon className="h-4 w-4 text-text-muted" aria-hidden="true" />
                  ) : (
                    <span className="h-4 w-4" />
                  )}
                  <span className="flex-1">{action.label}</span>
                  {action.hint && <span className="text-xs text-text-muted">{action.hint}</span>}
                </Command.Item>
              );
            })}
          </Command.Group>
        </Command.List>
      </div>
    </Command.Dialog>
  );
};
