import { type ReactNode, useEffect, useRef } from 'react';

/**
 * Two fixes the shared DataTable needs and cannot take from a product.
 *
 * 1. Its row-actions header is `<span class="sr-only">`, which is
 *    `position: absolute`. Nothing between it and the page is positioned, so it
 *    escapes the table's scroll frame and widens the document (765px of scroll
 *    at 390px). A positioned, clipping ancestor becomes its containing block.
 * 2. Its scroll frame scrolls sideways but is not focusable, so a table with no
 *    interactive cell (walk issues, neighbors) cannot be scrolled from the
 *    keyboard (axe `scrollable-region-focusable`, WCAG 2.1.1). The frame is made
 *    a labelled, focusable region after every render, since the frame only
 *    exists while there are rows.
 *
 * Wrap every DataTable. Goes when the shared DataTable positions its header
 * cell and makes its own frame focusable.
 */
export function ContainedTable({ label, children }: { label: string; children: ReactNode }) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const scroller = ref.current?.querySelector<HTMLElement>(
      '[data-phone-width-exempt="table-scroll"]',
    );
    scroller?.setAttribute('tabindex', '0');
    scroller?.setAttribute('role', 'region');
    scroller?.setAttribute('aria-label', label);
  });
  return (
    <div ref={ref} className="relative overflow-hidden rounded-lg">
      {children}
    </div>
  );
}
