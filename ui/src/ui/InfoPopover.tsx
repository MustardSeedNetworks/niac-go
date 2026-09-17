import { Info } from 'lucide-react';
import { type FC, type ReactNode, useEffect, useId, useRef, useState } from 'react';
import { iconSizes } from '../constants/sizes';

export interface InfoPopoverProps {
  /** Accessible label for the trigger button, e.g. "About BPF". Translated. */
  label: string;
  /** Popover heading. Usually the jargon term itself (do-not-translate). */
  title: string;
  /** Explanation content. Translated. */
  children: ReactNode;
  /** Optional class on the wrapper span. */
  className?: string;
}

/**
 * Click/focus-triggered jargon-disclosure popover.
 *
 * Unlike Tooltip.tsx (hover-only, for formatted enrichment on elements that
 * are already operable another way), InfoPopover is the trigger itself: a
 * small "i" button that opens a dismissable panel explaining a technical
 * term inline. Needed because hover-only disclosure is unusable for
 * keyboard/touch/screen-reader users, and jargon definitions are the
 * primary content here, not a bonus hint.
 *
 * a11y: button toggles `aria-expanded` and `aria-controls`; the panel is
 * `role="dialog"` with `aria-labelledby`/`aria-describedby`. Escape closes
 * and returns focus to the trigger; a mousedown outside the panel closes it.
 */
export const InfoPopover: FC<InfoPopoverProps> = ({ label, title, children, className = '' }) => {
  const [open, setOpen] = useState(false);
  const wrapperRef = useRef<HTMLSpanElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const titleId = useId();
  const bodyId = useId();

  useEffect(() => {
    if (!open) return;

    const onPointerDown = (e: MouseEvent) => {
      if (wrapperRef.current && !wrapperRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setOpen(false);
        triggerRef.current?.focus();
      }
    };

    document.addEventListener('mousedown', onPointerDown);
    document.addEventListener('keydown', onKeyDown);
    return () => {
      document.removeEventListener('mousedown', onPointerDown);
      document.removeEventListener('keydown', onKeyDown);
    };
  }, [open]);

  return (
    <span ref={wrapperRef} className={`relative inline-flex ${className}`}>
      <button
        ref={triggerRef}
        type="button"
        aria-expanded={open}
        aria-controls={bodyId}
        aria-label={label}
        onClick={() => setOpen((prev) => !prev)}
        className="inline-flex items-center justify-center rounded-full text-text-muted hover:text-text-primary focus:outline-none focus:ring-2 focus:ring-brand-primary/50"
      >
        <Info className={iconSizes.sm} aria-hidden="true" />
      </button>
      {open && (
        <span
          role="dialog"
          aria-labelledby={titleId}
          aria-describedby={bodyId}
          className="absolute left-0 top-full z-50 mt-tight w-64 rounded-lg border border-surface-border bg-bg-surface/95 p-cell text-xs text-text-primary shadow-lg"
        >
          <span id={titleId} className="mb-tight block font-semibold text-text-primary">
            {title}
          </span>
          <span id={bodyId} className="block text-text-secondary">
            {children}
          </span>
        </span>
      )}
    </span>
  );
};
