import { HelpCircle } from 'lucide-react';
import type { ReactNode } from 'react';
import { iconSizes } from '../../constants/sizes';

export interface FormFieldProps {
  label: ReactNode;
  children: ReactNode;
  helpText?: string;
  required?: boolean;
  className?: string;
  /**
   * The id of the control this field labels. Required: an optional
   * association is one a new field silently forgets, and the fallback that
   * used to render a bare <span> looked identical while leaving the control
   * with no accessible name and no click-to-focus target. A field that wraps
   * a group rather than a single control points at the group's primary
   * control.
   */
  htmlFor: string;
  /** Inline validation error rendered below the field, in the error color. */
  error?: string;
}

const rowClassName = 'flex items-center gap-compact text-sm font-medium text-text-secondary mb-2';

/**
 * Hover help for a field.
 *
 * It sits beside the label rather than inside it. A tooltip nested in the
 * <label> becomes part of the control's accessible name, so a field labelled
 * "Username" with help text announced as "Username Username is the account the
 * simulated CLI accepts" -- the name and the description run together. Keeping
 * it a sibling leaves the accessible name as the label alone.
 */
function HelpTip({ text }: { text: string }) {
  return (
    <span className="relative group">
      <HelpCircle className={`${iconSizes.sm} text-text-muted cursor-help`} aria-hidden="true" />
      <span className="absolute left-1/2 -translate-x-1/2 bottom-full mb-2 px-3 py-compact-md bg-bg-elevated text-text-primary text-xs rounded-lg opacity-0 group-hover:opacity-100 transition-opacity whitespace-nowrap z-10 pointer-events-none">
        {text}
      </span>
    </span>
  );
}

export function FormField({
  label,
  children,
  helpText,
  required,
  className = '',
  htmlFor,
  error,
}: FormFieldProps) {
  return (
    <div className={className}>
      <div className={rowClassName}>
        <label htmlFor={htmlFor} className="flex items-center gap-compact">
          {label}
          {required && <span className="text-status-error">*</span>}
        </label>
        {helpText && <HelpTip text={helpText} />}
      </div>
      {children}
      {error && (
        <p className="mt-1 text-xs text-status-error" role="alert">
          {error}
        </p>
      )}
    </div>
  );
}
