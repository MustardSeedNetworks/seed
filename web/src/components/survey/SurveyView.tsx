import { useState, useEffect, useCallback } from "react";
import { FloorPlanCanvas } from "./FloorPlanCanvas";
import { getAuthHeaders } from "../../hooks/useAuth";
import type {
  Survey,
  PassiveSample,
  ActiveSample,
  ThroughputSample,
} from "../../hooks/useSurvey";
import { X, Upload, Play, Pause, CheckCircle, Loader } from "../ui/Icons";

const API_BASE = import.meta.env.VITE_API_BASE || "";

interface SurveyViewProps {
  survey: Survey;
  onClose: () => void;
  onUpdate: () => void;
}

export function SurveyView({ survey: initialSurvey, onClose, onUpdate }: SurveyViewProps) {
  const [survey, setSurvey] = useState(initialSurvey);
  const [sampling, setSampling] = useState(false);
  const [uploadingFloorPlan, setUploadingFloorPlan] = useState(false);
  const [heatmapMetric, setHeatmapMetric] = useState<"rssi" | "throughput" | "latency" | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Poll for survey updates when in progress
  useEffect(() => {
    if (survey.status !== "in_progress") return;

    const interval = setInterval(async () => {
      try {
        const res = await fetch(`${API_BASE}/api/survey?id=${survey.id}`, {
          headers: getAuthHeaders(),
        });
        if (res.ok) {
          const updated = await res.json();
          setSurvey(updated);
        }
      } catch (err) {
        console.error("Failed to refresh survey:", err);
      }
    }, 3000);

    return () => clearInterval(interval);
  }, [survey.id, survey.status]);

  // Handle floor plan upload
  const handleFloorPlanUpload = useCallback(async (file: File) => {
    setUploadingFloorPlan(true);
    setError(null);

    try {
      // Read file as base64
      const reader = new FileReader();
      reader.onload = async (e) => {
        const imageData = e.target?.result as string;

        // Get image dimensions
        const img = new Image();
        img.onload = async () => {
          const floorPlan = {
            imageData,
            width: img.width,
            height: img.height,
            scaleM: 0.1, // Default: 10cm per pixel (adjust in settings)
          };

          // Upload to server
          const res = await fetch(`${API_BASE}/api/survey/floorplan?id=${survey.id}`, {
            method: "POST",
            headers: {
              ...getAuthHeaders(),
              "Content-Type": "application/json",
            },
            body: JSON.stringify(floorPlan),
          });

          if (res.ok) {
            // Refresh survey
            const updated = await res.json();
            setSurvey(updated);
            onUpdate();
          } else {
            throw new Error("Failed to upload floor plan");
          }
        };
        img.src = imageData;
      };
      reader.readAsDataURL(file);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to upload floor plan");
    } finally {
      setUploadingFloorPlan(false);
    }
  }, [survey.id, onUpdate]);

  // Handle taking a sample at clicked location
  const handlePointClick = useCallback(async (x: number, y: number) => {
    if (survey.status !== "in_progress") return;

    setSampling(true);
    setError(null);

    try {
      // Collect sample data based on survey type
      let sampleData: PassiveSample | ActiveSample | ThroughputSample;

      switch (survey.surveyType) {
        case "passive":
          // Fetch WiFi scan
          const scanRes = await fetch(`${API_BASE}/api/wifi/scan`, {
            headers: getAuthHeaders(),
          });
          if (!scanRes.ok) throw new Error("WiFi scan failed");
          const scanData = await scanRes.json();
          sampleData = { networks: scanData.networks || [] };
          break;

        case "active":
          // Fetch current WiFi status
          const wifiRes = await fetch(`${API_BASE}/api/wifi`, {
            headers: getAuthHeaders(),
          });
          if (!wifiRes.ok) throw new Error("WiFi status fetch failed");
          const wifiData = await wifiRes.json();

          // Check if BSSID changed (roaming)
          const lastSample = survey.samples[survey.samples.length - 1];
          const lastBssid = lastSample ? (lastSample.sampleData as ActiveSample).bssid : null;
          const roamingEvent = lastBssid !== null && lastBssid !== wifiData.bssid;

          sampleData = {
            ssid: wifiData.ssid || "",
            bssid: wifiData.bssid || "",
            rssi: wifiData.signal || 0,
            dataRate: wifiData.bitrate || 0,
            roamingEvent,
          };
          break;

        case "throughput":
          // Fetch WiFi status first
          const wifiRes2 = await fetch(`${API_BASE}/api/wifi`, {
            headers: getAuthHeaders(),
          });
          if (!wifiRes2.ok) throw new Error("WiFi status fetch failed");
          const wifiData2 = await wifiRes2.json();

          // Run iperf3 test
          if (!survey.iperfServer) {
            throw new Error("iperf3 server not configured for this survey");
          }

          const [host, port] = survey.iperfServer.split(":");
          const iperfRes = await fetch(`${API_BASE}/api/iperf/client`, {
            method: "POST",
            headers: {
              ...getAuthHeaders(),
              "Content-Type": "application/json",
            },
            body: JSON.stringify({
              host,
              port: port ? parseInt(port) : 5201,
              duration: survey.testDuration || 3,
              reverse: false,
            }),
          });

          if (!iperfRes.ok) throw new Error("iperf3 test failed");
          const iperfData = await iperfRes.json();

          sampleData = {
            ssid: wifiData2.ssid || "",
            bssid: wifiData2.bssid || "",
            rssi: wifiData2.signal || 0,
            downloadMbps: iperfData.summary?.sum_received?.bits_per_second
              ? iperfData.summary.sum_received.bits_per_second / 1_000_000
              : 0,
            uploadMbps: iperfData.summary?.sum_sent?.bits_per_second
              ? iperfData.summary.sum_sent.bits_per_second / 1_000_000
              : 0,
            latency: 0, // Not available in standard iperf3
            jitter: iperfData.summary?.sum_received?.jitter_ms || 0,
            packetLoss: iperfData.summary?.sum_received?.lost_percent || 0,
          };
          break;

        default:
          throw new Error("Unknown survey type");
      }

      // Submit sample to server
      const res = await fetch(`${API_BASE}/api/survey/sample?id=${survey.id}`, {
        method: "POST",
        headers: {
          ...getAuthHeaders(),
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ x, y, sampleData }),
      });

      if (!res.ok) throw new Error("Failed to save sample");

      // Refresh survey to get updated samples
      const refreshRes = await fetch(`${API_BASE}/api/survey?id=${survey.id}`, {
        headers: getAuthHeaders(),
      });
      if (refreshRes.ok) {
        const updated = await refreshRes.json();
        setSurvey(updated);
        onUpdate();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to take sample");
    } finally {
      setSampling(false);
    }
  }, [survey, onUpdate]);

  // Handle status changes
  const handleStatusChange = async (action: "start" | "pause" | "complete") => {
    try {
      const res = await fetch(`${API_BASE}/api/survey/${action}?id=${survey.id}`, {
        method: "POST",
        headers: getAuthHeaders(),
      });

      if (res.ok) {
        const updated = await res.json();
        setSurvey(updated);
        onUpdate();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : `Failed to ${action} survey`);
    }
  };

  return (
    <div className="fixed inset-0 bg-surface-base z-50 overflow-auto">
      {/* Header */}
      <div className="sticky top-0 bg-surface-raised border-b border-surface-border z-10">
        <div className="max-w-7xl mx-auto px-4 py-4 flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">{survey.name}</h1>
            <p className="text-sm text-text-muted mt-1">
              {survey.surveyType.charAt(0).toUpperCase() + survey.surveyType.slice(1)} Survey •{" "}
              {survey.samples.length} samples • {survey.status}
            </p>
          </div>

          <div className="flex items-center gap-2">
            {/* Status controls */}
            {survey.status === "created" && (
              <button
                onClick={() => handleStatusChange("start")}
                className="px-4 py-2 bg-brand-primary text-white rounded hover:bg-brand-primary/90 flex items-center gap-2"
              >
                <Play className="h-4 w-4" />
                Start Survey
              </button>
            )}

            {survey.status === "in_progress" && (
              <>
                <button
                  onClick={() => handleStatusChange("pause")}
                  className="px-4 py-2 border border-surface-border rounded hover:bg-surface-hover flex items-center gap-2"
                >
                  <Pause className="h-4 w-4" />
                  Pause
                </button>
                <button
                  onClick={() => handleStatusChange("complete")}
                  className="px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700 flex items-center gap-2"
                >
                  <CheckCircle className="h-4 w-4" />
                  Complete
                </button>
              </>
            )}

            {survey.status === "paused" && (
              <>
                <button
                  onClick={() => handleStatusChange("start")}
                  className="px-4 py-2 bg-brand-primary text-white rounded hover:bg-brand-primary/90 flex items-center gap-2"
                >
                  <Play className="h-4 w-4" />
                  Resume
                </button>
                <button
                  onClick={() => handleStatusChange("complete")}
                  className="px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700 flex items-center gap-2"
                >
                  <CheckCircle className="h-4 w-4" />
                  Complete
                </button>
              </>
            )}

            <button
              onClick={onClose}
              className="px-4 py-2 border border-surface-border rounded hover:bg-surface-hover flex items-center gap-2"
            >
              <X className="h-4 w-4" />
              Close
            </button>
          </div>
        </div>
      </div>

      {/* Main content */}
      <div className="max-w-7xl mx-auto px-4 py-6">
        {error && (
          <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded mb-4">
            {error}
          </div>
        )}

        {sampling && (
          <div className="bg-blue-50 border border-blue-200 text-blue-700 px-4 py-3 rounded mb-4 flex items-center gap-2">
            <Loader className="h-4 w-4 animate-spin" />
            Taking measurement...
          </div>
        )}

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Floor plan */}
          <div className="lg:col-span-2">
            <div className="bg-surface-raised rounded-lg border border-surface-border p-4">
              <div className="flex items-center justify-between mb-4">
                <h2 className="text-lg font-semibold">Floor Plan</h2>
                {heatmapMetric === null && survey.samples.length > 0 && (
                  <div className="flex gap-2">
                    <button
                      onClick={() => setHeatmapMetric("rssi")}
                      className="px-3 py-1 text-sm border border-surface-border rounded hover:bg-surface-hover"
                    >
                      RSSI Heatmap
                    </button>
                    {survey.surveyType === "throughput" && (
                      <>
                        <button
                          onClick={() => setHeatmapMetric("throughput")}
                          className="px-3 py-1 text-sm border border-surface-border rounded hover:bg-surface-hover"
                        >
                          Throughput
                        </button>
                        <button
                          onClick={() => setHeatmapMetric("latency")}
                          className="px-3 py-1 text-sm border border-surface-border rounded hover:bg-surface-hover"
                        >
                          Latency
                        </button>
                      </>
                    )}
                  </div>
                )}
                {heatmapMetric !== null && (
                  <button
                    onClick={() => setHeatmapMetric(null)}
                    className="px-3 py-1 text-sm bg-brand-primary text-white rounded hover:bg-brand-primary/90"
                  >
                    Hide Heatmap
                  </button>
                )}
              </div>

              {!survey.floorPlan ? (
                <div className="border-2 border-dashed border-surface-border rounded-lg p-12 text-center">
                  <Upload className="h-12 w-12 mx-auto text-text-muted mb-4" />
                  <p className="text-text-muted mb-4">Upload a floor plan to begin</p>
                  <label className="inline-block px-4 py-2 bg-brand-primary text-white rounded cursor-pointer hover:bg-brand-primary/90">
                    {uploadingFloorPlan ? "Uploading..." : "Choose File"}
                    <input
                      type="file"
                      accept="image/png,image/jpeg,image/jpg,image/gif,image/webp,image/svg+xml"
                      className="hidden"
                      onChange={(e) => {
                        const file = e.target.files?.[0];
                        if (file) handleFloorPlanUpload(file);
                      }}
                      disabled={uploadingFloorPlan}
                    />
                  </label>
                  <p className="text-xs text-text-muted mt-2">
                    PNG, JPG, GIF, WEBP, or SVG (max 10MB)
                  </p>
                </div>
              ) : (
                <div>
                  {survey.status === "in_progress" && (
                    <p className="text-sm text-text-muted mb-2">
                      Click on the floor plan to take a measurement at that location
                    </p>
                  )}
                  <FloorPlanCanvas
                    floorPlan={survey.floorPlan}
                    samples={survey.samples}
                    onPointClick={handlePointClick}
                    interactive={survey.status === "in_progress" && !sampling}
                    heatmapMetric={heatmapMetric}
                  />
                </div>
              )}
            </div>
          </div>

          {/* Sample list */}
          <div className="lg:col-span-1">
            <div className="bg-surface-raised rounded-lg border border-surface-border p-4">
              <h2 className="text-lg font-semibold mb-4">
                Samples ({survey.samples.length})
              </h2>
              <div className="space-y-2 max-h-[600px] overflow-y-auto">
                {survey.samples.length === 0 ? (
                  <p className="text-sm text-text-muted text-center py-8">
                    No samples yet. {survey.status === "in_progress" ? "Click on the floor plan to start." : "Start the survey to begin."}
                  </p>
                ) : (
                  survey.samples.map((sample, idx) => (
                    <div
                      key={idx}
                      className="border border-surface-border rounded p-3 text-sm"
                    >
                      <div className="flex items-center justify-between mb-2">
                        <span className="font-semibold">#{idx + 1}</span>
                        <span className="text-xs text-text-muted">
                          {new Date(sample.timestamp).toLocaleTimeString()}
                        </span>
                      </div>
                      <div className="text-xs space-y-1">
                        {renderSampleData(sample.sampleData, survey.surveyType)}
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

// Helper to render sample data
function renderSampleData(data: any, surveyType: string) {
  if (surveyType === "passive") {
    const passiveData = data as PassiveSample;
    return (
      <>
        <div>Networks: {passiveData.networks?.length || 0}</div>
        {passiveData.networks?.[0] && (
          <>
            <div>Strongest: {passiveData.networks[0].ssid}</div>
            <div>RSSI: {passiveData.networks[0].rssi} dBm</div>
          </>
        )}
      </>
    );
  }

  if (surveyType === "active") {
    const activeData = data as ActiveSample;
    return (
      <>
        <div>SSID: {activeData.ssid}</div>
        <div>RSSI: {activeData.rssi} dBm</div>
        <div>Rate: {activeData.dataRate} Mbps</div>
        {activeData.roamingEvent && (
          <div className="text-yellow-600 font-semibold">⚠ Roaming</div>
        )}
      </>
    );
  }

  if (surveyType === "throughput") {
    const throughputData = data as ThroughputSample;
    return (
      <>
        <div>RSSI: {throughputData.rssi} dBm</div>
        <div>↓ {throughputData.downloadMbps.toFixed(1)} Mbps</div>
        <div>↑ {throughputData.uploadMbps.toFixed(1)} Mbps</div>
        <div>Jitter: {throughputData.jitter.toFixed(1)} ms</div>
        {throughputData.packetLoss > 0 && (
          <div className="text-red-600">Loss: {throughputData.packetLoss.toFixed(1)}%</div>
        )}
      </>
    );
  }

  return null;
}
