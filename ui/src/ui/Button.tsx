import type { ButtonHTMLAttributes, FC, ReactNode, Ref } from 'react';
import { iconSizes } from '../constants/sizes';

type ButtonVariant = 'solid' | 'outline' | 'ghost' | 'secondary';
type ButtonTone = 'violet' | 'red' | 'green' | 'blue' | 'gray';
type ButtonSize = 'xs' | 'sm' | 'md' | 'lg';

// React 19: ref is now a regular prop, no forwardRef needed
interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  children: ReactNode;
  variant?: ButtonVariant;
  tone?: ButtonTone;
  size?: ButtonSize;
  leftIcon?: ReactNode;
  rightIcon?: ReactNode;
  loading?: boolean;
  className?: string;
  ref?: Ref<HTMLButtonElement>;
}

const baseStyles =
  'inline-flex items-center justify-center gap-2 font-medium rounded-lg transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-offset-gray-900 disabled:opacity-50 disabled:cursor-not-allowed active:scale-[0.98]';

const sizeStyles: Record<ButtonSize, string> = {
  xs: 'px-2 py-1 text-xs',
  sm: 'px-3 py-1.5 text-sm',
  md: 'px-4 py-2 text-sm',
  lg: 'px-6 py-3 text-base',
};

const variantStyles: Record<ButtonVariant, Record<ButtonTone, string>> = {
  solid: {
    violet:
      'bg-gradient-to-r from-brand-600 to-brand-500 hover:from-brand-500 hover:to-brand-400 text-text-primary shadow-lg shadow-brand-500/25 focus:ring-brand-500',
    red: 'bg-gradient-to-r from-red-600 to-red-500 hover:from-red-500 hover:to-red-400 text-text-primary shadow-lg shadow-red-500/25 focus:ring-status-error',
    green:
      'bg-gradient-to-r from-emerald-600 to-emerald-500 hover:from-emerald-500 hover:to-emerald-400 text-text-primary shadow-lg shadow-emerald-500/25 focus:ring-status-success',
    blue: 'bg-gradient-to-r from-blue-600 to-blue-500 hover:from-blue-500 hover:to-blue-400 text-text-primary shadow-lg shadow-blue-500/25 focus:ring-status-info',
    gray: 'bg-gradient-to-r from-gray-600 to-gray-500 hover:from-gray-500 hover:to-gray-400 text-text-primary shadow-lg shadow-gray-500/25 focus:ring-border-muted',
  },
  outline: {
    violet:
      'border border-brand-400/30 text-brand-300 hover:bg-brand-400/10 hover:border-brand-400/50 focus:ring-brand-500',
    red: 'border border-status-error/30 text-status-error hover:bg-status-error/10 hover:border-status-error/50 focus:ring-status-error',
    green:
      'border border-status-success/30 text-status-success hover:bg-status-success/10 hover:border-status-success/50 focus:ring-status-success',
    blue: 'border border-status-info/30 text-status-info hover:bg-status-info/10 hover:border-status-info/50 focus:ring-status-info',
    gray: 'border border-white/20 text-text-secondary hover:bg-white/10 hover:border-white/30 focus:ring-border-muted',
  },
  ghost: {
    violet: 'text-brand-300 hover:bg-brand-400/10 hover:text-brand-200 focus:ring-brand-500',
    red: 'text-status-error hover:bg-status-error/10 hover:text-status-error focus:ring-status-error',
    green:
      'text-status-success hover:bg-status-success/10 hover:text-status-success focus:ring-status-success',
    blue: 'text-status-info hover:bg-status-info/10 hover:text-status-info focus:ring-status-info',
    gray: 'text-text-muted hover:bg-white/10 hover:text-text-primary focus:ring-border-muted',
  },
  secondary: {
    violet: 'bg-white/10 hover:bg-white/15 text-text-primary focus:ring-brand-500',
    red: 'bg-white/10 hover:bg-white/15 text-text-primary focus:ring-status-error',
    green: 'bg-white/10 hover:bg-white/15 text-text-primary focus:ring-status-success',
    blue: 'bg-white/10 hover:bg-white/15 text-text-primary focus:ring-status-info',
    gray: 'bg-white/10 hover:bg-white/15 text-text-primary focus:ring-border-muted',
  },
};

// Loading spinner component
const LoadingSpinner: FC<{ size: ButtonSize }> = ({ size }) => {
  const spinnerSize = size === 'xs' || size === 'sm' ? iconSizes.xs : iconSizes.md;
  return (
    <svg
      className={`animate-spin ${spinnerSize}`}
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
    >
      <title>Loading</title>
      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
      <path
        className="opacity-75"
        fill="currentColor"
        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
      />
    </svg>
  );
};

// React 19: ref as a regular prop instead of forwardRef
export const Button: FC<ButtonProps> = ({
  children,
  variant = 'solid',
  tone = 'violet',
  size = 'md',
  leftIcon,
  rightIcon,
  loading = false,
  className = '',
  disabled,
  ref,
  ...props
}) => (
  <button
    type="button"
    ref={ref}
    className={`${baseStyles} ${sizeStyles[size]} ${variantStyles[variant][tone]} ${className}`}
    disabled={disabled || loading}
    {...props}
  >
    {loading ? <LoadingSpinner size={size} /> : leftIcon}
    {children}
    {!loading && rightIcon}
  </button>
);

// Icon button for compact actions
interface IconButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  icon: ReactNode;
  'aria-label': string;
  variant?: 'ghost' | 'outline' | 'solid';
  tone?: ButtonTone;
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}

export const IconButton: FC<IconButtonProps> = ({
  icon,
  variant = 'ghost',
  tone = 'gray',
  size = 'md',
  className = '',
  ...props
}) => {
  const iconSizeStyles = {
    sm: 'p-1.5',
    md: 'p-2',
    lg: 'p-3',
  };

  const variantBase = {
    ghost: 'hover:bg-white/10 rounded-lg',
    outline: 'border border-white/20 hover:bg-white/10 rounded-lg',
    solid: 'bg-white/10 hover:bg-white/15 rounded-lg',
  };

  const toneStyles: Record<ButtonTone, string> = {
    violet: 'text-brand-400 hover:text-brand-300',
    red: 'text-status-error hover:text-status-error',
    green: 'text-status-success hover:text-status-success',
    blue: 'text-status-info hover:text-status-info',
    gray: 'text-text-muted hover:text-text-primary',
  };

  return (
    <button
      type="button"
      className={`inline-flex items-center justify-center transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-brand-500/50 disabled:opacity-50 disabled:cursor-not-allowed ${iconSizeStyles[size]} ${variantBase[variant]} ${toneStyles[tone]} ${className}`}
      {...props}
    >
      {icon}
    </button>
  );
};
