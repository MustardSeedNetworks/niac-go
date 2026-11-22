import { act, fireEvent, render, screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { VirtualList } from './VirtualList';

const ITEM_HEIGHT = 50;
const items = Array.from({ length: 543 }, (_, i) => `device-${String(i).padStart(3, '0')}`);

function renderList(list: string[] = items) {
  return render(
    <VirtualList
      items={list}
      itemHeight={ITEM_HEIGHT}
      getKey={(item) => item}
      renderItem={(item) => <span>{item}</span>}
      ariaLabel="devices"
      data-testid="frame"
    />,
  );
}

const mounted = () => within(screen.getByTestId('frame')).queryAllByRole('listitem');

describe('VirtualList', () => {
  it('mounts a window of a 543-item list, not every item', () => {
    renderList();

    expect(mounted().length).toBeGreaterThan(0);
    expect(mounted().length).toBeLessThan(40);
    expect(screen.getByText('device-000')).toBeInTheDocument();
    expect(screen.queryByText('device-542')).not.toBeInTheDocument();
  });

  it('sizes the scrollable area for the whole list', () => {
    renderList();

    const [spacer] = Array.from(screen.getByTestId('frame').children);
    expect((spacer as HTMLElement).style.height).toBe(`${543 * ITEM_HEIGHT}px`);
  });

  it('moves the window with the scroll position and reaches the last item', () => {
    renderList();
    const frame = screen.getByTestId('frame');

    act(() => {
      frame.scrollTop = 543 * ITEM_HEIGHT;
      fireEvent.scroll(frame);
    });

    expect(screen.getByText('device-542')).toBeInTheDocument();
    expect(screen.queryByText('device-000')).not.toBeInTheDocument();
    expect(mounted().length).toBeLessThan(40);
  });

  it('numbers each mounted item within the full list for assistive technology', () => {
    renderList();
    const [first] = mounted();

    expect(first?.getAttribute('aria-posinset')).toBe('1');
    expect(first?.getAttribute('aria-setsize')).toBe('543');
  });

  it('does not reserve blank space for a short list', () => {
    renderList(items.slice(0, 3));

    expect(screen.getByTestId('frame').style.height).toBe(`${3 * ITEM_HEIGHT}px`);
    expect(mounted()).toHaveLength(3);
  });
});
