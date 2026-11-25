import type { ReactNode } from 'react';

/**
 * Contains the shared DataTable's screen-reader-only header cell.
 *
 * The shared DataTable names its row-actions column with
 * `<span class="sr-only">`, which is `position: absolute`. Nothing between it
 * and the page is positioned, so it escapes the table's scroll frame and
 * widens the document (765px of scroll at 390px). A positioned, clipping
 * ancestor is its containing block instead. Used wherever `rowActions` is.
 * Goes when the shared DataTable positions its own header cell.
 */
export function ContainedTable({ children }: { children: ReactNode }) {
  return <div className="relative overflow-hidden rounded-lg">{children}</div>;
}
