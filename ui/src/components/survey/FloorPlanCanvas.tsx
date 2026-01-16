// biome-ignore-all lint/complexity/noExcessiveCognitiveComplexity: Complex component
/**
 * FloorPlanCanvas Component (~234 lines)
 *
 * Purpose: HTML5 Canvas-based floor plan visualization for WiFi survey heatmaps.
 * Renders floor plan images with overlaid sample points and heatmap color gradients
 * showing signal strength or performance metrics at each location.
 *
 * Key Features:
 * - Image rendering: Displays floor plan images maintaining aspect ratio
 * - Sample markers: Shows measurement points as circles on the floor plan
 * - Heatmap visualization: Color-coded overlays (red=weak, yellow=medium, green=strong)
 * - Interactive mode: Click to add new sample points
 * - Responsive sizing: Automatically adapts to container width
 * - Metric selection: Display RSSI, throughput, or latency heatmaps
 * - Interpolation: Smooth color gradient between sample points
 * - Canvas optimization: Efficient rendering with minimal redraws
 *
 * Usage:
 * ```typescript
 * <FloorPlanCanvas
 *   floorPlan={floorPlanData}
 *   samples={samplePoints}
 *   onPointClick={handleClick}
 *   interactive={true}
 *   heatmapMetric="rssi"
 * />
 * ```
 *
 * Dependencies: Canvas API, floor plan and sample point types
 * State: Canvas dimensions, rendering state
 */

import type React from "react";
import { useCallback, useEffect, useRef, useState } from "react";
import type {
  ActiveSample,
  ApLocation,
  FloorPlan,
  HeatmapFilter,
  HeatmapMetric,
  PassiveSample,
  SamplePoint,
  ScannedNetwork,
  ThroughputSample,
} from "../../hooks/useSurvey";

/** Type alias for polymorphic sample data */
type SampleData = PassiveSample | ActiveSample | ThroughputSample;

import { cn, radius } from "../../styles/theme";

export interface CalibrationPoint {
  x: number;
  y: number;
}

interface FloorPlanCanvasProps {
  floorPlan: FloorPlan;
  samples: SamplePoint[];
  onPointClick?: (x: number, y: number) => void;
  interactive?: boolean;
  heatmapMetric?: HeatmapMetric;
  heatmapFilter?: HeatmapFilter;
  calibrationMode?: boolean;
  calibrationPoints?: CalibrationPoint[];
  onCalibrationClick?: (x: number, y: number) => void;
  apLocations?: ApLocation[];
  showApLabels?: boolean;
  apPlacementMode?: boolean;
  onApPlacementClick?: (x: number, y: number) => void;
  selectedApId?: string;
}

/**
 * FloorPlanCanvas Component
 * Renders floor plan with interactive sample points and heatmap visualization
 */
export function FloorPlanCanvas({
  floorPlan,
  samples,
  onPointClick,
  interactive = false,
  heatmapMetric = null,
  heatmapFilter,
  calibrationMode = false,
  calibrationPoints = [],
  onCalibrationClick,
  apLocations = [],
  showApLabels = true,
  apPlacementMode = false,
  onApPlacementClick,
  selectedApId,
}: FloorPlanCanvasProps): React.ReactElement {
  // Canvas DOM reference for drawing
  const canvasRef = useRef<HTMLCanvasElement>(null);
  // Container reference to measure available space
  const containerRef = useRef<HTMLDivElement>(null);
  // Track canvas dimensions for responsive sizing
  const [dimensions, setDimensions] = useState({ width: 0, height: 0 });
  // Track mouse position for calibration line preview
  const [mousePos, setMousePos] = useState<{ x: number; y: number } | null>(null);

  // Calculate canvas dimensions maintaining aspect ratio
  useEffect(() => {
    if (!(containerRef.current && floorPlan)) {
      return;
    }

    const container = containerRef.current;
    const containerWidth = container.clientWidth;
    const aspectRatio = floorPlan.height / floorPlan.width;

    const width = Math.min(containerWidth, 1200);
    const height = width * aspectRatio;

    setDimensions({ width, height });
  }, [floorPlan]);

  // Handle mouse move for calibration line preview
  const handleMouseMove = useCallback(
    (e: React.MouseEvent<HTMLCanvasElement>) => {
      // Only track mouse when in calibration mode with exactly one point set
      if (!calibrationMode || calibrationPoints.length !== 1) {
        if (mousePos !== null) {
          setMousePos(null);
        }
        return;
      }

      const canvas = canvasRef.current;
      if (!(canvas && floorPlan)) {
        return;
      }

      const rect = canvas.getBoundingClientRect();
      // Convert to floor plan coordinates
      const scaleX = floorPlan.width / dimensions.width;
      const scaleY = floorPlan.height / dimensions.height;

      setMousePos({
        x: (e.clientX - rect.left) * scaleX,
        y: (e.clientY - rect.top) * scaleY,
      });
    },
    [calibrationMode, calibrationPoints.length, floorPlan, dimensions, mousePos],
  );

  // Draw floor plan and samples
  useEffect(() => {
    if (!(canvasRef.current && floorPlan) || dimensions.width === 0) {
      return;
    }

    const canvas = canvasRef.current;
    const ctx = canvas.getContext("2d");
    if (!ctx) {
      return;
    }

    // Set canvas size
    canvas.width = dimensions.width;
    canvas.height = dimensions.height;

    // Draw floor plan image
    // Track Image object for cleanup on unmount/re-render (fixes #853)
    let isMounted = true;
    const img = new Image();
    img.onload = (): void => {
      // Skip render if component unmounted or effect re-ran (fixes #853)
      if (!isMounted) {
        return;
      }
      ctx.drawImage(img, 0, 0, dimensions.width, dimensions.height);

      // Calculate scale factor
      const scaleX = dimensions.width / floorPlan.width;
      const scaleY = dimensions.height / floorPlan.height;

      // Draw heatmap if requested
      if (heatmapMetric && samples.length > 0) {
        drawHeatmap(ctx, samples, heatmapMetric, scaleX, scaleY, heatmapFilter);
      }

      // Draw AP location markers
      for (const ap of apLocations) {
        const ax = ap.x * scaleX;
        const ay = ap.y * scaleY;
        const isSelected = ap.id === selectedApId;

        // Draw AP icon (antenna shape)
        ctx.save();
        ctx.translate(ax, ay);

        // Draw signal rings for selected AP
        if (isSelected) {
          ctx.strokeStyle = "rgba(34, 197, 94, 0.3)"; // green-500 with opacity
          ctx.lineWidth = 2;
          for (const r of [20, 35, 50]) {
            ctx.beginPath();
            ctx.arc(0, 0, r, 0, 2 * Math.PI);
            ctx.stroke();
          }
        }

        // Draw AP marker (triangle/antenna icon)
        ctx.beginPath();
        ctx.moveTo(0, -12);
        ctx.lineTo(-8, 6);
        ctx.lineTo(8, 6);
        ctx.closePath();
        ctx.fillStyle = isSelected
          ? "rgba(34, 197, 94, 0.9)" // green-500
          : "rgba(168, 85, 247, 0.9)"; // purple-500
        ctx.fill();
        ctx.strokeStyle = "#ffffff";
        ctx.lineWidth = 2;
        ctx.stroke();

        // Draw antenna line
        ctx.beginPath();
        ctx.moveTo(0, -12);
        ctx.lineTo(0, -18);
        ctx.strokeStyle = isSelected ? "#22c55e" : "#a855f7";
        ctx.lineWidth = 2;
        ctx.stroke();

        // Draw antenna top
        ctx.beginPath();
        ctx.arc(0, -18, 3, 0, 2 * Math.PI);
        ctx.fillStyle = isSelected ? "#22c55e" : "#a855f7";
        ctx.fill();

        ctx.restore();

        // Draw label if enabled
        if (showApLabels && ap.label) {
          ctx.fillStyle = "rgba(0, 0, 0, 0.8)";
          const labelWidth = ctx.measureText(ap.label).width + 8;
          ctx.fillRect(ax - labelWidth / 2, ay + 10, labelWidth, 16);
          ctx.fillStyle = "#ffffff";
          ctx.font = "bold 10px sans-serif";
          ctx.textAlign = "center";
          ctx.textBaseline = "top";
          ctx.fillText(ap.label, ax, ay + 12);
        }
      }

      // Draw sample points
      for (const sample of samples) {
        const x = sample.x * scaleX;
        const y = sample.y * scaleY;

        // Draw point
        // Note: Canvas API requires direct color values - CSS variables don't work
        // These colors are intentionally hardcoded for Canvas compatibility
        // See: https://developer.mozilla.org/en-US/docs/Web/API/CanvasRenderingContext2D/fillStyle
        ctx.beginPath();
        ctx.arc(x, y, 8, 0, 2 * Math.PI);
        ctx.fillStyle = heatmapMetric
          ? "rgba(255, 255, 255, 0.8)" // white for visibility on heatmap
          : "rgba(5, 104, 57, 0.8)"; // brand-primary green (#056839 at 80% opacity)
        ctx.fill();
        ctx.strokeStyle = "rgba(255, 255, 255, 1)"; // white border for visibility
        ctx.lineWidth = 2;
        ctx.stroke();

        // Draw point number - high contrast text for visibility
        // Canvas API limitation: must use direct color values
        ctx.fillStyle = heatmapMetric ? "#1e293b" : "#f8fafc"; // slate-800 / slate-50
        ctx.font = "bold 10px sans-serif";
        ctx.textAlign = "center";
        ctx.textBaseline = "middle";
        const pointNum = samples.indexOf(sample) + 1;
        ctx.fillText(pointNum.toString(), x, y);
      }

      // Draw calibration points and line if in calibration mode
      if (calibrationMode && calibrationPoints.length > 0) {
        // Draw calibration points
        for (const [index, point] of calibrationPoints.entries()) {
          const cx = point.x * scaleX;
          const cy = point.y * scaleY;

          // Draw point
          ctx.beginPath();
          ctx.arc(cx, cy, 10, 0, 2 * Math.PI);
          ctx.fillStyle = "rgba(234, 88, 12, 0.9)"; // orange-600
          ctx.fill();
          ctx.strokeStyle = "rgba(255, 255, 255, 1)";
          ctx.lineWidth = 2;
          ctx.stroke();

          // Draw label (A or B)
          ctx.fillStyle = "#ffffff";
          ctx.font = "bold 12px sans-serif";
          ctx.textAlign = "center";
          ctx.textBaseline = "middle";
          ctx.fillText(index === 0 ? "A" : "B", cx, cy);
        }

        // Draw preview line from first point to mouse cursor
        if (calibrationPoints.length === 1 && mousePos) {
          const [p1] = calibrationPoints;

          ctx.beginPath();
          ctx.moveTo(p1.x * scaleX, p1.y * scaleY);
          ctx.lineTo(mousePos.x * scaleX, mousePos.y * scaleY);
          ctx.strokeStyle = "rgba(234, 88, 12, 0.6)"; // orange-600 with less opacity
          ctx.lineWidth = 2;
          ctx.setLineDash([5, 5]);
          ctx.stroke();
          ctx.setLineDash([]); // Reset dash

          // Draw preview pixel distance at cursor
          const previewDist = Math.sqrt((mousePos.x - p1.x) ** 2 + (mousePos.y - p1.y) ** 2);
          ctx.fillStyle = "rgba(0, 0, 0, 0.6)";
          ctx.fillRect(mousePos.x * scaleX + 10, mousePos.y * scaleY - 10, 60, 20);
          ctx.fillStyle = "#ffffff";
          ctx.font = "bold 10px sans-serif";
          ctx.textAlign = "left";
          ctx.textBaseline = "middle";
          ctx.fillText(
            `${previewDist.toFixed(0)} px`,
            mousePos.x * scaleX + 15,
            mousePos.y * scaleY,
          );
        }

        // Draw line between points if we have two
        if (calibrationPoints.length === 2) {
          const [p1, p2] = calibrationPoints;

          ctx.beginPath();
          ctx.moveTo(p1.x * scaleX, p1.y * scaleY);
          ctx.lineTo(p2.x * scaleX, p2.y * scaleY);
          ctx.strokeStyle = "rgba(234, 88, 12, 0.9)"; // orange-600
          ctx.lineWidth = 2;
          ctx.setLineDash([5, 5]);
          ctx.stroke();
          ctx.setLineDash([]); // Reset dash

          // Draw pixel distance label at midpoint
          const midX = (p1.x * scaleX + p2.x * scaleX) / 2;
          const midY = (p1.y * scaleY + p2.y * scaleY) / 2;
          const pixelDist = Math.sqrt((p2.x - p1.x) ** 2 + (p2.y - p1.y) ** 2);

          ctx.fillStyle = "rgba(0, 0, 0, 0.7)";
          ctx.fillRect(midX - 40, midY - 10, 80, 20);
          ctx.fillStyle = "#ffffff";
          ctx.font = "bold 11px sans-serif";
          ctx.textAlign = "center";
          ctx.fillText(`${pixelDist.toFixed(0)} px`, midX, midY);
        }
      }
    };

    img.src = floorPlan.imageData;

    // Cleanup: cancel pending image load and prevent stale renders (fixes #853)
    return () => {
      isMounted = false;
      img.onload = null; // Prevent callback from firing after cleanup
      img.src = ""; // Cancel pending load
    };
  }, [
    floorPlan,
    samples,
    dimensions,
    heatmapMetric,
    heatmapFilter,
    calibrationMode,
    calibrationPoints,
    mousePos,
    apLocations,
    showApLabels,
    selectedApId,
  ]);

  // Handle canvas click
  const handleCanvasClick = (e: React.MouseEvent<HTMLCanvasElement>): void => {
    if (!(canvasRef.current && floorPlan)) {
      return;
    }

    const canvas = canvasRef.current;
    const rect = canvas.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;

    // Convert canvas coordinates to floor plan coordinates
    const scaleX = floorPlan.width / dimensions.width;
    const scaleY = floorPlan.height / dimensions.height;

    const floorPlanX = Math.round(x * scaleX);
    const floorPlanY = Math.round(y * scaleY);

    // Handle AP placement click if in AP placement mode
    if (apPlacementMode && onApPlacementClick) {
      onApPlacementClick(floorPlanX, floorPlanY);
      return;
    }

    // Handle calibration click if in calibration mode
    if (calibrationMode && onCalibrationClick) {
      onCalibrationClick(floorPlanX, floorPlanY);
      return;
    }

    // Handle regular point click if interactive
    if (interactive && onPointClick) {
      onPointClick(floorPlanX, floorPlanY);
    }
  };

  return (
    <div ref={containerRef} class="w-full">
      <canvas
        ref={canvasRef}
        onClick={handleCanvasClick}
        onMouseMove={handleMouseMove}
        class={cn(
          "border border-surface-border",
          radius.md,
          interactive || calibrationMode || apPlacementMode ? "cursor-crosshair" : "",
        )}
        width={dimensions.width}
        height={dimensions.height}
      />
    </div>
  );
}

/** Apply filter to sample data and extract matching networks */
function filterSampleData(sample: SamplePoint, filter?: HeatmapFilter): ScannedNetwork[] {
  const data = sample.sampleData as SampleData & {
    networks?: ScannedNetwork[];
    channel?: number;
    frequency?: number;
    channelWidth?: ScannedNetwork["channelWidth"];
    phyType?: ScannedNetwork["phyType"];
    security?: ScannedNetwork["security"];
    vendor?: string;
  };

  // For passive surveys, filter networks array
  if (data.networks && Array.isArray(data.networks)) {
    let networks = data.networks as ScannedNetwork[];

    if (filter) {
      if (filter.ssid) {
        networks = networks.filter((n) => n.ssid === filter.ssid || n.ssid.includes(filter.ssid));
      }
      if (filter.bssid) {
        networks = networks.filter((n) =>
          n.bssid.toLowerCase().includes(filter.bssid.toLowerCase()),
        );
      }
      if (filter.channel) {
        networks = networks.filter((n) => n.channel === filter.channel);
      }
      if (filter.band) {
        networks = networks.filter((n) => {
          if (filter.band === "2.4") {
            return n.frequency >= 2400 && n.frequency < 2500;
          }
          if (filter.band === "5") {
            return n.frequency >= 5000 && n.frequency < 6000;
          }
          if (filter.band === "6") {
            return n.frequency >= 5925 && n.frequency < 7125;
          }
          return true;
        });
      }
      if (filter.minRssi !== undefined) {
        networks = networks.filter((n) => filter.minRssi !== undefined && n.rssi >= filter.minRssi);
      }
      // New filters
      if (filter.channelWidth) {
        networks = networks.filter((n) => n.channelWidth === filter.channelWidth);
      }
      if (filter.phyType) {
        networks = networks.filter((n) => n.phyType === filter.phyType);
      }
      if (filter.security) {
        networks = networks.filter((n) => n.security === filter.security);
      }
      if (filter.vendor) {
        networks = networks.filter((n) => n.vendor === filter.vendor);
      }
    }

    return networks;
  }

  // For active/throughput surveys, return single network
  if (data.ssid && data.bssid) {
    const network: ScannedNetwork = {
      ssid: data.ssid,
      bssid: data.bssid,
      rssi: data.rssi || -100,
      channel: data.channel || 0,
      frequency: data.frequency || 0,
      channelWidth: data.channelWidth,
      phyType: data.phyType,
      security: data.security,
      vendor: data.vendor,
    };

    // Apply filter for active surveys
    if (filter) {
      if (filter.ssid && !network.ssid.includes(filter.ssid)) {
        return [];
      }
      if (filter.bssid && !network.bssid.toLowerCase().includes(filter.bssid.toLowerCase())) {
        return [];
      }
      if (filter.minRssi !== undefined && network.rssi < filter.minRssi) {
        return [];
      }
      if (filter.channelWidth && network.channelWidth !== filter.channelWidth) {
        return [];
      }
      if (filter.phyType && network.phyType !== filter.phyType) {
        return [];
      }
      if (filter.security && network.security !== filter.security) {
        return [];
      }
      if (filter.vendor && network.vendor !== filter.vendor) {
        return [];
      }
    }

    return [network];
  }

  return [];
}

/** Extended sample data type for metric extraction */
type ExtendedSampleData = SampleData & {
  networks?: ScannedNetwork[];
  downloadMbps?: number;
  latency?: number;
  noiseFloor?: number;
  rssi?: number;
  channelUtilization?: number;
};

/** Extract metric value from sample data */
function extractMetricValue(
  sample: SamplePoint,
  metric: HeatmapMetric,
  filter?: HeatmapFilter,
): number {
  const data = sample.sampleData as ExtendedSampleData;
  const filteredNetworks = filterSampleData(sample, filter);

  switch (metric) {
    case "rssi": {
      if (filteredNetworks.length > 0) {
        // Return best RSSI among filtered networks
        return Math.max(...filteredNetworks.map((n) => n.rssi));
      }
      return data.rssi || -100;
    }
    case "throughput":
      return data.downloadMbps || 0;
    case "latency":
      return data.latency || 0;
    case "snr": {
      // SNR = RSSI - Noise Floor (assume -95 dBm noise floor if not available)
      const rssi =
        filteredNetworks.length > 0
          ? Math.max(...filteredNetworks.map((n) => n.rssi))
          : data.rssi || -100;
      const noiseFloor = data.noiseFloor || -95;
      return rssi - noiseFloor;
    }
    case "noise":
      return data.noiseFloor || -95;
    case "cochannel": {
      // Count networks on same channel (co-channel interference)
      if (filteredNetworks.length === 0) {
        return 0;
      }
      const primaryChannel = filteredNetworks[0].channel;
      const allNetworks = (data.networks || []) as ScannedNetwork[];
      return allNetworks.filter((n) => n.channel === primaryChannel).length - 1;
    }
    case "adjacent": {
      // Count networks on adjacent channels
      if (filteredNetworks.length === 0) {
        return 0;
      }
      const primaryChannel = filteredNetworks[0].channel;
      const allNetworks = (data.networks || []) as ScannedNetwork[];
      return allNetworks.filter(
        (n) =>
          Math.abs(n.channel - primaryChannel) > 0 && Math.abs(n.channel - primaryChannel) <= 2,
      ).length;
    }
    case "channelUtil":
      // Channel utilization (if available)
      return data.channelUtilization || 0;
    case "apDensity": {
      // Count unique BSSIDs (APs) at this location
      if (data.networks && Array.isArray(data.networks)) {
        const uniqueBssiDs = new Set(data.networks.map((n: ScannedNetwork) => n.bssid));
        return uniqueBssiDs.size;
      }
      return data.uniqueBSSIDs || 0;
    }
    case "ssidCount": {
      // Count unique SSIDs at this location
      if (data.networks && Array.isArray(data.networks)) {
        const uniqueSsiDs = new Set(
          data.networks.map((n: ScannedNetwork) => n.ssid).filter(Boolean),
        );
        return uniqueSsiDs.size;
      }
      return data.uniqueSSIDs || 0;
    }
    default:
      return 0;
  }
}

// Helper function to draw heatmap
function drawHeatmap(
  ctx: CanvasRenderingContext2D,
  samples: SamplePoint[],
  metric: HeatmapMetric,
  scaleX: number,
  scaleY: number,
  filter?: HeatmapFilter,
): void {
  if (samples.length === 0 || !metric) {
    return;
  }

  // Extract metric values with filter applied
  const values = samples.map((s) => extractMetricValue(s, metric, filter));

  // Filter out invalid values
  const validValues = values.filter((v) => v !== null && !Number.isNaN(v));
  if (validValues.length === 0) {
    return;
  }

  const minValue = Math.min(...validValues);
  const maxValue = Math.max(...validValues);

  // Create gradient overlay
  const { canvas } = ctx;
  const imageData = ctx.createImageData(canvas.width, canvas.height);
  const { data } = imageData;

  // For each pixel, calculate interpolated value
  for (let y = 0; y < canvas.height; y++) {
    for (let x = 0; x < canvas.width; x++) {
      // Find nearest samples and interpolate
      let totalWeight = 0;
      let weightedValue = 0;

      // Use iterator pattern to avoid dynamic array indexing
      const valuesIterator = values[Symbol.iterator]();
      for (const sample of samples) {
        const sx = sample.x * scaleX;
        const sy = sample.y * scaleY;
        const distance = Math.sqrt((x - sx) ** 2 + (y - sy) ** 2);

        // Inverse distance weighting (IDW)
        const weight = distance === 0 ? 1000 : 1 / distance ** 2;
        totalWeight += weight;
        const valueResult = valuesIterator.next();
        const sampleValue = valueResult.done ? 0 : valueResult.value;
        weightedValue += weight * sampleValue;
      }

      const value = weightedValue / totalWeight;

      // Normalize to 0-1
      const normalized = (value - minValue) / (maxValue - minValue || 1);

      // Get color for this value
      const color = getHeatmapColor(normalized, metric);

      const pixelIndex = (y * canvas.width + x) * 4;
      // Bounds-checked pixel data assignment
      if (pixelIndex >= 0 && pixelIndex + 3 < data.length) {
        data.set([color.r, color.g, color.b, color.a], pixelIndex);
      }
    }
  }

  // Apply heatmap overlay
  ctx.globalAlpha = 0.5;
  ctx.putImageData(imageData, 0, 0);
  ctx.globalAlpha = 1.0;
}

// Get heatmap color based on normalized value (0-1)
function getHeatmapColor(
  value: number,
  metric: HeatmapMetric,
): { r: number; g: number; b: number; a: number } {
  // Determine if higher is better for this metric
  // Higher is better: RSSI, SNR, throughput
  // Lower is better: latency, noise, cochannel, adjacent, channelUtil
  const higherIsBetter = metric === "rssi" || metric === "snr" || metric === "throughput";

  // For metrics where higher is better, invert the normalization
  // so that high values appear green and low values appear red
  let normalizedValue = value;
  if (higherIsBetter) {
    normalizedValue = 1 - value;
  }

  // Color gradient: red (0/bad) -> yellow (0.5/medium) -> green (1/good)
  let r: number;
  let g: number;
  let b: number;

  // For interference metrics, use different color scheme (purple to blue)
  if (metric === "cochannel" || metric === "adjacent") {
    // Blue (low interference) to purple (high interference)
    if (normalizedValue < 0.5) {
      // Blue to cyan
      r = Math.round(100 * (normalizedValue * 2));
      g = Math.round(150 + 50 * (normalizedValue * 2));
      b = 255;
    } else {
      // Cyan to purple/magenta
      r = Math.round(100 + 155 * ((normalizedValue - 0.5) * 2));
      g = Math.round(200 - 150 * ((normalizedValue - 0.5) * 2));
      b = 255;
    }
    return { r, g, b, a: 180 };
  }

  // Standard red -> yellow -> green gradient
  if (normalizedValue < 0.5) {
    // Green to yellow (good to medium)
    r = Math.round(255 * (normalizedValue * 2));
    g = 255;
    b = 0;
  } else {
    // Yellow to red (medium to bad)
    r = 255;
    g = Math.round(255 * (1 - (normalizedValue - 0.5) * 2));
    b = 0;
  }

  return { r, g, b, a: 200 }; // Semi-transparent
}
