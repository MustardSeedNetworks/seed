import { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { Card, CardValue, Status } from "../ui/Card";
import { Skeleton } from "../ui/Skeleton";
import { layout, spacing } from "../../styles/theme";

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
  className?: string;
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
  className,
  onClick,
}: BaseCardProps<T>) {
  const { t } = useTranslation("common");
  const resolvedEmptyMessage = emptyMessage ?? t("status.noDataAvailable");

  // Loading state
  if (loading) {
    return (
      <Card title={title} subtitle={subtitle} icon={icon} status="loading" className={className}>
        {loadingContent || <DefaultLoadingSkeleton />}
      </Card>
    );
  }

  // Error state
  if (error) {
    return (
      <Card title={title} subtitle={subtitle} icon={icon} status="error" className={className}>
        <CardValue value={t("status.error")} size="md" status="error" />
        <p className="caption text-status-error mt-1">{error}</p>
      </Card>
    );
  }

  // No data state
  if (!data) {
    return (
      <Card title={title} subtitle={subtitle} icon={icon} status="unknown" className={className}>
        <CardValue value={resolvedEmptyMessage} size="md" />
      </Card>
    );
  }

  // Normal state with data
  const status = getStatus(data);

  return (
    <Card
      title={title}
      subtitle={subtitle}
      icon={icon}
      status={status}
      className={className}
      onClick={onClick}
    >
      {children(data)}
    </Card>
  );
}

/**
 * Default skeleton for loading state.
 * Cards can override with custom loadingContent prop.
 */
function DefaultLoadingSkeleton() {
  return (
    <>
      <Skeleton className={`h-8 w-32 ${spacing.margin.bottom.heading}`} />
      <div className={`${spacing.stack.sm} ${spacing.margin.top.content}`}>
        <div className={layout.flex.between}>
          <Skeleton className="h-3 w-16" />
          <Skeleton className="h-3 w-20" />
        </div>
        <div className={layout.flex.between}>
          <Skeleton className="h-3 w-12" />
          <Skeleton className="h-3 w-16" />
        </div>
        <div className={layout.flex.between}>
          <Skeleton className="h-3 w-20" />
          <Skeleton className="h-3 w-12" />
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
  className?: string;
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
  className,
  onClick,
}: SimpleBaseCardProps) {
  const { t } = useTranslation("common");

  // Loading state
  if (loading) {
    return (
      <Card title={title} subtitle={subtitle} icon={icon} status="loading" className={className}>
        {loadingContent || <DefaultLoadingSkeleton />}
      </Card>
    );
  }

  // Error state
  if (error) {
    return (
      <Card title={title} subtitle={subtitle} icon={icon} status="error" className={className}>
        <CardValue value={t("status.error")} size="md" status="error" />
        <p className="caption text-status-error mt-1">{error}</p>
      </Card>
    );
  }

  return (
    <Card
      title={title}
      subtitle={subtitle}
      icon={icon}
      status={status}
      className={className}
      onClick={onClick}
    >
      {children}
    </Card>
  );
}
