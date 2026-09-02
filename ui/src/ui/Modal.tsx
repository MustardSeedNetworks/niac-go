import { X } from 'lucide-react';
import { type FC, type ReactNode, useEffect, useId } from 'react';
import { iconSizes } from '../constants/sizes';
import { useFocusTrap } from '../hooks/useFocusTrap';

export type ModalSize = 'sm' | 'md' | 'lg' | 'xl' | 'full';

interface ModalBaseProps {
  isOpen: boolean;
  onClose: () => void;
  children: ReactNode;
  size?: ModalSize;
  showCloseButton?: boolean;
  closeOnBackdropClick?: boolean;
  closeOnEscape?: boolean;
  className?: string;
}

/**
 * A dialog must have an accessible name, so the type requires exactly one way
 * of giving it one. Before this, `title` was optional and a title-less Modal
 * compiled fine and shipped a dialog a screen reader announces as "dialog" and
 * nothing else — axe's aria-dialog-name, and eight of the nine violations the
 * Storybook a11y gate found when it was first turned on (#1668).
 *
 * - `title`      — Modal renders the heading and labels itself from it.
 * - `labelledBy` — the caller renders its own heading and passes its id, for
 *                  layouts Modal's header cannot express (an icon beside the
 *                  title, a heading with a subtitle beneath it).
 * - `ariaLabel`  — last resort, when there is no visible heading to point at.
 */
type ModalNaming =
  | { title: string; labelledBy?: never; ariaLabel?: never }
  | { title?: never; labelledBy: string; ariaLabel?: never }
  | { title?: never; labelledBy?: never; ariaLabel: string };

export type ModalProps = ModalBaseProps & ModalNaming;

const sizeClasses: Record<ModalSize, string> = {
  sm: 'max-w-sm',
  md: 'max-w-md',
  lg: 'max-w-lg',
  xl: 'max-w-xl',
  full: 'max-w-4xl',
};

export const Modal: FC<ModalProps> = ({
  isOpen,
  onClose,
  title,
  labelledBy,
  ariaLabel,
  children,
  size = 'md',
  showCloseButton = true,
  closeOnBackdropClick = true,
  closeOnEscape = true,
  className = '',
}) => {
  // Was the literal id "modal-title", which is a duplicate the moment two
  // modals are mounted at once — and aria-labelledby then resolves to whichever
  // the browser finds first.
  const headingId = useId();

  // Trap Tab/Shift+Tab focus inside the dialog and route Escape through onClose.
  const containerRef = useFocusTrap<HTMLDivElement>({
    isActive: isOpen,
    onEscape: closeOnEscape ? onClose : undefined,
  });

  useEffect(() => {
    if (isOpen) {
      // Prevent body scroll when modal is open
      document.body.style.overflow = 'hidden';
    }
    return () => {
      document.body.style.overflow = '';
    };
  }, [isOpen]);

  if (!isOpen) {
    return null;
  }

  return (
    <div className="fixed inset-0 z-50 flex-center">
      {closeOnBackdropClick ? (
        <button
          type="button"
          className="absolute inset-0 bg-scrim/70 backdrop-blur-sm"
          onClick={onClose}
          aria-label="Close modal"
        />
      ) : (
        <div className="absolute inset-0 bg-scrim/70 backdrop-blur-sm" />
      )}
      <div
        ref={containerRef}
        className={`relative z-10 mx-4 w-full ${sizeClasses[size]} rounded-2xl border border-surface-border bg-bg-surface/95 shadow-2xl ${className}`}
        role="dialog"
        aria-modal="true"
        aria-labelledby={title ? headingId : labelledBy}
        aria-label={ariaLabel}
      >
        {(title || showCloseButton) && (
          <div className="flex-between px-6 py-4 border-b border-surface-border">
            {title && (
              <h2 id={headingId} className="heading-3 text-text-primary">
                {title}
              </h2>
            )}
            {showCloseButton && (
              <button
                type="button"
                onClick={onClose}
                className="ml-auto p-1 text-text-muted hover:text-text-primary transition-colors rounded-lg hover:bg-surface-hover"
                aria-label="Close modal"
              >
                <X className={iconSizes.lg} />
              </button>
            )}
          </div>
        )}
        <div className="pad-lg">{children}</div>
      </div>
    </div>
  );
};

// Convenience components for modal sections
export const ModalHeader: FC<{ children: ReactNode; className?: string }> = ({
  children,
  className = '',
}) => <div className={`mb-content ${className}`}>{children}</div>;

export const ModalBody: FC<{ children: ReactNode; className?: string }> = ({
  children,
  className = '',
}) => <div className={`stack-lg ${className}`}>{children}</div>;

export const ModalFooter: FC<{ children: ReactNode; className?: string }> = ({
  children,
  className = '',
}) => (
  <div
    className={`flex justify-end gap-default pt-section mt-content border-t border-surface-border ${className}`}
  >
    {children}
  </div>
);
