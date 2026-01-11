import { type FC, type ReactNode } from 'react';

type CardVariant = 'default' | 'elevated' | 'outlined' | 'ghost';

interface CardProps {
  children: ReactNode;
  className?: string;
  variant?: CardVariant;
  hover?: boolean;
  padding?: 'none' | 'sm' | 'md' | 'lg';
}

const variantStyles: Record<CardVariant, string> = {
  default: 'backdrop-blur-xl bg-gradient-to-br from-gray-900/90 to-gray-900/70 border border-white/10 shadow-xl shadow-black/20',
  elevated: 'backdrop-blur-xl bg-gradient-to-br from-gray-800/90 to-gray-900/80 border border-white/15 shadow-2xl shadow-black/30',
  outlined: 'bg-transparent border border-white/20',
  ghost: 'bg-white/5 border border-transparent',
};

const hoverStyles = 'transition-all duration-300 hover:shadow-2xl hover:shadow-brand-500/5 hover:border-white/20 hover:-translate-y-0.5';

export const Card: FC<CardProps> = ({
  children,
  className = '',
  variant = 'default',
  hover = false,
  padding,
}) => {
  const paddingClass = padding === 'none' ? '' : padding === 'sm' ? 'p-4' : padding === 'lg' ? 'p-8' : '';

  return (
    <div
      className={`rounded-xl ${variantStyles[variant]} ${hover ? hoverStyles : ''} ${paddingClass} ${className}`}
    >
      {children}
    </div>
  );
};

interface CardContentProps {
  children: ReactNode;
  className?: string;
}

export const CardContent: FC<CardContentProps> = ({ children, className = '' }) => {
  return <div className={`p-6 ${className}`}>{children}</div>;
};

interface CardHeaderProps {
  children: ReactNode;
  className?: string;
  actions?: ReactNode;
}

export const CardHeader: FC<CardHeaderProps> = ({ children, className = '', actions }) => {
  return (
    <div className={`flex items-center justify-between px-6 py-4 border-b border-white/10 ${className}`}>
      <div className="flex items-center gap-3">{children}</div>
      {actions && <div className="flex items-center gap-2">{actions}</div>}
    </div>
  );
};

interface CardFooterProps {
  children: ReactNode;
  className?: string;
}

export const CardFooter: FC<CardFooterProps> = ({ children, className = '' }) => {
  return (
    <div className={`px-6 py-4 border-t border-white/10 bg-black/20 rounded-b-xl ${className}`}>
      {children}
    </div>
  );
};

// Stat card for dashboard metrics
interface StatCardProps {
  label: string;
  value: string | number;
  icon?: ReactNode;
  trend?: {
    value: number;
    label?: string;
  };
  className?: string;
}

export const StatCard: FC<StatCardProps> = ({
  label,
  value,
  icon,
  trend,
  className = '',
}) => {
  const trendColor = trend
    ? trend.value > 0
      ? 'text-emerald-400'
      : trend.value < 0
      ? 'text-red-400'
      : 'text-gray-400'
    : '';

  const trendIcon = trend
    ? trend.value > 0
      ? '↑'
      : trend.value < 0
      ? '↓'
      : '→'
    : '';

  return (
    <Card className={className} hover>
      <CardContent className="space-y-2">
        <div className="flex items-center justify-between">
          <span className="text-sm font-medium text-gray-400">{label}</span>
          {icon && <span className="text-brand-400">{icon}</span>}
        </div>
        <div className="flex items-end justify-between gap-2">
          <span className="text-3xl font-bold text-white">{value}</span>
          {trend && (
            <span className={`text-sm font-medium ${trendColor}`}>
              {trendIcon} {Math.abs(trend.value)}%
              {trend.label && <span className="text-gray-500 ml-1">{trend.label}</span>}
            </span>
          )}
        </div>
      </CardContent>
    </Card>
  );
};
