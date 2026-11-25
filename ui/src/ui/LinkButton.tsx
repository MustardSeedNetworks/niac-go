import type { ReactNode } from 'react';
import { Link, type LinkProps } from 'react-router';

type LinkButtonVariant = 'primary' | 'secondary';

interface LinkButtonProps extends Omit<LinkProps, 'className'> {
  children: ReactNode;
  variant?: LinkButtonVariant;
  icon?: ReactNode;
  className?: string;
}

const variants: Record<LinkButtonVariant, string> = {
  primary: 'border-transparent bg-brand-primary text-on-brand hover:bg-brand-primary/90',
  secondary: 'border-surface-border bg-surface-raised text-text-primary hover:bg-surface-hover',
};

/**
 * A button that navigates. A page whose action is a route change gets the
 * anchor semantics (middle-click, copy link) instead of a Button calling
 * navigate(). The shared Button is a <button>, so this carries the same
 * faces as a Link. Same data-variant, so "one primary per page" stays a
 * queryable fact.
 */
export function LinkButton({
  children,
  variant = 'primary',
  icon,
  className = '',
  ...props
}: LinkButtonProps) {
  return (
    <Link
      className={`inline-flex min-h-10 items-center justify-center gap-2 rounded-lg border px-4 py-2 text-sm font-medium no-underline transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-primary ${variants[variant]} ${className}`}
      data-variant={variant}
      {...props}
    >
      {icon ? <span aria-hidden="true">{icon}</span> : null}
      {children}
    </Link>
  );
}
