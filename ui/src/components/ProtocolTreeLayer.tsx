import { ChevronDown, ChevronRight } from 'lucide-react';
import { type FC, memo, useCallback, useState } from 'react';
import { iconSizes } from '../constants/sizes';
import type { ProtocolField, ProtocolLayer } from '../utils/protocol-layers';

interface ProtocolTreeLayerProps {
  layer: ProtocolLayer;
  onFieldSelect?: (byteStart: number, byteEnd: number) => void;
}

/**
 * Single expandable protocol layer in the dissection tree.
 * Clicking a field with byte range info fires onFieldSelect to highlight hex bytes.
 */
export const ProtocolTreeLayer: FC<ProtocolTreeLayerProps> = memo(({ layer, onFieldSelect }) => {
  const [isExpanded, setIsExpanded] = useState(layer.expanded ?? true);

  const toggleExpanded = useCallback(() => {
    setIsExpanded((prev) => !prev);
  }, []);

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
        className="flex items-center gap-1 w-full text-left py-0.5 hover:bg-white/5 rounded px-1 -ml-1"
      >
        {isExpanded ? (
          <ChevronDown className={`${iconSizes.sm} text-gray-500 flex-shrink-0`} />
        ) : (
          <ChevronRight className={`${iconSizes.sm} text-gray-500 flex-shrink-0`} />
        )}
        <span className="text-violet-300 font-medium">{layer.name}</span>
      </button>

      {isExpanded && (
        <div className="ml-5 border-l border-white/5 pl-2">
          {layer.fields.map((field) => {
            const hasRange = field.byteStart !== undefined && field.byteEnd !== undefined;
            return (
              <button
                key={`${field.name}-${field.value}`}
                type="button"
                onClick={() => handleFieldClick(field)}
                className={`block w-full text-left py-0.5 px-1 -ml-1 rounded text-xs ${
                  hasRange ? 'hover:bg-yellow-900/30 cursor-pointer' : 'cursor-default'
                }`}
                disabled={!hasRange}
              >
                <span className="text-gray-400">{field.name}: </span>
                <span className="text-white font-mono">{field.value}</span>
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
});

ProtocolTreeLayer.displayName = 'ProtocolTreeLayer';
