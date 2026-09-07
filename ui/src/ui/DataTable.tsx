import { type CSSProperties, type KeyboardEvent, type ReactNode, useMemo, useState } from 'react';
import { useVirtualScroll } from '../hooks/useVirtualScroll';

/**
 * DataTable — shared table primitive extracted from DeviceTable (Phase 7).
 *
 * Renders a typed list of rows as a semantic `<table>` with a header,
 * optional client-side sort, optional row selection, empty/loading
 * states, and optional virtual scrolling above a row threshold. Column
 * definitions own their own cell rendering so callers keep full control
 * over formatting (badges, links, conditional cells) while the table
 * chrome (header row, empty state, virtualization) is shared.
 *
 * Not every list view fits this shape — see Phase 7 PR summary for
 * which consumers were migrated and which were deliberately left as-is.
 * `PacketList` (a stack of button cards), `LogViewer` (expandable stream)
 * and `device-list/DeviceTableView` (a CSS grid) stay off it by design.
 */
export interface DataTableColumn<T> {
  /** Stable identifier, also used as the React key for the header cell. */
  key: string;
  header: ReactNode;
  cell: (row: T) => ReactNode;
  headerClassName?: string;
  cellClassName?: string;
  align?: 'left' | 'right';
  /** When set, the column header becomes clickable and sorts by this value. */
  sortAccessor?: (row: T) => string | number;
  'data-testid'?: string;
}

export interface DataTableSelection<T> {
  selectedKeys: ReadonlySet<string>;
  onToggleRow: (key: string) => void;
  onToggleAll: () => void;
  selectAllAriaLabel: string;
  selectRowAriaLabel: (row: T) => string;
}

export interface DataTableSort {
  key: string;
  direction: SortDirection;
}

export interface DataTableVirtualization {
  itemHeight: number;
  containerHeight: number;
  overscan?: number;
  /** Row-count threshold above which virtualization kicks in. */
  threshold: number;
  renderStatus: (visibleCount: number, totalCount: number) => ReactNode;
}

interface DataTableProps<T> {
  rows: T[];
  columns: DataTableColumn<T>[];
  getRowKey: (row: T) => string;
  emptyMessage: ReactNode;
  loading?: boolean;
  loadingMessage?: ReactNode;
  selection?: DataTableSelection<T>;
  virtualization?: DataTableVirtualization;
  rowClassName?: (row: T) => string;
  /** Per-row inline style, for callers whose row colour is data-driven. */
  rowStyle?: (row: T) => CSSProperties | undefined;
  rowTestId?: (row: T) => string;
  /**
   * Makes rows activatable. Rows become focusable and respond to Enter and
   * Space as well as click, so a click-to-drill-down table is reachable
   * from the keyboard.
   */
  onRowClick?: (row: T) => void;
  /** Sort applied before the user clicks any header. */
  defaultSort?: DataTableSort;
  /** Pins the header while the body scrolls. Needs a bounded container. */
  stickyHeader?: boolean;
  /** Extra classes on the scroll container, e.g. a height bound. */
  containerClassName?: string;
  'data-testid'?: string;
}

type SortDirection = 'asc' | 'desc';

function compareSortValues(a: string | number, b: string | number): number {
  if (typeof a === 'number' && typeof b === 'number') {
    return a - b;
  }
  return String(a).localeCompare(String(b));
}

export function DataTable<T>({
  rows,
  columns,
  getRowKey,
  emptyMessage,
  loading = false,
  loadingMessage,
  selection,
  virtualization,
  rowClassName,
  rowStyle,
  rowTestId,
  onRowClick,
  defaultSort,
  stickyHeader = false,
  containerClassName,
  'data-testid': testId,
}: DataTableProps<T>) {
  const [sort, setSort] = useState<DataTableSort | null>(defaultSort ?? null);

  const sortedRows = useMemo(() => {
    if (!sort) {
      return rows;
    }
    const column = columns.find((c) => c.key === sort.key);
    if (!column?.sortAccessor) {
      return rows;
    }
    const { sortAccessor } = column;
    const sign = sort.direction === 'asc' ? 1 : -1;
    return [...rows].sort((a, b) => sign * compareSortValues(sortAccessor(a), sortAccessor(b)));
  }, [rows, sort, columns]);

  const toggleSort = (columnKey: string) => {
    setSort((prev) => {
      if (!prev || prev.key !== columnKey) {
        return { key: columnKey, direction: 'asc' };
      }
      return prev.direction === 'asc' ? { key: columnKey, direction: 'desc' } : null;
    });
  };

  const useVirtualization = virtualization != null && rows.length >= virtualization.threshold;
  // Hook must run unconditionally; virtualization is disabled below the
  // threshold by simply not consuming its output.
  const virtualScroll = useVirtualScroll(sortedRows, {
    itemHeight: virtualization?.itemHeight ?? 1,
    containerHeight: virtualization?.containerHeight ?? 1,
    overscan: virtualization?.overscan,
  });

  if (loading) {
    return <>{loadingMessage}</>;
  }

  if (rows.length === 0) {
    return <>{emptyMessage}</>;
  }

  const allSelected = selection
    ? selection.selectedKeys.size === rows.length && rows.length > 0
    : false;

  const renderHeader = () => (
    <thead
      className={`text-xs uppercase tracking-wide text-text-muted ${
        stickyHeader ? 'sticky top-0 z-10 bg-bg-surface' : 'bg-bg-surface/60'
      }`}
    >
      <tr>
        {selection && (
          <th className="px-4 py-row-lg text-left">
            <input
              type="checkbox"
              checked={allSelected}
              onChange={selection.onToggleAll}
              className="h-4 w-4 rounded border-border-muted bg-bg-elevated text-brand-primary focus:ring-brand-primary"
              aria-label={selection.selectAllAriaLabel}
            />
          </th>
        )}
        {columns.map((column) => (
          <th
            key={column.key}
            className={`px-4 py-row-lg ${column.align === 'right' ? 'text-right' : 'text-left'} ${column.headerClassName ?? ''}`}
          >
            {column.sortAccessor ? (
              <button
                type="button"
                onClick={() => toggleSort(column.key)}
                className="inline-flex items-center gap-1 hover:text-text-secondary"
              >
                {column.header}
                {sort?.key === column.key && (sort.direction === 'asc' ? '▲' : '▼')}
              </button>
            ) : (
              column.header
            )}
          </th>
        ))}
      </tr>
    </thead>
  );

  const activateRow = (row: T) => (event: KeyboardEvent<HTMLTableRowElement>) => {
    if (event.key !== 'Enter' && event.key !== ' ') {
      return;
    }
    event.preventDefault();
    onRowClick?.(row);
  };

  const renderRow = (row: T) => {
    const key = getRowKey(row);
    return (
      <tr
        key={key}
        data-testid={rowTestId?.(row)}
        className={`${rowClassName?.(row) ?? ''} ${onRowClick ? 'cursor-pointer' : ''}`.trim()}
        style={rowStyle?.(row)}
        onClick={onRowClick ? () => onRowClick(row) : undefined}
        onKeyDown={onRowClick ? activateRow(row) : undefined}
        tabIndex={onRowClick ? 0 : undefined}
      >
        {selection && (
          <td className="px-4 py-row-lg">
            <input
              type="checkbox"
              checked={selection.selectedKeys.has(key)}
              onChange={() => selection.onToggleRow(key)}
              className="h-4 w-4 rounded border-border-muted bg-bg-elevated text-brand-primary focus:ring-brand-primary"
              aria-label={selection.selectRowAriaLabel(row)}
            />
          </td>
        )}
        {columns.map((column) => (
          <td
            key={column.key}
            data-testid={column['data-testid']}
            className={`px-4 py-row-lg ${column.align === 'right' ? 'text-right' : ''} ${column.cellClassName ?? ''}`}
          >
            {column.cell(row)}
          </td>
        ))}
      </tr>
    );
  };

  if (!useVirtualization) {
    return (
      <div
        className={`min-w-0 overflow-x-auto rounded-xl border border-surface-border ${containerClassName ?? ''}`.trimEnd()}
        data-testid={testId}
      >
        <table className="min-w-full divide-y divide-knob/10 text-sm">
          {renderHeader()}
          <tbody className="divide-y divide-knob/5 text-text-secondary">
            {sortedRows.map((row) => renderRow(row))}
          </tbody>
        </table>
      </div>
    );
  }

  return (
    <div className="rounded-xl border border-surface-border" data-testid={testId}>
      <div className="bg-bg-surface/60 px-4 py-row text-xs text-text-muted">
        {virtualization.renderStatus(virtualScroll.visibleItems.length, rows.length)}
      </div>
      <div {...virtualScroll.containerProps} className="overflow-auto">
        <div {...virtualScroll.spacerProps}>
          <div {...virtualScroll.contentProps}>
            <table className="min-w-full divide-y divide-knob/10 text-sm">
              {renderHeader()}
              <tbody className="divide-y divide-knob/5 text-text-secondary">
                {virtualScroll.visibleItems.map(({ item }) => renderRow(item))}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
}
