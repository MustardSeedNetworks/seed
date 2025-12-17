/**
 * Centralized frontend logging service with batch flushing to backend.
 *
 * This logger provides:
 * - Structured logging with levels (DEBUG, INFO, WARN, ERROR)
 * - Automatic batching and flushing to backend
 * - Session ID tracking for correlation
 * - Stack traces for errors
 * - Component-based categorization
 *
 * @example
 * import { logger } from '@/lib/logger';
 * logger.info('devices', 'Fetching devices list');
 * logger.error('auth', 'Login failed', error, { username: 'user' });
 */

export type LogLevel = "DEBUG" | "INFO" | "WARN" | "ERROR";

export interface LogEntry {
  timestamp: string;
  level: LogLevel;
  layer: "frontend";
  request_id?: string;
  session_id?: string;
  message: string;
  component: string;
  metadata?: Record<string, unknown>;
  stack?: string;
}

// Standard component names matching backend constants
export const LogComponents = {
  AUTH: "auth",
  DISCOVERY: "discovery",
  DEVICES: "devices",
  NETWORK: "network",
  SURVEY: "survey",
  WEBSOCKET: "websocket",
  SPEEDTEST: "speedtest",
  IPERF: "iperf",
  VULN: "vulnerabilities",
  CONFIG: "config",
  SYSTEM: "system",
  DNS: "dns",
  DHCP: "dhcp",
  GATEWAY: "gateway",
  VLAN: "vlan",
  WIFI: "wifi",
  CABLE: "cable",
  PUBLICIP: "publicip",
  EXPORT: "export",
  SETUP: "setup",
  UI: "ui",
  APP: "app",
} as const;

export type LogComponent = (typeof LogComponents)[keyof typeof LogComponents];

interface LoggerConfig {
  batchSize: number;
  flushInterval: number;
  enabled: boolean;
  consoleOutput: boolean;
  minLevel: LogLevel;
}

const DEFAULT_CONFIG: LoggerConfig = {
  batchSize: 50,
  flushInterval: 5000, // 5 seconds
  enabled: true,
  consoleOutput: true, // Also output to console
  minLevel: "DEBUG",
};

// Color styles for console output
const LOG_COLORS: Record<LogLevel, string> = {
  DEBUG: "color: #6B7280", // Gray
  INFO: "color: #3B82F6", // Blue
  WARN: "color: #EAB308", // Yellow
  ERROR: "color: #EF4444", // Red
};

/**
 * Logger class provides structured logging with batch flushing to backend.
 */
class Logger {
  private buffer: LogEntry[] = [];
  private flushTimer: ReturnType<typeof setInterval> | null = null;
  private sessionId: string;
  private config: LoggerConfig;
  private currentRequestId: string | undefined;

  constructor(config: Partial<LoggerConfig> = {}) {
    this.config = { ...DEFAULT_CONFIG, ...config };
    this.sessionId = this.generateSessionId();

    if (this.config.enabled) {
      this.startFlushTimer();
    }

    // Flush on page unload
    if (typeof window !== "undefined") {
      window.addEventListener("beforeunload", () => this.flush());
      window.addEventListener("visibilitychange", () => {
        if (document.visibilityState === "hidden") {
          this.flush();
        }
      });
    }
  }

  private generateSessionId(): string {
    return `${Date.now().toString(36)}-${Math.random().toString(36).substring(2, 9)}`;
  }

  private shouldLog(level: LogLevel): boolean {
    // Get numeric priority for comparison - level is type-safe LogLevel enum
    const levelPriority = this.getLevelPriority(level);
    const minLevelPriority = this.getLevelPriority(this.config.minLevel);
    return this.config.enabled && levelPriority >= minLevelPriority;
  }

  private getLevelPriority(level: LogLevel): number {
    switch (level) {
      case "ERROR":
        return 3;
      case "WARN":
        return 2;
      case "INFO":
        return 1;
      case "DEBUG":
      default:
        return 0;
    }
  }

  private createEntry(
    level: LogLevel,
    component: string,
    message: string,
    metadata?: Record<string, unknown>,
    stack?: string
  ): LogEntry {
    return {
      timestamp: new Date().toISOString(),
      level,
      layer: "frontend",
      session_id: this.sessionId,
      request_id: this.currentRequestId,
      message,
      component,
      metadata,
      stack,
    };
  }

  private logToConsole(entry: LogEntry): void {
    if (!this.config.consoleOutput) return;

    const style = LOG_COLORS[entry.level];
    const prefix = `%c[${entry.level}][${entry.component}]`;
    const args: unknown[] = [prefix, style, entry.message];

    if (entry.metadata && Object.keys(entry.metadata).length > 0) {
      args.push(entry.metadata);
    }

    if (entry.stack) {
      args.push("\n" + entry.stack);
    }

    // Use console.warn for all non-error levels since the linter restricts console.log/debug
    switch (entry.level) {
      case "ERROR":
        console.error(...args);
        break;
      default:
        console.warn(...args);
    }
  }

  private addToBuffer(entry: LogEntry): void {
    this.buffer.push(entry);
    this.logToConsole(entry);

    if (this.buffer.length >= this.config.batchSize) {
      this.flush();
    }
  }

  private startFlushTimer(): void {
    if (this.flushTimer) return;
    this.flushTimer = setInterval(() => this.flush(), this.config.flushInterval);
  }

  private stopFlushTimer(): void {
    if (this.flushTimer) {
      clearInterval(this.flushTimer);
      this.flushTimer = null;
    }
  }

  /**
   * Flush buffered logs to the backend.
   */
  async flush(): Promise<void> {
    if (this.buffer.length === 0) return;

    const entries = [...this.buffer];
    this.buffer = [];

    try {
      const response = await fetch("/api/logs/client", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ entries }),
        // Use keepalive for page unload scenarios
        keepalive: true,
      });

      if (!response.ok) {
        // Put entries back in buffer on failure (but don't retry indefinitely)
        if (entries.length < this.config.batchSize * 2) {
          this.buffer.unshift(...entries);
        }
      }
    } catch {
      // Silently fail - don't create infinite log loops
      // Put entries back for next attempt if buffer isn't too full
      if (this.buffer.length + entries.length <= this.config.batchSize * 3) {
        this.buffer.unshift(...entries);
      }
    }
  }

  /**
   * Set the current request ID for correlation with backend logs.
   */
  setRequestId(requestId: string | undefined): void {
    this.currentRequestId = requestId;
  }

  /**
   * Get the session ID for this logger instance.
   */
  getSessionId(): string {
    return this.sessionId;
  }

  /**
   * Log a debug message.
   */
  debug(component: string, message: string, metadata?: Record<string, unknown>): void {
    if (!this.shouldLog("DEBUG")) return;
    const entry = this.createEntry("DEBUG", component, message, metadata);
    this.addToBuffer(entry);
  }

  /**
   * Log an info message.
   */
  info(component: string, message: string, metadata?: Record<string, unknown>): void {
    if (!this.shouldLog("INFO")) return;
    const entry = this.createEntry("INFO", component, message, metadata);
    this.addToBuffer(entry);
  }

  /**
   * Log a warning message.
   */
  warn(component: string, message: string, metadata?: Record<string, unknown>): void {
    if (!this.shouldLog("WARN")) return;
    const entry = this.createEntry("WARN", component, message, metadata);
    this.addToBuffer(entry);
  }

  /**
   * Log an error message with optional Error object.
   */
  error(
    component: string,
    message: string,
    error?: Error | unknown,
    metadata?: Record<string, unknown>
  ): void {
    if (!this.shouldLog("ERROR")) return;

    let stack: string | undefined;
    let errorMeta = metadata || {};

    if (error instanceof Error) {
      stack = error.stack;
      errorMeta = {
        ...errorMeta,
        error_name: error.name,
        error_message: error.message,
      };
    } else if (error !== undefined) {
      errorMeta = {
        ...errorMeta,
        error_value: String(error),
      };
    }

    const entry = this.createEntry("ERROR", component, message, errorMeta, stack);
    this.addToBuffer(entry);
  }

  /**
   * Log the start of a timed operation. Returns a function to call when done.
   */
  timedOperation(component: string, operation: string): () => void {
    const startTime = performance.now();
    this.debug(component, `${operation} started`);

    return () => {
      const duration = Math.round(performance.now() - startTime);
      this.debug(component, `${operation} completed`, { duration_ms: duration });
    };
  }

  /**
   * Create a child logger with a fixed component.
   */
  withComponent(component: string): ComponentLogger {
    return new ComponentLogger(this, component);
  }

  /**
   * Update logger configuration.
   */
  configure(config: Partial<LoggerConfig>): void {
    this.config = { ...this.config, ...config };

    if (this.config.enabled && !this.flushTimer) {
      this.startFlushTimer();
    } else if (!this.config.enabled && this.flushTimer) {
      this.stopFlushTimer();
    }
  }

  /**
   * Disable the logger and flush remaining entries.
   */
  disable(): void {
    this.config.enabled = false;
    this.stopFlushTimer();
    this.flush();
  }
}

/**
 * A logger bound to a specific component.
 */
class ComponentLogger {
  constructor(
    private parent: Logger,
    private component: string
  ) {}

  debug(message: string, metadata?: Record<string, unknown>): void {
    this.parent.debug(this.component, message, metadata);
  }

  info(message: string, metadata?: Record<string, unknown>): void {
    this.parent.info(this.component, message, metadata);
  }

  warn(message: string, metadata?: Record<string, unknown>): void {
    this.parent.warn(this.component, message, metadata);
  }

  error(message: string, error?: Error | unknown, metadata?: Record<string, unknown>): void {
    this.parent.error(this.component, message, error, metadata);
  }

  timedOperation(operation: string): () => void {
    return this.parent.timedOperation(this.component, operation);
  }
}

// Export singleton instance
export const logger = new Logger();

// Export class for testing or custom instances
export { Logger, ComponentLogger };
