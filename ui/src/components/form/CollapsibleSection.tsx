import { ChevronDown, ChevronRight } from 'lucide-react';
import type { FC, ReactNode } from 'react';
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
}

export const CollapsibleSection: FC<CollapsibleSectionProps> = ({
  title,
  children,
  isExpanded,
  onToggle,
  required,
  enabled,
  onEnableChange,
}) => (
  <Card className="border-white/5 bg-gray-900/70 overflow-hidden">
    <div className="flex items-center justify-between px-6 py-4 hover:bg-white/5 transition-colors">
      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={onToggle}
          aria-expanded={isExpanded}
          aria-label={`${title} section, ${isExpanded ? 'expanded' : 'collapsed'}`}
          className="flex items-center gap-3 text-left focus:outline-none focus:ring-2 focus:ring-inset focus:ring-violet-500 rounded-lg px-1 -ml-1"
        >
          {isExpanded ? (
            <ChevronDown className="h-4 w-4 text-gray-400" />
          ) : (
            <ChevronRight className="h-4 w-4 text-gray-400" />
          )}
          <h3 className="font-semibold text-white">{title}</h3>
          {required && (
            <Tag colorScheme="red" className="text-xs">
              Required
            </Tag>
          )}
        </button>
        {onEnableChange && (
          <div className="ml-2">
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                checked={enabled ?? false}
                onChange={(e) => onEnableChange(e.target.checked)}
                className="sr-only peer"
              />
              <div className="w-9 h-5 bg-gray-700 rounded-full peer peer-checked:bg-violet-600 peer-focus:ring-2 peer-focus:ring-violet-500 transition-colors">
                <div
                  className={`absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full transition-transform ${enabled ? 'translate-x-4' : ''}`}
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
