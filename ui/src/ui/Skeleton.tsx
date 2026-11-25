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
