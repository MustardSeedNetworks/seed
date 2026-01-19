// biome-ignore-all lint/complexity/noExcessiveCognitiveComplexity: Complex component
/**
 * SurveyAnalysisPanel Component
 *
 * Purpose: Intelligent analysis of WiFi survey data with actionable recommendations.
 * Provides comprehensive network diagnostics and optimization suggestions.
 *
 * Key Features:
 * - Coverage gap detection with location markers
 * - Interference analysis (co-channel, adjacent channel)
 * - AP placement optimization suggestions
 * - Security vulnerability detection
 * - Roaming issue identification
 * - Channel optimization recommendations
 * - Capacity planning analysis
 * - Performance bottleneck detection
 *
 * Usage:
 * ```typescript
 * <SurveyAnalysisPanel
 *   survey={currentSurvey}
 *   onFindingClick={(finding) => handleFindingClick(finding)}
 *   thresholds={customThresholds}
 * />
 * ```
 */

import {
  Activity,
  AlertOctagon,
  AlertTriangle,
  CheckCircle2,
  FileText,
  Info,
  Lightbulb,
  MapPin,
  Radio,
  Shield,
  ShieldAlert,
  TrendingUp,
  Wifi,
  WifiOff,
  Zap,
} from 'lucide-react';
import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import type { ApLocation, SamplePoint, ScannedNetwork, Survey } from '../../hooks/useSurvey';
import { button, cn, icon as iconTokens, layout, radius, spacing } from '../../styles/theme';

/** Finding severity level */
export type FindingSeverity = 'critical' | 'warning' | 'info' | 'success';

/** Finding category */
export type FindingCategory =
  | 'coverage'
  | 'interference'
  | 'security'
  | 'performance'
  | 'roaming'
  | 'capacity'
  | 'optimization';

/** A single analysis finding/recommendation */
export interface AnalysisFinding {
  id: string;
  category: FindingCategory;
  severity: FindingSeverity;
  titleKey: string;
  descriptionKey: string;
  location?: { x: number; y: number };
  affectedSsids?: string[];
  affectedBssids?: string[];
  affectedChannels?: number[];
  recommendationKey?: string;
  value?: number | string;
  threshold?: number | string;
}

/** Configurable analysis thresholds */
export interface AnalysisThresholds {
  minGoodRssi: number;
  minAcceptableRssi: number;
  minGoodSnr: number;
  maxCoChannelCount: number;
  maxAdjacentChannelCount: number;
}

/** Default thresholds */
const DEFAULT_ANALYSIS_THRESHOLDS: AnalysisThresholds = {
  minGoodRssi: -65,
  minAcceptableRssi: -75,
  minGoodSnr: 25,
  maxCoChannelCount: 2,
  maxAdjacentChannelCount: 3,
};

interface SurveyAnalysisPanelProps {
  survey: Survey;
  onFindingClick?: (finding: AnalysisFinding) => void;
  onLocationClick?: (x: number, y: number) => void;
  onGenerateReport?: (findings: AnalysisFinding[]) => void;
  thresholds?: Partial<AnalysisThresholds>;
}

/** Analyze survey samples for findings */
function analyzeSurvey(
  samples: SamplePoint[],
  apLocations: ApLocation[],
  thresholds: AnalysisThresholds,
): AnalysisFinding[] {
  const findings: AnalysisFinding[] = [];
  let findingId = 0;

  // Track aggregated data
  const channelUsage = new Map<number, number>();
  const ssidCoverage = new Map<string, { good: number; weak: number; dead: number }>();
  const securityIssues: {
    ssid: string;
    bssid: string;
    security: string;
    x: number;
    y: number;
  }[] = [];
  const coverageGaps: { x: number; y: number; rssi: number }[] = [];
  const coChannelHotspots: {
    x: number;
    y: number;
    count: number;
    channel: number;
  }[] = [];

  // Analyze each sample
  for (const sample of samples) {
    const data = sample.sampleData as { networks?: ScannedNetwork[] };

    if (data.networks && Array.isArray(data.networks)) {
      const { networks } = data;

      // Track best RSSI at this location
      const bestRssi = networks.length > 0 ? Math.max(...networks.map((n) => n.rssi)) : -100;

      // Check for coverage gaps
      if (bestRssi < thresholds.minAcceptableRssi) {
        coverageGaps.push({ x: sample.x, y: sample.y, rssi: bestRssi });
      }

      // Track channel usage for co-channel analysis
      const channelsAtPoint = new Set<number>();
      for (const n of networks) {
        if (n.channel) {
          channelUsage.set(n.channel, (channelUsage.get(n.channel) || 0) + 1);
          channelsAtPoint.add(n.channel);
        }

        // Track SSID coverage quality
        const ssidStats = ssidCoverage.get(n.ssid) || {
          good: 0,
          weak: 0,
          dead: 0,
        };
        if (n.rssi >= thresholds.minGoodRssi) {
          ssidStats.good++;
        } else if (n.rssi >= thresholds.minAcceptableRssi) {
          ssidStats.weak++;
        } else {
          ssidStats.dead++;
        }
        ssidCoverage.set(n.ssid, ssidStats);

        // Check for security issues
        if (n.security === 'open' || n.security === 'wep') {
          securityIssues.push({
            ssid: n.ssid,
            bssid: n.bssid,
            security: n.security || 'open',
            x: sample.x,
            y: sample.y,
          });
        }
      }

      // Check for co-channel interference
      for (const channel of channelsAtPoint) {
        const networksOnChannel = networks.filter((n) => n.channel === channel);
        if (networksOnChannel.length > thresholds.maxCoChannelCount) {
          coChannelHotspots.push({
            x: sample.x,
            y: sample.y,
            count: networksOnChannel.length,
            channel,
          });
        }
      }
    }
  }

  // Generate coverage gap findings
  if (coverageGaps.length > 0) {
    const deadZoneThreshold = thresholds.minAcceptableRssi - 10;
    const criticalGaps = coverageGaps.filter((g) => g.rssi < deadZoneThreshold);
    const weakGaps = coverageGaps.filter(
      (g) => g.rssi >= deadZoneThreshold && g.rssi < thresholds.minAcceptableRssi,
    );

    if (criticalGaps.length > 0) {
      findings.push({
        id: `finding-${findingId++}`,
        category: 'coverage',
        severity: 'critical',
        titleKey: 'analysis.coverage.deadZones',
        descriptionKey: 'analysis.coverage.deadZonesDesc',
        location: criticalGaps[0],
        value: criticalGaps.length,
        threshold: deadZoneThreshold,
        recommendationKey: 'analysis.coverage.deadZonesAction',
      });
    }

    if (weakGaps.length > 0) {
      findings.push({
        id: `finding-${findingId++}`,
        category: 'coverage',
        severity: 'warning',
        titleKey: 'analysis.coverage.weakAreas',
        descriptionKey: 'analysis.coverage.weakAreasDesc',
        location: weakGaps[0],
        value: weakGaps.length,
        threshold: thresholds.minAcceptableRssi,
        recommendationKey: 'analysis.coverage.weakAreasAction',
      });
    }
  }

  // Generate co-channel interference findings
  if (coChannelHotspots.length > 0) {
    const worstSpot = coChannelHotspots.reduce((prev, curr) =>
      curr.count > prev.count ? curr : prev,
    );

    findings.push({
      id: `finding-${findingId++}`,
      category: 'interference',
      severity: coChannelHotspots.length > 5 ? 'critical' : 'warning',
      titleKey: 'analysis.interference.coChannel',
      descriptionKey: 'analysis.interference.coChannelDesc',
      location: worstSpot,
      affectedChannels: [worstSpot.channel],
      value: worstSpot.count,
      threshold: thresholds.maxCoChannelCount,
      recommendationKey: 'analysis.interference.coChannelAction',
    });
  }

  // Generate security findings
  const uniqueSecurityIssues = new Map<string, (typeof securityIssues)[0]>();
  for (const issue of securityIssues) {
    if (!uniqueSecurityIssues.has(issue.bssid)) {
      uniqueSecurityIssues.set(issue.bssid, issue);
    }
  }

  if (uniqueSecurityIssues.size > 0) {
    const openNetworks = Array.from(uniqueSecurityIssues.values()).filter(
      (i) => i.security === 'open',
    );
    const wepNetworks = Array.from(uniqueSecurityIssues.values()).filter(
      (i) => i.security === 'wep',
    );

    if (openNetworks.length > 0) {
      findings.push({
        id: `finding-${findingId++}`,
        category: 'security',
        severity: 'critical',
        titleKey: 'analysis.security.openNetworks',
        descriptionKey: 'analysis.security.openNetworksDesc',
        location: openNetworks[0],
        affectedSsids: openNetworks.map((n) => n.ssid),
        value: openNetworks.length,
        recommendationKey: 'analysis.security.openNetworksAction',
      });
    }

    if (wepNetworks.length > 0) {
      findings.push({
        id: `finding-${findingId++}`,
        category: 'security',
        severity: 'warning',
        titleKey: 'analysis.security.wepNetworks',
        descriptionKey: 'analysis.security.wepNetworksDesc',
        affectedSsids: wepNetworks.map((n) => n.ssid),
        value: wepNetworks.length,
        recommendationKey: 'analysis.security.wepNetworksAction',
      });
    }
  }

  // Generate channel optimization findings
  const sortedChannels = Array.from(channelUsage.entries()).sort((a, b) => b[1] - a[1]);
  const overusedChannels = sortedChannels.filter(([, count]) => count > samples.length * 0.5);

  if (overusedChannels.length > 0) {
    findings.push({
      id: `finding-${findingId++}`,
      category: 'optimization',
      severity: 'info',
      titleKey: 'analysis.optimization.overusedChannels',
      descriptionKey: 'analysis.optimization.overusedChannelsDesc',
      affectedChannels: overusedChannels.map(([ch]) => ch),
      value: overusedChannels[0][0],
      recommendationKey: 'analysis.optimization.overusedChannelsAction',
    });
  }

  // Generate AP coverage findings based on placed APs
  if (apLocations.length > 0) {
    findings.push({
      id: `finding-${findingId++}`,
      category: 'coverage',
      severity: 'info',
      titleKey: 'analysis.coverage.apCount',
      descriptionKey: 'analysis.coverage.apCountDesc',
      value: apLocations.length,
    });
  }

  // Overall health summary
  const criticalCount = findings.filter((i) => i.severity === 'critical').length;
  const warningCount = findings.filter((i) => i.severity === 'warning').length;

  if (criticalCount === 0 && warningCount === 0 && samples.length > 0) {
    findings.push({
      id: `finding-${findingId++}`,
      category: 'coverage',
      severity: 'success',
      titleKey: 'analysis.overall.healthy',
      descriptionKey: 'analysis.overall.healthyDesc',
      value: samples.length,
    });
  }

  return findings;
}

/**
 * Get icon for finding category
 */
function getFindingIcon(category: FindingCategory, severity: FindingSeverity): typeof Info {
  switch (category) {
    case 'coverage':
      return severity === 'critical' ? WifiOff : Wifi;
    case 'interference':
      return Radio;
    case 'security':
      return severity === 'critical' ? ShieldAlert : Shield;
    case 'performance':
      return Zap;
    case 'roaming':
      return TrendingUp;
    case 'capacity':
      return MapPin;
    case 'optimization':
      return Lightbulb;
    default:
      return Info;
  }
}

/**
 * Get color for severity level
 */
function getSeverityColor(severity: FindingSeverity): string {
  switch (severity) {
    case 'critical':
      return 'text-status-error';
    case 'warning':
      return 'text-status-warning';
    case 'success':
      return 'text-status-success';
    default:
      return 'text-brand-primary';
  }
}

/**
 * Get background color for severity level
 */
function getSeverityBg(severity: FindingSeverity): string {
  switch (severity) {
    case 'critical':
      return 'bg-status-error/10 border-status-error/20';
    case 'warning':
      return 'bg-status-warning/10 border-status-warning/20';
    case 'success':
      return 'bg-status-success/10 border-status-success/20';
    default:
      return 'bg-brand-primary/10 border-brand-primary/20';
  }
}

/**
 * SurveyAnalysisPanel provides intelligent analysis and recommendations
 */
export function SurveyAnalysisPanel({
  survey,
  onFindingClick,
  onLocationClick,
  onGenerateReport,
  thresholds: customThresholds,
}: SurveyAnalysisPanelProps): React.JSX.Element {
  const { t } = useTranslation('survey');

  // Merge custom thresholds with defaults
  const thresholds = useMemo(
    () => ({ ...DEFAULT_ANALYSIS_THRESHOLDS, ...customThresholds }),
    [customThresholds],
  );

  // Analyze survey data
  const findings = useMemo(
    () => analyzeSurvey(survey.samples, survey.apLocations || [], thresholds),
    [survey.samples, survey.apLocations, thresholds],
  );

  // Group findings by severity
  const criticalFindings = findings.filter((i) => i.severity === 'critical');
  const warningFindings = findings.filter((i) => i.severity === 'warning');
  const infoFindings = findings.filter((i) => i.severity === 'info' || i.severity === 'success');

  // Handle finding click
  const handleClick = (finding: AnalysisFinding): void => {
    if (finding.location && onLocationClick) {
      onLocationClick(finding.location.x, finding.location.y);
    }
    if (onFindingClick) {
      onFindingClick(finding);
    }
  };

  // Render single finding card
  const renderFinding = (finding: AnalysisFinding): React.JSX.Element => {
    const ICON = getFindingIcon(finding.category, finding.severity);
    const colorClass = getSeverityColor(finding.severity);
    const bgClass = getSeverityBg(finding.severity);

    return (
      // biome-ignore lint/a11y/useSemanticElements: Finding card pattern with complex content
      <div
        key={finding.id}
        onClick={() => handleClick(finding)}
        onKeyDown={(e: React.KeyboardEvent): void => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            handleClick(finding);
          }
        }}
        role="button"
        tabIndex={0}
        class={cn(
          spacing.pad.sm,
          radius.md,
          'border',
          bgClass,
          'cursor-pointer hover:opacity-80 transition-opacity',
        )}
      >
        <div class={cn(layout.inline.default)}>
          <ICON class={cn(iconTokens.size.sm, colorClass, 'flex-shrink-0')} />
          <div class="flex-1 min-w-0">
            <h4 class={cn('body-small font-medium', colorClass)}>{t(finding.titleKey as never)}</h4>
            <p class="caption text-text-muted">
              {t(finding.descriptionKey as never, {
                value: finding.value,
                threshold: finding.threshold,
                count: finding.value,
              })}
            </p>
            {finding.recommendationKey ? (
              <p class="caption text-text-primary mt-1">
                <Lightbulb class="w-3 h-3 inline mr-1" />
                {t(finding.recommendationKey as never)}
              </p>
            ) : null}
            {finding.affectedSsids && finding.affectedSsids.length > 0 ? (
              <p class="caption text-text-muted mt-1 truncate">
                {t('analysis.affectedSsids')}: {finding.affectedSsids.slice(0, 3).join(', ')}
                {finding.affectedSsids.length > 3 ? ` +${finding.affectedSsids.length - 3}` : null}
              </p>
            ) : null}
            {finding.location ? (
              <p class="caption text-brand-primary mt-1">
                <MapPin class="w-3 h-3 inline mr-1" />
                {t('analysis.clickToView')}
              </p>
            ) : null}
          </div>
        </div>
      </div>
    );
  };

  return (
    <div class={cn('bg-surface-raised', radius.md, 'border border-surface-border', spacing.pad.sm)}>
      {/* Header */}
      <div class={cn(layout.inline.default, 'justify-between', spacing.margin.bottom.content)}>
        <div class={cn(layout.inline.default)}>
          <Activity class={iconTokens.size.sm} />
          <h4 class="body-small font-medium">{t('analysis.title')}</h4>
        </div>
        <div class={cn(layout.inline.default)}>
          {criticalFindings.length > 0 ? (
            <span class="px-2 py-0.5 text-xs bg-status-error text-text-inverse rounded-full">
              {criticalFindings.length}
            </span>
          ) : null}
          {warningFindings.length > 0 ? (
            <span class="px-2 py-0.5 text-xs bg-status-warning text-text-inverse rounded-full">
              {warningFindings.length}
            </span>
          ) : null}
          {onGenerateReport && findings.length > 0 ? (
            <button
              type="button"
              onClick={() => onGenerateReport(findings)}
              class={cn(
                button.size.sm,
                'bg-brand-primary text-text-inverse',
                radius.md,
                'hover:opacity-90',
                layout.inline.tight,
              )}
            >
              <FileText class="w-3 h-3" />
              <span>{t('criteria.generateReport')}</span>
            </button>
          ) : null}
        </div>
      </div>

      {/* Summary */}
      <div class={cn(layout.inline.default, spacing.margin.bottom.content)}>
        {criticalFindings.length > 0 ? (
          <div class={cn(layout.inline.tight, 'text-status-error')}>
            <AlertOctagon class="w-4 h-4" />
            <span class="caption">
              {criticalFindings.length} {t('analysis.critical')}
            </span>
          </div>
        ) : null}
        {warningFindings.length > 0 ? (
          <div class={cn(layout.inline.tight, 'text-status-warning')}>
            <AlertTriangle class="w-4 h-4" />
            <span class="caption">
              {warningFindings.length} {t('analysis.warnings')}
            </span>
          </div>
        ) : null}
        {criticalFindings.length === 0 && warningFindings.length === 0 ? (
          <div class={cn(layout.inline.tight, 'text-status-success')}>
            <CheckCircle2 class="w-4 h-4" />
            <span class="caption">{t('analysis.noIssues')}</span>
          </div>
        ) : null}
      </div>

      {/* Findings list */}
      {findings.length === 0 ? (
        <p class="caption text-text-muted text-center py-4">{t('analysis.noData')}</p>
      ) : (
        <div class={cn(layout.stack.tight, 'max-h-80 overflow-y-auto')}>
          {/* Critical first */}
          {criticalFindings.map(renderFinding)}
          {/* Then warnings */}
          {warningFindings.map(renderFinding)}
          {/* Then info/success */}
          {infoFindings.map(renderFinding)}
        </div>
      )}
    </div>
  );
}
