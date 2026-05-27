/**
 * PathDiscoveryCard Component
 *
 * Purpose: Provides combined L2+L3 network path tracing functionality.
 * Displays hop-by-hop network path with latency, hostname resolution,
 * and L2 switch path with port details.
 *
 * Key Features:
 * - Traceroute (L3) with ICMP, UDP, or TCP protocols
 * - L2 switch path via LLDP/CDP/EDP + SNMP
 * - Device selector with discovered devices
 * - Quick target buttons for common destinations
 * - Visual RTT bar indicator for each hop
 * - L2 path diagram with port details
 * - Export results as JSON or CSV
 *
 * Usage:
 * ```typescript
 * <PathDiscoveryCard gateway="192.168.1.1" dnsServer="8.8.8.8" />
 * ```
 *
 * Dependencies: Card UI, DeviceSelector, theme utilities, path discovery API
 */

import { valibotResolver } from '@hookform/resolvers/valibot';
import type React from 'react';
import { memo, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useForm } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { api } from '../../api';
import { PathDiscoverySchema } from '../../schemas/auth';
import {
  button as buttonTokens,
  cn,
  icon as iconTokens,
  input as inputTokens,
  layout,
  radius,
  spacing,
  status as statusColor,
} from '../../styles/theme';
import type { PathResponse, TracerouteHop } from '../../types';
import { Card, CardDivider, CardValue, type Status } from '../ui/card';
import { Route } from '../ui/icons';
import { L2_PATH_DISPLAY } from './PathDiscoveryCardL2';
import { L3_PATH_DISPLAY } from './PathDiscoveryCardL3';
import { formatRtt, getMaxRtt } from './pathDiscoveryHelpers';

type Protocol = 'icmp' | 'udp' | 'tcp';

/** WebSocket message for streaming traceroute hops */
export interface TraceHopMessage {
  target: string;
  targetIp: string;
  protocol: string;
  hop: TracerouteHop;
  completed: boolean;
}

interface PathDiscoveryCardProps {
  gateway?: string;
  dnsServer?: string;
  /** Optional callback to register for traceHop WebSocket messages */
  onRegisterTraceHandler?: (handler: (msg: TraceHopMessage) => void) => () => void;
}

export const PathDiscoveryCard: React.NamedExoticComponent<PathDiscoveryCardProps> = memo(
  function pathDiscoveryCard({
    gateway,
    dnsServer,
    onRegisterTraceHandler,
  }: PathDiscoveryCardProps): React.ReactElement {
    const { t } = useTranslation('cards');

    const {
      register,
      handleSubmit,
      watch,
      setValue,
      formState: { errors },
    } = useForm<{ target: string; protocol: Protocol; port: number }>({
      resolver: valibotResolver(PathDiscoverySchema),
      defaultValues: { target: '', protocol: 'icmp', port: 80 },
      mode: 'onBlur',
    });
    const target = watch('target');
    const protocol = watch('protocol');
    const port = watch('port');
    const setTarget = useCallback(
      (next: string): void => {
        setValue('target', next, { shouldValidate: true, shouldDirty: true });
      },
      [setValue],
    );
    const [loading, setLoading] = useState(false);
    const [result, setResult] = useState<PathResponse | null>(null);
    const [error, setError] = useState<string | null>(null);
    const [expandedL2Hop, setExpandedL2Hop] = useState<number | null>(null);

    // Streaming hops received via WebSocket (accumulates as trace progresses)
    const [streamingHops, setStreamingHops] = useState<TracerouteHop[]>([]);
    const [_streamingTarget, setStreamingTarget] = useState<string>('');
    const activeTraceRef = useRef<string | null>(null);

    // Handle WebSocket trace hop messages for real-time updates
    const handleTraceHop = useCallback((msg: TraceHopMessage) => {
      // Only process if this is for our active trace
      if (activeTraceRef.current !== msg.target) {
        return;
      }

      setStreamingHops((prev) => {
        // Avoid duplicates by checking TTL
        if (prev.some((h) => h.ttl === msg.hop.ttl)) {
          return prev;
        }
        return [...prev, msg.hop].sort((a, b) => a.ttl - b.ttl);
      });
      setStreamingTarget(msg.target);

      if (msg.completed) {
        // Trace complete - the HTTP response will have the full result
        activeTraceRef.current = null;
      }
    }, []);

    // Register for WebSocket trace hop messages
    useEffect(() => {
      if (!onRegisterTraceHandler) {
        return;
      }
      return onRegisterTraceHandler(handleTraceHop);
    }, [onRegisterTraceHandler, handleTraceHop]);

    // Run path discovery (always L2+L3 combined)
    const runTrace = useCallback(
      async (traceTarget: string) => {
        if (!traceTarget.trim()) {
          return;
        }

        setLoading(true);
        setError(null);
        setResult(null);
        setExpandedL2Hop(null);
        setStreamingHops([]); // Clear streaming hops
        setStreamingTarget(traceTarget.trim());
        activeTraceRef.current = traceTarget.trim(); // Set active trace target

        try {
          const data = await api.post<PathResponse>('/api/v1/roots/path', {
            source: 'self',
            destination: traceTarget.trim(),
            method: 'both', // Always do both L2+L3
            protocol,
            port: protocol !== 'icmp' ? port : undefined,
          });
          setResult(data);
          setStreamingHops([]); // Clear streaming hops now that we have full result
          activeTraceRef.current = null;
        } catch (err) {
          setError(err instanceof Error ? err.message : 'Path discovery failed');
          activeTraceRef.current = null;
        } finally {
          setLoading(false);
        }
      },
      [protocol, port],
    );

    const onSubmit = useCallback(
      ({ target: traceTarget }: { target: string }): void => {
        runTrace(traceTarget).catch(() => {
          // Error handled in runTrace
        });
      },
      [runTrace],
    );

    // Quick target handlers
    const traceGateway = useCallback((): void => {
      if (gateway) {
        setTarget(gateway);
        runTrace(gateway).catch(() => {
          // Error handled in runTrace
        });
      }
    }, [gateway, runTrace]);

    const traceDns = useCallback((): void => {
      const dns = dnsServer || '8.8.8.8';
      setTarget(dns);
      runTrace(dns).catch(() => {
        // Error handled in runTrace
      });
    }, [dnsServer, runTrace]);

    const traceInternet = useCallback((): void => {
      const internetTarget = '8.8.8.8';
      setTarget(internetTarget);
      runTrace(internetTarget).catch(() => {
        // Error handled in runTrace
      });
    }, [runTrace]);

    // Export as JSON
    const exportJson = useCallback(() => {
      if (!result) {
        return;
      }
      const blob = new Blob([JSON.stringify(result, null, 2)], {
        type: 'application/json',
      });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `path-discovery-${target}-${Date.now()}.json`;
      a.click();
      URL.revokeObjectURL(url);
    }, [result, target]);

    // Export as CSV
    const exportCsv = useCallback(() => {
      if (!result) {
        return;
      }

      let csvContent = '';

      // L3 path section
      if (result.l3Path) {
        csvContent += 'L3 Path\n';
        csvContent += 'TTL,IP,Hostname,RTT (ms),State\n';
        csvContent += result.l3Path.hops
          .map(
            (h) =>
              `${h.ttl},${h.ip || '*'},${h.hostname || ''},${h.rtt > 0 ? (h.rtt / 1_000_000).toFixed(2) : ''},${h.state}`,
          )
          .join('\n');
      }

      // L2 path section
      if (result.l2Path) {
        if (csvContent) {
          csvContent += '\n\n';
        }
        csvContent += 'L2 Path\n';
        csvContent += 'Device,Device IP,Ingress Port,Egress Port,Source\n';
        csvContent += result.l2Path.hops
          .map(
            (h) =>
              `${h.device},${h.deviceIp},${h.ingressPort?.name || ''},${h.egressPort?.name || ''},${h.source}`,
          )
          .join('\n');
      }

      const blob = new Blob([csvContent], { type: 'text/csv' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `path-discovery-${target}-${Date.now()}.csv`;
      a.click();
      URL.revokeObjectURL(url);
    }, [result, target]);

    // Copy to clipboard
    const copyToClipboard = useCallback(() => {
      if (!result) {
        return;
      }
      navigator.clipboard.writeText(JSON.stringify(result, null, 2));
    }, [result]);

    // Determine card status based on worst hop result
    const cardStatus: Status = useMemo(() => {
      if (loading) {
        return 'loading';
      }
      if (error) {
        return 'error';
      }
      if (!result) {
        return 'unknown';
      }

      // Check L3 path for issues
      const l3Hops = result.l3Path?.hops || [];
      // L3Hop.state is 'timeout' | 'reply'; absence of reply = error state.
      const hasTimeouts = l3Hops.some((h) => h.state === 'timeout');
      const hasErrors = false;
      const hasHighLatency = l3Hops.some((h) => h.rtt > 100000000); // > 100ms

      if (hasErrors) {
        return 'error';
      }
      if (hasTimeouts || hasHighLatency) {
        return 'warning';
      }
      if (result.l3Path?.completed || result.l2Path) {
        return 'success';
      }
      return 'warning';
    }, [loading, error, result]);

    const maxRtt = result?.l3Path ? getMaxRtt(result.l3Path.hops) : 1;

    return (
      <Card
        title={t('pathDiscovery.title', 'Path Discovery')}
        icon={<Route className={iconTokens.size.md} />}
        status={cardStatus}
      >
        {/* Target Input Form - Responsive layout for various screen sizes */}
        <form
          onSubmit={handleSubmit(onSubmit)}
          className={cn('stack-sm', spacing.margin.bottom.content)}
        >
          {/* Target Input Row - Stack on mobile, inline on larger screens */}
          <div className="flex flex-col sm:flex-row gap-compact">
            {/* Target input - full width on mobile */}
            <input
              type="text"
              {...register('target')}
              placeholder={t('pathDiscovery.enterTarget', 'Enter IP or hostname...')}
              disabled={loading}
              className={cn(
                'flex-1 min-w-0',
                inputTokens.base,
                inputTokens.state.default,
                inputTokens.size.sm,
                'body-small',
              )}
            />

            {/* Protocol and Trace button group - inline always */}
            <div className="flex items-center gap-compact shrink-0">
              {/* Protocol selector - styled to match design system */}
              <select
                {...register('protocol')}
                disabled={loading}
                className={cn(
                  inputTokens.base,
                  inputTokens.state.default,
                  inputTokens.size.sm,
                  'w-20 body-small cursor-pointer',
                )}
                title={t('pathDiscovery.protocol', 'Traceroute protocol')}
              >
                <option value="icmp">ICMP</option>
                <option value="udp">UDP</option>
                <option value="tcp">TCP</option>
              </select>

              {/* Port input (only for TCP/UDP) */}
              {protocol !== 'icmp' && (
                <input
                  type="number"
                  {...register('port', { valueAsNumber: true })}
                  placeholder="Port"
                  min={1}
                  max={65535}
                  disabled={loading}
                  className={cn(
                    'w-16',
                    inputTokens.base,
                    inputTokens.state.default,
                    inputTokens.size.sm,
                    'body-small',
                  )}
                />
              )}

              <button
                type="submit"
                disabled={loading || !target?.trim()}
                className={cn(
                  buttonTokens.base,
                  buttonTokens.variant.primary,
                  buttonTokens.size.sm,
                  'whitespace-nowrap',
                )}
              >
                {loading ? '...' : t('pathDiscovery.trace', 'Trace')}
              </button>
            </div>
          </div>
          {errors.target ? (
            <p className={cn('caption', statusColor.text.error)}>{errors.target.message}</p>
          ) : null}
          {errors.port ? (
            <p className={cn('caption', statusColor.text.error)}>{errors.port.message}</p>
          ) : null}

          {/* Quick Targets - Wrap on small screens */}
          <div className="flex items-center gap-compact flex-wrap">
            <span className="caption text-text-muted shrink-0">
              {t('pathDiscovery.quick', 'Quick')}:
            </span>
            <div className="flex items-center gap-1.5 flex-wrap">
              <button
                type="button"
                onClick={traceGateway}
                disabled={loading || !gateway}
                className={cn(
                  buttonTokens.base,
                  buttonTokens.variant.ghost,
                  buttonTokens.size.xs,
                  'caption whitespace-nowrap',
                )}
              >
                {t('pathDiscovery.gateway', 'Gateway')}
              </button>
              <button
                type="button"
                onClick={traceDns}
                disabled={loading}
                className={cn(
                  buttonTokens.base,
                  buttonTokens.variant.ghost,
                  buttonTokens.size.xs,
                  'caption whitespace-nowrap',
                )}
              >
                {t('pathDiscovery.dns', 'DNS')}
              </button>
              <button
                type="button"
                onClick={traceInternet}
                disabled={loading}
                className={cn(
                  buttonTokens.base,
                  buttonTokens.variant.ghost,
                  buttonTokens.size.xs,
                  'caption whitespace-nowrap',
                )}
              >
                {t('pathDiscovery.internet', 'Internet')}
              </button>
            </div>
          </div>
        </form>
        <CardDivider />
        {/* Loading State with Streaming Hops */}
        {loading ? (
          <div className="stack-sm">
            <CardValue
              value={
                streamingHops.length > 0
                  ? t('pathDiscovery.tracingHops', 'Tracing... {{count}} hops', {
                      count: streamingHops.length,
                    })
                  : t('pathDiscovery.tracing', 'Tracing path...')
              }
              size="lg"
            />
            {/* Show streaming hops in real-time */}
            {streamingHops.length > 0 ? (
              <div className="stack-xs">
                {streamingHops.map((hop) => (
                  <div
                    key={hop.ttl}
                    className={cn(
                      'flex items-center gap-compact py-compact',
                      hop.state === 'timeout' && 'opacity-50',
                    )}
                  >
                    <span className="w-6 text-xs text-text-muted font-mono">{hop.ttl}</span>
                    <span className="flex-1 text-sm font-mono text-text-primary">
                      {hop.ip || '*'}
                    </span>
                    <span className="text-xs text-text-muted">{formatRtt(hop.rtt)}</span>
                  </div>
                ))}
                {/* Pulsing indicator for next hop */}
                <div className="flex items-center gap-compact py-compact animate-pulse">
                  <span className="w-6 text-xs text-text-muted font-mono">
                    {streamingHops.length + 1}
                  </span>
                  <span className="text-sm text-text-muted">...</span>
                </div>
              </div>
            ) : null}
          </div>
        ) : null}
        {/* Error State */}
        {error && !loading ? (
          <div className={cn(spacing.pad.sm, statusColor.bg.errorSoft, radius.default)}>
            <span className="body-small text-status-error">{error}</span>
          </div>
        ) : null}
        {/* Results */}
        {result && !loading ? (
          <div className="stack-md">
            {/* L3 Path Results */}
            {result.l3Path ? (
              <L3_PATH_DISPLAY result={result.l3Path} maxRtt={maxRtt} t={t} />
            ) : null}

            {/* L2 Path Results */}
            {result.l2Path ? (
              <L2_PATH_DISPLAY
                result={result.l2Path}
                expandedHop={expandedL2Hop}
                onToggleHop={setExpandedL2Hop}
                t={t}
              />
            ) : null}

            {/* Export Actions */}
            <div
              className={cn(layout.inline.default, spacing.gap.compact, spacing.margin.top.inline)}
            >
              <button
                type="button"
                onClick={exportJson}
                className={cn(
                  buttonTokens.base,
                  buttonTokens.variant.ghost,
                  buttonTokens.size.xs,
                  'caption',
                )}
              >
                {t('pathDiscovery.exportJSON', 'Export JSON')}
              </button>
              <button
                type="button"
                onClick={exportCsv}
                className={cn(
                  buttonTokens.base,
                  buttonTokens.variant.ghost,
                  buttonTokens.size.xs,
                  'caption',
                )}
              >
                {t('pathDiscovery.exportCSV', 'Export CSV')}
              </button>
              <button
                type="button"
                onClick={copyToClipboard}
                className={cn(
                  buttonTokens.base,
                  buttonTokens.variant.ghost,
                  buttonTokens.size.xs,
                  'caption',
                )}
              >
                {t('pathDiscovery.copy', 'Copy')}
              </button>
              <button
                type="button"
                onClick={(): void => {
                  runTrace(target).catch(() => {
                    // Error handled in runTrace
                  });
                }}
                disabled={loading}
                className={cn(
                  buttonTokens.base,
                  buttonTokens.variant.ghost,
                  buttonTokens.size.xs,
                  'caption',
                )}
              >
                {t('pathDiscovery.rerun', 'Re-run')}
              </button>
            </div>
          </div>
        ) : null}
        {/* Empty State - improved visual design */}
        {result || loading || error ? null : (
          <div
            className={cn(
              spacing.pad.default,
              'text-center',
              'bg-surface-base/50',
              radius.lg,
              'border border-dashed border-surface-border',
            )}
          >
            <div className="text-text-muted mb-2">
              <Route className={cn(iconTokens.size.lg, 'mx-auto opacity-40')} />
            </div>
            <p className="body-small text-text-muted">
              {t('pathDiscovery.enterTarget', 'Select a target to trace')}
            </p>
            <p className="caption text-text-muted mt-tight">
              {t(
                'pathDiscovery.emptyHint',
                'Enter an IP address or hostname, or use the quick buttons above',
              )}
            </p>
          </div>
        )}
      </Card>
    );
  },
);
