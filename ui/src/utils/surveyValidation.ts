/**
 * Survey Validation Utilities
 *
 * Purpose: Validate survey data against configurable pass/fail criteria.
 * Provides functions for running validation tests and generating results.
 *
 * Key Features:
 * - Validate surveys against configurable thresholds
 * - Calculate pass/fail percentages per criterion
 * - Identify failed sample locations for visualization
 * - Support for passive, active, and throughput survey types
 *
 * Usage:
 * ```typescript
 * import { validateSurvey, getDefaultCriteria } from './surveyValidation';
 *
 * const criteria = getDefaultCriteria('passive');
 * const results = validateSurvey(survey, criteria);
 *
 * if (results.overallPass) {
 *   // Survey passed all criteria
 * }
 * ```
 */

import type {
  ActiveSample,
  ComparisonOperator,
  HeatmapMetric,
  PassFailCriterion,
  PassFailResult,
  PassiveSample,
  SamplePoint,
  ScannedNetwork,
  Survey,
  SurveyValidation,
  ThroughputSample,
} from "../hooks/useSurvey";

/**
 * Extract metric value from a sample based on the criterion
 */
function extractMetricValue(sample: SamplePoint, criterion: PassFailCriterion): number | null {
  const data = sample.sampleData;

  // Handle passive samples (array of networks)
  if ("networks" in data && Array.isArray(data.networks)) {
    const networks = data.networks as ScannedNetwork[];
    if (networks.length === 0) return null;

    // Sort networks by RSSI (strongest first)
    const sortedNetworks = [...networks].sort((a, b) => b.rssi - a.rssi);

    switch (criterion.metric) {
      case "rssi": {
        // For apIndex, get the nth strongest AP
        const apIndex = criterion.apIndex ?? 0;
        const network = sortedNetworks.at(apIndex);
        if (!network) return null;
        return network.rssi;
      }

      case "snr": {
        // Use the strongest AP's SNR, or calculate from RSSI and noise
        const apIndex = criterion.apIndex ?? 0;
        const network = sortedNetworks.at(apIndex);
        if (!network) return null;
        if (network.snr !== undefined) return network.snr;
        if (network.noiseFloor !== undefined) {
          return network.rssi - network.noiseFloor;
        }
        // Default noise floor assumption if not available
        const passiveData = data as PassiveSample;
        const noiseFloor = passiveData.noiseFloor ?? -95;
        return network.rssi - noiseFloor;
      }

      case "noise": {
        const passiveData = data as PassiveSample;
        return passiveData.noiseFloor ?? sortedNetworks[0]?.noiseFloor ?? null;
      }

      case "cochannel": {
        // Count APs on the same channel as the strongest AP
        const primaryChannel = sortedNetworks[0]?.channel;
        if (!primaryChannel) return 0;
        // Count APs with strong signal on same channel (exclude the primary)
        return networks.filter((n) => n.channel === primaryChannel && n.rssi >= -85).length;
      }

      case "adjacent": {
        // Count APs on adjacent channels (within 4 channels for 2.4GHz)
        const primaryChannel = sortedNetworks[0]?.channel;
        if (!primaryChannel) return 0;
        return networks.filter(
          (n) =>
            n.channel !== primaryChannel &&
            Math.abs(n.channel - primaryChannel) <= 4 &&
            n.rssi >= -85,
        ).length;
      }

      case "channelUtil": {
        // Not typically available from passive scan
        return null;
      }

      case "throughput": {
        // Passive scans don't have throughput, use max TX rate
        const maxRate = Math.max(...networks.map((n) => n.txRate ?? 0));
        return maxRate > 0 ? maxRate : null;
      }

      default:
        return null;
    }
  }

  // Handle active samples
  if ("dataRate" in data) {
    const activeData = data as ActiveSample;

    switch (criterion.metric) {
      case "rssi":
        return activeData.rssi;
      case "throughput":
        return activeData.dataRate;
      default:
        return null;
    }
  }

  // Handle throughput samples
  if ("downloadMbps" in data) {
    const throughputData = data as ThroughputSample;

    switch (criterion.metric) {
      case "rssi":
        return throughputData.rssi;
      case "throughput":
        // Use download speed as the primary throughput metric
        return throughputData.downloadMbps;
      case "latency":
        // Check if this is the latency or jitter criterion
        if (criterion.name === "jitter") {
          return throughputData.jitter;
        }
        return throughputData.latency;
      default:
        return null;
    }
  }

  return null;
}

/**
 * Check if a value passes the threshold based on comparison operator
 */
function passesThreshold(
  value: number,
  threshold: number,
  comparison: ComparisonOperator,
): boolean {
  if (comparison === "gte") {
    return value >= threshold;
  }
  return value <= threshold;
}

/**
 * Validate a single criterion against all survey samples
 */
export function validateCriterion(
  samples: SamplePoint[],
  criterion: PassFailCriterion,
): PassFailResult {
  const values: Array<{ x: number; y: number; value: number }> = [];
  const failedLocations: Array<{ x: number; y: number; value: number }> = [];

  // Extract values from each sample
  for (const sample of samples) {
    const value = extractMetricValue(sample, criterion);
    if (value !== null) {
      values.push({ x: sample.x, y: sample.y, value });
      if (!passesThreshold(value, criterion.threshold, criterion.comparison)) {
        failedLocations.push({ x: sample.x, y: sample.y, value });
      }
    }
  }

  const totalSampleCount = values.length;
  const failedSampleCount = failedLocations.length;
  const passedSampleCount = totalSampleCount - failedSampleCount;
  const percentage = totalSampleCount > 0 ? (passedSampleCount / totalSampleCount) * 100 : 0;

  // Calculate statistics
  const allValues = values.map((v) => v.value);
  const averageValue =
    allValues.length > 0 ? allValues.reduce((a, b) => a + b, 0) / allValues.length : 0;

  // For "gte" comparisons, worst is minimum; for "lte", worst is maximum
  const worstValue =
    criterion.comparison === "gte"
      ? Math.min(...allValues, Number.POSITIVE_INFINITY)
      : Math.max(...allValues, Number.NEGATIVE_INFINITY);

  const bestValue =
    criterion.comparison === "gte"
      ? Math.max(...allValues, Number.NEGATIVE_INFINITY)
      : Math.min(...allValues, Number.POSITIVE_INFINITY);

  // Criterion passes if all samples pass (or we use a percentage threshold)
  // Using 100% pass rate as the default - all samples must pass
  const passed = failedSampleCount === 0;

  return {
    criterionId: criterion.id,
    criterionName: criterion.name,
    passed,
    averageValue: Number.isFinite(averageValue) ? averageValue : 0,
    worstValue: Number.isFinite(worstValue) ? worstValue : 0,
    bestValue: Number.isFinite(bestValue) ? bestValue : 0,
    threshold: criterion.threshold,
    comparison: criterion.comparison,
    suffix: criterion.suffix,
    failedSampleCount,
    totalSampleCount,
    failedLocations,
    percentage: Math.round(percentage * 10) / 10, // Round to 1 decimal
  };
}

/**
 * Validate a survey against all provided criteria
 */
export function validateSurvey(survey: Survey, criteria: PassFailCriterion[]): SurveyValidation {
  // Filter to enabled criteria that match the survey type
  const applicableCriteria = criteria.filter(
    (c) => c.enabled && (c.mode === "all" || c.mode === survey.surveyType),
  );

  // Validate each criterion
  const results: PassFailResult[] = applicableCriteria.map((criterion) =>
    validateCriterion(survey.samples, criterion),
  );

  // Calculate overall results
  const passedCount = results.filter((r) => r.passed).length;
  const failedCount = results.filter((r) => !r.passed).length;
  const overallPass = failedCount === 0;
  const overallPercentage = results.length > 0 ? (passedCount / results.length) * 100 : 100;

  return {
    overallPass,
    overallPercentage: Math.round(overallPercentage * 10) / 10,
    results,
    timestamp: new Date().toISOString(),
    criteria: applicableCriteria,
    passedCount,
    failedCount,
    surveyId: survey.id,
  };
}

/**
 * Get a summary of validation results suitable for display
 */
export function getValidationSummary(validation: SurveyValidation): {
  status: "pass" | "fail" | "warning";
  message: string;
  passedCount: number;
  totalCount: number;
} {
  const { passedCount, failedCount, results } = validation;
  const totalCount = results.length;

  if (failedCount === 0) {
    return {
      status: "pass",
      message: `All ${totalCount} criteria passed`,
      passedCount,
      totalCount,
    };
  }

  if (passedCount === 0) {
    return {
      status: "fail",
      message: `All ${totalCount} criteria failed`,
      passedCount,
      totalCount,
    };
  }

  return {
    status: "warning",
    message: `${passedCount} of ${totalCount} criteria passed`,
    passedCount,
    totalCount,
  };
}

/**
 * Calculate statistics for a specific metric across all samples
 */
export function calculateMetricStatistics(
  samples: SamplePoint[],
  metric: HeatmapMetric,
): {
  min: number;
  max: number;
  average: number;
  median: number;
  sampleCount: number;
} | null {
  if (!metric) return null;

  // Create a dummy criterion to extract values
  const dummyCriterion: PassFailCriterion = {
    id: "stats",
    name: "stats",
    displayKey: "stats",
    metric,
    comparison: "gte",
    threshold: 0,
    suffix: "",
    enabled: true,
    mode: "all",
  };

  const values: number[] = [];
  for (const sample of samples) {
    const value = extractMetricValue(sample, dummyCriterion);
    if (value !== null) {
      values.push(value);
    }
  }

  if (values.length === 0) return null;

  values.sort((a, b) => a - b);

  const min = values[0];
  const max = values[values.length - 1];
  const average = values.reduce((a, b) => a + b, 0) / values.length;
  const median =
    values.length % 2 === 0
      ? (values[values.length / 2 - 1] + values[values.length / 2]) / 2
      : values[Math.floor(values.length / 2)];

  return {
    min,
    max,
    average: Math.round(average * 10) / 10,
    median: Math.round(median * 10) / 10,
    sampleCount: values.length,
  };
}

/**
 * Get the percentage of samples meeting a threshold
 */
export function getPercentageMeetingThreshold(
  samples: SamplePoint[],
  metric: HeatmapMetric,
  threshold: number,
  comparison: ComparisonOperator = "gte",
): number {
  if (!metric) return 0;

  const dummyCriterion: PassFailCriterion = {
    id: "threshold",
    name: "threshold",
    displayKey: "threshold",
    metric,
    comparison,
    threshold,
    suffix: "",
    enabled: true,
    mode: "all",
  };

  let passing = 0;
  let total = 0;

  for (const sample of samples) {
    const value = extractMetricValue(sample, dummyCriterion);
    if (value !== null) {
      total++;
      if (passesThreshold(value, threshold, comparison)) {
        passing++;
      }
    }
  }

  return total > 0 ? Math.round((passing / total) * 1000) / 10 : 0;
}

/**
 * Format a criterion result for display
 */
export function formatCriterionResult(result: PassFailResult): string {
  const comparison = result.comparison === "gte" ? "\u2265" : "\u2264";
  const status = result.passed ? "\u2713" : "\u2717";
  return `${status} ${result.criterionName}: ${result.averageValue.toFixed(1)} ${result.suffix} (${comparison}${result.threshold})`;
}
