/**
 * useSurveyMutations
 *
 * Bundles the survey API mutation callbacks that SurveyView used to
 * declare inline: floor-plan updates, config updates, settings save,
 * status changes, and AirMapper-data import.
 *
 * The hook owns the savingSettings flag plus the fetch wiring, and
 * surfaces only the callbacks back to the component.
 */

import { useCallback, useState } from 'react';
import type { FloorPlan, Survey, SurveyConfig, SurveyType } from '../../hooks/useSurvey';
import type { AirMapperData } from '../../utils/airmapper';
import type { ImportOptions } from './AirMapperImport';

const API_BASE: string = import.meta.env.VITE_API_BASE || '';

/** Stable id derived from index. Backend treats id as opaque. */
function importedId(prefix: string, index: number): string {
  return `${prefix}-imported-${index}`;
}

/**
 * Pulls AP locations, client locations, and pass-fail criteria off the
 * parsed AirMapper payload and shapes them for the survey/imported-data
 * endpoint. Returns null when there's nothing to send so callers can
 * short-circuit.
 */
function buildImportedDataPayload(data: AirMapperData): {
  apLocations: Array<{ id: string; x: number; y: number; label?: string; imported: boolean }>;
  clientLocations: Array<{ id: string; x: number; y: number; label?: string; imported: boolean }>;
  passFailCriteria: Array<{
    option: string;
    name?: string;
    limit: number;
    suffix?: string;
    enabled: boolean;
    mode?: string;
    ap?: number;
    imported: boolean;
  }>;
} | null {
  // .locations.passive carries AP positions in AirMapper passive surveys.
  const apLocations = data.locations.passive.map((loc, idx) => ({
    id: importedId('ap', idx),
    x: Math.round(loc.x),
    y: Math.round(loc.y),
    label: loc.label || undefined,
    imported: true,
  }));

  const clientLocations = data.locations.client.map((loc, idx) => ({
    id: importedId('client', idx),
    x: Math.round(loc.x),
    y: Math.round(loc.y),
    label: loc.label || undefined,
    imported: true,
  }));

  // passFailCriteria lives on the raw backend response (when import went
  // through the backend route). Be defensive about the shape since
  // rawSerial is typed as unknown.
  const raw = data.rawSerial as { passFailCriteria?: unknown } | undefined;
  const rawCriteria = Array.isArray(raw?.passFailCriteria) ? raw.passFailCriteria : [];
  const passFailCriteria = rawCriteria
    .filter((c): c is Record<string, unknown> => typeof c === 'object' && c !== null)
    .map((c) => ({
      option: typeof c.option === 'string' ? c.option : '',
      name: typeof c.name === 'string' ? c.name : undefined,
      limit: typeof c.limit === 'number' ? c.limit : 0,
      suffix: typeof c.suffix === 'string' ? c.suffix : undefined,
      enabled: c.enabled !== false,
      mode: typeof c.mode === 'string' ? c.mode : undefined,
      ap: typeof c.ap === 'number' ? c.ap : undefined,
      imported: true,
    }))
    .filter((c) => c.option !== '');

  if (apLocations.length === 0 && clientLocations.length === 0 && passFailCriteria.length === 0) {
    return null;
  }
  return { apLocations, clientLocations, passFailCriteria };
}

interface UseSurveyMutationsArgs {
  survey: Survey;
  setSurvey: (s: Survey) => void;
  onUpdate: () => void;
  setError: (message: string | null) => void;
  currentFloorPlan: FloorPlan | null | undefined;
  editSurveyType: SurveyType;
  editIperfServer: string;
  editTestDuration: number;
  setShowImport: (show: boolean) => void;
}

interface UseSurveyMutationsResult {
  handleFloorPlanUpdate: (updates: Partial<FloorPlan>) => Promise<void>;
  handleConfigUpdate: (configUpdates: Partial<SurveyConfig>) => Promise<void>;
  handleAirMapperImport: (data: AirMapperData, options: ImportOptions) => Promise<void>;
  handleSaveSettings: () => Promise<void>;
  handleStatusChange: (action: 'start' | 'pause' | 'complete') => Promise<void>;
  savingSettings: boolean;
}

export function useSurveyMutations({
  survey,
  setSurvey,
  onUpdate,
  setError,
  currentFloorPlan,
  editSurveyType,
  editIperfServer,
  editTestDuration,
  setShowImport,
}: UseSurveyMutationsArgs): UseSurveyMutationsResult {
  const [savingSettings, setSavingSettings] = useState(false);

  const handleFloorPlanUpdate = useCallback(
    async (updates: Partial<FloorPlan>): Promise<void> => {
      if (!currentFloorPlan) {
        return;
      }

      try {
        const updatedFloorPlan: FloorPlan = {
          ...currentFloorPlan,
          ...updates,
        };

        const res: Response = await fetch(
          `${API_BASE}/api/v1/canopy/survey/floorplan?id=${survey.id}`,
          {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            credentials: 'include',
            body: JSON.stringify(updatedFloorPlan),
          },
        );

        if (!res.ok) {
          throw new Error('Failed to update floor plan settings');
        }

        const updated: Survey = await (res.json() as Promise<Survey>);
        setSurvey(updated);
        onUpdate();
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to update settings');
      }
    },
    [currentFloorPlan, survey.id, setSurvey, onUpdate, setError],
  );

  const handleConfigUpdate = useCallback(
    async (configUpdates: Partial<SurveyConfig>): Promise<void> => {
      try {
        const updatedConfig: SurveyConfig = {
          ...(survey.config || {}),
          ...configUpdates,
        } as SurveyConfig;

        const res: Response = await fetch(
          `${API_BASE}/api/v1/canopy/survey/config?id=${survey.id}`,
          {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            credentials: 'include',
            body: JSON.stringify(updatedConfig),
          },
        );

        if (!res.ok) {
          throw new Error('Failed to update survey config');
        }

        const updated: Survey = await (res.json() as Promise<Survey>);
        setSurvey(updated);
        onUpdate();
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to update config');
      }
    },
    [survey.config, survey.id, setSurvey, onUpdate, setError],
  );

  const handleSaveSettings = useCallback(async (): Promise<void> => {
    setSavingSettings(true);
    setError(null);

    try {
      const res: Response = await fetch(
        `${API_BASE}/api/v1/canopy/survey/settings?id=${survey.id}`,
        {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          credentials: 'include',
          body: JSON.stringify({
            surveyType: editSurveyType,
            iperfServer: editIperfServer,
            testDuration: editTestDuration,
          }),
        },
      );

      if (!res.ok) {
        const errorText: string = await (res.text() as Promise<string>);
        throw new Error(errorText || 'Failed to save settings');
      }

      const updated: Survey = await (res.json() as Promise<Survey>);
      setSurvey(updated);
      onUpdate();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save settings');
    } finally {
      setSavingSettings(false);
    }
  }, [survey.id, editSurveyType, editIperfServer, editTestDuration, setSurvey, onUpdate, setError]);

  const handleStatusChange = useCallback(
    async (action: 'start' | 'pause' | 'complete'): Promise<void> => {
      try {
        const res: Response = await fetch(
          `${API_BASE}/api/v1/canopy/survey/${action}?id=${survey.id}`,
          {
            method: 'POST',
            credentials: 'include',
          },
        );

        if (res.ok) {
          const updated: Survey = await (res.json() as Promise<Survey>);
          setSurvey(updated);
          onUpdate();
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : `Failed to ${action} survey`);
      }
    },
    [survey.id, setSurvey, onUpdate, setError],
  );

  const handleAirMapperImport = useCallback(
    // biome-ignore lint/complexity/noExcessiveCognitiveComplexity: AirMapper import branches on importFloorPlan vs calibration-only; mirrors the original inline structure
    async (data: AirMapperData, options: ImportOptions): Promise<void> => {
      try {
        // Build floor plan from imported data
        if (options.importFloorPlan && data.floorPlanImage) {
          // Get image dimensions from the data URL
          const img: HTMLImageElement = new Image();
          await new Promise<void>((resolve, reject) => {
            img.onload = (): void => resolve();
            img.onerror = (): void => reject(new Error('Failed to load imported image'));
            img.src = data.floorPlanImage;
          });

          const floorPlan: FloorPlan = {
            imageData: data.floorPlanImage,
            width: img.width,
            height: img.height,
            scaleM: options.importCalibration ? data.calibration.scaleM : 0.1,
            scaleSource: options.importCalibration ? 'imported' : 'default',
            propagationM: options.importCalibration ? data.calibration.propagationM : 10,
            originalFile: data.floorPlanFilename,
          };

          const res = await fetch(`${API_BASE}/api/v1/canopy/survey/floorplan?id=${survey.id}`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            credentials: 'include',
            body: JSON.stringify(floorPlan),
          });

          if (!res.ok) {
            throw new Error('Failed to import floor plan');
          }

          const updated = await (res.json() as Promise<Survey>);
          setSurvey(updated);
          onUpdate();
        } else if (options.importCalibration && currentFloorPlan) {
          // Just import calibration settings
          await handleFloorPlanUpdate({
            scaleM: data.calibration.scaleM,
            scaleSource: 'imported',
            propagationM: data.calibration.propagationM,
          });
        }

        // Fixes #727: persist AP / client placements and pass-fail criteria
        // from the parsed AirMapper data. Previously these were dropped on
        // the floor (literally) and the user saw a floor plan with no markers.
        const importedData = buildImportedDataPayload(data);
        if (importedData) {
          const dataRes = await fetch(
            `${API_BASE}/api/v1/canopy/survey/imported-data?id=${survey.id}`,
            {
              method: 'PUT',
              headers: { 'Content-Type': 'application/json' },
              credentials: 'include',
              body: JSON.stringify(importedData),
            },
          );
          if (dataRes.ok) {
            const updated = await (dataRes.json() as Promise<Survey>);
            setSurvey(updated);
            onUpdate();
          }
        }

        setShowImport(false);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to import AirMapper data');
      }
    },
    [
      survey.id,
      setSurvey,
      onUpdate,
      currentFloorPlan,
      handleFloorPlanUpdate,
      setShowImport,
      setError,
    ],
  );

  return {
    handleFloorPlanUpdate,
    handleConfigUpdate,
    handleAirMapperImport,
    handleSaveSettings,
    handleStatusChange,
    savingSettings,
  };
}
