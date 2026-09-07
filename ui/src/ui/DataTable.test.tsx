/**
 * DataTable.test.tsx — the shared table primitive extracted in Phase 7.
 *
 * Covers the behaviors the primitive owns independently of any consumer:
 * empty/loading states, testid passthrough, client-side sort toggling,
 * row selection (select-one / select-all), and — added with the U2b
 * migration of the seven hand-rolled tables — activatable rows, per-row
 * style and testid, and a caller-supplied initial sort.
 */

import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { required } from '../test/required';
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
      { ...required(baseColumns[0], 'the name column'), 'data-testid': 'name-cell' },
      required(baseColumns[1], 'the second column'),
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

  it('activates a row on click and on Enter and Space', () => {
    const onRowClick = vi.fn();
    render(
      <DataTable
        rows={rows}
        columns={baseColumns}
        getRowKey={(r) => r.id}
        emptyMessage={<div>Nothing here</div>}
        onRowClick={onRowClick}
      />,
    );

    const bravo = required(screen.getByText('Bravo').closest('tr'));
    // Reachable by keyboard, not mouse-only.
    expect(bravo).toHaveAttribute('tabindex', '0');

    fireEvent.click(bravo);
    fireEvent.keyDown(bravo, { key: 'Enter' });
    fireEvent.keyDown(bravo, { key: ' ' });
    expect(onRowClick).toHaveBeenCalledTimes(3);
    expect(onRowClick).toHaveBeenNthCalledWith(1, rows[0]);

    // Any other key is left to the browser.
    fireEvent.keyDown(bravo, { key: 'a' });
    expect(onRowClick).toHaveBeenCalledTimes(3);
  });

  it('leaves rows inert when no row handler is given', () => {
    render(
      <DataTable
        rows={rows}
        columns={baseColumns}
        getRowKey={(r) => r.id}
        emptyMessage={<div>Nothing here</div>}
      />,
    );

    expect(required(screen.getByText('Bravo').closest('tr'))).not.toHaveAttribute('tabindex');
  });

  it("applies the caller's per-row style and testid", () => {
    render(
      <DataTable
        rows={rows}
        columns={baseColumns}
        getRowKey={(r) => r.id}
        emptyMessage={<div>Nothing here</div>}
        rowStyle={(r) => (r.id === 'a' ? { backgroundColor: 'rgb(1, 2, 3)' } : undefined)}
        rowTestId={(r) => `row-${r.id}`}
      />,
    );

    expect(screen.getByTestId('row-a')).toHaveStyle({ backgroundColor: 'rgb(1, 2, 3)' });
    expect(screen.getByTestId('row-b')).not.toHaveStyle({ backgroundColor: 'rgb(1, 2, 3)' });
  });

  it('orders rows by the initial sort before any header is clicked', () => {
    render(
      <DataTable
        rows={rows}
        columns={baseColumns}
        getRowKey={(r) => r.id}
        emptyMessage={<div>Nothing here</div>}
        defaultSort={{ key: 'count', direction: 'desc' }}
      />,
    );

    const names = screen
      .getAllByRole('row')
      .slice(1)
      .map((row) => row.textContent);
    expect(names).toEqual(['Alpha5', 'Bravo2']);
  });

  it('pins the header only when asked to', () => {
    const { rerender } = render(
      <DataTable
        rows={rows}
        columns={baseColumns}
        getRowKey={(r) => r.id}
        emptyMessage={<div>Nothing here</div>}
      />,
    );
    expect(required(screen.getByText('Name').closest('thead')).className).not.toContain('sticky');

    rerender(
      <DataTable
        rows={rows}
        columns={baseColumns}
        getRowKey={(r) => r.id}
        emptyMessage={<div>Nothing here</div>}
        stickyHeader
      />,
    );
    expect(required(screen.getByText('Name').closest('thead')).className).toContain('sticky');
  });
});
