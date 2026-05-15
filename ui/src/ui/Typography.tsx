import type { FC, HTMLAttributes, ReactNode } from 'react';
import { Link } from 'react-router-dom';

/**
 * Typography primitives — one canonical class set per visual level so the
 * app doesn't drift into a dozen "almost h2" inline classNames.
 *
 * Visual scale (top to bottom, sized for our dark-theme baseline):
 *
 *   H1  — page header (set by Layout's <PageHeader>, rarely used in body)
 *         text-2xl / font-bold / text-white
 *   H2  — section header inside a card
 *         text-xl  / font-semibold / text-white
 *   H3  — sub-section header
 *         text-lg  / font-semibold / text-white
 *   H4  — small label / pill header (often with an icon)
 *         text-sm  / font-semibold / text-white
 *
 *   P         — body copy:                 text-gray-300 leading-relaxed
 *   SmallText — secondary inline text:     text-sm  text-gray-400
 *   Caption   — labels / captions / units: text-xs  text-gray-400
 *
 * Contrast — all foreground colors above hit WCAG AA against our
 * --color-bg-base / --color-bg-elevated surfaces. Avoid text-gray-500
 * for sustained body copy; reserve it for placeholders, disabled
 * states, and timestamps where lower contrast is intentional.
 *
 * Accent colors: brand violet (text-violet-300/400) is the only
 * "highlight" colour; status colours (red/amber/emerald) come from the
 * Tailwind palette directly. If you find yourself reaching for
 * text-purple/pink/sky for chrome, route it through this file first.
 */

interface TypographyProps extends HTMLAttributes<HTMLElement> {
  children: ReactNode;
  className?: string;
}

export const H1: FC<TypographyProps> = ({ children, className = '', ...props }) => (
  <h1 className={`text-2xl font-bold text-white ${className}`} {...props}>
    {children}
  </h1>
);

export const H2: FC<TypographyProps> = ({ children, className = '', ...props }) => (
  <h2 className={`text-xl font-semibold text-white mb-2 ${className}`} {...props}>
    {children}
  </h2>
);

export const H3: FC<TypographyProps> = ({ children, className = '', ...props }) => (
  <h3 className={`text-lg font-semibold text-white ${className}`} {...props}>
    {children}
  </h3>
);

export const H4: FC<TypographyProps> = ({ children, className = '', ...props }) => (
  <h4 className={`text-sm font-semibold text-white ${className}`} {...props}>
    {children}
  </h4>
);

export const P: FC<TypographyProps> = ({ children, className = '', ...props }) => (
  <p className={`text-gray-300 leading-relaxed ${className}`} {...props}>
    {children}
  </p>
);

export const SmallText: FC<TypographyProps> = ({ children, className = '', ...props }) => (
  <span className={`text-sm text-gray-400 ${className}`} {...props}>
    {children}
  </span>
);

export const Caption: FC<TypographyProps> = ({ children, className = '', ...props }) => (
  <span className={`text-xs text-gray-400 ${className}`} {...props}>
    {children}
  </span>
);

interface AccentLinkProps extends TypographyProps {
  href?: string;
  to?: string;
  onClick?: () => void;
}

export const AccentLink: FC<AccentLinkProps> = ({
  children,
  className = '',
  href,
  to,
  onClick,
  ...props
}) => {
  const linkClass = `text-violet-300 hover:text-violet-200 underline underline-offset-2 transition-colors ${className}`;

  if (to) {
    return (
      <Link to={to} className={linkClass} {...props}>
        {children}
      </Link>
    );
  }
  if (href) {
    return (
      <a href={href} className={linkClass} {...props}>
        {children}
      </a>
    );
  }
  return (
    <button type="button" onClick={onClick} className={linkClass} {...props}>
      {children}
    </button>
  );
};
