/**
 * Card Component System
 *
 * Base card container with glassmorphism styling for NIAC's dark theme.
 *
 * Features:
 * - Status badge with color coding (success, warning, error, loading, unknown)
 * - Header with title, optional subtitle, icon, and custom actions
 * - Keyboard accessibility (Enter/Space to activate)
 * - Optional click handler for interactive cards
 * - ARIA live regions for dynamic content updates
 * - Focus ring for keyboard navigation
 *
 * Usage:
 * ```tsx
 * <Card
 *   title="Device Status"
 *   subtitle="eth0"
 *   status="success"
 *   icon={<NetworkIcon />}
 *   onClick={() => handleCardClick()}
 * >
 *   <CardRow label="IP" value="192.168.1.1" />
 * </Card>
 * ```
 */

import type { FC, HTMLAttributes, KeyboardEvent, ReactNode } from 'react';
import { type Status, StatusBadge } from './StatusBadge';

// Re-export Status type for convenience
export type { Status };

// ============================================================================
// Variant Styles (preserved from original for backwards compatibility)
// ============================================================================

type CardVariant = 'default' | 'elevated' | 'outlined' | 'ghost';

const variantStyles: Record<CardVariant, string> = {
  default:
    'backdrop-blur-xl bg-gradient-to-br from-bg-surface/90 to-bg-surface/70 border border-surface-border shadow-xl shadow-scrim/20',
  elevated:
    'backdrop-blur-xl bg-gradient-to-br from-bg-elevated/90 to-bg-surface/80 border border-surface-border shadow-2xl shadow-scrim/30',
  outlined: 'bg-transparent border border-surface-border',
  ghost: 'bg-surface-hover border border-transparent',
};

const hoverStyles =
  'transition-all duration-300 hover:shadow-2xl hover:shadow-brand-primary/5 hover:border-surface-border hover:-translate-y-0.5';

const paddingClasses: Record<string, string> = {
  none: '',
  sm: 'pad',
  md: 'pad-lg',
  lg: 'pad-xl',
};

// ============================================================================
// Base Card Component (backwards compatible)
// ============================================================================

interface BaseCardProps extends Omit<HTMLAttributes<HTMLDivElement>, 'children'> {
  children: ReactNode;
  className?: string;
  variant?: CardVariant;
  hover?: boolean;
  padding?: 'none' | 'sm' | 'md' | 'lg';
}

/**
 * Basic Card container - use for simple layouts without status/title.
 * Preserves original API for backwards compatibility.
 */
export const Card: FC<BaseCardProps> = ({
  children,
  className = '',
  variant = 'default',
  hover = false,
  padding,
  ...rest
}) => {
  const paddingClass = padding ? (paddingClasses[padding] ?? '') : '';

  return (
    <div
      className={`rounded-xl ${variantStyles[variant]} ${hover ? hoverStyles : ''} ${paddingClass} ${className}`}
      {...rest}
    >
      {children}
    </div>
  );
};

// ============================================================================
// Enhanced Card with Status (new - matches Seed pattern)
// ============================================================================

interface StatusCardProps {
  /** Card title (required) */
  title: string;
  /** Optional subtitle or description */
  subtitle?: string;
  /** Status indicator (success, warning, error, loading, unknown, info) */
  status: Status;
  /** Card content */
  children: ReactNode;
  /** Additional CSS classes */
  className?: string;
  /** Callback when card is clicked */
  onClick?: () => void;
  /** Optional icon displayed in header */
  icon?: ReactNode;
  /** Optional action element in header (right side) */
  headerAction?: ReactNode;
  /** Enable aria-live region for dynamic content updates */
  enableLiveRegion?: boolean;
  /** ARIA label for the card */
  ariaLabel?: string;
  /** Card visual variant */
  variant?: CardVariant;
}

/**
 * StatusCard - Card with title, status badge, and accessibility features.
 *
 * Use this for data-driven cards that need status indication.
 */
export const StatusCard: FC<StatusCardProps> = ({
  title,
  subtitle,
  status,
  children,
  className = '',
  onClick,
  icon,
  headerAction,
  enableLiveRegion = false,
  ariaLabel,
  variant = 'default',
}) => {
  const isInteractive = typeof onClick === 'function';

  const handleKeyDown = (e: KeyboardEvent<HTMLDivElement>): void => {
    if (!isInteractive) return;
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      onClick?.();
    }
  };

  const ariaProps = {
    'aria-label': ariaLabel ?? `${title}${subtitle ? ` - ${subtitle}` : ''}`,
    ...(enableLiveRegion && {
      'aria-live': 'polite' as const,
      'aria-atomic': 'true' as const,
    }),
  };

  const titleId = `card-title-${title.replace(/\s+/g, '-').toLowerCase()}`;

  return (
    // biome-ignore lint/a11y/noStaticElementInteractions: Interactive role is conditionally applied based on onClick presence
    <div
      className={`rounded-xl ${variantStyles[variant]} pad sm:pad-lg
        transition-all hover:border-surface-border touch-manipulation
        focus-visible:ring-2 focus-visible:ring-brand-primary focus-visible:ring-offset-2 focus-visible:ring-offset-surface-base outline-none
        ${isInteractive ? 'cursor-pointer active:scale-[0.98]' : ''}
        ${className}`}
      onClick={onClick}
      onKeyDown={handleKeyDown}
      role={isInteractive ? 'button' : undefined}
      tabIndex={isInteractive ? 0 : undefined}
      {...ariaProps}
    >
      {/* Header */}
      <div className="flex-between">
        <div className="flex items-center gap-default">
          {icon && (
            <span className="text-text-muted shrink-0 w-5 h-5" aria-hidden="true">
              {icon}
            </span>
          )}
          <div className="flex flex-col">
            <h3 className="text-base font-semibold text-text-primary" id={titleId}>
              {title}
            </h3>
            {subtitle && <p className="text-xs text-text-muted leading-tight">{subtitle}</p>}
          </div>
        </div>
        <div className="flex items-center gap-compact">
          {headerAction}
          <StatusBadge status={status} size="md" />
        </div>
      </div>

      {/* Content */}
      <div className="mt-content" aria-describedby={titleId}>
        {children}
      </div>
    </div>
  );
};

// ============================================================================
// Card Sub-components
// ============================================================================

interface CardContentProps {
  children: ReactNode;
  className?: string;
}

export const CardContent: FC<CardContentProps> = ({ children, className = '' }) => (
  <div className={`pad-lg ${className}`}>{children}</div>
);

interface CardHeaderProps {
  children: ReactNode;
  className?: string;
  actions?: ReactNode;
}

export const CardHeader: FC<CardHeaderProps> = ({ children, className = '', actions }) => (
  <div className={`flex-between px-6 py-4 border-b border-surface-border ${className}`}>
    <div className="flex items-center gap-default">{children}</div>
    {actions && <div className="flex items-center gap-compact">{actions}</div>}
  </div>
);

interface CardFooterProps {
  children: ReactNode;
  className?: string;
}

export const CardFooter: FC<CardFooterProps> = ({ children, className = '' }) => (
  <div className={`px-6 py-4 border-t border-surface-border bg-scrim/20 rounded-b-xl ${className}`}>
    {children}
  </div>
);

// ============================================================================
// Card Value & Row Components (matches Seed pattern)
// ============================================================================

interface CardValueProps {
  /** Optional label above the value */
  label?: string;
  /** The value to display */
  value: string | number;
  /** Optional unit suffix */
  unit?: string;
  /** Text size */
  size?: 'sm' | 'md' | 'lg';
  /** Status-based coloring */
  status?: Status;
  /** Use monospace font */
  mono?: boolean;
}

const valueSizeClasses = {
  sm: 'text-sm',
  md: 'heading-4',
  lg: 'heading-3',
};

/**
 * CardValue - Displays a prominent value with optional label and status.
 */
export const CardValue: FC<CardValueProps> = ({
  label,
  value,
  unit,
  size = 'md',
  status,
  mono = false,
}) => {
  const statusColor = status
    ? {
        success: 'text-status-success',
        warning: 'text-status-warning',
        error: 'text-status-error',
        info: 'text-status-info',
        unknown: 'text-text-muted',
        loading: 'text-status-info',
      }[status]
    : 'text-text-primary';

  return (
    <div>
      {label && <p className="text-xs text-text-muted mb-tight">{label}</p>}
      <p
        className={`${valueSizeClasses[size]} ${statusColor} ${mono ? 'font-mono tabular-nums' : ''}`}
        data-testid="card-value"
      >
        <span>{value}</span>
        {unit && <span className="text-sm font-normal text-text-muted ml-tight">{unit}</span>}
      </p>
    </div>
  );
};

interface CardRowProps {
  /** Row label */
  label: string;
  /** Row value */
  value: string | number;
  /** Status-based coloring for the value */
  status?: Status;
  /** Use monospace font for value */
  mono?: boolean;
}

/**
 * CardRow - Displays a label-value pair in a horizontal row.
 */
export const CardRow: FC<CardRowProps> = ({ label, value, status, mono = false }) => {
  const statusColor = status
    ? {
        success: 'text-status-success',
        warning: 'text-status-warning',
        error: 'text-status-error',
        info: 'text-status-info',
        unknown: 'text-text-muted',
        loading: 'text-status-info',
      }[status]
    : 'text-text-primary';

  return (
    <div className="flex justify-between items-center py-compact">
      <span className="text-sm text-text-muted shrink-0">{label}</span>
      <span
        className={`text-sm font-medium ${statusColor} ${mono ? 'font-mono tabular-nums' : ''} truncate text-right`}
        title={String(value)}
        data-testid="card-row-value"
      >
        {value}
      </span>
    </div>
  );
};

/**
 * CardDivider - Horizontal divider for separating card sections.
 */
export const CardDivider: FC<{ className?: string }> = ({ className = '' }) => (
  <hr className={`border-surface-border my-3 ${className}`} />
);

// ============================================================================
// Stat Card (preserved from original)
// ============================================================================

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

export const StatCard: FC<StatCardProps> = ({ label, value, icon, trend, className = '' }) => {
  const trendColor = trend
    ? trend.value > 0
      ? 'text-status-success'
      : trend.value < 0
        ? 'text-status-error'
        : 'text-text-muted'
    : '';

  const trendIcon = trend ? (trend.value > 0 ? '↑' : trend.value < 0 ? '↓' : '→') : '';

  return (
    <Card className={className} hover={true}>
      <CardContent className="stack-sm">
        <div className="flex-between">
          <span className="text-sm font-medium text-text-muted">{label}</span>
          {icon && <span className="text-brand-accent">{icon}</span>}
        </div>
        <div className="flex items-end justify-between gap-compact">
          <span className="text-3xl font-bold text-text-primary">{value}</span>
          {trend && (
            <span className={`text-sm font-medium ${trendColor}`}>
              {trendIcon} {Math.abs(trend.value)}%
              {trend.label && <span className="text-text-muted ml-tight">{trend.label}</span>}
            </span>
          )}
        </div>
      </CardContent>
    </Card>
  );
};
