// biome-ignore-all lint/complexity/noExcessiveCognitiveComplexity: Complex component
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

// Import Ruler and FileArchive directly from lucide-react
import { Activity, Clock, FileArchive, Gauge, Hash, Radio, Ruler, Waves, Wifi } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useSettings } from "../../contexts/useSettings";
import type {
  ActiveSample,
  FloorPlan,
  HeatmapMetric,
  PassiveSample,
  SamplePoint,
  Survey,
  SurveyConfig,
  SurveyType,
  ThroughputSample,
} from "../../hooks/useSurvey";
import { LogComponents, logger } from "../../lib/logger";
import { button, cn, icon as iconTokens, layout, radius, spacing } from "../../styles/theme";
import type { AirMapperData } from "../../utils/airmapper";
import { CheckCircle, Loader, Pause, Play, Upload, X } from "../ui/Icons";
import { AirMapperImport, type ImportOptions } from "./AirMapperImport";
import { type CalibrationPoint, FloorPlanCanvas } from "./FloorPlanCanvas";
import { HeatmapLegend } from "./HeatmapLegend";
import { HeatmapStats } from "./HeatmapStats";
import { ScaleCalibrationPanel } from "./ScaleCalibrationPanel";
import { SurveyConfigPanel } from "./SurveyConfigPanel";

const API_BASE: string = import.meta.env.VITE_API_BASE || "";

interface SurveyViewProps {
  survey: Survey;
  onClose: () => void;
  onUpdate: () => void;
}

/**
 * SurveyView Component
 * Main survey interface with floor plan, sampling controls, and heatmap visualization
 */
// WiFi adapter status from /api/canopy/wifi/status
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
}: SurveyViewProps): React.JSX.Element {
  const { t } = useTranslation("survey");
  const { displayOptions } = useSettings();
  // Fixes #733: Use user's unit preference for calibration
  const useSae = displayOptions.unitSystem === "sae";
  // Current survey being edited
  const [survey, setSurvey] = useState(initialSurvey);

  // Get active floor (for multi-floor support)
  const getActiveFloor = useCallback(() => {
    if (!survey.floors || survey.floors.length === 0) {
      return null;
    }
    if (survey.activeFloorId) {
      const floor = survey.floors.find((f) => f.id === survey.activeFloorId);
      if (floor) {
        return floor;
      }
    }
    return survey.floors[0];
  }, [survey.floors, survey.activeFloorId]);

  // Get the current floor plan (from active floor or legacy field)
  const currentFloorPlan = getActiveFloor()?.floorPlan ?? survey.floorPlan;

  // Get samples for current floor (from active floor or legacy field)
  const currentSamples = useMemo(
    () => getActiveFloor()?.samples ?? survey.samples ?? [],
    [getActiveFloor, survey.samples],
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
  const [calibrationPoints, setCalibrationPoints] = useState<CalibrationPoint[]>([]);
  const [calibrationDistance, setCalibrationDistance] = useState<string>("");
  // Survey settings edit state
  const [editSurveyType, setEditSurveyType] = useState(initialSurvey.surveyType);
  const [editIperfServer, setEditIperfServer] = useState(initialSurvey.iperfServer || "");
  const [editTestDuration, setEditTestDuration] = useState(initialSurvey.testDuration || 3);
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
    const checkWifiStatus = async (): Promise<void> => {
      try {
        const res: Response = await fetch(`${API_BASE}/api/canopy/wifi/status`, {
          credentials: "include",
        });
        if (res.ok) {
          const status: WiFiStatus = await (res.json() as Promise<WiFiStatus>);
          setWifiStatus(status);
        }
      } catch (err) {
        logger.error(LogComponents.Wifi, "Failed to check WiFi status", err);
      }
    };
    checkWifiStatus().catch((err: unknown) => {
      logger.error(LogComponents.Wifi, "Error checking WiFi status", err);
    });
  }, []);

  // Poll for survey updates when in progress
  useEffect(() => {
    if (survey.status !== "in_progress") {
      return;
    }

    const interval = setInterval(async () => {
      try {
        const res = await fetch(`${API_BASE}/api/canopy/survey?id=${survey.id}`, {
          credentials: "include",
        });
        if (res.ok) {
          const updated = await (res.json() as Promise<Survey>);
          setSurvey(updated);
        }
      } catch (err) {
        logger.error(LogComponents.Survey, "Failed to refresh survey", err);
      }
    }, 3000);

    return (): void => clearInterval(interval);
  }, [survey.id, survey.status]);

  // Handle floor plan upload
  const handleFloorPlanUpload = useCallback(
    async (file: File) => {
      setUploadingFloorPlan(true);
      setError(null);

      try {
        // Read file as base64 using Promise wrapper
        const imageData: string = await new Promise<string>((resolve, reject) => {
          const reader: FileReader = new FileReader();
          reader.onload = (e: ProgressEvent<FileReader>): void => {
            const result: string | ArrayBuffer | null | undefined = e.target?.result;
            if (typeof result === "string") {
              resolve(result);
            } else {
              reject(new Error("Failed to read file as base64"));
            }
          };
          reader.onerror = (): void => {
            reject(new Error("Failed to read file"));
          };
          reader.readAsDataURL(file);
        });

        // Get image dimensions using Promise wrapper
        const { width, height } = await new Promise<{
          width: number;
          height: number;
        }>((resolve, reject) => {
          const img: HTMLImageElement = new Image();
          img.onload = (): void => {
            resolve({ width: img.width, height: img.height });
          };
          img.onerror = (): void => {
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
        const res = await fetch(`${API_BASE}/api/canopy/survey/floorplan?id=${survey.id}`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          credentials: "include",
          body: JSON.stringify(floorPlan),
        });

        if (!res.ok) {
          const errorText = await (res.text() as Promise<string>);
          throw new Error(errorText || "Failed to upload floor plan");
        }

        // Refresh survey
        const updated = await (res.json() as Promise<Survey>);
        setSurvey(updated);
        onUpdate();
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to upload floor plan");
      } finally {
        setUploadingFloorPlan(false);
      }
    },
    [survey.id, onUpdate],
  );

  // Handle taking a sample at clicked location
  const handlePointClick = useCallback(
    async (x: number, y: number) => {
      if (survey.status !== "in_progress") {
        return;
      }

      // Check WiFi availability before sampling
      if (!wifiStatus?.canScan) {
        setError(wifiStatus?.message || "No WiFi adapter available for scanning");
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
            const scanRes = await fetch(`${API_BASE}/api/canopy/wifi/scan`, {
              credentials: "include",
            });
            if (!scanRes.ok) {
              throw new Error("WiFi scan failed");
            }
            const scanData = await (scanRes.json() as Promise<Record<string, unknown>>);
            // Check if scan was successful
            if (!scanData.available) {
              throw new Error(scanData.error || "WiFi scan not available");
            }
            sampleData = { networks: scanData.networks || [] };
            break;
          }

          case "active": {
            // Fetch current WiFi status
            const wifiRes = await fetch(`${API_BASE}/api/canopy/wifi`, {
              credentials: "include",
            });
            if (!wifiRes.ok) {
              throw new Error("WiFi status fetch failed");
            }
            const wifiData = await (wifiRes.json() as Promise<Record<string, unknown>>);

            // Check if BSSID changed (roaming)
            const lastSample = currentSamples.at(-1);
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
          }

          case "throughput": {
            // Fetch WiFi status first
            const wifiRes2 = await fetch(`${API_BASE}/api/canopy/wifi`, {
              credentials: "include",
            });
            if (!wifiRes2.ok) {
              throw new Error("WiFi status fetch failed");
            }
            const wifiData2 = await (wifiRes2.json() as Promise<Record<string, unknown>>);

            // Run iperf3 test
            if (!survey.iperfServer) {
              throw new Error("iperf3 server not configured for this survey");
            }

            const [host, port] = survey.iperfServer.split(":");
            const iperfRes = await fetch(`${API_BASE}/api/sap/iperf/client`, {
              method: "POST",
              headers: {
                "Content-Type": "application/json",
              },
              credentials: "include",
              body: JSON.stringify({
                host,
                port: port ? Number.parseInt(port, 10) : 5201,
                duration: survey.testDuration || 3,
                reverse: false,
              }),
            });

            if (!iperfRes.ok) {
              throw new Error("iperf3 test failed");
            }
            const iperfData = await (iperfRes.json() as Promise<Record<string, unknown>>);

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
        const res = await fetch(`${API_BASE}/api/canopy/survey/sample?id=${survey.id}`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          credentials: "include",
          body: JSON.stringify({ x, y, sampleData }),
        });

        if (!res.ok) {
          throw new Error("Failed to save sample");
        }

        // Refresh survey to get updated samples
        const refreshRes = await fetch(`${API_BASE}/api/canopy/survey?id=${survey.id}`, {
          credentials: "include",
        });
        if (refreshRes.ok) {
          const updated = await (refreshRes.json() as Promise<Survey>);
          setSurvey(updated);
          onUpdate();
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to take sample");
      } finally {
        setSampling(false);
      }
    },
    [survey, onUpdate, wifiStatus, currentSamples],
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
  const handleSaveCalibration = async (): Promise<void> => {
    if (calibrationPoints.length !== 2 || !calibrationDistance) {
      setError("Please select two points and enter the distance");
      return;
    }

    const rawDistance: number = Number.parseFloat(calibrationDistance);
    if (Number.isNaN(rawDistance) || rawDistance <= 0) {
      setError("Please enter a valid positive distance");
      return;
    }

    // Fixes #733: Convert feet to meters if using SAE units
    const distance: number = useSae ? rawDistance * 0.3048 : rawDistance;

    // Calculate pixel distance
    const [p1, p2] = calibrationPoints;
    const pixelDist = Math.sqrt((p2.x - p1.x) ** 2 + (p2.y - p1.y) ** 2);

    if (pixelDist === 0) {
      setError("Please select two different points");
      return;
    }

    // Calculate scale (meters per pixel)
    const scaleM = distance / pixelDist;

    try {
      // Update floor plan scale on server
      const res = await fetch(`${API_BASE}/api/canopy/survey/floorplan?id=${survey.id}`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          ...currentFloorPlan,
          scaleM,
        }),
      });

      if (!res.ok) {
        throw new Error("Failed to update floor plan scale");
      }

      const updated = await (res.json() as Promise<Survey>);
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
  const handleCancelCalibration = (): void => {
    setCalibrationMode(false);
    setCalibrationPoints([]);
    setCalibrationDistance("");
  };

  // Handle floor plan scale/propagation updates from ScaleCalibrationPanel
  const handleFloorPlanUpdate = async (updates: Partial<FloorPlan>): Promise<void> => {
    if (!currentFloorPlan) {
      return;
    }

    try {
      const updatedFloorPlan: FloorPlan = {
        ...currentFloorPlan,
        ...updates,
      };

      const res: Response = await fetch(`${API_BASE}/api/canopy/survey/floorplan?id=${survey.id}`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify(updatedFloorPlan),
      });

      if (!res.ok) {
        throw new Error("Failed to update floor plan settings");
      }

      const updated: Survey = await (res.json() as Promise<Survey>);
      setSurvey(updated);
      onUpdate();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update settings");
    }
  };

  // Handle survey config updates from SurveyConfigPanel
  const handleConfigUpdate = async (configUpdates: Partial<SurveyConfig>): Promise<void> => {
    try {
      const updatedConfig: SurveyConfig = {
        ...(survey.config || {}),
        ...configUpdates,
      } as SurveyConfig;

      const res: Response = await fetch(`${API_BASE}/api/canopy/survey/config?id=${survey.id}`, {
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

      const updated: Survey = await (res.json() as Promise<Survey>);
      setSurvey(updated);
      onUpdate();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update config");
    }
  };

  // Handle survey type change from SurveyConfigPanel
  const handleSurveyTypeChange = (newType: SurveyType): void => {
    setEditSurveyType(newType);
    // Also update via settings endpoint
    handleSaveSettings().catch((err: unknown) => {
      setError(err instanceof Error ? err.message : "Failed to save settings");
    });
  };

  // Handle iperf settings change from SurveyConfigPanel
  const handleIperfSettingsChange = (server: string, duration: number): void => {
    setEditIperfServer(server);
    setEditTestDuration(duration);
  };

  // Handle AirMapper import
  const handleAirMapperImport = async (
    data: AirMapperData,
    options: ImportOptions,
  ): Promise<void> => {
    try {
      // Build floor plan from imported data
      if (options.importFloorPlan && data.floorPlanImage) {
        // Get image dimensions from the data URL
        const img: HTMLImageElement = new Image();
        await new Promise<void>((resolve, reject) => {
          img.onload = (): void => resolve();
          img.onerror = (): void => reject(new Error("Failed to load imported image"));
          img.src = data.floorPlanImage;
        });

        const floorPlan: FloorPlan = {
          imageData: data.floorPlanImage,
          width: img.width,
          height: img.height,
          scaleM: options.importCalibration ? data.calibration.scaleM : 0.1,
          scaleSource: options.importCalibration ? "imported" : "default",
          propagationM: options.importCalibration ? data.calibration.propagationM : 10,
          originalFile: data.floorPlanFilename,
        };

        // Upload floor plan to server
        const res = await fetch(`${API_BASE}/api/canopy/survey/floorplan?id=${survey.id}`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          credentials: "include",
          body: JSON.stringify(floorPlan),
        });

        if (!res.ok) {
          throw new Error("Failed to import floor plan");
        }

        const updated = await (res.json() as Promise<Survey>);
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
      setError(err instanceof Error ? err.message : "Failed to import AirMapper data");
    }
  };

  // Save survey settings
  const handleSaveSettings = async (): Promise<void> => {
    setSavingSettings(true);
    setError(null);

    try {
      const res: Response = await fetch(`${API_BASE}/api/canopy/survey/settings?id=${survey.id}`, {
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
      });

      if (!res.ok) {
        const errorText: string = await (res.text() as Promise<string>);
        throw new Error(errorText || "Failed to save settings");
      }

      const updated: Survey = await (res.json() as Promise<Survey>);
      setSurvey(updated);
      onUpdate();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save settings");
    } finally {
      setSavingSettings(false);
    }
  };

  // Get button title for start/resume buttons
  const getStartButtonTitle = (): string | undefined => {
    if (!wifiStatus?.canScan) {
      return t("wifi.requiredToStart");
    }
    if (!readyToStart) {
      return `${t("setup.readyToStart")}: ${missingSetupSteps.map((s) => s.label).join(", ")}`;
    }
    return;
  };

  // Handle status changes
  const handleStatusChange = async (action: "start" | "pause" | "complete"): Promise<void> => {
    try {
      const res: Response = await fetch(`${API_BASE}/api/canopy/survey/${action}?id=${survey.id}`, {
        method: "POST",
        credentials: "include",
      });

      if (res.ok) {
        const updated: Survey = await (res.json() as Promise<Survey>);
        setSurvey(updated);
        onUpdate();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : `Failed to ${action} survey`);
    }
  };

  return (
    <div class="fixed inset-0 bg-surface-base z-50 overflow-auto">
      {/* Header */}
      <div class="sticky top-0 bg-surface-raised border-b border-surface-border z-10">
        <div class={cn("max-w-7xl mx-auto pad", layout.flex.between)}>
          <div>
            <h1 class="heading-1">{survey.name}</h1>
            <p class={cn("body-small", spacing.margin.top.tight)}>
              {(survey.surveyType ?? "wifi").charAt(0).toUpperCase() +
                (survey.surveyType ?? "wifi").slice(1)}{" "}
              {t("status.survey")} • {currentSamples.length} {t("status.samples")} •{" "}
              {survey.status ?? "unknown"}
            </p>
          </div>

          <div class={layout.inline.default}>
            {/* Status controls */}
            {survey.status === "created" ? (
              <button
                type="button"
                onClick={(): void => {
                  handleStatusChange("start").catch(() => {
                    /* Error handled in handleStatusChange */
                  });
                }}
                disabled={!(wifiStatus?.canScan && readyToStart)}
                title={getStartButtonTitle()}
                class={cn(
                  button.size.md,
                  "bg-brand-primary text-text-inverse",
                  radius.md,
                  "hover:bg-brand-primary/90",
                  layout.inline.default,
                  "disabled:opacity-50 disabled:cursor-not-allowed",
                )}
              >
                <Play class={iconTokens.size.sm} />
                {t("buttons.startSurvey")}
              </button>
            ) : null}

            {survey.status === "in_progress" ? (
              <>
                <button
                  type="button"
                  onClick={(): void => {
                    handleStatusChange("pause").catch(() => {
                      /* Error handled in handleStatusChange */
                    });
                  }}
                  class={cn(
                    button.size.md,
                    "border border-surface-border",
                    radius.md,
                    "hover:bg-surface-hover",
                    layout.inline.default,
                  )}
                >
                  <Pause class={iconTokens.size.sm} />
                  {t("buttons.pause")}
                </button>
                <button
                  type="button"
                  onClick={(): void => {
                    handleStatusChange("complete").catch(() => {
                      /* Error handled in handleStatusChange */
                    });
                  }}
                  class={cn(
                    button.size.md,
                    "bg-status-success text-text-inverse",
                    radius.md,
                    "hover:bg-status-success/90",
                    layout.inline.default,
                  )}
                >
                  <CheckCircle class={iconTokens.size.sm} />
                  {t("buttons.complete")}
                </button>
              </>
            ) : null}

            {survey.status === "paused" ? (
              <>
                <button
                  type="button"
                  onClick={(): void => {
                    handleStatusChange("start").catch(() => {
                      /* Error handled in handleStatusChange */
                    });
                  }}
                  disabled={!(wifiStatus?.canScan && readyToStart)}
                  title={getStartButtonTitle()}
                  class={cn(
                    button.size.md,
                    "bg-brand-primary text-text-inverse",
                    radius.md,
                    "hover:bg-brand-primary/90",
                    layout.inline.default,
                    "disabled:opacity-50 disabled:cursor-not-allowed",
                  )}
                >
                  <Play class={iconTokens.size.sm} />
                  {t("buttons.resume")}
                </button>
                <button
                  type="button"
                  onClick={(): void => {
                    handleStatusChange("complete").catch(() => {
                      /* Error handled in handleStatusChange */
                    });
                  }}
                  class={cn(
                    button.size.md,
                    "bg-status-success text-text-inverse",
                    radius.md,
                    "hover:bg-status-success/90",
                    layout.inline.default,
                  )}
                >
                  <CheckCircle class={iconTokens.size.sm} />
                  {t("buttons.complete")}
                </button>
              </>
            ) : null}

            <button
              type="button"
              onClick={onClose}
              class={cn(
                button.size.md,
                "border border-surface-border",
                radius.md,
                "hover:bg-surface-hover",
                layout.inline.default,
              )}
            >
              <X class={iconTokens.size.sm} />
              {t("buttons.close")}
            </button>
          </div>
        </div>
      </div>

      {/* Main content */}
      <div class={cn("max-w-7xl mx-auto", spacing.pad.default, spacing.pad.lg)}>
        {error ? (
          <div
            class={cn(
              "bg-status-error/10 border border-status-error/20 text-status-error",
              spacing.pad.sm,
              radius.md,
              spacing.margin.bottom.content,
            )}
          >
            {error}
          </div>
        ) : null}

        {!readyToStart && (
          <div
            class={cn(
              "bg-status-warning/10 border border-status-warning/30 text-status-warning",
              spacing.pad.sm,
              radius.md,
              spacing.margin.bottom.content,
            )}
          >
            {t("setup.readyToStart")}: {missingSetupSteps.map((s) => s.label).join(", ")}
          </div>
        )}

        {/* AirMapper Import Modal */}
        {showImport ? (
          <div class={spacing.margin.bottom.content}>
            <AirMapperImport
              onImport={handleAirMapperImport}
              onCancel={(): void => setShowImport(false)}
            />
          </div>
        ) : null}

        {/* WiFi adapter status banner */}
        {wifiStatus && wifiStatus.status !== "ready" && (
          <div
            class={cn(
              wifiStatus.status === "unavailable"
                ? "bg-status-info/10 border-status-info/20 text-status-info"
                : "bg-status-info/10 border-status-info/20 text-status-info",
              "border",
              spacing.pad.sm,
              radius.md,
              spacing.margin.bottom.content,
            )}
          >
            <div class="font-medium">
              {wifiStatus.status === "unavailable"
                ? t("wifi.noAdapterSetup")
                : t("wifi.adapterAvailable")}
            </div>
            {wifiStatus.availableAdapters.length > 0 && (
              <div class={cn("caption", spacing.margin.top.tight)}>
                {t("wifi.availableAdapters")}: {wifiStatus.availableAdapters.join(", ")}
              </div>
            )}
            <div class={cn("caption", spacing.margin.top.tight)}>
              {t("wifi.setupWithoutAdapter")}
            </div>
          </div>
        )}

        {sampling ? (
          <div
            class={cn(
              "bg-status-info/10 border border-status-info/20 text-status-info",
              spacing.pad.sm,
              radius.md,
              spacing.margin.bottom.content,
              layout.inline.default,
            )}
          >
            <Loader class={cn(iconTokens.size.sm, "animate-spin")} />
            {t("progress.takingMeasurement")}
          </div>
        ) : null}

        <div class={cn("grid grid-cols-1 lg:grid-cols-3", spacing.gap.spacious)}>
          {/* Floor plan */}
          <div class="lg:col-span-2">
            <div class={cn("bg-surface-raised", radius.md, "border border-surface-border pad")}>
              <div class={cn(layout.flex.between, spacing.margin.bottom.content)}>
                <h2 class="heading-3">{t("floorPlan.title")}</h2>
                {heatmapMetric !== null && (
                  <button
                    type="button"
                    onClick={() => setHeatmapMetric(null)}
                    class={cn(
                      button.size.sm,
                      "body-small bg-brand-primary text-text-inverse",
                      radius.md,
                      "hover:bg-brand-primary/90",
                    )}
                  >
                    {t("buttons.hideHeatmap")}
                  </button>
                )}
              </div>

              {/* Heatmap metric selector - categorized */}
              {heatmapMetric === null && currentSamples.length > 0 && (
                <div class={cn(spacing.margin.bottom.content, spacing.stack.sm)}>
                  {/* Signal Category */}
                  <div>
                    <div class={cn("body-small text-text-muted", spacing.margin.bottom.tight)}>
                      {t("heatmaps.categories.signal")}
                    </div>
                    <div class={layout.inline.default}>
                      <button
                        type="button"
                        onClick={() => setHeatmapMetric("rssi")}
                        class={cn(
                          button.size.sm,
                          "body-small border border-surface-border",
                          radius.md,
                          "hover:bg-surface-hover",
                          layout.inline.tight,
                        )}
                      >
                        <Wifi class={iconTokens.size.sm} />
                        {t("heatmaps.rssi")}
                      </button>
                      <button
                        type="button"
                        onClick={() => setHeatmapMetric("snr")}
                        class={cn(
                          button.size.sm,
                          "body-small border border-surface-border",
                          radius.md,
                          "hover:bg-surface-hover",
                          layout.inline.tight,
                        )}
                      >
                        <Activity class={iconTokens.size.sm} />
                        {t("heatmaps.snr")}
                      </button>
                      <button
                        type="button"
                        onClick={() => setHeatmapMetric("noise")}
                        class={cn(
                          button.size.sm,
                          "body-small border border-surface-border",
                          radius.md,
                          "hover:bg-surface-hover",
                          layout.inline.tight,
                        )}
                      >
                        <Radio class={iconTokens.size.sm} />
                        {t("heatmaps.noise")}
                      </button>
                    </div>
                  </div>

                  {/* Interference Category */}
                  <div>
                    <div class={cn("body-small text-text-muted", spacing.margin.bottom.tight)}>
                      {t("heatmaps.categories.interference")}
                    </div>
                    <div class={layout.inline.default}>
                      <button
                        type="button"
                        onClick={() => setHeatmapMetric("cochannel")}
                        class={cn(
                          button.size.sm,
                          "body-small border border-surface-border",
                          radius.md,
                          "hover:bg-surface-hover",
                          layout.inline.tight,
                        )}
                      >
                        <Waves class={iconTokens.size.sm} />
                        {t("heatmaps.cochannel")}
                      </button>
                      <button
                        type="button"
                        onClick={() => setHeatmapMetric("adjacent")}
                        class={cn(
                          button.size.sm,
                          "body-small border border-surface-border",
                          radius.md,
                          "hover:bg-surface-hover",
                          layout.inline.tight,
                        )}
                      >
                        <Waves class={iconTokens.size.sm} />
                        {t("heatmaps.adjacent")}
                      </button>
                      <button
                        type="button"
                        onClick={() => setHeatmapMetric("apDensity")}
                        class={cn(
                          button.size.sm,
                          "body-small border border-surface-border",
                          radius.md,
                          "hover:bg-surface-hover",
                          layout.inline.tight,
                        )}
                      >
                        <Hash class={iconTokens.size.sm} />
                        {t("heatmaps.apDensity")}
                      </button>
                      <button
                        type="button"
                        onClick={() => setHeatmapMetric("ssidCount")}
                        class={cn(
                          button.size.sm,
                          "body-small border border-surface-border",
                          radius.md,
                          "hover:bg-surface-hover",
                          layout.inline.tight,
                        )}
                      >
                        <Hash class={iconTokens.size.sm} />
                        {t("heatmaps.ssidCount")}
                      </button>
                    </div>
                  </div>

                  {/* Performance Category - only for throughput surveys */}
                  {survey.surveyType === "throughput" && (
                    <div>
                      <div class={cn("body-small text-text-muted", spacing.margin.bottom.tight)}>
                        {t("heatmaps.categories.performance")}
                      </div>
                      <div class={layout.inline.default}>
                        <button
                          type="button"
                          onClick={() => setHeatmapMetric("throughput")}
                          class={cn(
                            button.size.sm,
                            "body-small border border-surface-border",
                            radius.md,
                            "hover:bg-surface-hover",
                            layout.inline.tight,
                          )}
                        >
                          <Gauge class={iconTokens.size.sm} />
                          {t("heatmaps.throughput")}
                        </button>
                        <button
                          type="button"
                          onClick={() => setHeatmapMetric("latency")}
                          class={cn(
                            button.size.sm,
                            "body-small border border-surface-border",
                            radius.md,
                            "hover:bg-surface-hover",
                            layout.inline.tight,
                          )}
                        >
                          <Clock class={iconTokens.size.sm} />
                          {t("heatmaps.latency")}
                        </button>
                      </div>
                    </div>
                  )}
                </div>
              )}

              {currentFloorPlan ? (
                <div>
                  {/* Calibration panel */}
                  {calibrationMode ? (
                    <div
                      class={cn(
                        "bg-status-warning/10 border border-status-warning/20",
                        spacing.pad.sm,
                        radius.md,
                        spacing.margin.bottom.content,
                      )}
                    >
                      <div
                        class={cn("font-medium text-status-warning", spacing.margin.bottom.inline)}
                      >
                        📐 {t("calibration.title")}
                      </div>
                      <p
                        class={cn("body-small text-text-secondary", spacing.margin.bottom.content)}
                      >
                        {t("calibration.instructions")}
                      </p>
                      <div class="stack-sm">
                        <div class={layout.inline.default}>
                          <span class="body-small text-text-muted w-20">
                            {t("calibration.pointA")}:
                          </span>
                          {calibrationPoints[0] ? (
                            <span class="body-small font-medium">
                              ({calibrationPoints[0].x}, {calibrationPoints[0].y})
                            </span>
                          ) : (
                            <span class="body-small text-text-muted italic">
                              {t("calibration.clickFloorPlan")}
                            </span>
                          )}
                        </div>
                        <div class={layout.inline.default}>
                          <span class="body-small text-text-muted w-20">
                            {t("calibration.pointB")}:
                          </span>
                          {calibrationPoints[1] ? (
                            <span class="body-small font-medium">
                              ({calibrationPoints[1].x}, {calibrationPoints[1].y})
                            </span>
                          ) : (
                            <span class="body-small text-text-muted italic">
                              {t("calibration.clickFloorPlan")}
                            </span>
                          )}
                        </div>
                        {calibrationPoints.length === 2 && (
                          <div class={layout.inline.default}>
                            <span class="body-small text-text-muted w-20">
                              {t("calibration.pixelDistance")}:
                            </span>
                            <span class="body-small font-medium">
                              {Math.sqrt(
                                (calibrationPoints[1].x - calibrationPoints[0].x) ** 2 +
                                  (calibrationPoints[1].y - calibrationPoints[0].y) ** 2,
                              ).toFixed(0)}{" "}
                              px
                            </span>
                          </div>
                        )}
                        <div class={cn(layout.inline.default, spacing.margin.top.inline)}>
                          <label for="calibration-distance" class="body-small text-text-muted w-20">
                            {t("calibration.distance")}:
                          </label>
                          <input
                            id="calibration-distance"
                            type="number"
                            step="0.1"
                            min="0"
                            value={calibrationDistance}
                            onChange={(
                              e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>,
                            ): void => setCalibrationDistance(e.target.value)}
                            placeholder={
                              useSae ? t("calibration.enterFeet") : t("calibration.enterMeters")
                            }
                            class={cn(
                              "flex-1",
                              button.size.sm,
                              "border border-surface-border",
                              radius.md,
                              "bg-surface-base text-text-primary",
                            )}
                          />
                          <span class="body-small text-text-muted">
                            {useSae ? t("calibration.feet") : t("calibration.meters")}
                          </span>
                        </div>
                        <div class={cn(layout.inline.default, spacing.margin.top.inline)}>
                          <button
                            type="button"
                            onClick={handleSaveCalibration}
                            disabled={calibrationPoints.length !== 2 || !calibrationDistance}
                            class={cn(
                              button.size.sm,
                              "bg-brand-primary text-text-inverse",
                              radius.md,
                              "hover:bg-brand-primary/90 disabled:opacity-50 disabled:cursor-not-allowed",
                            )}
                          >
                            {t("buttons.saveScale")}
                          </button>
                          <button
                            type="button"
                            onClick={handleCancelCalibration}
                            class={cn(
                              button.size.sm,
                              "border border-surface-border",
                              radius.md,
                              "hover:bg-surface-hover",
                            )}
                          >
                            {t("buttons.cancel")}
                          </button>
                          <button
                            type="button"
                            onClick={() => setCalibrationPoints([])}
                            class={cn(
                              button.size.sm,
                              "border border-surface-border",
                              radius.md,
                              "hover:bg-surface-hover",
                            )}
                          >
                            {t("buttons.resetPoints")}
                          </button>
                        </div>
                      </div>
                    </div>
                  ) : null}

                  {/* Calibrate button and current scale info */}
                  {!calibrationMode && currentFloorPlan && (
                    <div class={cn(layout.flex.between, spacing.margin.bottom.inline)}>
                      <div class="body-small text-text-muted">
                        {t("floorPlan.scale")}: {currentFloorPlan.scaleM.toFixed(3)} m/px
                        {survey.status === "in_progress" && ` • ${t("floorPlan.clickToMeasure")}`}
                      </div>
                      <button
                        type="button"
                        onClick={() => setCalibrationMode(true)}
                        class={cn(
                          button.size.sm,
                          "body-small border border-surface-border",
                          radius.md,
                          "hover:bg-surface-hover",
                          layout.inline.tight,
                        )}
                      >
                        <Ruler class={iconTokens.size.sm} />
                        {t("buttons.calibrateScale")}
                      </button>
                    </div>
                  )}

                  <FloorPlanCanvas
                    floorPlan={
                      currentFloorPlan ?? {
                        id: "",
                        name: "",
                        imageUrl: "",
                        width: 0,
                        height: 0,
                        scale: 1,
                      }
                    }
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

                  {/* Heatmap Legend and Stats - show when heatmap is active */}
                  {heatmapMetric !== null && currentSamples.length > 0 && (
                    <div class={spacing.margin.top.content}>
                      <HeatmapLegend
                        metric={heatmapMetric}
                        minValue={calculateMetricRange(currentSamples, heatmapMetric).min}
                        maxValue={calculateMetricRange(currentSamples, heatmapMetric).max}
                      />
                      <HeatmapStats samples={currentSamples} metric={heatmapMetric} />
                    </div>
                  )}
                </div>
              ) : (
                <div
                  class={cn(
                    "border-2 border-dashed border-surface-border",
                    radius.md,
                    "pad-lg text-center",
                  )}
                >
                  <Upload
                    class={cn(
                      iconTokens.size.xl,
                      "mx-auto text-text-muted",
                      spacing.margin.bottom.content,
                    )}
                  />
                  <p class={cn("text-text-muted", spacing.margin.bottom.content)}>
                    {t("floorPlan.uploadPrompt")}
                  </p>
                  <label
                    class={cn(
                      "inline-block",
                      button.size.md,
                      "bg-brand-primary text-text-inverse",
                      radius.md,
                      "cursor-pointer hover:bg-brand-primary/90",
                    )}
                  >
                    {uploadingFloorPlan ? t("floorPlan.uploading") : t("floorPlan.chooseFile")}
                    <input
                      type="file"
                      accept="image/png,image/jpeg,image/gif,image/webp,image/svg+xml"
                      class="hidden"
                      onChange={(
                        e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>,
                      ): void => {
                        const file = e.target.files?.[0];
                        if (file) {
                          handleFloorPlanUpload(file).catch(() => undefined);
                        }
                        // Reset input so same file can be selected again if needed
                        e.target.value = "";
                      }}
                      disabled={uploadingFloorPlan}
                    />
                  </label>
                  <p class={cn("caption text-text-muted", spacing.margin.top.inline)}>
                    {t("floorPlan.supportedFormats")}
                  </p>
                  <div
                    class={cn(
                      spacing.margin.top.content,
                      "border-t border-surface-border",
                      spacing.padding.top,
                    )}
                  >
                    <p class={cn("caption text-text-muted", spacing.margin.bottom.inline)}>
                      {t("import.description")}
                    </p>
                    <button
                      type="button"
                      onClick={() => setShowImport(true)}
                      class={cn(
                        button.size.sm,
                        "border border-surface-border",
                        radius.md,
                        "hover:bg-surface-hover",
                        layout.inline.default,
                      )}
                    >
                      <FileArchive class={iconTokens.size.sm} />
                      {t("import.button")}
                    </button>
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* Settings panel (shown when survey is in created status) and Sample list */}
          <div class={cn("lg:col-span-1", spacing.stack.default)}>
            {/* Setup checklist to guide users before starting a survey */}
            {survey.status === "created" && (
              <div class={cn("bg-surface-raised", radius.md, "border border-surface-border pad")}>
                <div class={cn(layout.flex.between, spacing.margin.bottom.inline)}>
                  <h2 class="heading-3">{t("setup.checklist")}</h2>
                  <span class="caption text-text-muted">
                    {completedSetupSteps}/{setupSteps.length}
                  </span>
                </div>
                <div class="stack-sm">
                  {setupSteps.map((step) => (
                    <div
                      key={step.key}
                      class={cn(
                        "flex items-center justify-between",
                        spacing.pad.xs,
                        radius.sm,
                        step.done ? "bg-surface-hover" : "bg-transparent",
                      )}
                    >
                      <div class={layout.inline.default}>
                        {step.done ? (
                          <CheckCircle class={cn(iconTokens.size.sm, "text-status-success")} />
                        ) : (
                          <Clock class={cn(iconTokens.size.sm, "text-text-muted")} />
                        )}
                        <span class="body-small">{step.label}</span>
                      </div>
                      {step.done ? <span class="caption text-status-success">✓</span> : null}
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Survey Settings Panel - only show when survey hasn't started */}
            {survey.status === "created" && (
              <div class={cn("bg-surface-raised", radius.md, "border border-surface-border pad")}>
                <h2 class={cn("heading-3", spacing.margin.bottom.content)}>
                  {t("settings.title")}
                </h2>
                <div class="stack">
                  {/* Survey Type */}
                  <div>
                    <label
                      for="survey-type-select"
                      class={cn("body-small text-text-muted block", spacing.margin.bottom.tight)}
                    >
                      {t("settings.surveyType")}
                    </label>
                    <select
                      id="survey-type-select"
                      value={editSurveyType}
                      onChange={(
                        e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>,
                      ): void =>
                        setEditSurveyType(e.target.value as "passive" | "active" | "throughput")
                      }
                      class={cn(
                        "w-full",
                        button.size.md,
                        "border border-surface-border",
                        radius.md,
                        "bg-surface-base text-text-primary",
                      )}
                    >
                      <option value="passive">{t("settings.types.passive")}</option>
                      <option value="active">{t("settings.types.active")}</option>
                      <option value="throughput">{t("settings.types.throughput")}</option>
                    </select>
                  </div>

                  {/* iperf Server - only show for throughput surveys */}
                  {editSurveyType === "throughput" && (
                    <>
                      <div>
                        <label
                          for="survey-iperf-server"
                          class={cn(
                            "body-small text-text-muted block",
                            spacing.margin.bottom.tight,
                          )}
                        >
                          {t("settings.iperfServer")}
                        </label>
                        <input
                          id="survey-iperf-server"
                          type="text"
                          value={editIperfServer}
                          onChange={(
                            e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>,
                          ): void => setEditIperfServer(e.target.value)}
                          placeholder="hostname:5201"
                          class={cn(
                            "w-full",
                            button.size.md,
                            "border border-surface-border",
                            radius.md,
                            "bg-surface-base text-text-primary",
                          )}
                        />
                        <p class={cn("caption text-text-muted", spacing.margin.top.tight)}>
                          {t("settings.iperfServerHint")}
                        </p>
                      </div>

                      <div>
                        <label
                          for="survey-test-duration"
                          class={cn(
                            "body-small text-text-muted block",
                            spacing.margin.bottom.tight,
                          )}
                        >
                          {t("settings.testDuration")}
                        </label>
                        <input
                          id="survey-test-duration"
                          type="number"
                          min="1"
                          max="60"
                          value={editTestDuration}
                          onChange={(
                            e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>,
                          ): void => setEditTestDuration(Number.parseInt(e.target.value, 10) || 3)}
                          class={cn(
                            "w-full",
                            button.size.md,
                            "border border-surface-border",
                            radius.md,
                            "bg-surface-base text-text-primary",
                          )}
                        />
                      </div>
                    </>
                  )}

                  {/* Save button */}
                  <button
                    type="button"
                    onClick={handleSaveSettings}
                    disabled={savingSettings}
                    class={cn(
                      "w-full",
                      button.size.md,
                      "bg-brand-primary text-text-inverse",
                      radius.md,
                      "hover:bg-brand-primary/90 disabled:opacity-50",
                    )}
                  >
                    {savingSettings ? t("buttons.saving") : t("buttons.saveSettings")}
                  </button>

                  {/* Survey type descriptions */}
                  <div
                    class={cn(
                      "caption text-text-muted border-t border-surface-border",
                      spacing.padding.top.section,
                      spacing.margin.top.inline,
                    )}
                  >
                    <p class={cn("font-medium", spacing.margin.bottom.inline)}>
                      {t("settings.typesDescription")}
                    </p>
                    <ul class={cn("list-disc list-inside", spacing.stack.xs)}>
                      <li>
                        <strong>Passive:</strong> {t("settings.passiveDesc")}
                      </li>
                      <li>
                        <strong>Active:</strong> {t("settings.activeDesc")}
                      </li>
                      <li>
                        <strong>Throughput:</strong> {t("settings.throughputDesc")}
                      </li>
                    </ul>
                  </div>
                </div>
              </div>
            )}

            {/* Scale Calibration Panel - show when floor plan exists */}
            {currentFloorPlan ? (
              <ScaleCalibrationPanel
                floorPlan={currentFloorPlan}
                onUpdate={handleFloorPlanUpdate}
                onStartCalibration={(): void => setCalibrationMode(true)}
                isCalibrating={calibrationMode}
              />
            ) : null}

            {/* Survey Configuration Panel - show when floor plan exists */}
            {currentFloorPlan && wifiStatus ? (
              <SurveyConfigPanel
                config={survey.config}
                surveyType={editSurveyType}
                availableAdapters={wifiStatus.availableAdapters || []}
                currentInterface={wifiStatus.currentInterface || survey.interface}
                iperfServer={editIperfServer}
                testDuration={editTestDuration}
                onUpdate={handleConfigUpdate}
                onSurveyTypeChange={handleSurveyTypeChange}
                onIperfSettingsChange={handleIperfSettingsChange}
              />
            ) : null}

            {/* Sample list */}
            <div class={cn("bg-surface-raised", radius.md, "border border-surface-border pad")}>
              <h2 class={cn("heading-3", spacing.margin.bottom.content)}>
                {t("samples.title")} ({currentSamples.length})
              </h2>
              <div class="stack-sm max-h-[70vh] overflow-y-auto">
                {currentSamples.length === 0 ? (
                  <p class={cn("body-small text-center", spacing.pad.lg)}>
                    {t("samples.noSamples")}{" "}
                    {survey.status === "in_progress"
                      ? t("samples.clickToStart")
                      : t("samples.startToBegin")}
                  </p>
                ) : (
                  currentSamples.map((sample, idx) => (
                    <div
                      key={sample.timestamp}
                      class={cn("border border-surface-border", radius.md, "pad-sm body-small")}
                    >
                      <div
                        class={cn(
                          "flex items-center justify-between",
                          spacing.margin.bottom.inline,
                        )}
                      >
                        <span class="font-semibold">#{idx + 1}</span>
                        <span class="caption">
                          {new Date(sample.timestamp).toLocaleTimeString()}
                        </span>
                      </div>
                      <div class="caption stack-xs">
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
  surveyType: string,
): React.JSX.Element {
  if (surveyType === "passive") {
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

  if (surveyType === "active") {
    const activeData = data as ActiveSample;
    return (
      <>
        <div>SSID: {activeData.ssid}</div>
        <div>RSSI: {activeData.rssi} dBm</div>
        <div>Rate: {activeData.dataRate} Mbps</div>
        {activeData.roamingEvent ? (
          <div class="text-status-warning font-semibold">⚠ Roaming</div>
        ) : null}
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
          <div class="text-status-error">Loss: {throughputData.packetLoss.toFixed(1)}%</div>
        )}
      </>
    );
  }

  return null;
}

// Helper to calculate min/max values for a heatmap metric
function calculateMetricRange(
  samples: SamplePoint[],
  metric: HeatmapMetric,
): { min: number; max: number } {
  if (!metric || samples.length === 0) {
    return { min: 0, max: 0 };
  }

  const values: number[] = [];

  for (const sample of samples) {
    const data = sample.sampleData as {
      networks?: { rssi: number; channel: number }[];
      rssi?: number;
      noiseFloor?: number;
      downloadMbps?: number;
      latency?: number;
      channelUtilization?: number;
      uniqueBssids?: number;
      uniqueSsids?: number;
    };

    switch (metric) {
      case "rssi":
        if (data.networks && Array.isArray(data.networks)) {
          const rssiValues = data.networks.map((n) => n.rssi);
          if (rssiValues.length > 0) {
            values.push(Math.max(...rssiValues));
          }
        } else if (data.rssi !== undefined) {
          values.push(data.rssi);
        }
        break;
      case "snr":
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
      case "noise":
        values.push(data.noiseFloor || -95);
        break;
      case "cochannel":
        if (data.networks && Array.isArray(data.networks) && data.networks.length > 0) {
          const primaryChannel = data.networks[0].channel;
          const count = data.networks.filter((n) => n.channel === primaryChannel).length - 1;
          values.push(count);
        }
        break;
      case "adjacent":
        if (data.networks && Array.isArray(data.networks) && data.networks.length > 0) {
          const primaryChannel = data.networks[0].channel;
          const count = data.networks.filter(
            (n) =>
              Math.abs(n.channel - primaryChannel) > 0 && Math.abs(n.channel - primaryChannel) <= 2,
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
          const uniqueBssids = new Set(data.networks.map((n: { bssid: string }) => n.bssid));
          values.push(uniqueBssids.size);
        } else if (data.uniqueBssids !== undefined) {
          values.push(data.uniqueBssids);
        }
        break;
      case "ssidCount":
        if (data.networks && Array.isArray(data.networks)) {
          const uniqueSsids = new Set(
            data.networks.map((n: { ssid: string }) => n.ssid).filter(Boolean),
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
