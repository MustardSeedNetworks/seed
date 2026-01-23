import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { cn, layout, spacing } from '../../styles/theme';
import { Card, CardValue, type Status } from '../ui/card';
import { Skeleton } from '../ui/skeleton';

interface BaseCardProps<T> {
  title: string;
  subtitle?: string;
  icon?: ReactNode;
  data: T | null;
  loading?: boolean;
  error?: string | null;
  getStatus: (data: T) => Status;
  children: (data: T) => ReactNode;
  loadingContent?: ReactNode;
  emptyMessage?: string;
  class?: string;
  onClick?: () => void;
}

/**
 * BaseCard provides a consistent wrapper for data-driven cards with:
 * - Loading state handling (skeleton or custom loading content)
 * - Error state display
 * - Empty/no-data state
 * - Status derivation from data
 *
 * Usage:
 * ```tsx
 * <BaseCard
 *   title="My Card"
 *   data={myData}
 *   loading={isLoading}
 *   getStatus={(data) => data.isHealthy ? 'success' : 'error'}
 * >
 *   {(data) => (
 *     <>
 *       <CardValue value={data.value} />
 *       <CardRow label="Label" value={data.info} />
 *     </>
 *   )}
 * </BaseCard>
 * ```
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
  emptyMessage,
  class: className,
  onClick,
}: BaseCardProps<T>): JSX.Element {
  const { t } = useTranslation('common');
  const resolvedEmptyMessage = emptyMessage ?? t('status.noDataAvailable');

  // Loading state
  if (loading) {
    return (
      <Card
        title={title}
        subtitle={subtitle}
        icon={icon}
        status="loading"
        class={className}
        enableLiveRegion={true}
      >
        {loadingContent || <DefaultLoadingSkeleton />}
      </Card>
    );
  }

  // Error state
  if (error) {
    return (
      <Card
        title={title}
        subtitle={subtitle}
        icon={icon}
        status="error"
        class={className}
        enableLiveRegion={true}
      >
        <CardValue value={t('status.error')} size="md" status="error" />
        <p class={cn('caption text-status-error', spacing.margin.top.tight)}>{error}</p>
      </Card>
    );
  }

  // No data state
  if (!data) {
    return (
      <Card
        title={title}
        subtitle={subtitle}
        icon={icon}
        status="unknown"
        class={className}
        enableLiveRegion={true}
      >
        <CardValue value={resolvedEmptyMessage} size="md" />
      </Card>
    );
  }

  // Normal state with data (fixes #674: enable live region for dynamic updates)
  const status = getStatus(data);

  return (
    <Card
      title={title}
      subtitle={subtitle}
      icon={icon}
      status={status}
      class={className}
      onClick={onClick}
      enableLiveRegion={true}
    >
      {children(data)}
    </Card>
  );
}

/**
 * Default skeleton for loading state.
 * Cards can override with custom loadingContent prop.
 */
function DefaultLoadingSkeleton(): JSX.Element {
  return (
    <>
      <Skeleton class={cn('h-8 w-32', spacing.margin.bottom.heading)} />
      <div class={cn(spacing.stack.sm, spacing.margin.top.content)}>
        <div class={layout.flex.between}>
          <Skeleton class="h-3 w-16" />
          <Skeleton class="h-3 w-20" />
        </div>
        <div class={layout.flex.between}>
          <Skeleton class="h-3 w-12" />
          <Skeleton class="h-3 w-16" />
        </div>
        <div class={layout.flex.between}>
          <Skeleton class="h-3 w-20" />
          <Skeleton class="h-3 w-12" />
        </div>
      </div>
    </>
  );
}

/**
 * Simpler BaseCard variant for cards that just need loading/error handling
 * without the render prop pattern.
 */
interface SimpleBaseCardProps {
  title: string;
  subtitle?: string;
  icon?: ReactNode;
  status: Status;
  loading?: boolean;
  error?: string | null;
  children: ReactNode;
  loadingContent?: ReactNode;
  class?: string;
  onClick?: () => void;
}

/**
 * Simplified card wrapper for basic display without data loading logic.
 */
export function SimpleBaseCard({
  title,
  subtitle,
  icon,
  status,
  loading = false,
  error = null,
  children,
  loadingContent,
  class: className,
  onClick,
}: SimpleBaseCardProps): JSX.Element {
  const { t } = useTranslation('common');

  // Loading state (fixes #674: enable live region for dynamic updates)
  if (loading) {
    return (
      <Card
        title={title}
        subtitle={subtitle}
        icon={icon}
        status="loading"
        class={className}
        enableLiveRegion={true}
      >
        {loadingContent || <DefaultLoadingSkeleton />}
      </Card>
    );
  }

  // Error state (fixes #674: enable live region for dynamic updates)
  if (error) {
    return (
      <Card
        title={title}
        subtitle={subtitle}
        icon={icon}
        status="error"
        class={className}
        enableLiveRegion={true}
      >
        <CardValue value={t('status.error')} size="md" status="error" />
        <p class={cn('caption text-status-error', spacing.margin.top.tight)}>{error}</p>
      </Card>
    );
  }

  return (
    <Card
      title={title}
      subtitle={subtitle}
      icon={icon}
      status={status}
      class={className}
      onClick={onClick}
      enableLiveRegion={true}
    >
      {children}
    </Card>
  );
}
