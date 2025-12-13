import { ReactNode } from "react";
import { StatusBadge, statusConfig, Status } from "./StatusBadge";

// Re-export Status and statusConfig for backwards compatibility
export type { Status };
export { statusConfig };

interface CardProps {
  title: string;
  subtitle?: string;
  status: Status;
  children: ReactNode;
  className?: string;
  onClick?: () => void;
  icon?: ReactNode;
}

export function Card({
  title,
  subtitle,
  status,
  children,
  className = "",
  onClick,
  icon,
}: CardProps) {
  const isInteractive = typeof onClick === "function";

  const handleKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
    if (!isInteractive) return;
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      onClick?.();
    }
  };

  return (
    <div
      className={`rounded-lg border border-surface-border bg-surface-raised p-3 sm:p-4 transition-all hover:border-brand-primary/40 touch-manipulation focus-visible:ring-2 focus-visible:ring-brand-primary focus-visible:ring-offset-2 focus-visible:ring-offset-surface-base outline-none ${
        isInteractive ? "cursor-pointer active:scale-[0.98]" : ""
      } ${className}`}
      onClick={onClick}
      onKeyDown={handleKeyDown}
      role={isInteractive ? "button" : undefined}
      tabIndex={isInteractive ? 0 : -1}
      aria-pressed={undefined}
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          {icon && (
            <span className="text-text-muted w-5 h-5 shrink-0">{icon}</span>
          )}
          <div className="flex flex-col">
            <h3 className="font-semibold text-text-primary text-base sm:text-lg leading-tight font-display">
              {title}
            </h3>
            {subtitle && (
              <p className="text-xs text-text-muted leading-tight">
                {subtitle}
              </p>
            )}
          </div>
        </div>
        <StatusBadge status={status} size="md" />
      </div>
      <div className="mt-2 sm:mt-3">{children}</div>
    </div>
  );
}

interface CardValueProps {
  label?: string;
  value: string | number;
  unit?: string;
  size?: "sm" | "md" | "lg";
  status?: Status;
  mono?: boolean;
  allowWrap?: boolean;
}

export function CardValue({
  label,
  value,
  unit,
  size = "md",
  status,
  mono = false,
  allowWrap = false,
}: CardValueProps) {
  const sizeClasses = {
    sm: "text-sm",
    md: "text-base font-medium leading-snug",
    lg: "text-lg font-semibold leading-snug",
  };

  const textMods = [
    status ? statusConfig[status].color : "text-text-primary",
    mono ? "font-mono tabular-nums" : "",
    allowWrap ? "break-all whitespace-pre-wrap" : "",
  ]
    .filter(Boolean)
    .join(" ");

  const statusIcon =
    status && statusConfig[status] ? statusConfig[status].icon : null;

  return (
    <div>
      {label && <p className="text-xs text-text-muted mb-1">{label}</p>}
      <p
        className={`${sizeClasses[size]} ${textMods} flex items-center gap-1.5`}
        data-testid="card-value"
      >
        {statusIcon && (
          <span className="inline-flex items-center justify-center w-3 h-3 shrink-0 text-current">
            {statusIcon}
          </span>
        )}
        <span className="flex items-baseline gap-1">
          <span>{value}</span>
          {unit && (
            <span className="text-sm font-normal text-text-muted">{unit}</span>
          )}
        </span>
      </p>
    </div>
  );
}

interface CardRowProps {
  label: string;
  value: string | number;
  status?: Status;
  wrap?: boolean;
  mono?: boolean;
  align?: "left" | "right";
}

export function CardRow({
  label,
  value,
  status,
  wrap = false,
  mono = false,
  align = "right",
}: CardRowProps) {
  const statusIcon =
    status && statusConfig[status] ? statusConfig[status].icon : null;
  const justifyClass = align === "right" ? "justify-end" : "justify-start";

  return (
    <div
      className={`flex ${wrap ? "items-start" : "items-center"} justify-between gap-2 py-1`}
    >
      <span className="text-sm text-text-muted shrink-0">{label}</span>
      <span
        className={`text-sm font-medium ${align === "right" ? "text-right" : "text-left"} ${wrap ? "break-all whitespace-pre-wrap" : "truncate"} ${mono ? "font-mono tabular-nums" : ""} ${status ? statusConfig[status].color : "text-text-primary"} flex items-center gap-1.5 ${justifyClass}`}
        title={String(value)}
        data-testid="card-row-value"
      >
        {statusIcon && (
          <span className="w-3 h-3 shrink-0 text-current">{statusIcon}</span>
        )}
        <span>{value}</span>
      </span>
    </div>
  );
}

interface CardDividerProps {
  className?: string;
}

export function CardDivider({ className = "" }: CardDividerProps) {
  return <hr className={`border-surface-border my-3 ${className}`} />;
}
