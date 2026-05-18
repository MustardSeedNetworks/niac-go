import type { FC } from 'react';

interface SkeletonProps {
  className?: string;
  variant?: 'text' | 'circular' | 'rectangular';
  width?: string | number;
  height?: string | number;
  lines?: number;
}

export const Skeleton: FC<SkeletonProps> = ({
  className = '',
  variant = 'text',
  width,
  height,
  lines = 1,
}) => {
  const baseClass =
    'skeleton bg-gradient-to-r from-white/5 via-white/10 to-white/5 bg-[length:200%_100%] animate-shimmer';

  const variantClasses = {
    text: 'h-4 rounded',
    circular: 'rounded-full',
    rectangular: 'rounded-lg',
  };

  const style = {
    width: width ? (typeof width === 'number' ? `${width}px` : width) : undefined,
    height: height ? (typeof height === 'number' ? `${height}px` : height) : undefined,
  };

  if (lines > 1) {
    const lineKeys = Array.from({ length: lines }, (_, idx) => `line-${idx}`);
    return (
      <div className={`space-y-2 ${className}`}>
        {lineKeys.map((lineKey, i) => (
          <div
            key={lineKey}
            className={`${baseClass} ${variantClasses[variant]}`}
            style={{
              ...style,
              width: i === lines - 1 ? '75%' : style.width,
            }}
          />
        ))}
      </div>
    );
  }

  return <div className={`${baseClass} ${variantClasses[variant]} ${className}`} style={style} />;
};

// Skeleton for card content
export const CardSkeleton: FC<{ className?: string }> = ({ className = '' }) => (
  <div className={`p-6 space-y-4 ${className}`}>
    <div className="flex items-center gap-3">
      <Skeleton variant="circular" width={40} height={40} />
      <div className="flex-1 space-y-2">
        <Skeleton width="40%" />
        <Skeleton width="60%" />
      </div>
    </div>
    <Skeleton lines={3} />
  </div>
);

// Skeleton for table rows
export const TableRowSkeleton: FC<{ columns?: number; className?: string }> = ({
  columns = 4,
  className = '',
}) => (
  <div className={`flex items-center gap-4 p-4 ${className}`}>
    {Array.from({ length: columns }, (_, idx) => `col-${idx}`).map((colKey, i) => (
      <Skeleton key={colKey} className="flex-1" width={i === 0 ? '30%' : undefined} />
    ))}
  </div>
);

// Skeleton for stat cards
export const StatCardSkeleton: FC<{ className?: string }> = ({ className = '' }) => (
  <div className={`p-6 space-y-3 ${className}`}>
    <div className="flex items-center justify-between">
      <Skeleton width="40%" />
      <Skeleton variant="circular" width={24} height={24} />
    </div>
    <Skeleton width="60%" height={32} />
  </div>
);

// Skeleton for device cards (card view)
export const DeviceCardSkeleton: FC<{ className?: string }> = ({ className = '' }) => (
  <div className={`p-4 space-y-3 rounded-xl border border-white/5 bg-bg-surface/70 ${className}`}>
    {/* Header with checkbox and icon */}
    <div className="flex items-start justify-between">
      <div className="flex items-center gap-3">
        <Skeleton variant="rectangular" width={16} height={16} />
        <Skeleton variant="rectangular" width={36} height={36} className="rounded-lg" />
      </div>
      <Skeleton width={60} height={20} className="rounded" />
    </div>
    {/* Name and IP */}
    <div className="space-y-2">
      <Skeleton width="70%" height={20} />
      <Skeleton width="50%" height={16} />
    </div>
    {/* MAC */}
    <Skeleton width="80%" height={12} />
    {/* Protocols */}
    <div className="flex gap-1">
      <Skeleton width={40} height={20} className="rounded" />
      <Skeleton width={40} height={20} className="rounded" />
      <Skeleton width={40} height={20} className="rounded" />
    </div>
    {/* Actions */}
    <div className="flex justify-end gap-1 pt-2 border-t border-white/5">
      <Skeleton variant="rectangular" width={32} height={32} className="rounded-lg" />
      <Skeleton variant="rectangular" width={32} height={32} className="rounded-lg" />
      <Skeleton variant="rectangular" width={32} height={32} className="rounded-lg" />
    </div>
  </div>
);

// Skeleton for device table rows
export const DeviceTableRowSkeleton: FC<{ className?: string }> = ({ className = '' }) => (
  <div className={`flex items-center gap-4 px-4 py-3 border-b border-white/5 ${className}`}>
    <Skeleton variant="rectangular" width={16} height={16} />
    <div className="flex-1 grid grid-cols-12 gap-4 items-center">
      {/* Hostname */}
      <div className="col-span-3 flex items-center gap-2">
        <Skeleton variant="rectangular" width={16} height={16} />
        <Skeleton width="70%" />
      </div>
      {/* Type */}
      <div className="col-span-2">
        <Skeleton width={60} height={20} className="rounded" />
      </div>
      {/* IP */}
      <div className="col-span-2">
        <Skeleton width="80%" />
      </div>
      {/* Protocols */}
      <div className="col-span-3 flex gap-1">
        <Skeleton width={40} height={20} className="rounded" />
        <Skeleton width={40} height={20} className="rounded" />
      </div>
      {/* Actions */}
      <div className="col-span-2 flex justify-end gap-1">
        <Skeleton variant="rectangular" width={32} height={32} className="rounded-lg" />
        <Skeleton variant="rectangular" width={32} height={32} className="rounded-lg" />
        <Skeleton variant="rectangular" width={32} height={32} className="rounded-lg" />
      </div>
    </div>
  </div>
);

// Multiple table rows skeleton
export const DeviceTableSkeleton: FC<{ rows?: number; className?: string }> = ({
  rows = 5,
  className = '',
}) => (
  <div className={className}>
    {Array.from({ length: rows }, (_, idx) => `row-${idx}`).map((rowKey) => (
      <DeviceTableRowSkeleton key={rowKey} />
    ))}
  </div>
);

// Multiple device cards skeleton
export const DeviceCardGridSkeleton: FC<{
  count?: number;
  className?: string;
}> = ({ count = 8, className = '' }) => (
  <div
    className={`grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4 ${className}`}
  >
    {Array.from({ length: count }, (_, idx) => `card-${idx}`).map((cardKey) => (
      <DeviceCardSkeleton key={cardKey} />
    ))}
  </div>
);
