import type { FC } from 'react';

/**
 * PageLoader is the Suspense fallback for lazy-loaded route components.
 * Sized to roughly match a typical page header so the layout doesn't
 * jump when a chunk finishes loading.
 */
export const PageLoader: FC = () => (
  <div className="flex items-center justify-center min-h-[400px]">
    <div className="flex flex-col items-center gap-3">
      <div className="h-8 w-8 animate-spin rounded-full border-4 border-violet-500 border-t-transparent" />
      <p className="text-sm text-gray-400">Loading...</p>
    </div>
  </div>
);
