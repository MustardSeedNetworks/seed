/**
 * Pure helpers used by SurveyView.
 *
 * - renderSampleData formats a single sample for the sidebar list.
 * - calculateMetricRange walks all samples to find the min/max for the
 *   currently selected heatmap metric.
 */

import type React from 'react';
import type {
  ActiveSample,
  HeatmapMetric,
  PassiveSample,
  SamplePoint,
  ThroughputSample,
} from '../../hooks/useSurvey';
import { status as statusColor } from '../../styles/theme';

// Helper to render sample data
export function renderSampleData(
  data: PassiveSample | ActiveSample | ThroughputSample,
  surveyType: string,
): React.JSX.Element | null {
  if (surveyType === 'passive') {
    const passiveData = data as PassiveSample;
    return (
      <>
        <div>Networks: {passiveData.networks?.length || 0}</div>
        {passiveData.networks?.[0] ? (
          <>
            <div>Strongest: {passiveData.networks[0].ssid}</div>
            <div>RSSI: {passiveData.networks[0].rssi} dBm</div>
          </>
        ) : null}
      </>
    );
  }

  if (surveyType === 'active') {
    const activeData = data as ActiveSample;
    return (
      <>
        <div>SSID: {activeData.ssid}</div>
        <div>RSSI: {activeData.rssi} dBm</div>
        <div>Rate: {activeData.dataRate} Mbps</div>
        {activeData.roamingEvent ? (
          <div className="text-status-warning font-semibold">⚠ Roaming</div>
        ) : null}
      </>
    );
  }

  if (surveyType === 'throughput') {
    const throughputData = data as ThroughputSample;
    return (
      <>
        <div>RSSI: {throughputData.rssi} dBm</div>
        <div>↓ {throughputData.downloadMbps.toFixed(1)} Mbps</div>
        <div>↑ {throughputData.uploadMbps.toFixed(1)} Mbps</div>
        <div>Jitter: {throughputData.jitter.toFixed(1)} ms</div>
        {throughputData.packetLoss > 0 && (
          <div className={statusColor.text.error}>
            Loss: {throughputData.packetLoss.toFixed(1)}%
          </div>
        )}
      </>
    );
  }

  return null;
}

// Helper to calculate min/max values for a heatmap metric
// biome-ignore lint/complexity/noExcessiveCognitiveComplexity: Switch covers every metric we render
export function calculateMetricRange(
  samples: SamplePoint[],
  metric: HeatmapMetric,
): { min: number; max: number } {
  if (!metric || samples.length === 0) {
    return { min: 0, max: 0 };
  }

  const values: number[] = [];

  for (const sample of samples) {
    const data = sample.sampleData as {
      networks?: { rssi: number; channel: number; bssid?: string; ssid?: string }[];
      rssi?: number;
      noiseFloor?: number;
      downloadMbps?: number;
      latency?: number;
      channelUtilization?: number;
      uniqueBssids?: number;
      uniqueSsids?: number;
    };

    switch (metric) {
      case 'rssi':
        if (data.networks && Array.isArray(data.networks)) {
          const rssiValues = data.networks.map((n) => n.rssi);
          if (rssiValues.length > 0) {
            values.push(Math.max(...rssiValues));
          }
        } else if (data.rssi !== undefined) {
          values.push(data.rssi);
        }
        break;
      case 'snr':
        if (data.networks && Array.isArray(data.networks)) {
          const rssiValues = data.networks.map((n) => n.rssi);
          if (rssiValues.length > 0) {
            const rssi = Math.max(...rssiValues);
            const noise = data.noiseFloor || -95;
            values.push(rssi - noise);
          }
        } else if (data.rssi !== undefined) {
          const noise = data.noiseFloor || -95;
          values.push(data.rssi - noise);
        }
        break;
      case 'noise':
        values.push(data.noiseFloor || -95);
        break;
      case 'cochannel':
        if (data.networks && Array.isArray(data.networks) && data.networks.length > 0) {
          const primaryChannel = data.networks[0].channel;
          const count = data.networks.filter((n) => n.channel === primaryChannel).length - 1;
          values.push(count);
        }
        break;
      case 'adjacent':
        if (data.networks && Array.isArray(data.networks) && data.networks.length > 0) {
          const primaryChannel = data.networks[0].channel;
          const count = data.networks.filter(
            (n) =>
              Math.abs(n.channel - primaryChannel) > 0 && Math.abs(n.channel - primaryChannel) <= 2,
          ).length;
          values.push(count);
        }
        break;
      case 'throughput':
        if (data.downloadMbps !== undefined) {
          values.push(data.downloadMbps);
        }
        break;
      case 'latency':
        if (data.latency !== undefined) {
          values.push(data.latency);
        }
        break;
      case 'channelUtil':
        if (data.channelUtilization !== undefined) {
          values.push(data.channelUtilization);
        }
        break;
      case 'apDensity':
        if (data.networks && Array.isArray(data.networks)) {
          const uniqueBssids = new Set(
            data.networks
              .map((n) => n.bssid)
              .filter((bssid): bssid is string => typeof bssid === 'string'),
          );
          values.push(uniqueBssids.size);
        } else if (data.uniqueBssids !== undefined) {
          values.push(data.uniqueBssids);
        }
        break;
      case 'ssidCount':
        if (data.networks && Array.isArray(data.networks)) {
          const uniqueSsids = new Set(
            data.networks.map((n) => n.ssid).filter((ssid): ssid is string => Boolean(ssid)),
          );
          values.push(uniqueSsids.size);
        } else if (data.uniqueSsids !== undefined) {
          values.push(data.uniqueSsids);
        }
        break;
      default:
        // Unknown metric, skip
        break;
    }
  }

  if (values.length === 0) {
    return { min: 0, max: 0 };
  }

  return {
    min: Math.min(...values),
    max: Math.max(...values),
  };
}
