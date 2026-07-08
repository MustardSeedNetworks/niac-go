/**
 * DataTable.test.tsx — the shared table primitive extracted in Phase 7.
 *
 * Covers the behaviors the primitive owns independently of any consumer:
 * empty/loading states, testid passthrough, client-side sort toggling,
 * and row selection (select-one / select-all).
 */
import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { DataTable, type DataTableColumn } from './DataTable';

interface Row {
  id: string;
  name: string;
  count: number;
}

const rows: Row[] = [
  { id: 'a', name: 'Bravo', count: 2 },
  { id: 'b', name: 'Alpha', count: 5 },
];

const baseColumns: DataTableColumn<Row>[] = [
  { key: 'name', header: 'Name', cell: (r) => r.name, sortAccessor: (r) => r.name },
  { key: 'count', header: 'Count', cell: (r) => r.count, sortAccessor: (r) => r.count },
];

describe('DataTable', () => {
  it('renders the empty message when there are no rows', () => {
    render(
      <DataTable
        rows={[]}
        columns={baseColumns}
        getRowKey={(r) => r.id}
        emptyMessage={<div>Nothing here</div>}
      />,
    );

    expect(screen.getByText('Nothing here')).toBeInTheDocument();
    expect(screen.queryByRole('table')).not.toBeInTheDocument();
  });

  it('renders the loading message instead of rows or the empty state while loading', () => {
    render(
      <DataTable
        rows={[]}
        columns={baseColumns}
        getRowKey={(r) => r.id}
        emptyMessage={<div>Nothing here</div>}
        loading
        loadingMessage={<div>Loading…</div>}
      />,
    );

    expect(screen.getByText('Loading…')).toBeInTheDocument();
    expect(screen.queryByText('Nothing here')).not.toBeInTheDocument();
  });

  it('renders one row per item and passes through column data-testid', () => {
    const columns: DataTableColumn<Row>[] = [
      { ...baseColumns[0], 'data-testid': 'name-cell' },
      baseColumns[1],
    ];
    render(
      <DataTable
        rows={rows}
        columns={columns}
        getRowKey={(r) => r.id}
        emptyMessage={<div>Nothing here</div>}
        data-testid="my-table"
      />,
    );

    expect(screen.getByTestId('my-table')).toBeInTheDocument();
    expect(screen.getAllByTestId('name-cell')).toHaveLength(2);
    expect(screen.getByText('Bravo')).toBeInTheDocument();
    expect(screen.getByText('Alpha')).toBeInTheDocument();
  });

  it('sorts ascending then descending then back to insertion order on repeated header clicks', () => {
    render(
      <DataTable
        rows={rows}
        columns={baseColumns}
        getRowKey={(r) => r.id}
        emptyMessage={<div>Nothing here</div>}
      />,
    );

    const nameHeaderButton = screen.getByRole('button', { name: 'Name' });
    const cellTexts = () =>
      screen
        .getAllByRole('row')
        .slice(1)
        .map((row) => row.textContent);

    // Insertion order: Bravo, Alpha
    expect(cellTexts()[0]).toContain('Bravo');

    fireEvent.click(nameHeaderButton);
    expect(cellTexts()[0]).toContain('Alpha');

    fireEvent.click(nameHeaderButton);
    expect(cellTexts()[0]).toContain('Bravo');

    fireEvent.click(nameHeaderButton);
    expect(cellTexts()[0]).toContain('Bravo');
  });

  it('supports row and select-all selection', () => {
    const onToggleRow = vi.fn();
    const onToggleAll = vi.fn();
    render(
      <DataTable
        rows={rows}
        columns={baseColumns}
        getRowKey={(r) => r.id}
        emptyMessage={<div>Nothing here</div>}
        selection={{
          selectedKeys: new Set(['a']),
          onToggleRow,
          onToggleAll,
          selectAllAriaLabel: 'Select all',
          selectRowAriaLabel: (r) => `Select ${r.name}`,
        }}
      />,
    );

    const selectAll = screen.getByRole('checkbox', { name: 'Select all' });
    expect(selectAll).not.toBeChecked();

    const rowCheckbox = screen.getByRole('checkbox', { name: 'Select Bravo' });
    expect(rowCheckbox).toBeChecked();

    fireEvent.click(rowCheckbox);
    expect(onToggleRow).toHaveBeenCalledWith('a');

    fireEvent.click(selectAll);
    expect(onToggleAll).toHaveBeenCalledTimes(1);
  });

  it('marks select-all checked only when every row is selected', () => {
    render(
      <DataTable
        rows={rows}
        columns={baseColumns}
        getRowKey={(r) => r.id}
        emptyMessage={<div>Nothing here</div>}
        selection={{
          selectedKeys: new Set(['a', 'b']),
          onToggleRow: vi.fn(),
          onToggleAll: vi.fn(),
          selectAllAriaLabel: 'Select all',
          selectRowAriaLabel: (r) => `Select ${r.name}`,
        }}
      />,
    );

    expect(screen.getByRole('checkbox', { name: 'Select all' })).toBeChecked();
  });

  it('virtualizes only once the row count reaches the configured threshold', () => {
    const manyRows: Row[] = Array.from({ length: 3 }, (_, i) => ({
      id: `r${i}`,
      name: `Row ${i}`,
      count: i,
    }));
    const renderStatus = vi.fn((visible: number, total: number) => `${visible}/${total}`);

    const { rerender } = render(
      <DataTable
        rows={manyRows}
        columns={baseColumns}
        getRowKey={(r) => r.id}
        emptyMessage={<div>Nothing here</div>}
        virtualization={{
          itemHeight: 10,
          containerHeight: 50,
          threshold: 100,
          renderStatus,
        }}
      />,
    );
    expect(renderStatus).not.toHaveBeenCalled();

    rerender(
      <DataTable
        rows={manyRows}
        columns={baseColumns}
        getRowKey={(r) => r.id}
        emptyMessage={<div>Nothing here</div>}
        virtualization={{
          itemHeight: 10,
          containerHeight: 50,
          threshold: 2,
          renderStatus,
        }}
      />,
    );
    expect(renderStatus).toHaveBeenCalledWith(expect.any(Number), 3);
    expect(screen.getByText('3/3')).toBeInTheDocument();
  });
});
