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

import type React from 'react';
import { memo, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { api } from '../../api';
import {
  button as buttonTokens,
  cn,
  icon as iconTokens,
  input as inputTokens,
  layout,
  radius,
  spacing,
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
  // biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Complex path discovery with trace and hop visualization
  function pathDiscoveryCard({
    gateway,
    dnsServer,
    onRegisterTraceHandler,
  }: PathDiscoveryCardProps): React.ReactElement {
    const { t } = useTranslation('cards');

    const [target, setTarget] = useState('');
    const [protocol, setProtocol] = useState<Protocol>('icmp');
    const [port, setPort] = useState<number>(80);
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

    // Handle form submit
    const handleSubmit = useCallback(
      (e: React.FormEvent): void => {
        e.preventDefault();
        runTrace(target).catch(() => {
          // Error handled in runTrace
        });
      },
      [target, runTrace],
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
      const hasErrors = l3Hops.some((h) => h.state === 'error' || h.state === 'unreachable');
      const hasTimeouts = l3Hops.some((h) => h.state === 'timeout');
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
        icon={<Route class={iconTokens.size.md} />}
        status={cardStatus}
      >
        {/* Target Input Form - Responsive layout for various screen sizes */}
        <form onSubmit={handleSubmit} class={cn('stack-sm', spacing.margin.bottom.content)}>
          {/* Target Input Row - Stack on mobile, inline on larger screens */}
          <div class="flex flex-col sm:flex-row gap-2">
            {/* Target input - full width on mobile */}
            <input
              type="text"
              value={target}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                setTarget(e.target.value)
              }
              placeholder={t('pathDiscovery.enterTarget', 'Enter IP or hostname...')}
              disabled={loading}
              class={cn(
                'flex-1 min-w-0',
                inputTokens.base,
                inputTokens.state.default,
                inputTokens.size.sm,
                'body-small',
              )}
              onKeyDown={(e: React.KeyboardEvent): void => {
                if (e.key === 'Enter' && target.trim()) {
                  e.preventDefault();
                  handleSubmit(e as unknown as React.FormEvent);
                }
              }}
            />

            {/* Protocol and Trace button group - inline always */}
            <div class="flex items-center gap-2 shrink-0">
              {/* Protocol selector - styled to match design system */}
              <select
                value={protocol}
                onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                  setProtocol(e.target.value as Protocol)
                }
                disabled={loading}
                class={cn(
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
                  value={port}
                  onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                    setPort(Number.parseInt(e.target.value, 10) || 80)
                  }
                  placeholder="Port"
                  min={1}
                  max={65535}
                  disabled={loading}
                  class={cn(
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
                disabled={loading || !target.trim()}
                class={cn(
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

          {/* Quick Targets - Wrap on small screens */}
          <div class="flex items-center gap-2 flex-wrap">
            <span class="caption text-text-muted shrink-0">
              {t('pathDiscovery.quick', 'Quick')}:
            </span>
            <div class="flex items-center gap-1.5 flex-wrap">
              <button
                type="button"
                onClick={traceGateway}
                disabled={loading || !gateway}
                class={cn(
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
                class={cn(
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
                class={cn(
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
          <div class="stack-sm">
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
              <div class="stack-xs">
                {streamingHops.map((hop) => (
                  <div
                    key={hop.ttl}
                    class={cn(
                      'flex items-center gap-2 py-1',
                      hop.state === 'timeout' && 'opacity-50',
                    )}
                  >
                    <span class="w-6 text-xs text-text-muted font-mono">{hop.ttl}</span>
                    <span class="flex-1 text-sm font-mono text-text-primary">{hop.ip || '*'}</span>
                    <span class="text-xs text-text-muted">{formatRtt(hop.rtt)}</span>
                  </div>
                ))}
                {/* Pulsing indicator for next hop */}
                <div class="flex items-center gap-2 py-1 animate-pulse">
                  <span class="w-6 text-xs text-text-muted font-mono">
                    {streamingHops.length + 1}
                  </span>
                  <span class="text-sm text-text-muted">...</span>
                </div>
              </div>
            ) : null}
          </div>
        ) : null}

        {/* Error State */}
        {error && !loading ? (
          <div class={cn(spacing.pad.sm, 'bg-status-error/10', radius.default)}>
            <span class="body-small text-status-error">{error}</span>
          </div>
        ) : null}

        {/* Results */}
        {result && !loading ? (
          <div class="stack-md">
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
            <div class={cn(layout.inline.default, spacing.gap.compact, spacing.margin.top.inline)}>
              <button
                type="button"
                onClick={exportJson}
                class={cn(
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
                class={cn(
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
                class={cn(
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
                class={cn(
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
            class={cn(
              spacing.pad.md,
              'text-center',
              'bg-surface-base/50',
              radius.lg,
              'border border-dashed border-surface-border',
            )}
          >
            <div class="text-text-muted mb-2">
              <Route class={cn(iconTokens.size.lg, 'mx-auto opacity-40')} />
            </div>
            <p class="body-small text-text-muted">
              {t('pathDiscovery.enterTarget', 'Select a target to trace')}
            </p>
            <p class="caption text-text-muted mt-1">
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
