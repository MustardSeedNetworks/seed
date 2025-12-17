/**
 * Survey Report Generator
 *
 * Purpose: Generate comprehensive survey report data from survey results.
 * Creates structured report data that can be rendered to PDF or HTML.
 *
 * Key Features:
 * - Executive summary with overall status
 * - Pass/fail criteria results table
 * - Heatmap statistics and descriptions
 * - Analysis findings and recommendations
 * - AP inventory listing
 *
 * Usage:
 * ```typescript
 * import { generateReport } from './reportGenerator';
 *
 * const report = generateReport(survey, validation, options);
 * console.log(report.summary.overallStatus);
 * ```
 */

import type {
  Survey,
  SurveyValidation,
  HeatmapMetric,
  SamplePoint,
  PassiveSample,
} from "../hooks/useSurvey";
import type { AnalysisFinding } from "../components/survey/SurveyAnalysisPanel";
import { calculateMetricStatistics, getPercentageMeetingThreshold } from "./surveyValidation";

/** Report metadata */
export interface ReportMetadata {
  title: string;
  surveyName: string;
  surveyId: string;
  generatedAt: string;
  generatedBy: string;
  surveyType: string;
  sampleCount: number;
  facilitySize: string;
  scaleInfo: string;
}

/** Report summary section */
export interface ReportSummary {
  overallStatus: "pass" | "fail";
  passedCriteria: number;
  totalCriteria: number;
  overallPercentage: number;
  keyFindings: string[];
}

/** Statistics for a single heatmap metric */
export interface HeatmapStatistics {
  min: number;
  max: number;
  average: number;
  median: number;
  sampleCount: number;
  percentMeetingThreshold: number;
}

/** Heatmap section in report */
export interface ReportHeatmap {
  type: HeatmapMetric;
  displayName: string;
  descriptionKey: string;
  statistics: HeatmapStatistics | null;
  threshold?: number;
  unit: string;
}

/** AP inventory entry */
export interface APInventoryEntry {
  ssid: string;
  bssid: string;
  channel: number;
  band: string;
  security: string;
  vendor: string;
  avgRssi: number;
  sampleCount: number;
}

/** Complete report structure */
export interface SurveyReport {
  metadata: ReportMetadata;
  summary: ReportSummary;
  validation: SurveyValidation | null;
  heatmaps: ReportHeatmap[];
  analysis: AnalysisFinding[];
  apInventory: APInventoryEntry[];
  recommendations: string[];
}

/** Report generation options */
export interface ReportOptions {
  includeHeatmaps?: HeatmapMetric[];
  includeAnalysis?: boolean;
  includeAPInventory?: boolean;
  locale?: string;
  companyName?: string;
}

/** Default heatmaps to include based on survey type */
const DEFAULT_HEATMAPS: Record<string, HeatmapMetric[]> = {
  passive: ["rssi", "snr", "noise", "cochannel", "adjacent"],
  active: ["rssi", "snr", "throughput"],
  throughput: ["rssi", "throughput", "latency"],
};

/** Heatmap display names */
const HEATMAP_NAMES: Record<string, string> = {
  rssi: "Signal Strength (RSSI)",
  snr: "Signal-to-Noise Ratio (SNR)",
  noise: "Noise Floor",
  throughput: "Throughput",
  latency: "Latency",
  cochannel: "Co-Channel Interference",
  adjacent: "Adjacent Channel Interference",
  channelUtil: "Channel Utilization",
};

/** Heatmap units */
const HEATMAP_UNITS: Record<string, string> = {
  rssi: "dBm",
  snr: "dB",
  noise: "dBm",
  throughput: "Mbps",
  latency: "ms",
  cochannel: "APs",
  adjacent: "APs",
  channelUtil: "%",
};

/** Default thresholds for percentage calculations */
const DEFAULT_THRESHOLDS: Record<string, { value: number; comparison: "gte" | "lte" }> = {
  rssi: { value: -65, comparison: "gte" },
  snr: { value: 25, comparison: "gte" },
  noise: { value: -90, comparison: "lte" },
  throughput: { value: 100, comparison: "gte" },
  latency: { value: 50, comparison: "lte" },
  cochannel: { value: 4, comparison: "lte" },
  adjacent: { value: 2, comparison: "lte" },
};

/**
 * Calculate facility size from floor plan dimensions
 */
function calculateFacilitySize(survey: Survey): string {
  if (!survey.floorPlan) {
    return "Unknown";
  }

  const { width, height, scaleM } = survey.floorPlan;
  const widthM = width * scaleM;
  const heightM = height * scaleM;
  const areaM2 = widthM * heightM;

  return `${widthM.toFixed(1)} x ${heightM.toFixed(1)} m (${areaM2.toFixed(0)} m\u00B2)`;
}

/**
 * Get scale information string
 */
function getScaleInfo(survey: Survey): string {
  if (!survey.floorPlan) {
    return "No floor plan";
  }

  const { scaleM, scaleSource } = survey.floorPlan;
  const ppm = 1 / scaleM;
  const sourceLabel = scaleSource || "default";

  return `${scaleM.toFixed(4)} m/px (${ppm.toFixed(1)} px/m) - ${sourceLabel}`;
}

/**
 * Generate key findings from validation results
 */
function generateKeyFindings(
  validation: SurveyValidation | null,
  analysis: AnalysisFinding[]
): string[] {
  const findings: string[] = [];

  // Add validation-based findings
  if (validation) {
    const failedCriteria = validation.results.filter((r) => !r.passed);
    if (failedCriteria.length === 0) {
      findings.push("All pass/fail criteria met");
    } else {
      failedCriteria.forEach((result) => {
        findings.push(
          `${result.criterionName}: ${result.percentage.toFixed(1)}% pass rate (${result.failedSampleCount} locations failed)`
        );
      });
    }
  }

  // Add critical analysis findings
  const criticalFindings = analysis.filter((f) => f.severity === "critical");
  criticalFindings.forEach((finding) => {
    findings.push(finding.titleKey); // Will be translated during render
  });

  return findings.slice(0, 5); // Limit to top 5 findings
}

/**
 * Generate heatmap statistics for a metric
 */
function generateHeatmapData(samples: SamplePoint[], metric: HeatmapMetric): ReportHeatmap {
  if (!metric) {
    return {
      type: null,
      displayName: "Unknown",
      descriptionKey: "report.heatmapDescriptions.rssi",
      statistics: null,
      unit: "",
    };
  }

  const stats = calculateMetricStatistics(samples, metric);
  const threshold = DEFAULT_THRESHOLDS[metric];
  const percentMeeting = threshold
    ? getPercentageMeetingThreshold(samples, metric, threshold.value, threshold.comparison)
    : 0;

  return {
    type: metric,
    displayName: HEATMAP_NAMES[metric] || metric,
    descriptionKey: `report.heatmapDescriptions.${metric}`,
    statistics: stats
      ? {
          ...stats,
          percentMeetingThreshold: percentMeeting,
        }
      : null,
    threshold: threshold?.value,
    unit: HEATMAP_UNITS[metric] || "",
  };
}

/**
 * Build AP inventory from survey samples
 */
function buildAPInventory(samples: SamplePoint[]): APInventoryEntry[] {
  const apMap = new Map<
    string,
    {
      ssid: string;
      bssid: string;
      channels: number[];
      rssiValues: number[];
      security: string;
      vendor: string;
    }
  >();

  // Collect data from all samples
  for (const sample of samples) {
    const data = sample.sampleData;
    if ("networks" in data && Array.isArray(data.networks)) {
      const passiveData = data as PassiveSample;
      for (const network of passiveData.networks) {
        const key = network.bssid;
        if (!apMap.has(key)) {
          apMap.set(key, {
            ssid: network.ssid || "(Hidden)",
            bssid: network.bssid,
            channels: [],
            rssiValues: [],
            security: network.security || "unknown",
            vendor: network.vendor || "Unknown",
          });
        }
        const entry = apMap.get(key)!;
        if (network.channel && !entry.channels.includes(network.channel)) {
          entry.channels.push(network.channel);
        }
        entry.rssiValues.push(network.rssi);
      }
    }
  }

  // Convert to inventory entries
  const inventory: APInventoryEntry[] = [];
  for (const [, data] of apMap) {
    const avgRssi =
      data.rssiValues.length > 0
        ? data.rssiValues.reduce((a, b) => a + b, 0) / data.rssiValues.length
        : 0;

    // Determine band from channel
    const primaryChannel = data.channels[0] || 0;
    let band = "Unknown";
    if (primaryChannel >= 1 && primaryChannel <= 14) {
      band = "2.4 GHz";
    } else if (primaryChannel >= 36 && primaryChannel <= 177) {
      band = "5 GHz";
    } else if (primaryChannel >= 1 && primaryChannel <= 233) {
      band = "6 GHz";
    }

    inventory.push({
      ssid: data.ssid,
      bssid: data.bssid,
      channel: primaryChannel,
      band,
      security: data.security,
      vendor: data.vendor,
      avgRssi: Math.round(avgRssi * 10) / 10,
      sampleCount: data.rssiValues.length,
    });
  }

  // Sort by average RSSI (strongest first)
  inventory.sort((a, b) => b.avgRssi - a.avgRssi);

  return inventory;
}

/**
 * Generate recommendations based on validation and analysis
 */
function generateRecommendations(
  validation: SurveyValidation | null,
  analysis: AnalysisFinding[]
): string[] {
  const recommendations: string[] = [];

  // Add recommendations from analysis findings
  for (const finding of analysis) {
    if (finding.recommendationKey) {
      recommendations.push(finding.recommendationKey);
    }
  }

  // Add recommendations based on failed criteria
  if (validation) {
    for (const result of validation.results) {
      if (!result.passed && result.failedSampleCount > 0) {
        if (result.criterionName.includes("signal") || result.criterionName.includes("Signal")) {
          recommendations.push("analysis.coverage.weakAreasAction");
        } else if (
          result.criterionName.includes("channel") ||
          result.criterionName.includes("Channel")
        ) {
          recommendations.push("analysis.interference.coChannelAction");
        }
      }
    }
  }

  // Remove duplicates
  return [...new Set(recommendations)];
}

/**
 * Generate a complete survey report
 */
export function generateReport(
  survey: Survey,
  validation: SurveyValidation | null,
  analysis: AnalysisFinding[] = [],
  options: ReportOptions = {}
): SurveyReport {
  const {
    includeHeatmaps = DEFAULT_HEATMAPS[survey.surveyType] || DEFAULT_HEATMAPS.passive,
    includeAnalysis = true,
    includeAPInventory = true,
    companyName = "Mustard Seed Networks",
  } = options;

  // Build metadata
  const metadata: ReportMetadata = {
    title: `WiFi Survey Report: ${survey.name}`,
    surveyName: survey.name,
    surveyId: survey.id,
    generatedAt: new Date().toISOString(),
    generatedBy: companyName,
    surveyType: survey.surveyType,
    sampleCount: survey.samples.length,
    facilitySize: calculateFacilitySize(survey),
    scaleInfo: getScaleInfo(survey),
  };

  // Build summary
  const summary: ReportSummary = {
    overallStatus: validation?.overallPass ? "pass" : "fail",
    passedCriteria: validation?.passedCount || 0,
    totalCriteria: validation?.results.length || 0,
    overallPercentage: validation?.overallPercentage || 0,
    keyFindings: generateKeyFindings(validation, analysis),
  };

  // Generate heatmap data
  const heatmaps: ReportHeatmap[] = includeHeatmaps
    .filter((m): m is HeatmapMetric => m !== null)
    .map((metric) => generateHeatmapData(survey.samples, metric));

  // Build AP inventory
  const apInventory = includeAPInventory ? buildAPInventory(survey.samples) : [];

  // Generate recommendations
  const recommendations = generateRecommendations(validation, analysis);

  return {
    metadata,
    summary,
    validation,
    heatmaps,
    analysis: includeAnalysis ? analysis : [],
    apInventory,
    recommendations,
  };
}

/**
 * Export report data to CSV format
 */
export function exportReportToCSV(report: SurveyReport): string {
  const lines: string[] = [];

  // Header
  lines.push("WiFi Survey Report");
  lines.push(`Survey Name,${report.metadata.surveyName}`);
  lines.push(`Generated,${report.metadata.generatedAt}`);
  lines.push(`Survey Type,${report.metadata.surveyType}`);
  lines.push(`Sample Count,${report.metadata.sampleCount}`);
  lines.push(`Facility Size,${report.metadata.facilitySize}`);
  lines.push("");

  // Summary
  lines.push("Summary");
  lines.push(`Overall Status,${report.summary.overallStatus.toUpperCase()}`);
  lines.push(`Criteria Passed,${report.summary.passedCriteria}/${report.summary.totalCriteria}`);
  lines.push(`Overall Percentage,${report.summary.overallPercentage}%`);
  lines.push("");

  // Validation results
  if (report.validation) {
    lines.push("Pass/Fail Criteria Results");
    lines.push("Criterion,Status,Average,Threshold,Pass Rate,Failed Locations");
    for (const result of report.validation.results) {
      const status = result.passed ? "PASS" : "FAIL";
      const comparison = result.comparison === "gte" ? ">=" : "<=";
      lines.push(
        `${result.criterionName},${status},${result.averageValue.toFixed(1)} ${result.suffix},${comparison}${result.threshold} ${result.suffix},${result.percentage}%,${result.failedSampleCount}`
      );
    }
    lines.push("");
  }

  // Heatmap statistics
  lines.push("Heatmap Statistics");
  lines.push("Metric,Min,Max,Average,Median,Samples,% Meeting Threshold");
  for (const heatmap of report.heatmaps) {
    if (heatmap.statistics) {
      const { min, max, average, median, sampleCount, percentMeetingThreshold } =
        heatmap.statistics;
      lines.push(
        `${heatmap.displayName},${min.toFixed(1)},${max.toFixed(1)},${average.toFixed(1)},${median.toFixed(1)},${sampleCount},${percentMeetingThreshold}%`
      );
    }
  }
  lines.push("");

  // AP Inventory
  if (report.apInventory.length > 0) {
    lines.push("AP Inventory");
    lines.push("SSID,BSSID,Channel,Band,Security,Vendor,Avg RSSI,Sample Count");
    for (const ap of report.apInventory) {
      lines.push(
        `"${ap.ssid}",${ap.bssid},${ap.channel},${ap.band},${ap.security},${ap.vendor},${ap.avgRssi},${ap.sampleCount}`
      );
    }
  }

  return lines.join("\n");
}
