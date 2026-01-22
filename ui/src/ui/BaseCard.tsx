// Copyright (c) 2025 Mustard Seed Networks. All rights reserved.

/**
 * BaseCard Component
 *
 * Data-driven card wrapper with built-in state handling.
 * Provides consistent loading, error, and empty state patterns.
 *
 * Pattern ported from Seed UI for NIAC harmonization.
 *
 * Features:
 * - Generic type support for type-safe data rendering
 * - Loading state with skeleton or custom content
 * - Error state with message display
 * - Empty/no-data state handling
 * - Status derivation from data via getStatus callback
 * - Full accessibility with ARIA live regions
 *
 * Usage:
 * ```tsx
 * <BaseCard
 *   title="Device Status"
 *   data={deviceData}
 *   loading={isLoading}
 *   error={error}
 *   getStatus={(data) => data.online ? 'success' : 'error'}
 * >
 *   {(data) => (
 *     <>
 *       <CardValue value={data.name} />
 *       <CardRow label="IP" value={data.ip} />
 *     </>
 *   )}
 * </BaseCard>
 * ```
 */

import type React from 'react';
import type { FC, ReactNode } from 'react';
import { CardValue, type Status, StatusCard } from './Card';
import { Skeleton } from './Skeleton';

interface BaseCardProps<T> {
  /** Card title */
  title: string;
  /** Optional subtitle */
  subtitle?: string;
  /** Optional icon in header */
  icon?: ReactNode;
  /** The data to render (null = no data state) */
  data: T | null;
  /** Loading state flag */
  loading?: boolean;
  /** Error message (if any) */
  error?: string | null;
  /** Function to derive status from data */
  getStatus: (data: T) => Status;
  /** Render function - receives data when available */
  children: (data: T) => ReactNode;
  /** Custom loading content (default: skeleton) */
  loadingContent?: ReactNode;
  /** Custom empty message */
  emptyMessage?: string;
  /** Additional CSS classes */
  className?: string;
  /** Click handler for interactive cards */
  onClick?: () => void;
}

/**
 * Default loading skeleton for cards.
 * Can be overridden with loadingContent prop.
 */
const DefaultLoadingSkeleton: FC = () => (
  <div className="space-y-3">
    <Skeleton width="50%" height={24} />
    <div className="space-y-2">
      <div className="flex justify-between">
        <Skeleton width="30%" />
        <Skeleton width="40%" />
      </div>
      <div className="flex justify-between">
        <Skeleton width="25%" />
        <Skeleton width="35%" />
      </div>
      <div className="flex justify-between">
        <Skeleton width="35%" />
        <Skeleton width="30%" />
      </div>
    </div>
  </div>
);

/**
 * BaseCard - Generic data-driven card with state handling.
 *
 * State priority: loading > error > no data > data
 *
 * Type parameter T represents the data shape this card displays.
 */
export function BaseCard<T>({
  title,
  subtitle,
  icon,
  data,
  loading = false,
  error = null,
  getStatus,
  children,
  loadingContent,
  emptyMessage = 'No data available',
  className,
  onClick,
}: BaseCardProps<T>): React.JSX.Element {
  // Loading state (highest priority)
  if (loading) {
    return (
      <StatusCard
        title={title}
        subtitle={subtitle}
        icon={icon}
        status="loading"
        className={className}
        enableLiveRegion={true}
      >
        {loadingContent ?? <DefaultLoadingSkeleton />}
      </StatusCard>
    );
  }

  // Error state
  if (error) {
    return (
      <StatusCard
        title={title}
        subtitle={subtitle}
        icon={icon}
        status="error"
        className={className}
        enableLiveRegion={true}
      >
        <CardValue value="Error" size="md" status="error" />
        <p className="text-sm text-red-400/80 mt-1">{error}</p>
      </StatusCard>
    );
  }

  // No data state
  if (data === null || data === undefined) {
    return (
      <StatusCard
        title={title}
        subtitle={subtitle}
        icon={icon}
        status="unknown"
        className={className}
        enableLiveRegion={true}
      >
        <CardValue value={emptyMessage} size="md" />
      </StatusCard>
    );
  }

  // Normal state with data
  const status = getStatus(data);

  return (
    <StatusCard
      title={title}
      subtitle={subtitle}
      icon={icon}
      status={status}
      className={className}
      onClick={onClick}
      enableLiveRegion={true}
    >
      {children(data)}
    </StatusCard>
  );
}

// ============================================================================
// SimpleBaseCard - For cards without data loading logic
// ============================================================================

interface SimpleBaseCardProps {
  /** Card title */
  title: string;
  /** Optional subtitle */
  subtitle?: string;
  /** Optional icon in header */
  icon?: ReactNode;
  /** Pre-determined status */
  status: Status;
  /** Loading state flag */
  loading?: boolean;
  /** Error message (if any) */
  error?: string | null;
  /** Card content */
  children: ReactNode;
  /** Custom loading content */
  loadingContent?: ReactNode;
  /** Additional CSS classes */
  className?: string;
  /** Click handler */
  onClick?: () => void;
}

/**
 * SimpleBaseCard - Card wrapper for static content with loading/error handling.
 *
 * Use this when you have a pre-determined status and don't need the
 * render prop pattern of BaseCard.
 */
export const SimpleBaseCard: FC<SimpleBaseCardProps> = ({
  title,
  subtitle,
  icon,
  status,
  loading = false,
  error = null,
  children,
  loadingContent,
  className,
  onClick,
}) => {
  // Loading state
  if (loading) {
    return (
      <StatusCard
        title={title}
        subtitle={subtitle}
        icon={icon}
        status="loading"
        className={className}
        enableLiveRegion={true}
      >
        {loadingContent ?? <DefaultLoadingSkeleton />}
      </StatusCard>
    );
  }

  // Error state
  if (error) {
    return (
      <StatusCard
        title={title}
        subtitle={subtitle}
        icon={icon}
        status="error"
        className={className}
        enableLiveRegion={true}
      >
        <CardValue value="Error" size="md" status="error" />
        <p className="text-sm text-red-400/80 mt-1">{error}</p>
      </StatusCard>
    );
  }

  // Normal state
  return (
    <StatusCard
      title={title}
      subtitle={subtitle}
      icon={icon}
      status={status}
      className={className}
      onClick={onClick}
      enableLiveRegion={true}
    >
      {children}
    </StatusCard>
  );
};
