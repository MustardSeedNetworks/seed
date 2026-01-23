/**
 * PerformanceCard Component
 *
 * Purpose: Internet speed and network performance testing with speedtest.net integration and
 * iperf3 support. Displays download/upload speeds, latency, jitter, and packet loss metrics.
 * (~670 lines - complex testing orchestration)
 *
 * Key Features:
 * - Speedtest.net integration: download/upload speeds, latency, server info, distance
 * - iperf3 support: UDP and TCP throughput testing, bandwidth, jitter, packet loss
 * - Real-time progress: shows test phase and progress percentage
 * - Multiple test result history: displays last completed test results
 * - Visual gauges: SpeedGauge components for visual speed representation
 * - Protocol metrics: separate IPv4/IPv6 results (when available)
 * - Test controls: start/stop test, clear results
 * - Latency thresholds: warning/critical levels from settings
 * - System info: iperf3 installation status and version
 *
 * Usage:
 * ```typescript
 * <PerformanceCard
 *   data={performanceData}
 *   loading={isRunningTest}
 * />
 * ```
 *
 * Dependencies: Card UI components, SpeedGauge, useSettings hook, auth hooks,
 *              Icons, theme utilities
 * State: Manages test state, results history, current phase, progress tracking
 */

import type React from 'react';
import { memo, useCallback, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { api } from '../../api';
import { useSettings } from '../../contexts/useSettings';
import { LogComponents, logger } from '../../lib/logger';
import { cn, icon as iconTokens, layout, radius, spacing } from '../../styles/theme';
import { Card, CardDivider, CardRow, CardValue, type Status } from '../ui/card';
import { Gauge } from '../ui/icons';
import { ProgressRing, PulsingDot, SpeedGauge } from '../ui/SpeedGauge';

// Speedtest types
interface SpeedtestData {
  download: number;
  upload: number;
  latency: number;
  server: string;
  location: string;
  host: string;
  distance: number;
  timestamp: string;
  testDuration: number;
}

interface SpeedtestStatus {
  running: boolean;
  phase: string;
  progress: number;
  currentDownload: number; // Live download speed during test
  currentUpload: number; // Live upload speed during test
  last?: SpeedtestData;
}

// iperf3 types
interface IperfInfo {
  installed: boolean;
  version?: string;
  error?: string;
}

interface IperfResult {
  bandwidth: number;
  transfer: number;
  retransmits: number;
  jitter: number;
  lostPackets: number;
  lostPercent: number;
  protocol: string;
  direction: string;
  duration: number;
  server: string;
  port: number;
  timestamp: string;
  downloadBandwidth?: number;
  uploadBandwidth?: number;
  downloadTransfer?: number;
  uploadTransfer?: number;
}

interface IperfClientStatus {
  running: boolean;
  phase: string;
  progress: number;
  last?: IperfResult;
}

interface IperfServerStatus {
  running: boolean;
  port: number;
  pid: number;
  error?: string;
}

interface PerformanceCardProps {
  loading?: boolean;
  runSpeedtestEnabled?: boolean;
  runIperfEnabled?: boolean;
}

// Phase label translation keys
type SpeedtestPhase =
  | 'idle'
  | 'finding_server'
  | 'testing_latency'
  | 'testing_download'
  | 'testing_upload'
  | 'complete';
type IperfPhase = 'idle' | 'connecting' | 'testing' | 'complete';

export const PerformanceCard: React.NamedExoticComponent<PerformanceCardProps> = memo(
  // biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Performance card manages speedtest and iperf state machines with multiple polling effects and UI states
  function performanceCard({
    loading,
    runSpeedtestEnabled = true,
    runIperfEnabled = true,
  }: PerformanceCardProps): React.ReactElement {
    const { t } = useTranslation('cards');
    // Get iperf settings from context
    const { iperfSettings } = useSettings();

    // Helper to get speedtest phase label
    const getSpeedtestPhaseLabel = (phase: string): string => {
      switch (phase as SpeedtestPhase) {
        case 'idle':
          return t('performance.phaseReady');
        case 'finding_server':
          return t('performance.phaseFindingServer');
        case 'testing_latency':
          return t('performance.phaseTestingLatency');
        case 'testing_download':
          return t('performance.phaseTestingDownload');
        case 'testing_upload':
          return t('performance.phaseTestingUpload');
        case 'complete':
          return t('performance.phaseComplete');
        default:
          return phase;
      }
    };

    // Helper to get iperf phase label
    const getIperfPhaseLabel = (phase: string): string => {
      switch (phase as IperfPhase) {
        case 'idle':
          return t('performance.phaseReady');
        case 'connecting':
          return t('performance.phaseConnecting');
        case 'testing':
          return t('performance.phaseTesting');
        case 'complete':
          return t('performance.phaseComplete');
        default:
          return phase;
      }
    };

    // Speedtest state
    const [speedtestStatus, setSpeedtestStatus] = useState<SpeedtestStatus | null>(null);
    const [speedtestResult, setSpeedtestResult] = useState<SpeedtestData | null>(null);
    const [speedtestError, setSpeedtestError] = useState<string | null>(null);
    const [speedtestRunning, setSpeedtestRunning] = useState(false);

    // iperf3 state
    const [iperfInfo, setIperfInfo] = useState<IperfInfo | null>(null);
    const [iperfClientStatus, setIperfClientStatus] = useState<IperfClientStatus | null>(null);
    const [iperfResult, setIperfResult] = useState<IperfResult | null>(null);
    const [iperfServerStatus, setIperfServerStatus] = useState<IperfServerStatus | null>(null);
    const [iperfError, setIperfError] = useState<string | null>(null);
    const [iperfClientRunning, setIperfClientRunning] = useState(false);

    // Start/stop iperf server based on settings
    const manageIperfServer = useCallback(async (shouldRun: boolean, port: number) => {
      try {
        const action = shouldRun ? 'start' : 'stop';
        // Use api.post() for CSRF token inclusion (#CSRF-FIX)
        const res = await api.post('/api/v1/sap/iperf/server', { action, port });
        if (res.ok) {
          const statusRes = await fetch('/api/v1/sap/iperf/server/status', {
            credentials: 'include',
          });
          if (statusRes.ok) {
            // biome-ignore lint/nursery/useAwaitThenable: Response.json() returns a Promise
            setIperfServerStatus(await statusRes.json());
          }
        }
      } catch (err) {
        logger.error(LogComponents.Iperf, 'Failed to manage iperf server', err);
      }
    }, []);

    // Fetch initial status
    useEffect(() => {
      // biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Initial fetch requires checking multiple API endpoints
      const fetchStatus = async (): Promise<void> => {
        try {
          // Fetch speedtest status
          const speedRes = await fetch('/api/v1/sap/speedtest/status', {
            credentials: 'include',
          });
          if (speedRes.ok) {
            // biome-ignore lint/nursery/useAwaitThenable: Response.json() returns a Promise
            const data = await speedRes.json();
            setSpeedtestStatus(data);
            if (data.last) {
              setSpeedtestResult(data.last);
            }
            setSpeedtestRunning(data.running);
          }

          // Fetch iperf3 info
          const iperfInfoRes = await fetch('/api/v1/sap/iperf/info', {
            credentials: 'include',
          });
          if (iperfInfoRes.ok) {
            // biome-ignore lint/nursery/useAwaitThenable: Response.json() returns a Promise
            setIperfInfo(await iperfInfoRes.json());
          }

          // Fetch iperf3 client status
          const iperfClientRes = await fetch('/api/v1/sap/iperf/client/status', {
            credentials: 'include',
          });
          if (iperfClientRes.ok) {
            // biome-ignore lint/nursery/useAwaitThenable: Response.json() returns a Promise
            const data = await iperfClientRes.json();
            setIperfClientStatus(data);
            if (data.last) {
              setIperfResult(data.last);
            }
            setIperfClientRunning(data.running);
          }

          // Fetch iperf3 server status
          const iperfServerRes = await fetch('/api/v1/sap/iperf/server/status', {
            credentials: 'include',
          });
          if (iperfServerRes.ok) {
            // biome-ignore lint/nursery/useAwaitThenable: Response.json() returns a Promise
            setIperfServerStatus(await iperfServerRes.json());
          }
        } catch (err) {
          logger.error(LogComponents.Speedtest, 'Failed to fetch performance status', err);
        }
      };
      fetchStatus().catch(() => {
        // Error already logged in fetchStatus
      });
    }, []);

    // Track if we've done initial server sync
    const initialServerSyncDone = useRef(false);

    // Sync iperf server state on initial load and when settings change
    useEffect(() => {
      if (!iperfInfo?.installed) {
        return;
      }

      // On initial load or settings change, ensure server state matches settings
      if (!initialServerSyncDone.current || iperfSettings.enableServer) {
        // Check current server status and sync if needed
        // biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Server sync requires checking status and conditionally starting/stopping
        const syncServerState = async (): Promise<void> => {
          try {
            const res = await fetch('/api/v1/sap/iperf/server/status', {
              credentials: 'include',
            });
            if (res.ok) {
              // biome-ignore lint/nursery/useAwaitThenable: Response.json() returns a Promise
              const data = await res.json();
              const serverRunning = data.running === true;
              const shouldBeRunning = iperfSettings.enableServer;

              // If state doesn't match settings, fix it
              if (shouldBeRunning && !serverRunning) {
                await manageIperfServer(true, iperfSettings.serverPort);
              } else if (!shouldBeRunning && serverRunning) {
                await manageIperfServer(false, iperfSettings.serverPort);
              }
            }
          } catch {
            // If we can't check status and server should be running, try to start it
            if (iperfSettings.enableServer) {
              manageIperfServer(true, iperfSettings.serverPort).catch(() => {
                // Error already logged
              });
            }
          }
          initialServerSyncDone.current = true;
        };
        syncServerState().catch(() => {
          // Error already logged
        });
      } else if (!iperfSettings.enableServer) {
        // Server should be stopped
        manageIperfServer(false, iperfSettings.serverPort).catch(() => {
          // Error already logged
        });
      }
    }, [
      iperfSettings.enableServer,
      iperfSettings.serverPort,
      iperfInfo?.installed,
      manageIperfServer,
    ]);

    // Poll speedtest status while running (300ms for smooth gauge updates)
    useEffect(() => {
      if (!speedtestRunning) {
        return;
      }

      const interval = setInterval((): void => {
        // biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Speedtest poll handles status update and completion signaling
        (async (): Promise<void> => {
          try {
            const res = await fetch('/api/v1/sap/speedtest/status', {
              credentials: 'include',
            });
            if (res.ok) {
              // biome-ignore lint/nursery/useAwaitThenable: Response.json() returns a Promise
              const data = await res.json();
              setSpeedtestStatus(data);
              if (!data.running) {
                setSpeedtestRunning(false);
                if (data.last) {
                  setSpeedtestResult(data.last);
                }
                // Signal FAB that speedtest is complete
                window.dispatchEvent(
                  new CustomEvent('cardTestComplete', {
                    detail: { test: 'speedtest' },
                  }),
                );
              }
            }
          } catch (err) {
            logger.error(LogComponents.Speedtest, 'Failed to poll speedtest status', err);
          }
        })().catch(() => {
          // Error already logged
        });
      }, 300);

      return (): void => clearInterval(interval);
    }, [speedtestRunning]);

    // Poll iperf3 client status while running
    useEffect(() => {
      if (!iperfClientRunning) {
        return;
      }

      const interval = setInterval((): void => {
        // biome-ignore lint/complexity/noExcessiveCognitiveComplexity: iPerf poll handles status update and completion signaling
        (async (): Promise<void> => {
          try {
            const res = await fetch('/api/v1/sap/iperf/client/status', {
              credentials: 'include',
            });
            if (res.ok) {
              // biome-ignore lint/nursery/useAwaitThenable: Response.json() returns a Promise
              const data = await res.json();
              setIperfClientStatus(data);
              if (!data.running) {
                setIperfClientRunning(false);
                if (data.last) {
                  setIperfResult(data.last);
                }
                // Signal FAB that iperf is complete
                window.dispatchEvent(
                  new CustomEvent('cardTestComplete', {
                    detail: { test: 'iperf' },
                  }),
                );
              }
            }
          } catch (err) {
            logger.error(LogComponents.Iperf, 'Failed to poll iperf status', err);
          }
        })().catch(() => {
          // Error already logged
        });
      }, 1000);

      return (): void => clearInterval(interval);
    }, [iperfClientRunning]);

    const runSpeedtest = useCallback(async () => {
      if (!runSpeedtestEnabled) {
        setSpeedtestError(t('performance.testsDisabled'));
        return;
      }

      setSpeedtestError(null);
      setSpeedtestRunning(true);
      setSpeedtestStatus({ running: true, phase: 'finding_server', progress: 0 });

      try {
        await api.post('/api/v1/sap/speedtest');
      } catch (err) {
        setSpeedtestError(err instanceof Error ? err.message : t('performance.speedtestFailed'));
        setSpeedtestStatus({ running: false, phase: 'idle', progress: 0 });
        setSpeedtestRunning(false);
      }
    }, [runSpeedtestEnabled, t]);

    const runIperfClient = useCallback(async () => {
      if (!runIperfEnabled) {
        setIperfError(t('performance.testsDisabled'));
        return;
      }

      if (!iperfSettings.server) {
        setIperfError(t('performance.serverNotConfigured'));
        return;
      }

      setIperfError(null);
      setIperfClientRunning(true);
      setIperfClientStatus({ running: true, phase: 'connecting', progress: 0 });

      try {
        await api.post('/api/v1/sap/iperf/client', {
          server: iperfSettings.server,
          port: iperfSettings.port,
          protocol: iperfSettings.protocol,
          direction: iperfSettings.direction,
          reverse: iperfSettings.direction === 'download',
          duration: iperfSettings.duration,
          parallel: 1,
        });
      } catch (err) {
        setIperfError(err instanceof Error ? err.message : t('performance.iperfFailed'));
        setIperfClientStatus({ running: false, phase: 'idle', progress: 0 });
        setIperfClientRunning(false);
      }
    }, [iperfSettings, runIperfEnabled, t]);

    // Listen for FAB "run all tests" event
    useEffect(() => {
      const handleRunAllTests = (): void => {
        // Run speedtest if enabled
        if (runSpeedtestEnabled && !speedtestRunning) {
          runSpeedtest().catch(() => {
            // Error handled in runSpeedtest
          });
        }
        // Run iperf client test if enabled and configured
        if (
          runIperfEnabled &&
          !iperfClientRunning &&
          iperfSettings.server &&
          iperfInfo?.installed
        ) {
          // Delay slightly so tests don't all hammer at once
          setTimeout((): void => {
            runIperfClient().catch(() => {
              // Error handled in runIperfClient
            });
          }, 500);
        }
      };

      window.addEventListener('runAllTests', handleRunAllTests);
      return (): void => {
        window.removeEventListener('runAllTests', handleRunAllTests);
      };
    }, [
      runSpeedtest,
      runIperfClient,
      speedtestRunning,
      iperfClientRunning,
      iperfSettings.server,
      iperfInfo?.installed,
      runSpeedtestEnabled,
      runIperfEnabled,
    ]);

    const formatSpeed = (mbps: number): string => {
      if (mbps >= 1000) {
        return `${(mbps / 1000).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} Gbps`;
      }
      return `${mbps.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} Mbps`;
    };

    const getStatus = (): Status => {
      if (!(runSpeedtestEnabled || runIperfEnabled)) {
        return 'unknown';
      }
      if (loading || speedtestRunning || iperfClientRunning) {
        return 'loading';
      }
      if (speedtestError || iperfError) {
        return 'error';
      }
      if (speedtestResult || iperfResult) {
        return 'success';
      }
      return 'unknown';
    };

    return (
      <Card
        title={t('performance.title')}
        subtitle={t('performance.subtitle')}
        icon={<Gauge class={iconTokens.size.md} />}
        status={getStatus()}
      >
        <div>
          {/* Internet Speed Section */}
          <p class={cn('caption font-medium', spacing.margin.bottom.inline)}>
            {t('performance.internetSpeed')}
          </p>

          {speedtestRunning && speedtestStatus ? (
            <div class={spacing.margin.bottom.heading}>
              <div
                class={cn(layout.flex.center, spacing.gap.spacious, spacing.padding.bottom.inline)}
              >
                <SpeedGauge
                  value={speedtestStatus.currentDownload || 0}
                  label={t('performance.download')}
                  size="md"
                  isRunning={speedtestStatus.phase === 'testing_download'}
                />
                <SpeedGauge
                  value={speedtestStatus.currentUpload || 0}
                  label={t('performance.upload')}
                  size="md"
                  isRunning={speedtestStatus.phase === 'testing_upload'}
                />
              </div>
              <div class={cn(layout.inline.default, spacing.pad.xs, 'bg-surface-hover', radius.md)}>
                <PulsingDot color="primary" size="sm" />
                <span class="body-small font-medium">
                  {getSpeedtestPhaseLabel(speedtestStatus.phase)}
                </span>
                <span class="body-small text-text-muted ml-auto">
                  {Math.round(speedtestStatus.progress)}%
                </span>
              </div>
            </div>
          ) : null}

          {!speedtestRunning && speedtestResult ? (
            <div class={spacing.margin.bottom.heading}>
              <div
                class={cn(layout.flex.center, spacing.gap.spacious, spacing.padding.bottom.inline)}
              >
                <SpeedGauge
                  value={speedtestResult.download}
                  label={t('performance.download')}
                  size="md"
                />
                <SpeedGauge
                  value={speedtestResult.upload}
                  label={t('performance.upload')}
                  size="md"
                />
              </div>
              <CardRow
                label={t('performance.latency')}
                value={`${speedtestResult.latency.toFixed(0)} ms`}
              />
              <CardRow label={t('performance.server')} value={speedtestResult.location} />
            </div>
          ) : null}

          {speedtestRunning || speedtestResult || speedtestError ? null : (
            <p class={cn('body-small', spacing.margin.bottom.inline)}>
              {t('performance.noResults')}
            </p>
          )}

          {speedtestError ? <p class="body-small text-status-error">{speedtestError}</p> : null}

          <CardDivider />

          {/* LAN Speed (iperf3) Section */}
          <p
            class={cn(
              'caption font-medium',
              spacing.margin.bottom.inline,
              spacing.margin.top.inline,
            )}
          >
            {t('performance.lanSpeed')}
            {iperfInfo?.version ? (
              <span class={cn('text-text-muted font-normal', spacing.margin.left.inline)}>
                {iperfInfo.version}
              </span>
            ) : null}
          </p>

          {iperfInfo?.installed ? null : (
            <p class={cn('body-small text-status-warning', spacing.margin.bottom.heading)}>
              {t('performance.iperfNotInstalled')}
            </p>
          )}

          {iperfInfo?.installed ? (
            <>
              {/* Config Summary */}
              {iperfSettings.server ? (
                <div
                  class={cn(
                    'caption',
                    spacing.margin.bottom.heading,
                    spacing.pad.sm,
                    'bg-surface-hover',
                    radius.default,
                  )}
                >
                  <div class={layout.flex.between}>
                    <span>{t('performance.server')}:</span>
                    <span class="text-text-primary">
                      {iperfSettings.server}:{iperfSettings.port}
                    </span>
                  </div>
                  <div class={layout.flex.between}>
                    <span>{t('performance.test')}:</span>
                    <span class="text-text-primary">
                      {iperfSettings.protocol.toUpperCase()}{' '}
                      {iperfSettings.direction === 'bidirectional'
                        ? t('performance.both')
                        : iperfSettings.direction}
                    </span>
                  </div>
                </div>
              ) : (
                <p class={cn('caption', spacing.margin.bottom.heading)}>
                  {t('performance.configureServer')}
                </p>
              )}

              {/* Client Status/Results */}
              {iperfClientRunning && iperfClientStatus ? (
                <div
                  class={cn(
                    layout.inline.spacious,
                    spacing.margin.bottom.heading,
                    spacing.pad.sm,
                    'bg-surface-hover',
                    radius.lg,
                  )}
                >
                  <ProgressRing progress={iperfClientStatus.progress} size={56} strokeWidth={5} />
                  <div class="flex-1">
                    <div class={layout.inline.default}>
                      <PulsingDot color="primary" size="sm" />
                      <span class="body-small font-medium">
                        {getIperfPhaseLabel(iperfClientStatus.phase)}
                      </span>
                    </div>
                    {(() => {
                      const pp = Math.min(Math.max(iperfClientStatus.progress, 0), 100);
                      return (
                        <progress
                          value={pp}
                          max={100}
                          aria-label="iPerf progress"
                          class={cn(spacing.margin.top.inline, 'w-full', radius.full)}
                        />
                      );
                    })()}
                  </div>
                </div>
              ) : null}

              {!iperfClientRunning && iperfResult ? (
                <div class={cn(spacing.margin.bottom.heading, 'stack-sm')}>
                  {iperfResult.direction === 'bidirectional' ? (
                    <div class={cn('grid grid-cols-1 sm:grid-cols-2', spacing.gap.default)}>
                      <CardValue
                        label={t('performance.download')}
                        value={formatSpeed(iperfResult.downloadBandwidth ?? iperfResult.bandwidth)}
                        size="md"
                        status="success"
                      />
                      <CardValue
                        label={t('performance.upload')}
                        value={formatSpeed(iperfResult.uploadBandwidth ?? iperfResult.bandwidth)}
                        size="md"
                        status="success"
                      />
                    </div>
                  ) : (
                    <CardValue
                      label={
                        iperfResult.direction === 'download'
                          ? t('performance.download')
                          : t('performance.upload')
                      }
                      value={formatSpeed(iperfResult.bandwidth)}
                      size="md"
                      status="success"
                    />
                  )}

                  {iperfResult.direction === 'bidirectional' ? (
                    <div class={cn('grid grid-cols-1 sm:grid-cols-2', spacing.gap.default)}>
                      {iperfResult.downloadTransfer !== undefined ? (
                        <CardRow
                          label={t('performance.downloadTransfer')}
                          value={`${iperfResult.downloadTransfer.toFixed(1)} MB`}
                        />
                      ) : null}
                      {iperfResult.uploadTransfer !== undefined ? (
                        <CardRow
                          label={t('performance.uploadTransfer')}
                          value={`${iperfResult.uploadTransfer.toFixed(1)} MB`}
                        />
                      ) : null}
                    </div>
                  ) : (
                    <CardRow
                      label={t('performance.transfer')}
                      value={`${iperfResult.transfer.toFixed(1)} MB`}
                    />
                  )}

                  {iperfResult.protocol === 'tcp' && iperfResult.retransmits > 0 ? (
                    <CardRow
                      label={t('performance.retransmits')}
                      value={iperfResult.retransmits.toString()}
                    />
                  ) : null}
                  {iperfResult.protocol === 'udp' ? (
                    <>
                      <CardRow
                        label={t('performance.jitter')}
                        value={`${iperfResult.jitter.toFixed(2)} ms`}
                      />
                      <CardRow
                        label={t('performance.packetLoss')}
                        value={`${iperfResult.lostPercent.toFixed(2)}%`}
                      />
                    </>
                  ) : null}
                </div>
              ) : null}

              {iperfError ? <p class="body-small text-status-error">{iperfError}</p> : null}

              {/* Server status indicator (if enabled) */}
              {iperfSettings.enableServer ? (
                <div
                  class={cn(
                    'caption',
                    layout.flex.between,
                    'pad-sm bg-surface-hover',
                    radius.default,
                  )}
                >
                  <span>{t('performance.serverMode')}</span>
                  <span
                    class={iperfServerStatus?.running ? 'text-status-success' : 'text-text-muted'}
                  >
                    {iperfServerStatus?.running
                      ? t('performance.listening', {
                          port: iperfServerStatus.port,
                        })
                      : t('performance.stopped')}
                  </span>
                </div>
              ) : null}
            </>
          ) : null}
        </div>
      </Card>
    );
  },
);
