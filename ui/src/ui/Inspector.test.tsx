import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { Inspector, InspectorPane, InspectorPanes, InspectorRecords } from './Inspector';

describe('Inspector', () => {
  it('renders the filter above the columns, and only when given one', () => {
    const { container, rerender } = render(
      <Inspector filter={<input aria-label="Display filter" />}>
        <InspectorRecords>rows</InspectorRecords>
      </Inspector>,
    );
    expect(screen.getByLabelText('Display filter')).toBeInTheDocument();

    rerender(
      <Inspector>
        <InspectorRecords>rows</InspectorRecords>
      </Inspector>,
    );
    expect(container.querySelector('input')).toBeNull();
  });

  it('gives the record list the wider column', () => {
    // The two packet inspectors had drifted into mirror images of each other —
    // the live view gave the list 4/12 and the pcap view 8/12, on two tabs of
    // one route. The list is what an operator scans, so it takes the wide half;
    // this asserts the two can no longer disagree.
    const { container } = render(
      <Inspector>
        <InspectorRecords>rows</InspectorRecords>
        <InspectorPanes>
          <InspectorPane label="Hex dump">bytes</InspectorPane>
        </InspectorPanes>
      </Inspector>,
    );
    const records = container.querySelector('.lg\\:col-span-7');
    const panes = container.querySelector('.lg\\:col-span-5');
    expect(records).not.toBeNull();
    expect(panes).not.toBeNull();
    expect(records?.textContent).toBe('rows');
  });

  it('shares the column height evenly between panes', () => {
    // Hand-set pixel heights are what let the same two panes ship as 350/220
    // in one implementation and 280/280 in the other.
    const { container } = render(
      <InspectorPanes>
        <InspectorPane label="Hex dump">bytes</InspectorPane>
        <InspectorPane label="Packet details">fields</InspectorPane>
      </InspectorPanes>,
    );
    const panes = container.querySelectorAll('.flex-1.min-h-0');
    expect(panes.length).toBeGreaterThanOrEqual(2);
    for (const pane of panes) {
      expect(pane.className).not.toMatch(/h-\[\d+px\]/);
    }
  });

  it('scrolls only the pane that asks to', () => {
    const { container } = render(
      <InspectorPanes>
        <InspectorPane label="Hex dump">bytes</InspectorPane>
        <InspectorPane label="Packet details" scroll>
          fields
        </InspectorPane>
      </InspectorPanes>,
    );
    expect(container.querySelectorAll('.overflow-y-auto')).toHaveLength(1);
  });

  it('labels each pane as prose, not as a figure', () => {
    render(<InspectorPane label="Packet details">fields</InspectorPane>);
    const label = screen.getByText('Packet details');
    expect(label.className).not.toMatch(/font-mono/);
  });
});
