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

import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useSettings } from '../../contexts/useSettings';
import type {
  ActiveSample,
  HeatmapMetric,
  PassiveSample,
  Survey,
  SurveyType,
  ThroughputSample,
} from '../../hooks/useSurvey';
import { LogComponents, logger } from '../../lib/logger';
import { cn, icon as iconTokens, layout, radius, spacing } from '../../styles/theme';
import { Loader } from '../ui/icons';
import { AirMapperImport } from './AirMapperImport';
import type { CalibrationPoint } from './FloorPlanCanvas';
import { SurveyViewFloorPlanPanel } from './SurveyViewFloorPlanPanel';
import { SurveyViewHeader } from './SurveyViewHeader';
import { SurveyViewSidePanel } from './SurveyViewSidePanel';
import { useSurveyMutations } from './useSurveyMutations';

const API_BASE: string = import.meta.env.VITE_API_BASE || '';

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
  status: 'unavailable' | 'available' | 'ready';
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
  const { t } = useTranslation('survey');
  const { displayOptions } = useSettings();
  // Fixes #733: Use user's unit preference for calibration
  const useSae = displayOptions.unitSystem === 'sae';
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
  const [calibrationDistance, setCalibrationDistance] = useState<string>('');
  // Survey settings edit state
  const [editSurveyType, setEditSurveyType] = useState(initialSurvey.surveyType);
  const [editIperfServer, setEditIperfServer] = useState(initialSurvey.iperfServer || '');
  const [editTestDuration, setEditTestDuration] = useState(initialSurvey.testDuration || 3);
  // AirMapper import state
  const [showImport, setShowImport] = useState(false);
  // Setup progress helpers
  const hasFloorPlan = !!currentFloorPlan;
  const hasCalibration =
    hasFloorPlan &&
    currentFloorPlan?.scaleM > 0 &&
    currentFloorPlan?.scaleSource &&
    currentFloorPlan?.scaleSource !== 'default';
  const interfaceReady = wifiStatus?.canScan === true;
  const configReady =
    hasFloorPlan &&
    ((survey.config?.bands && survey.config.bands.length > 0) ||
      (survey.config?.adapters && survey.config.adapters.length > 0));
  const setupSteps = [
    { key: 'floorPlan', label: t('setup.uploadFloorPlan'), done: hasFloorPlan },
    {
      key: 'calibration',
      label: t('setup.calibrateScale'),
      done: hasCalibration,
    },
    { key: 'wifi', label: t('setup.wifiInterface'), done: interfaceReady },
    { key: 'config', label: t('setup.configureSurvey'), done: configReady },
    {
      key: 'start',
      label: t('setup.readyToStart'),
      done: survey.status !== 'created',
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
          credentials: 'include',
        });
        if (res.ok) {
          const status: WiFiStatus = await (res.json() as Promise<WiFiStatus>);
          setWifiStatus(status);
        }
      } catch (err) {
        logger.error(LogComponents.Wifi, 'Failed to check WiFi status', err);
      }
    };
    checkWifiStatus().catch((err: unknown) => {
      logger.error(LogComponents.Wifi, 'Error checking WiFi status', err);
    });
  }, []);

  // Poll for survey updates when in progress
  useEffect(() => {
    if (survey.status !== 'in_progress') {
      return;
    }

    const interval = setInterval(async () => {
      try {
        const res = await fetch(`${API_BASE}/api/canopy/survey?id=${survey.id}`, {
          credentials: 'include',
        });
        if (res.ok) {
          const updated = await (res.json() as Promise<Survey>);
          setSurvey(updated);
        }
      } catch (err) {
        logger.error(LogComponents.Survey, 'Failed to refresh survey', err);
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
            if (typeof result === 'string') {
              resolve(result);
            } else {
              reject(new Error('Failed to read file as base64'));
            }
          };
          reader.onerror = (): void => {
            reject(new Error('Failed to read file'));
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
            reject(new Error('Failed to load image - file may be corrupted'));
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
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          credentials: 'include',
          body: JSON.stringify(floorPlan),
        });

        if (!res.ok) {
          const errorText = await (res.text() as Promise<string>);
          throw new Error(errorText || 'Failed to upload floor plan');
        }

        // Refresh survey
        const updated = await (res.json() as Promise<Survey>);
        setSurvey(updated);
        onUpdate();
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to upload floor plan');
      } finally {
        setUploadingFloorPlan(false);
      }
    },
    [survey.id, onUpdate],
  );

  // Handle taking a sample at clicked location
  const handlePointClick = useCallback(
    async (x: number, y: number) => {
      if (survey.status !== 'in_progress') {
        return;
      }

      // Check WiFi availability before sampling
      if (!wifiStatus?.canScan) {
        setError(wifiStatus?.message || 'No WiFi adapter available for scanning');
        return;
      }

      setSampling(true);
      setError(null);

      try {
        // Collect sample data based on survey type
        let sampleData: PassiveSample | ActiveSample | ThroughputSample;

        switch (survey.surveyType) {
          case 'passive': {
            // Fetch WiFi scan
            const scanRes = await fetch(`${API_BASE}/api/canopy/wifi/scan`, {
              credentials: 'include',
            });
            if (!scanRes.ok) {
              throw new Error('WiFi scan failed');
            }
            const scanData = await (scanRes.json() as Promise<Record<string, unknown>>);
            // Check if scan was successful
            if (!scanData.available) {
              throw new Error(scanData.error || 'WiFi scan not available');
            }
            sampleData = { networks: scanData.networks || [] };
            break;
          }

          case 'active': {
            // Fetch current WiFi status
            const wifiRes = await fetch(`${API_BASE}/api/canopy/wifi`, {
              credentials: 'include',
            });
            if (!wifiRes.ok) {
              throw new Error('WiFi status fetch failed');
            }
            const wifiData = await (wifiRes.json() as Promise<Record<string, unknown>>);

            // Check if BSSID changed (roaming)
            const lastSample = currentSamples.at(-1);
            const lastBssid = lastSample ? (lastSample.sampleData as ActiveSample).bssid : null;
            const roamingEvent = lastBssid !== null && lastBssid !== wifiData.bssid;

            sampleData = {
              ssid: wifiData.ssid || '',
              bssid: wifiData.bssid || '',
              rssi: wifiData.signal || 0,
              dataRate: wifiData.bitrate || 0,
              roamingEvent,
            };
            break;
          }

          case 'throughput': {
            // Fetch WiFi status first
            const wifiRes2 = await fetch(`${API_BASE}/api/canopy/wifi`, {
              credentials: 'include',
            });
            if (!wifiRes2.ok) {
              throw new Error('WiFi status fetch failed');
            }
            const wifiData2 = await (wifiRes2.json() as Promise<Record<string, unknown>>);

            // Run iperf3 test
            if (!survey.iperfServer) {
              throw new Error('iperf3 server not configured for this survey');
            }

            const [host, port] = survey.iperfServer.split(':');
            const iperfRes = await fetch(`${API_BASE}/api/sap/iperf/client`, {
              method: 'POST',
              headers: {
                'Content-Type': 'application/json',
              },
              credentials: 'include',
              body: JSON.stringify({
                host,
                port: port ? Number.parseInt(port, 10) : 5201,
                duration: survey.testDuration || 3,
                reverse: false,
              }),
            });

            if (!iperfRes.ok) {
              throw new Error('iperf3 test failed');
            }
            const iperfData = await (iperfRes.json() as Promise<Record<string, unknown>>);

            sampleData = {
              ssid: wifiData2.ssid || '',
              bssid: wifiData2.bssid || '',
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
            throw new Error('Unknown survey type');
        }

        // Submit sample to server
        const res = await fetch(`${API_BASE}/api/canopy/survey/sample?id=${survey.id}`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          credentials: 'include',
          body: JSON.stringify({ x, y, sampleData }),
        });

        if (!res.ok) {
          throw new Error('Failed to save sample');
        }

        // Refresh survey to get updated samples
        const refreshRes = await fetch(`${API_BASE}/api/canopy/survey?id=${survey.id}`, {
          credentials: 'include',
        });
        if (refreshRes.ok) {
          const updated = await (refreshRes.json() as Promise<Survey>);
          setSurvey(updated);
          onUpdate();
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to take sample');
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
      setError('Please select two points and enter the distance');
      return;
    }

    const rawDistance: number = Number.parseFloat(calibrationDistance);
    if (Number.isNaN(rawDistance) || rawDistance <= 0) {
      setError('Please enter a valid positive distance');
      return;
    }

    // Fixes #733: Convert feet to meters if using SAE units
    const distance: number = useSae ? rawDistance * 0.3048 : rawDistance;

    // Calculate pixel distance
    const [p1, p2] = calibrationPoints;
    const pixelDist = Math.sqrt((p2.x - p1.x) ** 2 + (p2.y - p1.y) ** 2);

    if (pixelDist === 0) {
      setError('Please select two different points');
      return;
    }

    // Calculate scale (meters per pixel)
    const scaleM = distance / pixelDist;

    try {
      // Update floor plan scale on server
      const res = await fetch(`${API_BASE}/api/canopy/survey/floorplan?id=${survey.id}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify({
          ...currentFloorPlan,
          scaleM,
        }),
      });

      if (!res.ok) {
        throw new Error('Failed to update floor plan scale');
      }

      const updated = await (res.json() as Promise<Survey>);
      setSurvey(updated);
      onUpdate();

      // Exit calibration mode
      setCalibrationMode(false);
      setCalibrationPoints([]);
      setCalibrationDistance('');
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save scale');
    }
  };

  // Cancel calibration
  const handleCancelCalibration = (): void => {
    setCalibrationMode(false);
    setCalibrationPoints([]);
    setCalibrationDistance('');
  };

  // Survey API mutations live in their own hook
  const {
    handleFloorPlanUpdate,
    handleConfigUpdate,
    handleAirMapperImport,
    handleSaveSettings,
    handleStatusChange,
    savingSettings,
  } = useSurveyMutations({
    survey,
    setSurvey,
    onUpdate,
    setError,
    currentFloorPlan,
    editSurveyType,
    editIperfServer,
    editTestDuration,
    setShowImport,
  });

  // Handle survey type change from SurveyConfigPanel
  const handleSurveyTypeChange = (newType: SurveyType): void => {
    setEditSurveyType(newType);
    // Also update via settings endpoint
    handleSaveSettings().catch((err: unknown) => {
      setError(err instanceof Error ? err.message : 'Failed to save settings');
    });
  };

  // Handle iperf settings change from SurveyConfigPanel
  const handleIperfSettingsChange = (server: string, duration: number): void => {
    setEditIperfServer(server);
    setEditTestDuration(duration);
  };

  // Get button title for start/resume buttons
  const getStartButtonTitle = (): string | undefined => {
    if (!wifiStatus?.canScan) {
      return t('wifi.requiredToStart');
    }
    if (!readyToStart) {
      return `${t('setup.readyToStart')}: ${missingSetupSteps.map((s) => s.label).join(', ')}`;
    }
    return;
  };

  return (
    <div class="fixed inset-0 bg-surface-base z-50 overflow-auto">
      <SurveyViewHeader
        survey={survey}
        sampleCount={currentSamples.length}
        wifiStatus={wifiStatus}
        readyToStart={readyToStart}
        getStartButtonTitle={getStartButtonTitle}
        handleStatusChange={handleStatusChange}
        onClose={onClose}
      />

      {/* Main content */}
      <div class={cn('max-w-7xl mx-auto', spacing.pad.default, spacing.pad.lg)}>
        {error ? (
          <div
            class={cn(
              'bg-status-error/10 border border-status-error/20 text-status-error',
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
              'bg-status-warning/10 border border-status-warning/30 text-status-warning',
              spacing.pad.sm,
              radius.md,
              spacing.margin.bottom.content,
            )}
          >
            {t('setup.readyToStart')}: {missingSetupSteps.map((s) => s.label).join(', ')}
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
        {wifiStatus && wifiStatus.status !== 'ready' && (
          <div
            class={cn(
              wifiStatus.status === 'unavailable'
                ? 'bg-status-info/10 border-status-info/20 text-status-info'
                : 'bg-status-info/10 border-status-info/20 text-status-info',
              'border',
              spacing.pad.sm,
              radius.md,
              spacing.margin.bottom.content,
            )}
          >
            <div class="font-medium">
              {wifiStatus.status === 'unavailable'
                ? t('wifi.noAdapterSetup')
                : t('wifi.adapterAvailable')}
            </div>
            {wifiStatus.availableAdapters.length > 0 && (
              <div class={cn('caption', spacing.margin.top.tight)}>
                {t('wifi.availableAdapters')}: {wifiStatus.availableAdapters.join(', ')}
              </div>
            )}
            <div class={cn('caption', spacing.margin.top.tight)}>
              {t('wifi.setupWithoutAdapter')}
            </div>
          </div>
        )}

        {sampling ? (
          <div
            class={cn(
              'bg-status-info/10 border border-status-info/20 text-status-info',
              spacing.pad.sm,
              radius.md,
              spacing.margin.bottom.content,
              layout.inline.default,
            )}
          >
            <Loader class={cn(iconTokens.size.sm, 'animate-spin')} />
            {t('progress.takingMeasurement')}
          </div>
        ) : null}

        <div class={cn('grid grid-cols-1 lg:grid-cols-3', spacing.gap.spacious)}>
          <SurveyViewFloorPlanPanel
            survey={survey}
            currentFloorPlan={currentFloorPlan}
            currentSamples={currentSamples}
            heatmapMetric={heatmapMetric}
            setHeatmapMetric={setHeatmapMetric}
            calibrationMode={calibrationMode}
            setCalibrationMode={setCalibrationMode}
            calibrationPoints={calibrationPoints}
            setCalibrationPoints={setCalibrationPoints}
            calibrationDistance={calibrationDistance}
            setCalibrationDistance={setCalibrationDistance}
            useSae={useSae}
            handleSaveCalibration={handleSaveCalibration}
            handleCancelCalibration={handleCancelCalibration}
            handlePointClick={handlePointClick}
            handleCalibrationClick={handleCalibrationClick}
            handleFloorPlanUpload={handleFloorPlanUpload}
            sampling={sampling}
            wifiStatus={wifiStatus}
            uploadingFloorPlan={uploadingFloorPlan}
            setShowImport={setShowImport}
          />

          <SurveyViewSidePanel
            survey={survey}
            currentSamples={currentSamples}
            currentFloorPlan={currentFloorPlan}
            setupSteps={setupSteps}
            completedSetupSteps={completedSetupSteps}
            editSurveyType={editSurveyType}
            setEditSurveyType={setEditSurveyType}
            editIperfServer={editIperfServer}
            setEditIperfServer={setEditIperfServer}
            editTestDuration={editTestDuration}
            setEditTestDuration={setEditTestDuration}
            savingSettings={savingSettings}
            handleSaveSettings={handleSaveSettings}
            handleFloorPlanUpdate={handleFloorPlanUpdate}
            setCalibrationMode={setCalibrationMode}
            calibrationMode={calibrationMode}
            wifiStatus={wifiStatus}
            handleConfigUpdate={handleConfigUpdate}
            handleSurveyTypeChange={handleSurveyTypeChange}
            handleIperfSettingsChange={handleIperfSettingsChange}
          />
        </div>
      </div>
    </div>
  );
}
