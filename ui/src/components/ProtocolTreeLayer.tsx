import { ChevronDown, ChevronRight } from 'lucide-react';
import { type FC, memo, useCallback } from 'react';
import { iconSizes } from '../constants/sizes';
import type { ProtocolField, ProtocolLayer } from '../utils/protocol-layers';

interface ProtocolTreeLayerProps {
  layer: ProtocolLayer;
  isExpanded: boolean;
  onToggle: (name: string) => void;
  onFieldSelect?: (byteStart: number, byteEnd: number) => void;
}

/**
 * Single expandable protocol layer in the dissection tree.
 * Clicking a field with byte range info fires onFieldSelect to highlight hex bytes.
 * Expanded state is owned by ProtocolTree so expand-all/collapse-all can drive every layer at once.
 */
export const ProtocolTreeLayer: FC<ProtocolTreeLayerProps> = memo(
  ({ layer, isExpanded, onToggle, onFieldSelect }) => {
    const toggleExpanded = useCallback(() => {
      onToggle(layer.name);
    }, [layer.name, onToggle]);

    const handleFieldClick = useCallback(
      (field: ProtocolField) => {
        if (field.byteStart !== undefined && field.byteEnd !== undefined && onFieldSelect) {
          onFieldSelect(field.byteStart, field.byteEnd);
        }
      },
      [onFieldSelect],
    );

    return (
      <div className="text-sm">
        <button
          type="button"
          onClick={toggleExpanded}
          className="flex items-center gap-tight w-full text-left py-0.5 hover:bg-surface-hover rounded px-1 -ml-1"
        >
          {isExpanded ? (
            <ChevronDown className={`${iconSizes.sm} text-text-muted flex-shrink-0`} />
          ) : (
            <ChevronRight className={`${iconSizes.sm} text-text-muted flex-shrink-0`} />
          )}
          <span className="text-brand-accent font-medium">{layer.name}</span>
        </button>

        {isExpanded && (
          <div className="ml-5 border-l border-surface-border pl-2">
            {layer.fields.map((field) => {
              const hasRange = field.byteStart !== undefined && field.byteEnd !== undefined;
              return (
                <button
                  key={`${field.name}-${field.value}`}
                  type="button"
                  onClick={() => handleFieldClick(field)}
                  className={`block w-full text-left py-0.5 px-1 -ml-1 rounded text-xs ${
                    hasRange ? 'hover:bg-status-warning/30 cursor-pointer' : 'cursor-default'
                  }`}
                  disabled={!hasRange}
                >
                  <span className="text-text-muted">{field.name}: </span>
                  <span className="text-text-primary font-mono">{field.value}</span>
                </button>
              );
            })}
          </div>
        )}
      </div>
    );
  },
);

ProtocolTreeLayer.displayName = 'ProtocolTreeLayer';
