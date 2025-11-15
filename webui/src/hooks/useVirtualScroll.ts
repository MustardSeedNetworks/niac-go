import { useState, useEffect, useRef, useMemo } from 'react';

interface VirtualScrollOptions {
  itemHeight: number;
  containerHeight: number;
  overscan?: number;
}

/**
 * FEATURE #126: Virtual scrolling hook for large lists
 *
 * Renders only visible items plus overscan buffer for smooth scrolling.
 * Reduces DOM nodes and improves performance with 1000+ items.
 *
 * @param itemCount - Total number of items in the list
 * @param options - Configuration for virtual scrolling
 * @returns Virtual scroll state and container props
 */
export function useVirtualScroll<T>(
  items: T[],
  options: VirtualScrollOptions
) {
  const { itemHeight, containerHeight, overscan = 3 } = options;
  const [scrollTop, setScrollTop] = useState(0);
  const containerRef = useRef<HTMLDivElement>(null);

  const { visibleItems, offsetY, totalHeight } = useMemo(() => {
    const itemCount = items.length;
    const visibleCount = Math.ceil(containerHeight / itemHeight);
    const startIndex = Math.max(0, Math.floor(scrollTop / itemHeight) - overscan);
    const endIndex = Math.min(itemCount, startIndex + visibleCount + 2 * overscan);

    return {
      visibleItems: items.slice(startIndex, endIndex).map((item, index) => ({
        item,
        index: startIndex + index,
      })),
      offsetY: startIndex * itemHeight,
      totalHeight: itemCount * itemHeight,
    };
  }, [items, scrollTop, itemHeight, containerHeight, overscan]);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const handleScroll = () => {
      setScrollTop(container.scrollTop);
    };

    container.addEventListener('scroll', handleScroll, { passive: true });
    return () => container.removeEventListener('scroll', handleScroll);
  }, []);

  return {
    containerRef,
    visibleItems,
    offsetY,
    totalHeight,
    containerProps: {
      ref: containerRef,
      style: {
        height: `${containerHeight}px`,
        overflow: 'auto',
        position: 'relative' as const,
      },
    },
    spacerProps: {
      style: {
        height: `${totalHeight}px`,
        position: 'relative' as const,
      },
    },
    contentProps: {
      style: {
        transform: `translateY(${offsetY}px)`,
        position: 'absolute' as const,
        top: 0,
        left: 0,
        right: 0,
      },
    },
  };
}
