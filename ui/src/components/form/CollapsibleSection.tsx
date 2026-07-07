import { ChevronDown, ChevronRight } from 'lucide-react';
import type { FC, ReactNode } from 'react';
import { iconSizes } from '../../constants/sizes';
import { Card, CardContent } from '../../ui/Card';
import { Tag } from '../../ui/Tag';

export interface CollapsibleSectionProps {
  title: string;
  children: ReactNode;
  isExpanded: boolean;
  onToggle: () => void;
  required?: boolean;
  enabled?: boolean;
  onEnableChange?: (enabled: boolean) => void;
  /** Anchor id for deep-linking to this section (e.g. `#snmp` from the Running Devices walk browser). */
  id?: string;
}

export const CollapsibleSection: FC<CollapsibleSectionProps> = ({
  title,
  children,
  isExpanded,
  onToggle,
  required,
  enabled,
  onEnableChange,
  id,
}) => (
  <Card id={id} className="border-surface-border bg-bg-surface/70 overflow-hidden">
    <div className="flex-between px-6 py-4 hover:bg-surface-hover transition-colors">
      <div className="flex items-center gap-default">
        <button
          type="button"
          onClick={onToggle}
          aria-expanded={isExpanded}
          aria-label={`${title} section, ${isExpanded ? 'expanded' : 'collapsed'}`}
          className="flex items-center gap-default text-left focus:outline-none focus:ring-2 focus:ring-inset focus:ring-brand-primary rounded-lg px-1 -ml-1"
        >
          {isExpanded ? (
            <ChevronDown className={`${iconSizes.md} text-text-muted`} />
          ) : (
            <ChevronRight className={`${iconSizes.md} text-text-muted`} />
          )}
          <h3 className="font-semibold text-text-primary">{title}</h3>
          {required && (
            <Tag colorScheme="red" className="text-xs">
              Required
            </Tag>
          )}
        </button>
        {onEnableChange && (
          <div className="ml-inline">
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                checked={enabled ?? false}
                onChange={(e) => onEnableChange(e.target.checked)}
                className="sr-only peer"
              />
              <div className="w-9 h-5 bg-bg-elevated rounded-full peer peer-checked:bg-brand-primary peer-focus:ring-2 peer-focus:ring-brand-primary transition-colors">
                <div
                  className={`absolute top-0.5 left-0.5 w-4 h-4 bg-knob rounded-full transition-transform ${enabled ? 'translate-x-4' : ''}`}
                />
              </div>
            </label>
          </div>
        )}
      </div>
      {enabled !== undefined && (
        <Tag colorScheme={enabled ? 'green' : 'gray'} className="text-xs">
          {enabled ? 'Enabled' : 'Disabled'}
        </Tag>
      )}
    </div>
    {isExpanded && <CardContent className="pt-0 pb-6 px-6">{children}</CardContent>}
  </Card>
);
