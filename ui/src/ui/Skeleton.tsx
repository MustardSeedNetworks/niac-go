interface SkeletonProps {
  className?: string;
  variant?: 'text' | 'circular' | 'rectangular';
  width?: string | number;
  height?: string | number;
  lines?: number;
}

export function Skeleton({
  className = '',
  variant = 'text',
  width,
  height,
  lines = 1,
}: SkeletonProps) {
  const baseClass =
    'skeleton bg-gradient-to-r from-knob/5 via-knob/10 to-knob/5 bg-[length:200%_100%] animate-shimmer';

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
      <div className={`stack-sm ${className}`}>
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
}

// Skeleton for card content
export function CardSkeleton({ className = '' }: { className?: string }) {
  return (
    <div className={`pad-lg stack-lg ${className}`}>
      <div className="flex items-center gap-default">
        <Skeleton variant="circular" width={40} height={40} />
        <div className="flex-1 stack-sm">
          <Skeleton width="40%" />
          <Skeleton width="60%" />
        </div>
      </div>
      <Skeleton lines={3} />
    </div>
  );
}

// Skeleton for table rows
export function TableRowSkeleton({
  columns = 4,
  className = '',
}: {
  columns?: number;
  className?: string;
}) {
  return (
    <div className={`flex items-center gap-comfortable pad ${className}`}>
      {Array.from({ length: columns }, (_, idx) => `col-${idx}`).map((colKey, i) => (
        <Skeleton key={colKey} className="flex-1" width={i === 0 ? '30%' : undefined} />
      ))}
    </div>
  );
}

// Skeleton for stat cards
export function StatCardSkeleton({ className = '' }: { className?: string }) {
  return (
    <div className={`pad-lg stack ${className}`}>
      <div className="flex-between">
        <Skeleton width="40%" />
        <Skeleton variant="circular" width={24} height={24} />
      </div>
      <Skeleton width="60%" height={32} />
    </div>
  );
}

// Skeleton for device cards (card view)
export function DeviceCardSkeleton({ className = '' }: { className?: string }) {
  return (
    <div
      className={`pad stack rounded-xl border border-surface-border bg-bg-surface/70 ${className}`}
    >
      {/* Header with checkbox and icon */}
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-default">
          <Skeleton variant="rectangular" width={16} height={16} />
          <Skeleton variant="rectangular" width={36} height={36} className="rounded-lg" />
        </div>
        <Skeleton width={60} height={20} className="rounded" />
      </div>
      {/* Name and IP */}
      <div className="stack-sm">
        <Skeleton width="70%" height={20} />
        <Skeleton width="50%" height={16} />
      </div>
      {/* MAC */}
      <Skeleton width="80%" height={12} />
      {/* Protocols */}
      <div className="flex gap-tight">
        <Skeleton width={40} height={20} className="rounded" />
        <Skeleton width={40} height={20} className="rounded" />
        <Skeleton width={40} height={20} className="rounded" />
      </div>
      {/* Actions */}
      <div className="flex justify-end gap-tight pt-2 border-t border-surface-border">
        <Skeleton variant="rectangular" width={32} height={32} className="rounded-lg" />
        <Skeleton variant="rectangular" width={32} height={32} className="rounded-lg" />
        <Skeleton variant="rectangular" width={32} height={32} className="rounded-lg" />
      </div>
    </div>
  );
}

// Skeleton for device table rows
export function DeviceTableRowSkeleton({ className = '' }: { className?: string }) {
  return (
    <div
      className={`flex items-center gap-comfortable px-4 py-row-lg border-b border-surface-border ${className}`}
    >
      <Skeleton variant="rectangular" width={16} height={16} />
      <div className="flex-1 grid grid-cols-12 gap-comfortable items-center">
        {/* Hostname */}
        <div className="col-span-3 flex items-center gap-compact">
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
        <div className="col-span-3 flex gap-tight">
          <Skeleton width={40} height={20} className="rounded" />
          <Skeleton width={40} height={20} className="rounded" />
        </div>
        {/* Actions */}
        <div className="col-span-2 flex justify-end gap-tight">
          <Skeleton variant="rectangular" width={32} height={32} className="rounded-lg" />
          <Skeleton variant="rectangular" width={32} height={32} className="rounded-lg" />
          <Skeleton variant="rectangular" width={32} height={32} className="rounded-lg" />
        </div>
      </div>
    </div>
  );
}

// Multiple table rows skeleton
export function DeviceTableSkeleton({
  rows = 5,
  className = '',
}: {
  rows?: number;
  className?: string;
}) {
  return (
    <div className={className}>
      {Array.from({ length: rows }, (_, idx) => `row-${idx}`).map((rowKey) => (
        <DeviceTableRowSkeleton key={rowKey} />
      ))}
    </div>
  );
}

// Multiple device cards skeleton
export function DeviceCardGridSkeleton({
  count = 8,
  className = '',
}: {
  count?: number;
  className?: string;
}) {
  return (
    <div
      className={`grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-comfortable ${className}`}
    >
      {Array.from({ length: count }, (_, idx) => `card-${idx}`).map((cardKey) => (
        <DeviceCardSkeleton key={cardKey} />
      ))}
    </div>
  );
}
