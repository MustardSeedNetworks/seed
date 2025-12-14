/**
 * Skeleton Component
 *
 * Purpose: Provides reusable loading placeholder components for data that hasn't loaded yet.
 * Uses CSS animation to create a pulsing skeleton effect while content is being fetched.
 *
 * Key Features:
 * - Multiple variants: text (rounded), circular (for avatars), rectangular (for images/blocks)
 * - Flexible sizing via width/height props (accepts px numbers or string values)
 * - CardSkeleton: Pre-configured skeleton for card layouts with title, rows, and value skeletons
 * - Accessible: Uses aria-hidden="true" to hide from screen readers during loading
 *
 * Usage:
 * ```typescript
 * // Text skeleton (for paragraphs)
 * <Skeleton variant="text" className="h-4 w-32" />
 *
 * // Circular skeleton (for avatars)
 * <Skeleton variant="circular" width={40} height={40} />
 *
 * // Rectangular skeleton (for images)
 * <Skeleton variant="rectangular" width={200} height={150} />
 *
 * // Full card skeleton
 * <CardSkeleton />
 * ```
 *
 * Dependencies: theme utilities (cn, radius, card, layout), React
 * State: None - purely presentational component
 */

import { cn, radius, card, layout } from "../../styles/theme";

interface SkeletonProps {
  className?: string;
  variant?: "text" | "circular" | "rectangular";
  width?: string | number;
  height?: string | number;
}

export function Skeleton({
  className = "",
  variant = "text",
  width,
  height,
}: SkeletonProps) {
  const baseClasses = "animate-pulse bg-surface-hover";

  const variantClasses = {
    text: radius.default,
    circular: radius.full,
    rectangular: radius.lg,
  };

  const sizeClasses = [
    width
      ? typeof width === "number"
        ? `w-[${width}px]`
        : `w-[${width}]`
      : "",
    height
      ? typeof height === "number"
        ? `h-[${height}px]`
        : `h-[${height}]`
      : "",
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <div
      className={cn(baseClasses, variantClasses[variant], sizeClasses, className)}
      aria-hidden="true"
    />
  );
}

export function CardSkeleton() {
  return (
    <div className={cn(card.base, card.variant.default, card.padding.md)}>
      <div className={cn(layout.flex.between, "mb-3")}>
        <Skeleton className="h-4 w-24" />
        <Skeleton variant="circular" className="h-3 w-3" />
      </div>
      <Skeleton className="h-8 w-32 mb-2" />
      <div className="stack-sm mt-4">
        <div className={layout.flex.between}>
          <Skeleton className="h-3 w-16" />
          <Skeleton className="h-3 w-20" />
        </div>
        <div className={layout.flex.between}>
          <Skeleton className="h-3 w-12" />
          <Skeleton className="h-3 w-16" />
        </div>
      </div>
    </div>
  );
}

export function TextSkeleton({ lines = 3 }: { lines?: number }) {
  return (
    <div className="stack-sm">
      {Array.from({ length: lines }).map((_, i) => (
        <Skeleton
          key={i}
          className="h-4"
          width={i === lines - 1 ? "60%" : "100%"}
        />
      ))}
    </div>
  );
}
