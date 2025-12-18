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

import { useState, useCallback } from "react";
import { api } from "../lib/api";
import { logger, LogComponents } from "../lib/logger";

/** DNS test result */
export interface DNSTestResult {
  server: string;
  hostname: string;
  responseTime: number;
  status: "success" | "timeout" | "error";
  resolvedIP?: string;
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
  type: "dns" | "http" | "tcp" | "icmp";
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
  dns: DNSTestResult[];
  gateway: GatewayTestResult | null;
  custom: CustomTestResult[];
  timestamp: string;
  overall: "healthy" | "degraded" | "unhealthy";
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
  types?: ("dns" | "gateway" | "custom")[];
}

/**
 * Custom hook for managing health check operations.
 *
 * Provides functions to run various network health checks and retrieve results.
 *
 * @returns Health check state and control functions
 */
export function useHealthChecks() {
  const [results, setResults] = useState<HealthCheckResults | null>(null);
  const [isRunning, setIsRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);

  /**
   * Runs DNS resolution tests.
   */
  const runDNSTests = useCallback(async (): Promise<DNSTestResult[]> => {
    try {
      const data = await api.get<{ results: DNSTestResult[] }>("/api/dns");
      return data.results || [];
    } catch (err) {
      logger.error(LogComponents.DNS, "DNS tests failed", err, { endpoint: "/api/dns" });
      return [];
    }
  }, []);

  /**
   * Runs gateway connectivity test.
   */
  const runGatewayTest = useCallback(async (): Promise<GatewayTestResult | null> => {
    try {
      return await api.get<GatewayTestResult>("/api/gateway");
    } catch (err) {
      logger.error(LogComponents.GATEWAY, "Gateway test failed", err, { endpoint: "/api/gateway" });
      return null;
    }
  }, []);

  /**
   * Runs custom tests.
   */
  const runCustomTests = useCallback(async (): Promise<CustomTestResult[]> => {
    try {
      setIsRunning(true);
      const data = await api.post<{ results: CustomTestResult[] }>("/api/health-checks/run");
      return data.results || [];
    } catch (err) {
      logger.error(LogComponents.SYSTEM, "Custom tests failed", err, {
        endpoint: "/api/health-checks/run",
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
      const { types = ["dns", "gateway", "custom"] } = options;

      try {
        setError(null);
        setIsRunning(true);

        // Run selected tests in parallel
        const testPromises: Promise<unknown>[] = [];
        const testTypes: string[] = [];

        if (types.includes("dns")) {
          testPromises.push(runDNSTests());
          testTypes.push("dns");
        }
        if (types.includes("gateway")) {
          testPromises.push(runGatewayTest());
          testTypes.push("gateway");
        }
        if (types.includes("custom")) {
          testPromises.push(runCustomTests());
          testTypes.push("custom");
        }

        const results = await Promise.allSettled(testPromises);

        // Organize results
        const healthResults: HealthCheckResults = {
          dns: [],
          gateway: null,
          custom: [],
          timestamp: new Date().toISOString(),
          overall: "healthy",
        };

        // Process results - map indices to test types
        // Using forEach index which is safe as we control both arrays
        const resultMap = new Map<string, PromiseSettledResult<unknown>>();
        testTypes.forEach((type, idx) => {
          // eslint-disable-next-line security/detect-object-injection
          resultMap.set(type, results[idx]!);
        });

        // Extract results by type
        const dnsResult = resultMap.get("dns");
        if (dnsResult?.status === "fulfilled") {
          healthResults.dns = dnsResult.value as DNSTestResult[];
        }

        const gatewayResult = resultMap.get("gateway");
        if (gatewayResult?.status === "fulfilled") {
          healthResults.gateway = gatewayResult.value as GatewayTestResult;
        }

        const customResult = resultMap.get("custom");
        if (customResult?.status === "fulfilled") {
          healthResults.custom = customResult.value as CustomTestResult[];
        }

        // Determine overall health status
        const dnsOk = healthResults.dns.every((d) => d.status === "success");
        const gatewayOk = healthResults.gateway?.reachable ?? true;
        const customOk = healthResults.custom.every((c) => c.success);

        if (!dnsOk || !gatewayOk) {
          healthResults.overall = "unhealthy";
        } else if (!customOk) {
          healthResults.overall = "degraded";
        }

        setResults(healthResults);
        return healthResults;
      } catch (err) {
        const message = err instanceof Error ? err.message : "Health check failed";
        setError(message);
        logger.error(LogComponents.SYSTEM, "Health check failed", err, {
          types,
        });
        return null;
      } finally {
        setIsRunning(false);
      }
    },
    [runDNSTests, runGatewayTest, runCustomTests]
  );

  /**
   * Fetches test settings.
   */
  const fetchSettings = useCallback(async (): Promise<TestsSettings | null> => {
    try {
      return await api.get<TestsSettings>("/api/health-checks/settings");
    } catch (err) {
      logger.error(LogComponents.CONFIG, "Failed to fetch test settings", err, {
        endpoint: "/api/health-checks/settings",
      });
      return null;
    }
  }, []);

  /**
   * Updates test settings.
   */
  const updateSettings = useCallback(async (settings: Partial<TestsSettings>): Promise<boolean> => {
    try {
      await api.put("/api/health-checks/settings", settings);
      return true;
    } catch (err) {
      logger.error(LogComponents.CONFIG, "Failed to update test settings", err, {
        endpoint: "/api/health-checks/settings",
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
    runDNSTests,
    runGatewayTest,
    runCustomTests,

    // Settings operations
    fetchSettings,
    updateSettings,
  };
}
