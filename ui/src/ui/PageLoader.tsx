import type { FC } from 'react';

/**
 * Suspense fallback for lazy-loaded routed pages. Sized to match a
 * typical page header so the layout doesn't jump when the chunk lands.
 */
export const PageLoader: FC = () => (
  <div class="flex items-center justify-center min-h-[400px]">
    <div class="flex flex-col items-center gap-3">
      <div class="h-8 w-8 animate-spin rounded-full border-4 border-brand-primary border-t-transparent" />
      <p class="text-sm text-text-muted">Loading...</p>
    </div>
  </div>
);
