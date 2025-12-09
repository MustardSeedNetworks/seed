import { ReactNode } from "react";

export type Status = "success" | "warning" | "error" | "unknown" | "loading";

interface CardProps {
  title: string;
  subtitle?: string;
  status: Status;
  children: ReactNode;
  className?: string;
  onClick?: () => void;
}

const statusConfig: Record<
  Status,
  { icon: ReactNode; color: string; bgColor: string; label: string }
> = {
  success: {
    icon: (
      <svg
        className="w-4 h-4"
        viewBox="0 0 20 20"
        fill="currentColor"
        aria-hidden="true"
      >
        <path
          fillRule="evenodd"
          d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-10.707a1 1 0 00-1.414-1.414L9 9.172 7.707 7.879a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
          clipRule="evenodd"
        />
      </svg>
    ),
    color: "text-status-success",
    bgColor: "bg-status-success/10",
    label: "Status: success",
  },
  warning: {
    icon: (
      <svg
        className="w-4 h-4"
        viewBox="0 0 20 20"
        fill="currentColor"
        aria-hidden="true"
      >
        <path
          fillRule="evenodd"
          d="M8.257 3.099c.765-1.36 2.72-1.36 3.485 0l6.518 11.6c.75 1.334-.214 3.001-1.742 3.001H3.48c-1.528 0-2.492-1.667-1.742-3.001l6.52-11.6zM11 14a1 1 0 11-2 0 1 1 0 012 0zm-1-2a1 1 0 01-1-1V8a1 1 0 112 0v3a1 1 0 01-1 1z"
          clipRule="evenodd"
        />
      </svg>
    ),
    color: "text-status-warning",
    bgColor: "bg-status-warning/10",
    label: "Status: warning",
  },
  error: {
    icon: (
      <svg
        className="w-4 h-4"
        viewBox="0 0 20 20"
        fill="currentColor"
        aria-hidden="true"
      >
        <path
          fillRule="evenodd"
          d="M10 18a8 8 0 100-16 8 8 0 000 16zm-1.293-5.293a1 1 0 011.414 0L10 12.586l.879-.879a1 1 0 111.414 1.414L11.414 14l.879.879a1 1 0 01-1.414 1.414L10 15.414l-.879.879a1 1 0 11-1.414-1.414L8.586 14l-.879-.879a1 1 0 010-1.414z"
          clipRule="evenodd"
        />
      </svg>
    ),
    color: "text-status-error",
    bgColor: "bg-status-error/10",
    label: "Status: error",
  },
  unknown: {
    icon: (
      <svg
        className="w-4 h-4"
        viewBox="0 0 20 20"
        fill="currentColor"
        aria-hidden="true"
      >
        <path d="M9 7a1 1 0 012 0c0 1.5-2 1.5-2 3h2c0-1.5 2-1.5 2-3a3 3 0 10-6 0h2z" />
        <circle cx="10" cy="14" r="1" />
      </svg>
    ),
    color: "text-text-muted",
    bgColor: "bg-surface-hover",
    label: "Status: unknown",
  },
  loading: {
    icon: (
      <svg
        className="w-4 h-4 animate-spin"
        viewBox="0 0 20 20"
        fill="none"
        aria-hidden="true"
      >
        <circle
          className="opacity-25"
          cx="10"
          cy="10"
          r="8"
          stroke="currentColor"
          strokeWidth="3"
        />
        <path
          className="opacity-75"
          fill="currentColor"
          d="M18 10a8 8 0 00-8-8v4a4 4 0 014 4h4z"
        />
      </svg>
    ),
    color: "text-status-info",
    bgColor: "bg-status-info/10",
    label: "Status: loading",
  },
};

export function Card({
  title,
  subtitle,
  status,
  children,
  className = "",
  onClick,
}: CardProps) {
  const config = statusConfig[status];
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
        <div className="flex flex-col">
          <h3 className="font-semibold text-text-primary text-base sm:text-lg leading-tight font-display">
            {title}
          </h3>
          {subtitle && (
            <p className="text-xs text-text-muted leading-tight">{subtitle}</p>
          )}
        </div>
        <span
          className={`inline-flex items-center justify-center rounded-full ${config.color} ${config.bgColor} p-1`}
          aria-label={config.label}
        >
          {config.icon}
        </span>
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
