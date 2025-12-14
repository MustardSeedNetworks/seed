import { useRef, useEffect, useState } from "react";
import type { FloorPlan, SamplePoint } from "../../hooks/useSurvey";

interface FloorPlanCanvasProps {
  floorPlan: FloorPlan;
  samples: SamplePoint[];
  onPointClick?: (x: number, y: number) => void;
  interactive?: boolean;
  heatmapMetric?: "rssi" | "throughput" | "latency" | null;
}

export function FloorPlanCanvas({
  floorPlan,
  samples,
  onPointClick,
  interactive = false,
  heatmapMetric = null,
}: FloorPlanCanvasProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const [dimensions, setDimensions] = useState({ width: 0, height: 0 });

  // Calculate canvas dimensions maintaining aspect ratio
  useEffect(() => {
    if (!containerRef.current || !floorPlan) return;

    const container = containerRef.current;
    const containerWidth = container.clientWidth;
    const aspectRatio = floorPlan.height / floorPlan.width;

    const width = Math.min(containerWidth, 1200);
    const height = width * aspectRatio;

    setDimensions({ width, height });
  }, [floorPlan]);

  // Draw floor plan and samples
  useEffect(() => {
    if (!canvasRef.current || !floorPlan || dimensions.width === 0) return;

    const canvas = canvasRef.current;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    // Set canvas size
    canvas.width = dimensions.width;
    canvas.height = dimensions.height;

    // Draw floor plan image
    const img = new Image();
    img.onload = () => {
      ctx.drawImage(img, 0, 0, dimensions.width, dimensions.height);

      // Calculate scale factor
      const scaleX = dimensions.width / floorPlan.width;
      const scaleY = dimensions.height / floorPlan.height;

      // Draw heatmap if requested
      if (heatmapMetric && samples.length > 0) {
        drawHeatmap(ctx, samples, heatmapMetric, scaleX, scaleY);
      }

      // Draw sample points
      samples.forEach((sample) => {
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
          : "rgba(37, 99, 235, 0.8)"; // brand-primary (#2563eb at 80% opacity)
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
      });
    };

    img.src = floorPlan.imageData;
  }, [floorPlan, samples, dimensions, heatmapMetric]);

  // Handle canvas click
  const handleCanvasClick = (e: React.MouseEvent<HTMLCanvasElement>) => {
    if (!interactive || !onPointClick || !canvasRef.current || !floorPlan)
      return;

    const canvas = canvasRef.current;
    const rect = canvas.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;

    // Convert canvas coordinates to floor plan coordinates
    const scaleX = floorPlan.width / dimensions.width;
    const scaleY = floorPlan.height / dimensions.height;

    const floorPlanX = Math.round(x * scaleX);
    const floorPlanY = Math.round(y * scaleY);

    onPointClick(floorPlanX, floorPlanY);
  };

  return (
    <div ref={containerRef} className="w-full">
      <canvas
        ref={canvasRef}
        onClick={handleCanvasClick}
        className={`border border-surface-border rounded ${
          interactive ? "cursor-crosshair" : ""
        }`}
        style={{ maxWidth: "100%" }}
      />
    </div>
  );
}

// Helper function to draw heatmap
function drawHeatmap(
  ctx: CanvasRenderingContext2D,
  samples: SamplePoint[],
  metric: "rssi" | "throughput" | "latency",
  scaleX: number,
  scaleY: number,
) {
  if (samples.length === 0) return;

  // Extract metric values
  const values = samples.map((s) => {
    const data = s.sampleData as any;
    switch (metric) {
      case "rssi":
        return data.rssi || data.networks?.[0]?.rssi || -100;
      case "throughput":
        return data.downloadMbps || 0;
      case "latency":
        return data.latency || 0;
      default:
        return 0;
    }
  });

  const minValue = Math.min(...values);
  const maxValue = Math.max(...values);

  // Create gradient overlay
  const canvas = ctx.canvas;
  const imageData = ctx.createImageData(canvas.width, canvas.height);
  const data = imageData.data;

  // For each pixel, calculate interpolated value
  for (let y = 0; y < canvas.height; y++) {
    for (let x = 0; x < canvas.width; x++) {
      // Find nearest samples and interpolate
      let totalWeight = 0;
      let weightedValue = 0;

      samples.forEach((sample, idx) => {
        const sx = sample.x * scaleX;
        const sy = sample.y * scaleY;
        const distance = Math.sqrt((x - sx) ** 2 + (y - sy) ** 2);

        // Inverse distance weighting (IDW)
        const weight = distance === 0 ? 1000 : 1 / distance ** 2;
        totalWeight += weight;
        weightedValue += weight * values[idx];
      });

      const value = weightedValue / totalWeight;

      // Normalize to 0-1
      const normalized = (value - minValue) / (maxValue - minValue || 1);

      // Get color for this value
      const color = getHeatmapColor(normalized, metric);

      const idx = (y * canvas.width + x) * 4;
      data[idx] = color.r;
      data[idx + 1] = color.g;
      data[idx + 2] = color.b;
      data[idx + 3] = color.a;
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
  metric: "rssi" | "throughput" | "latency",
): { r: number; g: number; b: number; a: number } {
  // For RSSI, lower is worse (invert)
  if (metric === "rssi") {
    value = 1 - value;
  }
  // For latency, lower is better (invert)
  if (metric === "latency") {
    value = 1 - value;
  }

  // Color gradient: red (0) -> yellow (0.5) -> green (1)
  let r, g, b;

  if (value < 0.5) {
    // Red to yellow
    r = 255;
    g = Math.round(255 * (value * 2));
    b = 0;
  } else {
    // Yellow to green
    r = Math.round(255 * (1 - (value - 0.5) * 2));
    g = 255;
    b = 0;
  }

  return { r, g, b, a: 200 }; // Semi-transparent
}
