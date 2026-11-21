import { type ReactNode, useEffect, useState } from 'react';
import { useVirtualScroll } from '../../hooks/useVirtualScroll';

/** Room the page keeps for its header, filters and padding around the list. */
const PAGE_CHROME_PX = 320;
const MIN_LIST_HEIGHT_PX = 320;
const MAX_LIST_HEIGHT_PX = 720;

function viewportListHeight(): number {
  return Math.min(
    MAX_LIST_HEIGHT_PX,
    Math.max(MIN_LIST_HEIGHT_PX, window.innerHeight - PAGE_CHROME_PX),
  );
}

function useListHeightBudget(): number {
  const [height, setHeight] = useState(viewportListHeight);
  useEffect(() => {
    const onResize = () => setHeight(viewportListHeight());
    window.addEventListener('resize', onResize);
    return () => window.removeEventListener('resize', onResize);
  }, []);
  return height;
}

interface VirtualListProps<T> {
  items: T[];
  /** Every item renders at exactly this height; windowing depends on it. */
  itemHeight: number;
  getKey: (item: T) => string;
  renderItem: (item: T) => ReactNode;
  ariaLabel: string;
  'data-testid'?: string;
}

/**
 * Scrolls a long list inside a bounded frame and mounts only the rows near the
 * viewport. A scenario pack runs up to 543 devices; mounting each one made the
 * list page the slowest surface in the UI.
 */
export function VirtualList<T>({
  items,
  itemHeight,
  getKey,
  renderItem,
  ariaLabel,
  'data-testid': testId,
}: VirtualListProps<T>) {
  const budget = useListHeightBudget();
  const { visibleItems, containerProps, spacerProps, contentProps } = useVirtualScroll(items, {
    itemHeight,
    containerHeight: Math.min(budget, items.length * itemHeight),
    overscan: 4,
  });

  return (
    <div {...containerProps} data-testid={testId}>
      <div {...spacerProps}>
        <ul {...contentProps} aria-label={ariaLabel} className="m-0 list-none p-0">
          {visibleItems.map(({ item, index }) => (
            <li
              key={getKey(item)}
              aria-posinset={index + 1}
              aria-setsize={items.length}
              style={{ height: itemHeight }}
            >
              {renderItem(item)}
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}
