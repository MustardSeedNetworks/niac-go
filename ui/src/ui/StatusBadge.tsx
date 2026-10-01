/**
 * StatusBadge Component
 *
 * Displays system status with visual indicators (icon or dot).
 * Provides consistent color-coding and symbols across the UI.
 *
 * Features:
 * - Variants: "icon" (large symbol) or "dot" (small indicator)
 * - Sizes: "sm", "md", "lg"
 * - Status types: success, warning, error, info, unknown, loading
 * - Accessible with proper ARIA labels
 *
 * Usage:
 * ```tsx
 * <StatusBadge status="success" variant="icon" />
 * <StatusBadge status="warning" variant="dot" size="sm" />
 * ```
 */

import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import { getSizeConfig, getStatusConfig, type SizeKey, type Status } from './StatusConfig';

export type { Status };

interface StatusBadgeProps {
  /** Status type determines color and icon */
  status: Status;
  /** Visual variant - icon shows full symbol, dot shows small indicator */
  variant?: 'icon' | 'dot';
  /** Size of the badge */
  size?: SizeKey;
  /** Additional CSS classes */
  className?: string;
}

/**
 * StatusBadge - Unified status indicator component
 *
 * Variants:
 * - icon: Shows checkmark/triangle/X icon (default)
 * - dot: Shows small colored dot
 */
export const StatusBadge: FC<StatusBadgeProps> = ({
  status,
  variant = 'icon',
  size = 'md',
  className = '',
}) => {
  const { t } = useTranslation('common');
  const config = getStatusConfig(status);
  const labels: Record<Status, string> = {
    success: t('statusLabel.success'),
    warning: t('statusLabel.warning'),
    error: t('statusLabel.error'),
    unknown: t('statusLabel.unknown'),
    loading: t('statusLabel.loading'),
  };
  const label = labels[status];
  const sizes = getSizeConfig(size);

  if (variant === 'dot') {
    // Replace /15 opacity with solid color for dot variant
    const dotBgColor = config.bgColor.replace('/15', '');

    return (
      <span
        className={`inline-block rounded-full ${sizes.dot} ${dotBgColor} ${className}`}
        role="img"
        aria-label={label}
      />
    );
  }

  return (
    <span
      className={`inline-flex items-center justify-center rounded-full ${config.color} ${config.bgColor} ${sizes.padding} ${className}`}
      role="img"
      aria-label={label}
    >
      <span className={sizes.icon} aria-hidden="true">
        {config.icon}
      </span>
    </span>
  );
};
