/**
 * SurveyView Component (~539 lines)
 *
 * Purpose: Full-featured WiFi site survey editor and viewer. Allows users to create
 * detailed signal strength and performance heatmaps by recording samples at physical
 * locations on a floor plan, supporting passive scanning, active testing, and iperf3.
 *
 * Key Features:
 * - Floor plan canvas: Interactive image-based coordinate system for sample placement
 * - Floor plan upload: Users can upload custom floor plan images
 * - Passive sampling: Record RSSI (signal strength) measurements from WiFi beacons
 * - Active sampling: Measure latency and packet loss to target server
 * - Throughput sampling: Use iperf3 server for bandwidth measurements
 * - Heatmap visualization: Color-coded overlays showing signal/performance gradients
 * - Sample management: Create, update, delete sample points
 * - Export data: Save survey results for analysis
 * - Real-time updates: Reflect changes immediately in UI
 *
 * Usage:
 * ```typescript
 * <SurveyView
 *   survey={surveyData}
 *   onClose={handleClose}
 *   onUpdate={handleUpdate}
 * />
 * ```
 *
 * Dependencies: FloorPlanCanvas, useAuth, useSurvey hook, API communication
 * State: survey data, sampling status, heatmap metric selection, upload progress
 */

import { useState, useEffect, useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { FloorPlanCanvas, type CalibrationPoint } from "./FloorPlanCanvas";
import { logger, LogComponents } from "../../lib/logger";
import { ScaleCalibrationPanel } from "./ScaleCalibrationPanel";
import { SurveyConfigPanel } from "./SurveyConfigPanel";
import { AirMapperImport, type ImportOptions } from "./AirMapperImport";
import { HeatmapLegend } from "./HeatmapLegend";
import type { AirMapperData } from "../../utils/airmapper";
// Fix #669: Removed deprecated getAuthHeaders - using credentials: 'include' for cookie auth
import { useSettings } from "../../contexts/useSettings";
import type {
  Survey,
  PassiveSample,
  ActiveSample,
  ThroughputSample,
  FloorPlan,
  SurveyConfig,
  SurveyType,
  HeatmapMetric,
  SamplePoint,
} from "../../hooks/useSurvey";
import { X, Upload, Play, Pause, CheckCircle, Loader } from "../ui/Icons";
// Import Ruler and FileArchive directly from lucide-react
import {
  Ruler,
  FileArchive,
  Wifi,
  Activity,
  Radio,
  Gauge,
  Clock,
  Waves,
  Hash,
} from "lucide-react";
import {
  radius,
  layout,
  spacing,
  button,
  icon as iconTokens,
} from "../../styles/theme";

const API_BASE = import.meta.env.VITE_API_BASE || "";

interface SurveyViewProps {
  survey: Survey;
  onClose: () => void;
  onUpdate: () => void;
}

/**
 * SurveyView Component
 * Main survey interface with floor plan, sampling controls, and heatmap visualization
 */
// WiFi adapter status from /api/wifi/status
interface WiFiStatus {
  status: "unavailable" | "available" | "ready";
  message: string;
  currentInterface: string;
  isWireless: boolean;
  availableAdapters: string[];
  canScan: boolean;
}

/**
 * SurveyView Component
 *
 * Main survey interface with floor plan, sampling controls, heatmap visualization, and legend.
 */
export function SurveyView({
  survey: initialSurvey,
  onClose,
  onUpdate,
}: SurveyViewProps) {
  const { t } = useTranslation("survey");
  const { displayOptions } = useSettings();
  // Fixes #733: Use user's unit preference for calibration
  const useSAE = displayOptions.unitSystem === "sae";
  // Current survey being edited
  const [survey, setSurvey] = useState(initialSurvey);

  // Get active floor (for multi-floor support)
  const getActiveFloor = useCallback(() => {
    if (!survey.floors || survey.floors.length === 0) {
      return null;
    }
    if (survey.activeFloorId) {
      const floor = survey.floors.find((f) => f.id === survey.activeFloorId);
      if (floor) return floor;
    }
    return survey.floors[0];
  }, [survey.floors, survey.activeFloorId]);

  // Get the current floor plan (from active floor or legacy field)
  const currentFloorPlan = getActiveFloor()?.floorPlan ?? survey.floorPlan;

  // Get samples for current floor (from active floor or legacy field)
  const currentSamples = useMemo(
    () => getActiveFloor()?.samples ?? survey.samples ?? [],
    [getActiveFloor, survey.samples]
  );
  // Indicates if a sampling operation is in progress
  const [sampling, setSampling] = useState(false);
  // Indicates if floor plan upload is in progress
  const [uploadingFloorPlan, setUploadingFloorPlan] = useState(false);
  // Selected metric for heatmap visualization
  const [heatmapMetric, setHeatmapMetric] = useState<HeatmapMetric>(null);
  const [error, setError] = useState<string | null>(null);
  // WiFi adapter status
  const [wifiStatus, setWifiStatus] = useState<WiFiStatus | null>(null);
  // Calibration mode state
  const [calibrationMode, setCalibrationMode] = useState(false);
  const [calibrationPoints, setCalibrationPoints] = useState<
    CalibrationPoint[]
  >([]);
  const [calibrationDistance, setCalibrationDistance] = useState<string>("");
  // Survey settings edit state
  const [editSurveyType, setEditSurveyType] = useState(
    initialSurvey.surveyType
  );
  const [editIperfServer, setEditIperfServer] = useState(
    initialSurvey.iperfServer || ""
  );
  const [editTestDuration, setEditTestDuration] = useState(
    initialSurvey.testDuration || 3
  );
  const [savingSettings, setSavingSettings] = useState(false);
  // AirMapper import state
  const [showImport, setShowImport] = useState(false);
  // Setup progress helpers
  const hasFloorPlan = !!currentFloorPlan;
  const hasCalibration =
    hasFloorPlan &&
    currentFloorPlan?.scaleM > 0 &&
    currentFloorPlan?.scaleSource &&
    currentFloorPlan?.scaleSource !== "default";
  const interfaceReady = wifiStatus?.canScan === true;
  const configReady =
    hasFloorPlan &&
    ((survey.config?.bands && survey.config.bands.length > 0) ||
      (survey.config?.adapters && survey.config.adapters.length > 0));
  const setupSteps = [
    { key: "floorPlan", label: t("setup.uploadFloorPlan"), done: hasFloorPlan },
    {
      key: "calibration",
      label: t("setup.calibrateScale"),
      done: hasCalibration,
    },
    { key: "wifi", label: t("setup.wifiInterface"), done: interfaceReady },
    { key: "config", label: t("setup.configureSurvey"), done: configReady },
    {
      key: "start",
      label: t("setup.readyToStart"),
      done: survey.status !== "created",
    },
  ];
  const missingSetupSteps = setupSteps.filter((s) => !s.done);
  const readyToStart = missingSetupSteps.length === 0;
  const completedSetupSteps = setupSteps.filter((s) => s.done).length;

  // Check WiFi adapter status on mount
  useEffect(() => {
    const checkWifiStatus = async () => {
      try {
        const res = await fetch(`${API_BASE}/api/wifi/status`, {
          credentials: "include",
        });
        if (res.ok) {
          const status = await res.json();
          setWifiStatus(status);
        }
      } catch (err) {
        logger.error(LogComponents.WIFI, "Failed to check WiFi status", err);
      }
    };
    checkWifiStatus();
  }, []);

  // Poll for survey updates when in progress
  useEffect(() => {
    if (survey.status !== "in_progress") return;

    const interval = setInterval(async () => {
      try {
        const res = await fetch(`${API_BASE}/api/survey?id=${survey.id}`, {
          credentials: "include",
        });
        if (res.ok) {
          const updated = await res.json();
          setSurvey(updated);
        }
      } catch (err) {
        logger.error(LogComponents.SURVEY, "Failed to refresh survey", err);
      }
    }, 3000);

    return () => clearInterval(interval);
  }, [survey.id, survey.status]);

  // Handle floor plan upload
  const handleFloorPlanUpload = useCallback(
    async (file: File) => {
      setUploadingFloorPlan(true);
      setError(null);

      try {
        // Read file as base64 using Promise wrapper
        const imageData = await new Promise<string>((resolve, reject) => {
          const reader = new FileReader();
          reader.onload = (e) => {
            const result = e.target?.result;
            if (typeof result === "string") {
              resolve(result);
            } else {
              reject(new Error("Failed to read file as base64"));
            }
          };
          reader.onerror = () => {
            reject(new Error("Failed to read file"));
          };
          reader.readAsDataURL(file);
        });

        // Get image dimensions using Promise wrapper
        const { width, height } = await new Promise<{
          width: number;
          height: number;
        }>((resolve, reject) => {
          const img = new Image();
          img.onload = () => {
            resolve({ width: img.width, height: img.height });
          };
          img.onerror = () => {
            reject(new Error("Failed to load image - file may be corrupted"));
          };
          img.src = imageData;
        });

        const floorPlan = {
          imageData,
          width,
          height,
          scaleM: 0.1, // Default: 10cm per pixel (adjust in settings)
        };

        // Upload to server
        const res = await fetch(
          `${API_BASE}/api/survey/floorplan?id=${survey.id}`,
          {
            method: "POST",
            headers: {
              "Content-Type": "application/json",
            },
            credentials: "include",
            body: JSON.stringify(floorPlan),
          }
        );

        if (!res.ok) {
          const errorText = await res.text();
          throw new Error(errorText || "Failed to upload floor plan");
        }

        // Refresh survey
        const updated = await res.json();
        setSurvey(updated);
        onUpdate();
      } catch (err) {
        setError(
          err instanceof Error ? err.message : "Failed to upload floor plan"
        );
      } finally {
        setUploadingFloorPlan(false);
      }
    },
    [survey.id, onUpdate]
  );

  // Handle taking a sample at clicked location
  const handlePointClick = useCallback(
    async (x: number, y: number) => {
      if (survey.status !== "in_progress") return;

      // Check WiFi availability before sampling
      if (!wifiStatus?.canScan) {
        setError(
          wifiStatus?.message || "No WiFi adapter available for scanning"
        );
        return;
      }

      setSampling(true);
      setError(null);

      try {
        // Collect sample data based on survey type
        let sampleData: PassiveSample | ActiveSample | ThroughputSample;

        switch (survey.surveyType) {
          case "passive": {
            // Fetch WiFi scan
            const scanRes = await fetch(`${API_BASE}/api/wifi/scan`, {
              credentials: "include",
            });
            if (!scanRes.ok) throw new Error("WiFi scan failed");
            const scanData = await scanRes.json();
            // Check if scan was successful
            if (!scanData.available) {
              throw new Error(scanData.error || "WiFi scan not available");
            }
            sampleData = { networks: scanData.networks || [] };
            break;
          }

          case "active": {
            // Fetch current WiFi status
            const wifiRes = await fetch(`${API_BASE}/api/wifi`, {
              credentials: "include",
            });
            if (!wifiRes.ok) throw new Error("WiFi status fetch failed");
            const wifiData = await wifiRes.json();

            // Check if BSSID changed (roaming)
            const lastSample = currentSamples[currentSamples.length - 1];
            const lastBssid = lastSample
              ? (lastSample.sampleData as ActiveSample).bssid
              : null;
            const roamingEvent =
              lastBssid !== null && lastBssid !== wifiData.bssid;

            sampleData = {
              ssid: wifiData.ssid || "",
              bssid: wifiData.bssid || "",
              rssi: wifiData.signal || 0,
              dataRate: wifiData.bitrate || 0,
              roamingEvent,
            };
            break;
          }

          case "throughput": {
            // Fetch WiFi status first
            const wifiRes2 = await fetch(`${API_BASE}/api/wifi`, {
              credentials: "include",
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
                "Content-Type": "application/json",
              },
              credentials: "include",
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
          }

          default:
            throw new Error("Unknown survey type");
        }

        // Submit sample to server
        const res = await fetch(
          `${API_BASE}/api/survey/sample?id=${survey.id}`,
          {
            method: "POST",
            headers: {
              "Content-Type": "application/json",
            },
            credentials: "include",
            body: JSON.stringify({ x, y, sampleData }),
          }
        );

        if (!res.ok) throw new Error("Failed to save sample");

        // Refresh survey to get updated samples
        const refreshRes = await fetch(
          `${API_BASE}/api/survey?id=${survey.id}`,
          {
            credentials: "include",
          }
        );
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
    },
    [survey, onUpdate, wifiStatus, currentSamples]
  );

  // Handle calibration click - collect two points
  const handleCalibrationClick = useCallback((x: number, y: number) => {
    setCalibrationPoints((prev) => {
      if (prev.length >= 2) {
        // Reset if we already have 2 points
        return [{ x, y }];
      }
      return [...prev, { x, y }];
    });
  }, []);

  // Calculate and save scale from calibration
  const handleSaveCalibration = async () => {
    if (calibrationPoints.length !== 2 || !calibrationDistance) {
      setError("Please select two points and enter the distance");
      return;
    }

    const rawDistance = parseFloat(calibrationDistance);
    if (isNaN(rawDistance) || rawDistance <= 0) {
      setError("Please enter a valid positive distance");
      return;
    }

    // Fixes #733: Convert feet to meters if using SAE units
    const distance = useSAE ? rawDistance * 0.3048 : rawDistance;

    // Calculate pixel distance
    const p1 = calibrationPoints[0];
    const p2 = calibrationPoints[1];
    const pixelDist = Math.sqrt((p2.x - p1.x) ** 2 + (p2.y - p1.y) ** 2);

    if (pixelDist === 0) {
      setError("Please select two different points");
      return;
    }

    // Calculate scale (meters per pixel)
    const scaleM = distance / pixelDist;

    try {
      // Update floor plan scale on server
      const res = await fetch(
        `${API_BASE}/api/survey/floorplan?id=${survey.id}`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          credentials: "include",
          body: JSON.stringify({
            ...currentFloorPlan,
            scaleM,
          }),
        }
      );

      if (!res.ok) {
        throw new Error("Failed to update floor plan scale");
      }

      const updated = await res.json();
      setSurvey(updated);
      onUpdate();

      // Exit calibration mode
      setCalibrationMode(false);
      setCalibrationPoints([]);
      setCalibrationDistance("");
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save scale");
    }
  };

  // Cancel calibration
  const handleCancelCalibration = () => {
    setCalibrationMode(false);
    setCalibrationPoints([]);
    setCalibrationDistance("");
  };

  // Handle floor plan scale/propagation updates from ScaleCalibrationPanel
  const handleFloorPlanUpdate = async (updates: Partial<FloorPlan>) => {
    if (!currentFloorPlan) return;

    try {
      const updatedFloorPlan = {
        ...currentFloorPlan,
        ...updates,
      };

      const res = await fetch(
        `${API_BASE}/api/survey/floorplan?id=${survey.id}`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          credentials: "include",
          body: JSON.stringify(updatedFloorPlan),
        }
      );

      if (!res.ok) {
        throw new Error("Failed to update floor plan settings");
      }

      const updated = await res.json();
      setSurvey(updated);
      onUpdate();
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to update settings"
      );
    }
  };

  // Handle survey config updates from SurveyConfigPanel
  const handleConfigUpdate = async (configUpdates: Partial<SurveyConfig>) => {
    try {
      const updatedConfig = {
        ...(survey.config || {}),
        ...configUpdates,
      };

      const res = await fetch(`${API_BASE}/api/survey/config?id=${survey.id}`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify(updatedConfig),
      });

      if (!res.ok) {
        throw new Error("Failed to update survey config");
      }

      const updated = await res.json();
      setSurvey(updated);
      onUpdate();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update config");
    }
  };

  // Handle survey type change from SurveyConfigPanel
  const handleSurveyTypeChange = (newType: SurveyType) => {
    setEditSurveyType(newType);
    // Also update via settings endpoint
    handleSaveSettings();
  };

  // Handle iperf settings change from SurveyConfigPanel
  const handleIperfSettingsChange = (server: string, duration: number) => {
    setEditIperfServer(server);
    setEditTestDuration(duration);
  };

  // Handle AirMapper import
  const handleAirMapperImport = async (
    data: AirMapperData,
    options: ImportOptions
  ) => {
    try {
      // Build floor plan from imported data
      if (options.importFloorPlan && data.floorPlanImage) {
        // Get image dimensions from the data URL
        const img = new Image();
        await new Promise<void>((resolve, reject) => {
          img.onload = () => resolve();
          img.onerror = () =>
            reject(new Error("Failed to load imported image"));
          img.src = data.floorPlanImage;
        });

        const floorPlan: FloorPlan = {
          imageData: data.floorPlanImage,
          width: img.width,
          height: img.height,
          scaleM: options.importCalibration ? data.calibration.scaleM : 0.1,
          scaleSource: options.importCalibration ? "imported" : "default",
          propagationM: options.importCalibration
            ? data.calibration.propagationM
            : 10,
          originalFile: data.floorPlanFilename,
        };

        // Upload floor plan to server
        const res = await fetch(
          `${API_BASE}/api/survey/floorplan?id=${survey.id}`,
          {
            method: "POST",
            headers: {
              "Content-Type": "application/json",
            },
            credentials: "include",
            body: JSON.stringify(floorPlan),
          }
        );

        if (!res.ok) {
          throw new Error("Failed to import floor plan");
        }

        const updated = await res.json();
        setSurvey(updated);
        onUpdate();
      } else if (options.importCalibration && currentFloorPlan) {
        // Just import calibration settings
        await handleFloorPlanUpdate({
          scaleM: data.calibration.scaleM,
          scaleSource: "imported",
          propagationM: data.calibration.propagationM,
        });
      }

      setShowImport(false);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to import AirMapper data"
      );
    }
  };

  // Save survey settings
  const handleSaveSettings = async () => {
    setSavingSettings(true);
    setError(null);

    try {
      const res = await fetch(
        `${API_BASE}/api/survey/settings?id=${survey.id}`,
        {
          method: "PUT",
          headers: {
            "Content-Type": "application/json",
          },
          credentials: "include",
          body: JSON.stringify({
            surveyType: editSurveyType,
            iperfServer: editIperfServer,
            testDuration: editTestDuration,
          }),
        }
      );

      if (!res.ok) {
        const errorText = await res.text();
        throw new Error(errorText || "Failed to save settings");
      }

      const updated = await res.json();
      setSurvey(updated);
      onUpdate();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save settings");
    } finally {
      setSavingSettings(false);
    }
  };

  // Handle status changes
  const handleStatusChange = async (action: "start" | "pause" | "complete") => {
    try {
      const res = await fetch(
        `${API_BASE}/api/survey/${action}?id=${survey.id}`,
        {
          method: "POST",
          credentials: "include",
        }
      );

      if (res.ok) {
        const updated = await res.json();
        setSurvey(updated);
        onUpdate();
      }
    } catch (err) {
      setError(
        err instanceof Error ? err.message : `Failed to ${action} survey`
      );
    }
  };

  return (
    <div className="fixed inset-0 bg-surface-base z-50 overflow-auto">
      {/* Header */}
      <div className="sticky top-0 bg-surface-raised border-b border-surface-border z-10">
        <div className={`max-w-7xl mx-auto pad ${layout.flex.between}`}>
          <div>
            <h1 className="heading-1">{survey.name}</h1>
            <p className={`body-small ${spacing.margin.top.tight}`}>
              {(survey.surveyType ?? "wifi").charAt(0).toUpperCase() +
                (survey.surveyType ?? "wifi").slice(1)}{" "}
              {t("status.survey")} • {currentSamples.length}{" "}
              {t("status.samples")} • {survey.status ?? "unknown"}
            </p>
          </div>

          <div className={`${layout.inline.default}`}>
            {/* Status controls */}
            {survey.status === "created" && (
              <button
                onClick={() => handleStatusChange("start")}
                disabled={!wifiStatus?.canScan || !readyToStart}
                title={
                  !wifiStatus?.canScan
                    ? t("wifi.requiredToStart")
                    : !readyToStart
                      ? `${t("setup.readyToStart")}: ${missingSetupSteps
                          .map((s) => s.label)
                          .join(", ")}`
                      : undefined
                }
                className={`${button.size.md} bg-brand-primary text-text-inverse ${radius.md} hover:bg-brand-primary/90 ${layout.inline.default} disabled:opacity-50 disabled:cursor-not-allowed`}
              >
                <Play className={iconTokens.size.sm} />
                {t("buttons.startSurvey")}
              </button>
            )}

            {survey.status === "in_progress" && (
              <>
                <button
                  onClick={() => handleStatusChange("pause")}
                  className={`${button.size.md} border border-surface-border ${radius.md} hover:bg-surface-hover ${layout.inline.default}`}
                >
                  <Pause className={iconTokens.size.sm} />
                  {t("buttons.pause")}
                </button>
                <button
                  onClick={() => handleStatusChange("complete")}
                  className={`${button.size.md} bg-status-success text-text-inverse ${radius.md} hover:bg-status-success/90 ${layout.inline.default}`}
                >
                  <CheckCircle className={iconTokens.size.sm} />
                  {t("buttons.complete")}
                </button>
              </>
            )}

            {survey.status === "paused" && (
              <>
                <button
                  onClick={() => handleStatusChange("start")}
                  disabled={!wifiStatus?.canScan || !readyToStart}
                  title={
                    !wifiStatus?.canScan
                      ? t("wifi.requiredToStart")
                      : !readyToStart
                        ? `${t("setup.readyToStart")}: ${missingSetupSteps
                            .map((s) => s.label)
                            .join(", ")}`
                        : undefined
                  }
                  className={`${button.size.md} bg-brand-primary text-text-inverse ${radius.md} hover:bg-brand-primary/90 ${layout.inline.default} disabled:opacity-50 disabled:cursor-not-allowed`}
                >
                  <Play className={iconTokens.size.sm} />
                  {t("buttons.resume")}
                </button>
                <button
                  onClick={() => handleStatusChange("complete")}
                  className={`${button.size.md} bg-status-success text-text-inverse ${radius.md} hover:bg-status-success/90 ${layout.inline.default}`}
                >
                  <CheckCircle className={iconTokens.size.sm} />
                  {t("buttons.complete")}
                </button>
              </>
            )}

            <button
              onClick={onClose}
              className={`${button.size.md} border border-surface-border ${radius.md} hover:bg-surface-hover ${layout.inline.default}`}
            >
              <X className={iconTokens.size.sm} />
              {t("buttons.close")}
            </button>
          </div>
        </div>
      </div>

      {/* Main content */}
      <div
        className={`max-w-7xl mx-auto ${spacing.pad.default} ${spacing.pad.lg}`}
      >
        {error && (
          <div
            className={`bg-status-error/10 border border-status-error/20 text-status-error ${spacing.pad.sm} ${radius.md} ${spacing.margin.bottom.content}`}
          >
            {error}
          </div>
        )}

        {!readyToStart && (
          <div
            className={`bg-status-warning/10 border border-status-warning/30 text-status-warning ${spacing.pad.sm} ${radius.md} ${spacing.margin.bottom.content}`}
          >
            {t("setup.readyToStart")}:{" "}
            {missingSetupSteps.map((s) => s.label).join(", ")}
          </div>
        )}

        {/* AirMapper Import Modal */}
        {showImport && (
          <div className={`${spacing.margin.bottom.content}`}>
            <AirMapperImport
              onImport={handleAirMapperImport}
              onCancel={() => setShowImport(false)}
            />
          </div>
        )}

        {/* WiFi adapter status banner */}
        {wifiStatus && wifiStatus.status !== "ready" && (
          <div
            className={`${
              wifiStatus.status === "unavailable"
                ? "bg-status-info/10 border-status-info/20 text-status-info"
                : "bg-status-info/10 border-status-info/20 text-status-info"
            } border ${spacing.pad.sm} ${radius.md} ${spacing.margin.bottom.content}`}
          >
            <div className="font-medium">
              {wifiStatus.status === "unavailable"
                ? t("wifi.noAdapterSetup")
                : t("wifi.adapterAvailable")}
            </div>
            {wifiStatus.availableAdapters.length > 0 && (
              <div className={`caption ${spacing.margin.top.tight}`}>
                {t("wifi.availableAdapters")}:{" "}
                {wifiStatus.availableAdapters.join(", ")}
              </div>
            )}
            <div className={`caption ${spacing.margin.top.tight}`}>
              {t("wifi.setupWithoutAdapter")}
            </div>
          </div>
        )}

        {sampling && (
          <div
            className={`bg-status-info/10 border border-status-info/20 text-status-info ${spacing.pad.sm} ${radius.md} ${spacing.margin.bottom.content} ${layout.inline.default}`}
          >
            <Loader className={`${iconTokens.size.sm} animate-spin`} />
            {t("progress.takingMeasurement")}
          </div>
        )}

        <div
          className={`grid grid-cols-1 lg:grid-cols-3 ${spacing.gap.spacious}`}
        >
          {/* Floor plan */}
          <div className="lg:col-span-2">
            <div
              className={`bg-surface-raised ${radius.md} border border-surface-border pad`}
            >
              <div
                className={`${layout.flex.between} ${spacing.margin.bottom.content}`}
              >
                <h2 className="heading-3">{t("floorPlan.title")}</h2>
                {heatmapMetric !== null && (
                  <button
                    onClick={() => setHeatmapMetric(null)}
                    className={`${button.size.sm} body-small bg-brand-primary text-text-inverse ${radius.md} hover:bg-brand-primary/90`}
                  >
                    {t("buttons.hideHeatmap")}
                  </button>
                )}
              </div>

              {/* Heatmap metric selector - categorized */}
              {heatmapMetric === null && currentSamples.length > 0 && (
                <div
                  className={`${spacing.margin.bottom.content} ${spacing.stack.sm}`}
                >
                  {/* Signal Category */}
                  <div>
                    <div
                      className={`body-small text-text-muted ${spacing.margin.bottom.tight}`}
                    >
                      {t("heatmaps.categories.signal")}
                    </div>
                    <div className={layout.inline.default}>
                      <button
                        onClick={() => setHeatmapMetric("rssi")}
                        className={`${button.size.sm} body-small border border-surface-border ${radius.md} hover:bg-surface-hover ${layout.inline.tight}`}
                      >
                        <Wifi className={iconTokens.size.sm} />
                        {t("heatmaps.rssi")}
                      </button>
                      <button
                        onClick={() => setHeatmapMetric("snr")}
                        className={`${button.size.sm} body-small border border-surface-border ${radius.md} hover:bg-surface-hover ${layout.inline.tight}`}
                      >
                        <Activity className={iconTokens.size.sm} />
                        {t("heatmaps.snr")}
                      </button>
                      <button
                        onClick={() => setHeatmapMetric("noise")}
                        className={`${button.size.sm} body-small border border-surface-border ${radius.md} hover:bg-surface-hover ${layout.inline.tight}`}
                      >
                        <Radio className={iconTokens.size.sm} />
                        {t("heatmaps.noise")}
                      </button>
                    </div>
                  </div>

                  {/* Interference Category */}
                  <div>
                    <div
                      className={`body-small text-text-muted ${spacing.margin.bottom.tight}`}
                    >
                      {t("heatmaps.categories.interference")}
                    </div>
                    <div className={layout.inline.default}>
                      <button
                        onClick={() => setHeatmapMetric("cochannel")}
                        className={`${button.size.sm} body-small border border-surface-border ${radius.md} hover:bg-surface-hover ${layout.inline.tight}`}
                      >
                        <Waves className={iconTokens.size.sm} />
                        {t("heatmaps.cochannel")}
                      </button>
                      <button
                        onClick={() => setHeatmapMetric("adjacent")}
                        className={`${button.size.sm} body-small border border-surface-border ${radius.md} hover:bg-surface-hover ${layout.inline.tight}`}
                      >
                        <Waves className={iconTokens.size.sm} />
                        {t("heatmaps.adjacent")}
                      </button>
                      <button
                        onClick={() => setHeatmapMetric("apDensity")}
                        className={`${button.size.sm} body-small border border-surface-border ${radius.md} hover:bg-surface-hover ${layout.inline.tight}`}
                      >
                        <Hash className={iconTokens.size.sm} />
                        {t("heatmaps.apDensity")}
                      </button>
                      <button
                        onClick={() => setHeatmapMetric("ssidCount")}
                        className={`${button.size.sm} body-small border border-surface-border ${radius.md} hover:bg-surface-hover ${layout.inline.tight}`}
                      >
                        <Hash className={iconTokens.size.sm} />
                        {t("heatmaps.ssidCount")}
                      </button>
                    </div>
                  </div>

                  {/* Performance Category - only for throughput surveys */}
                  {survey.surveyType === "throughput" && (
                    <div>
                      <div
                        className={`body-small text-text-muted ${spacing.margin.bottom.tight}`}
                      >
                        {t("heatmaps.categories.performance")}
                      </div>
                      <div className={layout.inline.default}>
                        <button
                          onClick={() => setHeatmapMetric("throughput")}
                          className={`${button.size.sm} body-small border border-surface-border ${radius.md} hover:bg-surface-hover ${layout.inline.tight}`}
                        >
                          <Gauge className={iconTokens.size.sm} />
                          {t("heatmaps.throughput")}
                        </button>
                        <button
                          onClick={() => setHeatmapMetric("latency")}
                          className={`${button.size.sm} body-small border border-surface-border ${radius.md} hover:bg-surface-hover ${layout.inline.tight}`}
                        >
                          <Clock className={iconTokens.size.sm} />
                          {t("heatmaps.latency")}
                        </button>
                      </div>
                    </div>
                  )}
                </div>
              )}

              {!currentFloorPlan ? (
                <div
                  className={`border-2 border-dashed border-surface-border ${radius.md} pad-lg text-center`}
                >
                  <Upload
                    className={`${iconTokens.size.xl} mx-auto text-text-muted ${spacing.margin.bottom.content}`}
                  />
                  <p
                    className={`text-text-muted ${spacing.margin.bottom.content}`}
                  >
                    {t("floorPlan.uploadPrompt")}
                  </p>
                  <label
                    className={`inline-block ${button.size.md} bg-brand-primary text-text-inverse ${radius.md} cursor-pointer hover:bg-brand-primary/90`}
                  >
                    {uploadingFloorPlan
                      ? t("floorPlan.uploading")
                      : t("floorPlan.chooseFile")}
                    <input
                      type="file"
                      accept="image/png,image/jpeg,image/gif,image/webp,image/svg+xml"
                      className="hidden"
                      onChange={(e) => {
                        const file = e.target.files?.[0];
                        if (file) {
                          handleFloorPlanUpload(file);
                        }
                        // Reset input so same file can be selected again if needed
                        e.target.value = "";
                      }}
                      disabled={uploadingFloorPlan}
                    />
                  </label>
                  <p
                    className={`caption text-text-muted ${spacing.margin.top.inline}`}
                  >
                    {t("floorPlan.supportedFormats")}
                  </p>
                  <div
                    className={`${spacing.margin.top.content} border-t border-surface-border ${spacing.padding.top}`}
                  >
                    <p
                      className={`caption text-text-muted ${spacing.margin.bottom.inline}`}
                    >
                      {t("import.description")}
                    </p>
                    <button
                      onClick={() => setShowImport(true)}
                      className={`${button.size.sm} border border-surface-border ${radius.md} hover:bg-surface-hover ${layout.inline.default}`}
                    >
                      <FileArchive className={iconTokens.size.sm} />
                      {t("import.button")}
                    </button>
                  </div>
                </div>
              ) : (
                <div>
                  {/* Calibration panel */}
                  {calibrationMode && (
                    <div
                      className={`bg-status-warning/10 border border-status-warning/20 ${spacing.pad.sm} ${radius.md} ${spacing.margin.bottom.content}`}
                    >
                      <div
                        className={`font-medium text-status-warning ${spacing.margin.bottom.inline}`}
                      >
                        📐 {t("calibration.title")}
                      </div>
                      <p
                        className={`body-small text-text-secondary ${spacing.margin.bottom.content}`}
                      >
                        {t("calibration.instructions")}
                      </p>
                      <div className="stack-sm">
                        <div className={`${layout.inline.default}`}>
                          <span className="body-small text-text-muted w-20">
                            {t("calibration.pointA")}:
                          </span>
                          {calibrationPoints[0] ? (
                            <span className="body-small font-medium">
                              ({calibrationPoints[0].x},{" "}
                              {calibrationPoints[0].y})
                            </span>
                          ) : (
                            <span className="body-small text-text-muted italic">
                              {t("calibration.clickFloorPlan")}
                            </span>
                          )}
                        </div>
                        <div className={`${layout.inline.default}`}>
                          <span className="body-small text-text-muted w-20">
                            {t("calibration.pointB")}:
                          </span>
                          {calibrationPoints[1] ? (
                            <span className="body-small font-medium">
                              ({calibrationPoints[1].x},{" "}
                              {calibrationPoints[1].y})
                            </span>
                          ) : (
                            <span className="body-small text-text-muted italic">
                              {t("calibration.clickFloorPlan")}
                            </span>
                          )}
                        </div>
                        {calibrationPoints.length === 2 && (
                          <div className={`${layout.inline.default}`}>
                            <span className="body-small text-text-muted w-20">
                              {t("calibration.pixelDistance")}:
                            </span>
                            <span className="body-small font-medium">
                              {Math.sqrt(
                                (calibrationPoints[1].x -
                                  calibrationPoints[0].x) **
                                  2 +
                                  (calibrationPoints[1].y -
                                    calibrationPoints[0].y) **
                                    2
                              ).toFixed(0)}{" "}
                              px
                            </span>
                          </div>
                        )}
                        <div
                          className={`${layout.inline.default} ${spacing.margin.top.inline}`}
                        >
                          <label className="body-small text-text-muted w-20">
                            {t("calibration.distance")}:
                          </label>
                          <input
                            type="number"
                            step="0.1"
                            min="0"
                            value={calibrationDistance}
                            onChange={(e) =>
                              setCalibrationDistance(e.target.value)
                            }
                            placeholder={
                              useSAE
                                ? t("calibration.enterFeet")
                                : t("calibration.enterMeters")
                            }
                            className={`flex-1 ${button.size.sm} border border-surface-border ${radius.md} bg-surface-base text-text-primary`}
                          />
                          <span className="body-small text-text-muted">
                            {useSAE
                              ? t("calibration.feet")
                              : t("calibration.meters")}
                          </span>
                        </div>
                        <div
                          className={`${layout.inline.default} ${spacing.margin.top.inline}`}
                        >
                          <button
                            onClick={handleSaveCalibration}
                            disabled={
                              calibrationPoints.length !== 2 ||
                              !calibrationDistance
                            }
                            className={`${button.size.sm} bg-brand-primary text-text-inverse ${radius.md} hover:bg-brand-primary/90 disabled:opacity-50 disabled:cursor-not-allowed`}
                          >
                            {t("buttons.saveScale")}
                          </button>
                          <button
                            onClick={handleCancelCalibration}
                            className={`${button.size.sm} border border-surface-border ${radius.md} hover:bg-surface-hover`}
                          >
                            {t("buttons.cancel")}
                          </button>
                          <button
                            onClick={() => setCalibrationPoints([])}
                            className={`${button.size.sm} border border-surface-border ${radius.md} hover:bg-surface-hover`}
                          >
                            {t("buttons.resetPoints")}
                          </button>
                        </div>
                      </div>
                    </div>
                  )}

                  {/* Calibrate button and current scale info */}
                  {!calibrationMode && currentFloorPlan && (
                    <div
                      className={`${layout.flex.between} ${spacing.margin.bottom.inline}`}
                    >
                      <div className="body-small text-text-muted">
                        {t("floorPlan.scale")}:{" "}
                        {currentFloorPlan.scaleM.toFixed(3)} m/px
                        {survey.status === "in_progress" &&
                          ` • ${t("floorPlan.clickToMeasure")}`}
                      </div>
                      <button
                        onClick={() => setCalibrationMode(true)}
                        className={`${button.size.sm} body-small border border-surface-border ${radius.md} hover:bg-surface-hover ${layout.inline.tight}`}
                      >
                        <Ruler className={iconTokens.size.sm} />
                        {t("buttons.calibrateScale")}
                      </button>
                    </div>
                  )}

                  <FloorPlanCanvas
                    floorPlan={currentFloorPlan!}
                    samples={currentSamples}
                    onPointClick={handlePointClick}
                    interactive={
                      survey.status === "in_progress" &&
                      !sampling &&
                      !calibrationMode &&
                      wifiStatus?.canScan === true
                    }
                    heatmapMetric={heatmapMetric}
                    calibrationMode={calibrationMode}
                    calibrationPoints={calibrationPoints}
                    onCalibrationClick={handleCalibrationClick}
                  />

                  {/* Heatmap Legend - show when heatmap is active */}
                  {heatmapMetric !== null && currentSamples.length > 0 && (
                    <div className={spacing.margin.top.content}>
                      <HeatmapLegend
                        metric={heatmapMetric}
                        minValue={
                          calculateMetricRange(currentSamples, heatmapMetric)
                            .min
                        }
                        maxValue={
                          calculateMetricRange(currentSamples, heatmapMetric)
                            .max
                        }
                      />
                    </div>
                  )}
                </div>
              )}
            </div>
          </div>

          {/* Settings panel (shown when survey is in created status) and Sample list */}
          <div className={`lg:col-span-1 ${spacing.stack.default}`}>
            {/* Setup checklist to guide users before starting a survey */}
            {survey.status === "created" && (
              <div
                className={`bg-surface-raised ${radius.md} border border-surface-border pad`}
              >
                <div
                  className={`${layout.flex.between} ${spacing.margin.bottom.inline}`}
                >
                  <h2 className="heading-3">{t("setup.checklist")}</h2>
                  <span className="caption text-text-muted">
                    {completedSetupSteps}/{setupSteps.length}
                  </span>
                </div>
                <div className="stack-sm">
                  {setupSteps.map((step) => (
                    <div
                      key={step.key}
                      className={`flex items-center justify-between ${spacing.pad.xs} ${radius.sm} ${
                        step.done ? "bg-surface-hover" : "bg-transparent"
                      }`}
                    >
                      <div className={layout.inline.default}>
                        {step.done ? (
                          <CheckCircle
                            className={`${iconTokens.size.sm} text-status-success`}
                          />
                        ) : (
                          <Clock
                            className={`${iconTokens.size.sm} text-text-muted`}
                          />
                        )}
                        <span className="body-small">{step.label}</span>
                      </div>
                      {step.done && (
                        <span className="caption text-status-success">✓</span>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Survey Settings Panel - only show when survey hasn't started */}
            {survey.status === "created" && (
              <div
                className={`bg-surface-raised ${radius.md} border border-surface-border pad`}
              >
                <h2 className={`heading-3 ${spacing.margin.bottom.content}`}>
                  {t("settings.title")}
                </h2>
                <div className="stack">
                  {/* Survey Type */}
                  <div>
                    <label
                      className={`body-small text-text-muted block ${spacing.margin.bottom.tight}`}
                    >
                      {t("settings.surveyType")}
                    </label>
                    <select
                      value={editSurveyType}
                      onChange={(e) =>
                        setEditSurveyType(
                          e.target.value as "passive" | "active" | "throughput"
                        )
                      }
                      className={`w-full ${button.size.md} border border-surface-border ${radius.md} bg-surface-base text-text-primary`}
                    >
                      <option value="passive">
                        {t("settings.types.passive")}
                      </option>
                      <option value="active">
                        {t("settings.types.active")}
                      </option>
                      <option value="throughput">
                        {t("settings.types.throughput")}
                      </option>
                    </select>
                  </div>

                  {/* iperf Server - only show for throughput surveys */}
                  {editSurveyType === "throughput" && (
                    <>
                      <div>
                        <label
                          className={`body-small text-text-muted block ${spacing.margin.bottom.tight}`}
                        >
                          {t("settings.iperfServer")}
                        </label>
                        <input
                          type="text"
                          value={editIperfServer}
                          onChange={(e) => setEditIperfServer(e.target.value)}
                          placeholder="hostname:5201"
                          className={`w-full ${button.size.md} border border-surface-border ${radius.md} bg-surface-base text-text-primary`}
                        />
                        <p
                          className={`caption text-text-muted ${spacing.margin.top.tight}`}
                        >
                          {t("settings.iperfServerHint")}
                        </p>
                      </div>

                      <div>
                        <label
                          className={`body-small text-text-muted block ${spacing.margin.bottom.tight}`}
                        >
                          {t("settings.testDuration")}
                        </label>
                        <input
                          type="number"
                          min="1"
                          max="60"
                          value={editTestDuration}
                          onChange={(e) =>
                            setEditTestDuration(parseInt(e.target.value) || 3)
                          }
                          className={`w-full ${button.size.md} border border-surface-border ${radius.md} bg-surface-base text-text-primary`}
                        />
                      </div>
                    </>
                  )}

                  {/* Save button */}
                  <button
                    onClick={handleSaveSettings}
                    disabled={savingSettings}
                    className={`w-full ${button.size.md} bg-brand-primary text-text-inverse ${radius.md} hover:bg-brand-primary/90 disabled:opacity-50`}
                  >
                    {savingSettings
                      ? t("buttons.saving")
                      : t("buttons.saveSettings")}
                  </button>

                  {/* Survey type descriptions */}
                  <div
                    className={`caption text-text-muted border-t border-surface-border ${spacing.padding.top.section} ${spacing.margin.top.inline}`}
                  >
                    <p
                      className={`font-medium ${spacing.margin.bottom.inline}`}
                    >
                      {t("settings.typesDescription")}
                    </p>
                    <ul className={`list-disc list-inside ${spacing.stack.xs}`}>
                      <li>
                        <strong>Passive:</strong> {t("settings.passiveDesc")}
                      </li>
                      <li>
                        <strong>Active:</strong> {t("settings.activeDesc")}
                      </li>
                      <li>
                        <strong>Throughput:</strong>{" "}
                        {t("settings.throughputDesc")}
                      </li>
                    </ul>
                  </div>
                </div>
              </div>
            )}

            {/* Scale Calibration Panel - show when floor plan exists */}
            {currentFloorPlan && (
              <ScaleCalibrationPanel
                floorPlan={currentFloorPlan}
                onUpdate={handleFloorPlanUpdate}
                onStartCalibration={() => setCalibrationMode(true)}
                isCalibrating={calibrationMode}
              />
            )}

            {/* Survey Configuration Panel - show when floor plan exists */}
            {currentFloorPlan && wifiStatus && (
              <SurveyConfigPanel
                config={survey.config}
                surveyType={editSurveyType}
                availableAdapters={wifiStatus.availableAdapters || []}
                currentInterface={
                  wifiStatus.currentInterface || survey.interface
                }
                iperfServer={editIperfServer}
                testDuration={editTestDuration}
                onUpdate={handleConfigUpdate}
                onSurveyTypeChange={handleSurveyTypeChange}
                onIperfSettingsChange={handleIperfSettingsChange}
              />
            )}

            {/* Sample list */}
            <div
              className={`bg-surface-raised ${radius.md} border border-surface-border pad`}
            >
              <h2 className={`heading-3 ${spacing.margin.bottom.content}`}>
                {t("samples.title")} ({currentSamples.length})
              </h2>
              <div className="stack-sm max-h-[70vh] overflow-y-auto">
                {currentSamples.length === 0 ? (
                  <p className={`body-small text-center ${spacing.pad.lg}`}>
                    {t("samples.noSamples")}{" "}
                    {survey.status === "in_progress"
                      ? t("samples.clickToStart")
                      : t("samples.startToBegin")}
                  </p>
                ) : (
                  currentSamples.map((sample, idx) => (
                    <div
                      key={idx}
                      className={`border border-surface-border ${radius.md} pad-sm body-small`}
                    >
                      <div
                        className={`flex items-center justify-between ${spacing.margin.bottom.inline}`}
                      >
                        <span className="font-semibold">#{idx + 1}</span>
                        <span className="caption">
                          {new Date(sample.timestamp).toLocaleTimeString()}
                        </span>
                      </div>
                      <div className="caption stack-xs">
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
function renderSampleData(
  data: PassiveSample | ActiveSample | ThroughputSample,
  surveyType: string
) {
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
          <div className="text-status-warning font-semibold">⚠ Roaming</div>
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
          <div className="text-status-error">
            Loss: {throughputData.packetLoss.toFixed(1)}%
          </div>
        )}
      </>
    );
  }

  return null;
}

// Helper to calculate min/max values for a heatmap metric
function calculateMetricRange(
  samples: SamplePoint[],
  metric: HeatmapMetric
): { min: number; max: number } {
  if (!metric || samples.length === 0) {
    return { min: 0, max: 0 };
  }

  const values: number[] = [];

  samples.forEach((sample) => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any -- Polymorphic sample data
    const data = sample.sampleData as any;

    switch (metric) {
      case "rssi":
        if (data.networks && Array.isArray(data.networks)) {
          const rssiValues = data.networks.map((n: { rssi: number }) => n.rssi);
          if (rssiValues.length > 0) {
            values.push(Math.max(...rssiValues));
          }
        } else if (data.rssi !== undefined) {
          values.push(data.rssi);
        }
        break;
      case "snr":
        if (data.networks && Array.isArray(data.networks)) {
          const rssiValues = data.networks.map((n: { rssi: number }) => n.rssi);
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
      case "noise":
        values.push(data.noiseFloor || -95);
        break;
      case "cochannel":
        if (
          data.networks &&
          Array.isArray(data.networks) &&
          data.networks.length > 0
        ) {
          const primaryChannel = data.networks[0].channel;
          const count =
            data.networks.filter(
              (n: { channel: number }) => n.channel === primaryChannel
            ).length - 1;
          values.push(count);
        }
        break;
      case "adjacent":
        if (
          data.networks &&
          Array.isArray(data.networks) &&
          data.networks.length > 0
        ) {
          const primaryChannel = data.networks[0].channel;
          const count = data.networks.filter(
            (n: { channel: number }) =>
              Math.abs(n.channel - primaryChannel) > 0 &&
              Math.abs(n.channel - primaryChannel) <= 2
          ).length;
          values.push(count);
        }
        break;
      case "throughput":
        if (data.downloadMbps !== undefined) {
          values.push(data.downloadMbps);
        }
        break;
      case "latency":
        if (data.latency !== undefined) {
          values.push(data.latency);
        }
        break;
      case "channelUtil":
        if (data.channelUtilization !== undefined) {
          values.push(data.channelUtilization);
        }
        break;
      case "apDensity":
        if (data.networks && Array.isArray(data.networks)) {
          const uniqueBSSIDs = new Set(
            data.networks.map((n: { bssid: string }) => n.bssid)
          );
          values.push(uniqueBSSIDs.size);
        } else if (data.uniqueBSSIDs !== undefined) {
          values.push(data.uniqueBSSIDs);
        }
        break;
      case "ssidCount":
        if (data.networks && Array.isArray(data.networks)) {
          const uniqueSSIDs = new Set(
            data.networks.map((n: { ssid: string }) => n.ssid).filter(Boolean)
          );
          values.push(uniqueSSIDs.size);
        } else if (data.uniqueSSIDs !== undefined) {
          values.push(data.uniqueSSIDs);
        }
        break;
    }
  });

  if (values.length === 0) {
    return { min: 0, max: 0 };
  }

  return {
    min: Math.min(...values),
    max: Math.max(...values),
  };
}
