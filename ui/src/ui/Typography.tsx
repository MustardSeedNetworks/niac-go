import type { FC, HTMLAttributes, ReactNode } from "react";
import { Link } from "react-router-dom";

interface TypographyProps extends HTMLAttributes<HTMLElement> {
  children: ReactNode;
  className?: string;
}

export const H2: FC<TypographyProps> = ({ children, className = "", ...props }) => {
  return (
    <h2 className={`text-xl font-semibold text-white mb-2 ${className}`} {...props}>
      {children}
    </h2>
  );
};

export const P: FC<TypographyProps> = ({ children, className = "", ...props }) => {
  return (
    <p className={`text-gray-300 leading-relaxed ${className}`} {...props}>
      {children}
    </p>
  );
};

export const SmallText: FC<TypographyProps> = ({ children, className = "", ...props }) => {
  return (
    <span className={`text-sm text-gray-400 ${className}`} {...props}>
      {children}
    </span>
  );
};

interface AccentLinkProps extends TypographyProps {
  href?: string;
  to?: string;
  onClick?: () => void;
}

export const AccentLink: FC<AccentLinkProps> = ({
  children,
  className = "",
  href,
  to,
  onClick,
  ...props
}) => {
  const linkClass = `text-violet-400 hover:text-violet-300 underline underline-offset-2 transition-colors ${className}`;

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
    <button onClick={onClick} className={linkClass} {...props}>
      {children}
    </button>
  );
};
