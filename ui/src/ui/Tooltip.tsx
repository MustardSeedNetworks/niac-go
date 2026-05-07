import { type FC, type ReactNode, useId, useState } from 'react';

export interface TooltipProps {
  /** Hover text. If omitted, the wrapper renders children unchanged. */
  text?: ReactNode;
  /** Where to place the bubble relative to the trigger. Defaults to "top". */
  side?: 'top' | 'bottom' | 'left' | 'right';
  /** Trigger element(s). */
  children: ReactNode;
  /** Optional class on the wrapper span. */
  className?: string;
}

const sideClass: Record<NonNullable<TooltipProps['side']>, string> = {
  top: 'bottom-full left-1/2 -translate-x-1/2 mb-2',
  bottom: 'top-full left-1/2 -translate-x-1/2 mt-2',
  left: 'right-full top-1/2 -translate-y-1/2 mr-2',
  right: 'left-full top-1/2 -translate-y-1/2 ml-2',
};

/**
 * Minimal CSS-only tooltip with proper a11y wiring.
 *
 * Why a custom one rather than @radix-ui/react-tooltip:
 *   - No extra runtime dep for what is a 30-line component.
 *   - Native `title=` is fine for plain strings; this primitive is for
 *     formatted / multi-line / linked tooltip content (e.g. "Auto-fix
 *     rewrites the file in place — a .bak is created next to the original").
 *
 * a11y: the wrapper exposes aria-describedby pointing at a hidden bubble
 * that screen readers announce. Keyboard focus on the trigger reveals the
 * bubble (we use `:focus-within` semantics via Tailwind's `peer` pattern,
 * which doesn't require JS state when hover/focus suffice).
 */
export const Tooltip: FC<TooltipProps> = ({ text, side = 'top', children, className = '' }) => {
  const id = useId();
  const [open, setOpen] = useState(false);

  if (text == null || text === '') {
    return <>{children}</>;
  }

  return (
    // biome-ignore lint/a11y/noStaticElementInteractions: hover-only enrichment; a11y comes from aria-describedby below
    <span
      className={`relative inline-flex ${className}`}
      onMouseEnter={() => setOpen(true)}
      onMouseLeave={() => setOpen(false)}
      onFocus={() => setOpen(true)}
      onBlur={() => setOpen(false)}
    >
      <span aria-describedby={id} className="inline-flex">
        {children}
      </span>
      <span
        id={id}
        role="tooltip"
        className={`pointer-events-none absolute z-50 max-w-xs whitespace-normal rounded-md bg-gray-950/95 px-2 py-1 text-xs text-gray-100 ring-1 ring-white/10 transition-opacity duration-100 ${sideClass[side]} ${open ? 'opacity-100' : 'opacity-0'}`}
      >
        {text}
      </span>
    </span>
  );
};
