import { type KeyboardEvent, type ReactNode, useId, useRef } from 'react';
import { cn } from '../../styles/theme';

export interface HelpTab {
  id: string;
  label: string;
  icon?: ReactNode;
}

interface HelpTabsProps {
  tabs: readonly HelpTab[];
  value: string;
  onChange: (id: string) => void;
  label: string;
  children: ReactNode;
}

/**
 * The Help sections as a tab list that wraps. The shared SettingsTabs scrolls
 * sideways, which is right for Settings' five tabs and wrong for Help's eight:
 * the strip is about 850px inside a drawer of about 455px, so Glossary,
 * Shortcuts and FAQ sit off-screen where nobody finds them (D1,
 * e2e/reachability.spec.ts). Same roles, same keyboard model as SettingsTabs;
 * only the overflow differs. Goes if the shared tabs gain a wrapping mode.
 */
export function HelpTabs({ tabs, value, onChange, label, children }: HelpTabsProps) {
  const prefix = useId();
  const listRef = useRef<HTMLDivElement>(null);
  const tabId = (id: string) => `${prefix}-tab-${id}`;

  const onKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    const current = tabs.findIndex((tab) => tab.id === value);
    const last = tabs.length - 1;
    const target = {
      ArrowRight: current === last ? 0 : current + 1,
      ArrowLeft: current <= 0 ? last : current - 1,
      Home: 0,
      End: last,
    }[event.key];
    const next = target === undefined ? undefined : tabs[target];
    if (!next) return;
    event.preventDefault();
    onChange(next.id);
    listRef.current?.querySelector<HTMLElement>(`[id="${tabId(next.id)}"]`)?.focus();
  };

  return (
    <div>
      <div className="border-b border-hairline">
        <div
          ref={listRef}
          role="tablist"
          aria-label={label}
          aria-orientation="horizontal"
          onKeyDown={onKeyDown}
          className="flex flex-wrap gap-x-1"
        >
          {tabs.map((tab) => {
            const selected = tab.id === value;
            return (
              <button
                key={tab.id}
                id={tabId(tab.id)}
                type="button"
                role="tab"
                aria-selected={selected}
                aria-controls={`${prefix}-panel`}
                tabIndex={selected ? 0 : -1}
                onClick={() => onChange(tab.id)}
                className={cn(
                  '-mb-px inline-flex min-h-11 items-center gap-2 border-b-2 px-3 text-sm font-medium focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-brand-primary',
                  selected
                    ? 'border-brand-primary text-text-primary'
                    : 'border-transparent text-text-muted hover:text-text-primary',
                )}
              >
                {tab.icon ? <span aria-hidden="true">{tab.icon}</span> : null}
                {tab.label}
              </button>
            );
          })}
        </div>
      </div>
      {/* tabIndex 0 (WAI-ARIA tabs pattern): a section with nothing tabbable, like
          Overview, still ends the dialog's tab order on something the focus trap
          can wrap from. Without it Tab leaves the modal drawer. */}
      <div
        id={`${prefix}-panel`}
        role="tabpanel"
        tabIndex={0}
        aria-labelledby={tabId(value)}
        className="pt-4 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-primary"
      >
        {children}
      </div>
    </div>
  );
}
