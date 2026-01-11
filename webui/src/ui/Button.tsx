import { type FC, type ReactNode, type ButtonHTMLAttributes } from 'react';

type ButtonVariant = 'solid' | 'outline' | 'ghost' | 'secondary';
type ButtonTone = 'violet' | 'red' | 'green' | 'gray';
type ButtonSize = 'sm' | 'md' | 'lg';

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  children: ReactNode;
  variant?: ButtonVariant;
  tone?: ButtonTone;
  size?: ButtonSize;
  leftIcon?: ReactNode;
  rightIcon?: ReactNode;
  className?: string;
}

const baseStyles =
  'inline-flex items-center justify-center gap-2 font-medium rounded-lg transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-offset-gray-900 disabled:opacity-50 disabled:cursor-not-allowed';

const sizeStyles: Record<ButtonSize, string> = {
  sm: 'px-3 py-1.5 text-sm',
  md: 'px-4 py-2 text-sm',
  lg: 'px-6 py-3 text-base',
};

const variantStyles: Record<ButtonVariant, Record<ButtonTone, string>> = {
  solid: {
    violet: 'bg-violet-600 hover:bg-violet-700 text-white focus:ring-violet-500',
    red: 'bg-red-600 hover:bg-red-700 text-white focus:ring-red-500',
    green: 'bg-green-600 hover:bg-green-700 text-white focus:ring-green-500',
    gray: 'bg-gray-600 hover:bg-gray-700 text-white focus:ring-gray-500',
  },
  outline: {
    violet: 'border border-violet-400/30 text-violet-300 hover:bg-violet-400/10 focus:ring-violet-500',
    red: 'border border-red-400/30 text-red-300 hover:bg-red-400/10 focus:ring-red-500',
    green: 'border border-green-400/30 text-green-300 hover:bg-green-400/10 focus:ring-green-500',
    gray: 'border border-white/20 text-gray-300 hover:bg-white/10 focus:ring-gray-500',
  },
  ghost: {
    violet: 'text-violet-300 hover:bg-violet-400/10 focus:ring-violet-500',
    red: 'text-red-300 hover:bg-red-400/10 focus:ring-red-500',
    green: 'text-green-300 hover:bg-green-400/10 focus:ring-green-500',
    gray: 'text-gray-300 hover:bg-white/10 focus:ring-gray-500',
  },
  secondary: {
    violet: 'bg-gray-700 hover:bg-gray-600 text-gray-200 focus:ring-gray-500',
    red: 'bg-gray-700 hover:bg-gray-600 text-gray-200 focus:ring-gray-500',
    green: 'bg-gray-700 hover:bg-gray-600 text-gray-200 focus:ring-gray-500',
    gray: 'bg-gray-700 hover:bg-gray-600 text-gray-200 focus:ring-gray-500',
  },
};

export const Button: FC<ButtonProps> = ({
  children,
  variant = 'solid',
  tone = 'violet',
  size = 'md',
  leftIcon,
  rightIcon,
  className = '',
  ...props
}) => {
  return (
    <button
      className={`${baseStyles} ${sizeStyles[size]} ${variantStyles[variant][tone]} ${className}`}
      {...props}
    >
      {leftIcon}
      {children}
      {rightIcon}
    </button>
  );
};
