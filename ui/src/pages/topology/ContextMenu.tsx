import type { FC, MouseEvent as ReactMouseEvent } from 'react';

/**
 * ContextMenu is the small popover shown on right-click of a node,
 * edge, or canvas in TopologyPage. Each action is a button; clicking
 * an action invokes the handler and the parent closes the menu via
 * the supplied onClose. Click outside / Esc are handled by the page
 * (one overlay listener catches every dismiss case rather than each
 * menu installing its own).
 *
 * Positioning uses absolute placement from the parent (the canvas
 * div). The parent passes the event's clientX/clientY adjusted for
 * the canvas's bounding rect so the menu pops where the user clicked.
 *
 * No external dep — niac doesn't ship Headless UI and a 60-line
 * absolute-positioned div with role="menu" is sufficient for the
 * three menus this needs. Keyboard navigation is intentionally not
 * implemented since right-click without pointer doesn't have a
 * common keyboard equivalent in this UI.
 */

export interface ContextMenuItem {
  /** Stable key for React list reconciliation. */
  key: string;
  /** Visible label. */
  label: string;
  /** Optional muted sub-label (e.g. shortcut hint or supporting info). */
  hint?: string;
  /** Action to run on click. The parent closes the menu after. */
  onSelect: () => void;
  /** Render as a destructive (red) item — used for Hide / Clear ops. */
  destructive?: boolean;
  /** Disable the item (greyed out, no click). */
  disabled?: boolean;
  /** Insert a separator BEFORE this item (instead of rendering an item). */
  separatorBefore?: boolean;
}

interface Props {
  /** Anchor x/y in the parent's coordinate space (clientX - rect.left). */
  x: number;
  y: number;
  items: ContextMenuItem[];
  onClose: () => void;
}

export const ContextMenu: FC<Props> = ({ x, y, items, onClose }) => {
  const handleItemClick = (e: ReactMouseEvent, item: ContextMenuItem) => {
    e.stopPropagation();
    if (item.disabled) return;
    item.onSelect();
    onClose();
  };

  return (
    <div
      className="absolute z-50 min-w-[180px] rounded-md border border-white/10 bg-gray-950/95 py-1 text-xs text-gray-200 shadow-xl backdrop-blur"
      style={{ left: x, top: y }}
      // Stop right-click on the menu itself from re-opening the pane
      // menu through ReactFlow's onPaneContextMenu.
      onContextMenu={(e) => e.preventDefault()}
      role="menu"
    >
      {items.map((item, idx) => (
        <div key={item.key}>
          {item.separatorBefore && idx > 0 && <hr className="my-1 h-px border-0 bg-white/10" />}
          <button
            type="button"
            role="menuitem"
            disabled={item.disabled}
            onClick={(e) => handleItemClick(e, item)}
            className={`flex w-full items-center justify-between gap-3 px-3 py-1.5 text-left transition-colors ${
              item.disabled
                ? 'text-gray-600 cursor-not-allowed'
                : item.destructive
                  ? 'text-red-300 hover:bg-red-500/15 hover:text-red-200'
                  : 'hover:bg-white/5 hover:text-white'
            }`}
          >
            <span>{item.label}</span>
            {item.hint && <span className="text-[10px] text-gray-500">{item.hint}</span>}
          </button>
        </div>
      ))}
    </div>
  );
};
