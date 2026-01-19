// biome-ignore-all lint/complexity/noExcessiveCognitiveComplexity: Complex component
/**
 * Health Checks Hook
 *
 * Manages network and system health check operations including DNS testing,
 * gateway reachability, and custom test execution.
 *
 * Features:
 * - Run DNS resolution tests
 * - Check gateway connectivity
 * - Execute custom health check tests
 * - Track test progress and results
 * - Configure test settings
 *
 * Usage:
 * ```typescript
 * const { runTests, testResults, isRunning } = useHealthChecks();
 *
 * // Run all configured tests
 * await runTests();
 *
 * // Run specific test types
 * await runTests({ types: ['dns', 'gateway'] });
 * ```
 */

import { useCallback, useState } from 'react';
import { api } from '../api';
import { LogComponents, logger } from '../lib/logger';

/** DNS test result */
export interface DnsTestResult {
  server: string;
  hostname: string;
  responseTime: number;
  status: 'success' | 'timeout' | 'error';
  resolvedIp?: string;
  error?: string;
}

/** Gateway test result */
export interface GatewayTestResult {
  gateway: string;
  latency: number;
  packetLoss: number;
  reachable: boolean;
  interface: string;
}

/** Custom test configuration */
export interface TestConfig {
  name: string;
  type: 'dns' | 'http' | 'tcp' | 'icmp';
  target: string;
  port?: number;
  timeout?: number;
  enabled: boolean;
}

/** Custom test result */
export interface CustomTestResult {
  name: string;
  type: string;
  target: string;
  success: boolean;
  responseTime: number;
  error?: string;
  details?: Record<string, unknown>;
}

/** Combined test results */
export interface HealthCheckResults {
  dns: DnsTestResult[];
  gateway: GatewayTestResult | null;
  custom: CustomTestResult[];
  timestamp: string;
  overall: 'healthy' | 'degraded' | 'unhealthy';
}

/** Test settings */
export interface TestsSettings {
  dnsServers: Array<{
    address: string;
    enabled: boolean;
  }>;
  dnsTestHostname: string;
  gatewayPingCount: number;
  customTests: TestConfig[];
}

/** Options for running tests */
interface RunTestsOptions {
  types?: ('dns' | 'gateway' | 'custom')[];
}

/**
 * Custom hook for managing health check operations.
 *
 * Provides functions to run various network health checks and retrieve results.
 *
 * @returns Health check state and control functions
 */
export function useHealthChecks(): {
  results: HealthCheckResults | null;
  isRunning: boolean;
  error: string | null;
  runTests: (options?: RunTestsOptions) => Promise<HealthCheckResults | null>;
  runDnsTests: () => Promise<DnsTestResult[]>;
  runGatewayTest: () => Promise<GatewayTestResult | null>;
  runCustomTests: () => Promise<CustomTestResult[]>;
  fetchSettings: () => Promise<TestsSettings | null>;
  updateSettings: (settings: Partial<TestsSettings>) => Promise<boolean>;
} {
  const [results, setResults] = useState<HealthCheckResults | null>(null);
  const [isRunning, setIsRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);

  /**
   * Runs DNS resolution tests.
   */
  const runDnsTests = useCallback(async (): Promise<DnsTestResult[]> => {
    try {
      const data = await api.get<{ results: DnsTestResult[] }>('/api/v1/sap/dns');
      return data.results || [];
    } catch (err) {
      logger.error(LogComponents.Dns, 'DNS tests failed', err, {
        endpoint: '/api/v1/sap/dns',
      });
      return [];
    }
  }, []);

  /**
   * Runs gateway connectivity test.
   */
  const runGatewayTest = useCallback(async (): Promise<GatewayTestResult | null> => {
    try {
      return await api.get<GatewayTestResult>('/api/v1/sap/gateway');
    } catch (err) {
      logger.error(LogComponents.Gateway, 'Gateway test failed', err, {
        endpoint: '/api/v1/sap/gateway',
      });
      return null;
    }
  }, []);

  /**
   * Runs custom tests.
   */
  const runCustomTests = useCallback(async (): Promise<CustomTestResult[]> => {
    try {
      setIsRunning(true);
      const data = await api.post<{ results: CustomTestResult[] }>('/api/v1/sap/health-checks/run');
      return data.results || [];
    } catch (err) {
      logger.error(LogComponents.System, 'Custom tests failed', err, {
        endpoint: '/api/v1/sap/health-checks/run',
      });
      return [];
    } finally {
      setIsRunning(false);
    }
  }, []);

  /**
   * Runs specified health check tests.
   */
  const runTests = useCallback(
    async (options: RunTestsOptions = {}): Promise<HealthCheckResults | null> => {
      const { types = ['dns', 'gateway', 'custom'] } = options;

      try {
        setError(null);
        setIsRunning(true);

        // Run selected tests in parallel
        const testPromises: Promise<unknown>[] = [];
        const testTypes: string[] = [];

        if (types.includes('dns')) {
          testPromises.push(runDnsTests());
          testTypes.push('dns');
        }
        if (types.includes('gateway')) {
          testPromises.push(runGatewayTest());
          testTypes.push('gateway');
        }
        if (types.includes('custom')) {
          testPromises.push(runCustomTests());
          testTypes.push('custom');
        }

        const settledResults = await Promise.allSettled(testPromises);

        // Organize results
        const healthResults: HealthCheckResults = {
          dns: [],
          gateway: null,
          custom: [],
          timestamp: new Date().toISOString(),
          overall: 'healthy',
        };

        // Process results - map indices to test types
        // Using forEach index which is safe as we control both arrays
        const resultMap = new Map<string, PromiseSettledResult<unknown>>();
        for (const [idx, type] of testTypes.entries()) {
          const result = settledResults[idx];
          if (result) {
            resultMap.set(type, result);
          }
        }

        // Extract results by type
        const dnsResult = resultMap.get('dns');
        if (dnsResult?.status === 'fulfilled') {
          healthResults.dns = dnsResult.value as DnsTestResult[];
        }

        const gatewayResult = resultMap.get('gateway');
        if (gatewayResult?.status === 'fulfilled') {
          healthResults.gateway = gatewayResult.value as GatewayTestResult;
        }

        const customResult = resultMap.get('custom');
        if (customResult?.status === 'fulfilled') {
          healthResults.custom = customResult.value as CustomTestResult[];
        }

        // Determine overall health status
        const dnsOk = healthResults.dns.every((d) => d.status === 'success');
        const gatewayOk = healthResults.gateway?.reachable ?? true;
        const customOk = healthResults.custom.every((c) => c.success);

        if (!(dnsOk && gatewayOk)) {
          healthResults.overall = 'unhealthy';
        } else if (!customOk) {
          healthResults.overall = 'degraded';
        }

        setResults(healthResults);
        return healthResults;
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Health check failed';
        setError(message);
        logger.error(LogComponents.System, 'Health check failed', err, {
          types,
        });
        return null;
      } finally {
        setIsRunning(false);
      }
    },
    [runDnsTests, runGatewayTest, runCustomTests],
  );

  /**
   * Fetches test settings.
   */
  const fetchSettings = useCallback(async (): Promise<TestsSettings | null> => {
    try {
      return await api.get<TestsSettings>('/api/v1/sap/health-checks/settings');
    } catch (err) {
      logger.error(LogComponents.CONFIG, 'Failed to fetch test settings', err, {
        endpoint: '/api/v1/sap/health-checks/settings',
      });
      return null;
    }
  }, []);

  /**
   * Updates test settings.
   */
  const updateSettings = useCallback(async (settings: Partial<TestsSettings>): Promise<boolean> => {
    try {
      await api.put('/api/v1/sap/health-checks/settings', settings);
      return true;
    } catch (err) {
      logger.error(LogComponents.CONFIG, 'Failed to update test settings', err, {
        endpoint: '/api/v1/sap/health-checks/settings',
        updates: settings,
      });
      return false;
    }
  }, []);

  return {
    // State
    results,
    isRunning,
    error,

    // Test operations
    runTests,
    runDnsTests,
    runGatewayTest,
    runCustomTests,

    // Settings operations
    fetchSettings,
    updateSettings,
  };
}
