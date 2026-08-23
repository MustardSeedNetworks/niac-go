/**
 * @fileoverview Focus Trap Hook for Accessibility
 * @description Custom hook that traps keyboard focus within a modal/drawer.
 *              Implements WCAG 2.1 AA compliance for modal dialogs.
 */

import { type RefObject, useEffect, useRef } from 'react';

/** Focusable element selectors */
const FOCUSABLE_SELECTORS: string = [
  'button:not([disabled])',
  'input:not([disabled])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  'a[href]',
  '[tabindex]:not([tabindex="-1"])',
].join(', ');

interface UseFocusTrapOptions {
  /** Whether the focus trap is active */
  isActive: boolean;
  /** Callback when Escape key is pressed */
  onEscape?: () => void;
  /** Whether to auto-focus the first focusable element */
  autoFocus?: boolean;
  /** Whether to restore focus on deactivation */
  restoreFocus?: boolean;
}

/** Get all focusable elements within a container */
function getFocusableElements(container: HTMLElement): HTMLElement[] {
  const elements = container.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTORS);
  return Array.from(elements).filter(
    (el) => el.offsetParent !== null && !el.hasAttribute('aria-hidden'),
  );
}

/** Handle Tab key navigation within the trap */
function handleTabKey(
  event: KeyboardEvent,
  container: HTMLElement,
  focusableElements: HTMLElement[],
): void {
  const firstElement = focusableElements.at(0);
  const lastElement = focusableElements.at(-1);
  // An empty list has no ends to trap focus between. Checking the ends rather
  // than the length is the same condition, stated so the type system can hold
  // us to it.
  if (!firstElement || !lastElement) {
    return;
  }

  const isAtFirst = document.activeElement === firstElement;
  const isAtLast = document.activeElement === lastElement;
  const isOutside = !container.contains(document.activeElement);

  // Shift+Tab from first element -> go to last
  if (event.shiftKey && isAtFirst) {
    event.preventDefault();
    lastElement.focus();
    return;
  }

  // Tab from last element -> go to first
  if (!event.shiftKey && isAtLast) {
    event.preventDefault();
    firstElement.focus();
    return;
  }

  // If focus is outside container, bring it back
  if (isOutside) {
    event.preventDefault();
    firstElement.focus();
  }
}

/**
 * Custom hook that traps keyboard focus within a container.
 * Implements modal dialog accessibility requirements per WCAG 2.1 AA.
 *
 * @param options - Configuration options
 * @returns RefObject to attach to the container element
 *
 * @example
 * ```tsx
 * function Modal({ isOpen, onClose }) {
 *   const containerRef = useFocusTrap({
 *     isActive: isOpen,
 *     onEscape: onClose,
 *   });
 *
 *   return (
 *     <div ref={containerRef} role="dialog" aria-modal="true">
 *       ...
 *     </div>
 *   );
 * }
 * ```
 */
export function useFocusTrap<T extends HTMLElement = HTMLDivElement>(
  options: UseFocusTrapOptions,
): RefObject<T | null> {
  const { isActive, onEscape, autoFocus = true, restoreFocus = true } = options;
  const containerRef = useRef<T>(null);
  const previousActiveElement = useRef<HTMLElement | null>(null);

  // Consumers pass inline arrows for onEscape (e.g. `onCancel={() => setOpen(false)}`),
  // so its identity changes on every parent render. Holding it in a ref keeps it out
  // of the effect's dependency list: otherwise the whole trap tears down and re-arms
  // on each render, and the teardown's restoreFocus() yanks focus back out of the
  // dialog. On a continuously re-rendering page (the log stream at Trace level) that
  // thrashes focus many times a second and makes Escape unreliable. (D18)
  const onEscapeRef = useRef(onEscape);
  onEscapeRef.current = onEscape;

  useEffect(() => {
    if (!isActive) {
      return;
    }

    // Store the currently focused element to restore later
    if (restoreFocus) {
      previousActiveElement.current = document.activeElement as HTMLElement;
    }

    const container = containerRef.current;
    if (!container) {
      return;
    }

    // Auto-focus the first focusable element
    if (autoFocus) {
      const focusableElements = getFocusableElements(container);
      const [firstElement] = focusableElements;
      if (firstElement) {
        requestAnimationFrame(() => {
          firstElement.focus();
        });
      }
    }

    // Handle keyboard navigation
    function handleKeyDown(event: KeyboardEvent): void {
      // Handle Escape key
      const escapeHandler = onEscapeRef.current;
      if (event.key === 'Escape' && escapeHandler) {
        event.preventDefault();
        event.stopPropagation();
        escapeHandler();
        return;
      }

      // Handle Tab key for focus trapping
      if (event.key === 'Tab' && container) {
        const focusableElements = getFocusableElements(container);
        handleTabKey(event, container, focusableElements);
      }
    }

    // Handle focus leaving the container
    function handleFocusOut(event: FocusEvent): void {
      if (!container) return;
      const isLeavingContainer = !container.contains(event.relatedTarget as Node);
      if (!isLeavingContainer) {
        return;
      }

      const focusableElements = getFocusableElements(container);
      if (focusableElements.length === 0) {
        return;
      }

      const firstFocusable = focusableElements.at(0);
      requestAnimationFrame(() => {
        if (firstFocusable && container && !container.contains(document.activeElement)) {
          firstFocusable.focus();
        }
      });
    }

    document.addEventListener('keydown', handleKeyDown);
    container.addEventListener('focusout', handleFocusOut);

    return () => {
      document.removeEventListener('keydown', handleKeyDown);
      container.removeEventListener('focusout', handleFocusOut);

      // Restore focus to the previously focused element
      if (restoreFocus && previousActiveElement.current) {
        previousActiveElement.current.focus();
      }
    };
  }, [isActive, autoFocus, restoreFocus]);

  return containerRef;
}

export default useFocusTrap;
