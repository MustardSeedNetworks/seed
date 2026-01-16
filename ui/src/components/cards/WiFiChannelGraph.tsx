/**
 * WiFi Channel Overlap Graph Component
 *
 * Displays a visual representation of WiFi channel usage and overlap.
 *
 * Features:
 * - X-axis: Channel numbers (1-14 for 2.4GHz, 36-165 for 5GHz, 1-233 for 6GHz)
 * - Y-axis: Signal strength (-100 to -30 dBm)
 * - Each network rendered as a curve centered on its channel
 * - Curve width represents channel width (20, 40, 80, 160, 320 MHz)
 * - Curve height represents signal strength
 * - Connected network highlighted in primary brand color
 * - Band selection tabs to switch between 2.4GHz, 5GHz, and 6GHz
 * - Hover tooltip showing network details
 *
 * The graph helps visualize channel congestion and optimal channel selection.
 */

import type React from "react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { cn, icon as iconTokens, layout, spacing } from "../../styles/theme";
import { CardValue } from "../ui/Card";
import { Wifi } from "../ui/Icons";
import { SimpleBaseCard } from "./BaseCard";

/**
 * Network data for channel graph visualization
 */
interface ChannelNetwork {
  ssid: string;
  bssid: string;
  channel: number;
  centerFreq: number;
  channelWidth: number; // MHz (20, 40, 80, 160, 320)
  signal: number; // dBm
  band: string; // "2.4GHz", "5GHz", "6GHz"
  isConnected: boolean;
}

/**
 * Channel graph data organized by band (normalized to camelCase)
 */
interface ChannelGraphData {
  networks24Ghz: ChannelNetwork[];
  networks5Ghz: ChannelNetwork[];
  networks6Ghz: ChannelNetwork[];
  connectedBssid?: string;
  scanTime: string;
}

/**
 * API response structure
 */
interface ChannelGraphResponse {
  available: boolean;
  error?: string;
  data?: ChannelGraphData;
}

/**
 * Props for WiFi Channel Graph Card
 */
interface WifiChannelGraphProps {
  data: ChannelGraphResponse | null;
  loading?: boolean;
  visible?: boolean;
}

/**
 * Band selection type
 */
type BandType = "2.4GHz" | "5GHz" | "6GHz";

/**
 * Get channel range for a given band
 */
function getChannelRange(band: BandType): {
  min: number;
  max: number;
  step: number;
} {
  switch (band) {
    case "2.4GHz":
      return { min: 1, max: 14, step: 1 };
    case "5GHz":
      return { min: 36, max: 165, step: 4 }; // 5GHz channels: 36, 40, 44, 48, ...
    case "6GHz":
      return { min: 1, max: 233, step: 4 }; // 6GHz channels
    default:
      return { min: 1, max: 14, step: 1 }; // Default to 2.4GHz
  }
}

/**
 * Convert signal strength to Y coordinate (inverted, stronger signal = higher on graph)
 */
function signalToY(signal: number, height: number): number {
  // Signal range: -100 (weakest) to -30 (strongest)
  // Map to 0 (bottom) to height (top)
  const normalized = (signal + 100) / 70; // Normalize to 0-1 (stronger = higher value)
  return height * (1 - normalized); // Invert Y axis (SVG 0 is top)
}

/**
 * Generate SVG path for a network's channel coverage
 * Creates a bell curve centered on the channel, with width based on channel width
 */
function generateNetworkPath(
  network: ChannelNetwork,
  channelRange: { min: number; max: number },
  width: number,
  height: number,
): string {
  const { channel, channelWidth, signal } = network;
  const { min, max } = channelRange;
  const channelSpan = max - min;

  // Calculate X position for the center of the channel
  const centerX = ((channel - min) / channelSpan) * width;

  // Calculate width in channels (5 MHz per channel for 2.4GHz, 20 MHz spacing for 5/6GHz)
  const channelWidthInChannels = channelWidth / 5; // Approximate

  // Calculate half-width in pixels
  const halfWidth = (channelWidthInChannels / channelSpan) * width * 0.5;

  // Calculate peak height based on signal strength
  const peakY = signalToY(signal, height);
  const baseY = height;

  // Generate Gaussian-like curve using quadratic bezier
  const leftX = Math.max(0, centerX - halfWidth);
  const rightX = Math.min(width, centerX + halfWidth);

  // Create path: Start at base left, curve to peak, curve to base right
  return `M ${leftX},${baseY} Q ${leftX},${peakY} ${centerX},${peakY} Q ${rightX},${peakY} ${rightX},${baseY} Z`;
}

/** Helper to compute opacity for network path */
function getNetworkOpacity(isHovered: boolean, isConnected: boolean): number {
  if (isHovered) {
    return 0.9;
  }
  if (isConnected) {
    return 0.7;
  }
  return 0.4;
}

/** Helper to compute stroke width for network path */
function getNetworkStrokeWidth(isHovered: boolean, isConnected: boolean): number {
  if (isHovered) {
    return 2;
  }
  if (isConnected) {
    return 2;
  }
  return 1;
}

/** Helper to determine card status */
function getCardStatus(
  loading: boolean | undefined,
  available: boolean | undefined,
): "loading" | "success" | "error" {
  if (loading) {
    return "loading";
  }
  if (available) {
    return "success";
  }
  return "error";
}

/**
 * Render the channel graph for a specific band
 */
function _channelGraph({
  networks,
  band,
  connectedBssid,
}: {
  networks: ChannelNetwork[];
  band: BandType;
  connectedBssid?: string;
}): React.ReactElement {
  const { t: tCards } = useTranslation("cards");
  const { t: tCommon } = useTranslation("common");
  const [hoveredNetwork, setHoveredNetwork] = useState<ChannelNetwork | null>(null);

  const channelRange = getChannelRange(band);
  const width = 600;
  const height = 300;
  const padding = { top: 20, right: 20, bottom: 40, left: 50 };
  const graphWidth = width - padding.left - padding.right;
  const graphHeight = height - padding.top - padding.bottom;

  // Generate channel markers
  const channelMarkers = useMemo(() => {
    const markers: { channel: number; x: number }[] = [];
    const { min, max, step } = channelRange;
    for (let ch = min; ch <= max; ch += step) {
      const x = padding.left + ((ch - min) / (max - min)) * graphWidth;
      markers.push({ channel: ch, x });
    }
    return markers;
  }, [channelRange, graphWidth]);

  // Signal markers (Y-axis)
  const signalMarkers = [-90, -70, -50, -30];

  if (networks.length === 0) {
    return (
      <div class={layout.flex.center} style={{ height: `${height}px` }}>
        <p class="body-small text-text-muted">
          {tCards("wifi.channelGraph.noNetworksDetected", { band })}
        </p>
      </div>
    );
  }

  return (
    <div class="relative">
      <svg
        width={width}
        height={height}
        class="w-full"
        viewBox={`0 0 ${width} ${height}`}
        role="img"
        aria-label="WiFi channel signal graph"
      >
        {/* Background grid */}
        <g class="opacity-10">
          {/* Horizontal lines (signal strength) */}
          {signalMarkers.map((signal) => {
            const y = padding.top + signalToY(signal, graphHeight);
            return (
              <line
                key={signal}
                x1={padding.left}
                y1={y}
                x2={width - padding.right}
                y2={y}
                stroke="currentColor"
                strokeWidth="1"
              />
            );
          })}
          {/* Vertical lines (channels) */}
          {channelMarkers.map(({ channel, x }) => (
            <line
              key={channel}
              x1={x}
              y1={padding.top}
              x2={x}
              y2={height - padding.bottom}
              stroke="currentColor"
              strokeWidth="1"
            />
          ))}
        </g>

        {/* Y-axis labels (signal strength) */}
        <g class="text-text-muted" style={{ fontSize: "10px" }}>
          {signalMarkers.map((signal) => {
            const y = padding.top + signalToY(signal, graphHeight);
            return (
              <text key={signal} x={padding.left - 10} y={y + 3} textAnchor="end">
                {signal}
              </text>
            );
          })}
        </g>

        {/* X-axis labels (channels) */}
        <g class="text-text-muted" style={{ fontSize: "10px" }}>
          {channelMarkers.map(({ channel, x }) => (
            <text key={channel} x={x} y={height - padding.bottom + 15} textAnchor="middle">
              {channel}
            </text>
          ))}
        </g>

        {/* Axis labels */}
        <text
          x={padding.left / 2}
          y={height / 2}
          textAnchor="middle"
          transform={`rotate(-90, ${padding.left / 2}, ${height / 2})`}
          class="body-small text-text-muted"
        >
          Signal (dBm)
        </text>
        <text x={width / 2} y={height - 5} textAnchor="middle" class="body-small text-text-muted">
          Channel
        </text>

        {/* Network curves */}
        <g transform={`translate(${padding.left}, ${padding.top})`}>
          {networks.map((network) => {
            const path = generateNetworkPath(network, channelRange, graphWidth, graphHeight);
            const isConnected = network.bssid === connectedBssid;
            const isHovered = hoveredNetwork?.bssid === network.bssid;

            return (
              // biome-ignore lint/a11y/noStaticElementInteractions: SVG path element with hover events for data visualization
              <path
                key={network.bssid}
                d={path}
                fill={isConnected ? "var(--color-brand-primary)" : "var(--color-status-info)"}
                opacity={getNetworkOpacity(isHovered, isConnected)}
                stroke={isConnected ? "var(--color-brand-primary)" : "var(--color-status-info)"}
                strokeWidth={getNetworkStrokeWidth(isHovered, isConnected)}
                class="transition-all cursor-pointer"
                onMouseEnter={(): void => setHoveredNetwork(network)}
                onMouseLeave={(): void => setHoveredNetwork(null)}
                aria-label={`Network ${network.ssid || network.bssid} on channel ${network.channel}`}
              />
            );
          })}
        </g>
      </svg>

      {/* Hover tooltip */}
      {hoveredNetwork ? (
        <div
          class="absolute bg-surface-raised border border-surface-border rounded shadow-lg p-2 z-10"
          style={{ top: "10px", right: "10px" }}
        >
          <p class="body-small font-semibold">{hoveredNetwork.ssid || "(Hidden)"}</p>
          <p class="caption text-text-muted">
            {tCards("wifi.channelGraph.tooltipChannel", {
              channel: hoveredNetwork.channel,
            })}
          </p>
          <p class="caption text-text-muted">{hoveredNetwork.signal} dBm</p>
          <p class="caption text-text-muted">{hoveredNetwork.channelWidth} MHz</p>
          {hoveredNetwork.isConnected ? (
            <p class="caption text-brand-primary font-medium">{tCommon("status.connected")}</p>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}

/**
 * WiFi Channel Graph Card
 * Displays channel overlap visualization for WiFi networks
 */
export function WifiChannelGraph({
  data,
  loading,
  visible = true,
}: WifiChannelGraphProps): React.ReactElement | null {
  const { t: tr } = useTranslation("cards");
  const { t: tc } = useTranslation("common");
  const [selectedBand, setSelectedBand] = useState<BandType>("2.4GHz");

  // Get networks for selected band
  // Note: All hooks must be called before any early returns to follow React rules
  const networks = useMemo(() => {
    if (!data?.data) {
      return [];
    }
    switch (selectedBand) {
      case "2.4GHz":
        return data.data.networks24Ghz;
      case "5GHz":
        return data.data.networks5Ghz;
      case "6GHz":
        return data.data.networks6Ghz;
      default:
        return data.data.networks24Ghz;
    }
  }, [data, selectedBand]);

  // Determine which bands have networks
  const availableBands = useMemo(() => {
    if (!data?.data) {
      return [];
    }
    const bands: BandType[] = [];
    if (data.data.networks24Ghz.length > 0) {
      bands.push("2.4GHz");
    }
    if (data.data.networks5Ghz.length > 0) {
      bands.push("5GHz");
    }
    if (data.data.networks6Ghz.length > 0) {
      bands.push("6GHz");
    }
    return bands;
  }, [data]);

  // Auto-select first available band
  if (availableBands.length > 0 && !availableBands.includes(selectedBand)) {
    setSelectedBand(availableBands[0]);
  }

  // Don't render if not visible (e.g., no WiFi adapter)
  if (!visible) {
    return null;
  }

  return (
    <SimpleBaseCard
      title={tr("wifi.channelGraph.title")}
      icon={<Wifi class={iconTokens.size.md} />}
      status={getCardStatus(loading, data?.available)}
      loading={loading}
      loadingContent={<CardValue value={tc("status.scanning")} size="lg" />}
    >
      {data?.available ? (
        <>
          {/* Band selection tabs */}
          {availableBands.length > 1 ? (
            <div class={cn(layout.inline.default, spacing.margin.bottom.inline)}>
              {availableBands.map((band) => (
                <button
                  type="button"
                  key={band}
                  onClick={(): void => setSelectedBand(band)}
                  class={cn(
                    spacing.chip.md,
                    "rounded",
                    "transition-colors",
                    selectedBand === band
                      ? "bg-brand-primary text-text-inverse"
                      : "bg-surface-hover text-text-primary hover:bg-surface-border",
                  )}
                >
                  {band}
                </button>
              ))}
            </div>
          ) : null}

          {/* Channel graph */}
          <channelGraph
            networks={networks}
            band={selectedBand}
            connectedBssid={data.data?.connectedBssid}
          />

          {/* Legend */}
          <div class={cn(layout.inline.default, spacing.margin.top.inline)}>
            <div class={layout.inline.tight}>
              <div class="w-4 h-4 bg-brand-primary opacity-70 rounded" />
              <span class="caption text-text-muted">{tc("status.connected")}</span>
            </div>
            <div class={layout.inline.tight}>
              <div class="w-4 h-4 bg-status-info opacity-40 rounded" />
              <span class="caption text-text-muted">{tr("wifi.channelGraph.otherNetworks")}</span>
            </div>
          </div>
        </>
      ) : (
        <CardValue value={data?.error || tc("status.unavailable")} size="md" status="error" />
      )}
    </SimpleBaseCard>
  );
}
