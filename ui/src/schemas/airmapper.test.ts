import { describe, expect, it } from 'vitest';
import { parseSerialJson } from './airmapper';

describe('parseSerialJson', () => {
  it('accepts a minimal valid payload', () => {
    const result = parseSerialJson('{}');
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.value).toEqual({});
    }
  });

  it('accepts a fully-populated payload', () => {
    const payload = {
      fileName: 'survey.serial',
      floorPlanFilename: 'plan.jpg',
      floorPlanScalePpf: 10,
      floorPlanWidthPx: 1024,
      floorPlanHeightPx: 768,
      propagation: 8,
      propagationUnit: 'ft',
      surveyName: 'Office Q4',
      surveyMode: 'passive',
      surveyPointCount: 42,
      surveyItemsCount: 7,
      surveyStartTime: '2026-05-25T00:00:00Z',
      surveyActive1x1: false,
      unitName: 'AirCheck',
      unitType: 'G3',
      unitSerial: 'ACG3-001',
      labels: ['floor-1', 'office'],
      views: [
        {
          name: 'coverage',
          option: 'rssi',
          mode: 'heatmap',
          limit: -65,
          filters: [{ key: 'band', value: 5 }],
        },
      ],
      locations: {
        passive: [{ x: 100, y: 200, label: 'AP-1' }],
      },
    };
    const result = parseSerialJson(JSON.stringify(payload));
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.value.surveyName).toBe('Office Q4');
      expect(result.value.locations?.passive?.[0]?.label).toBe('AP-1');
    }
  });

  it('rejects malformed JSON', () => {
    const result = parseSerialJson('{not json}');
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.reason).toMatch(/JSON/);
    }
  });

  it('rejects wrong field types', () => {
    // floorPlanScalePpf must be a number; sending a string trips the schema.
    const result = parseSerialJson(JSON.stringify({ floorPlanScalePpf: '10' }));
    expect(result.ok).toBe(false);
    if (!result.ok) {
      const flat = result.issues.join(' ');
      expect(flat.toLowerCase()).toContain('floorplanscaleppf');
    }
  });

  it('rejects malformed nested location objects', () => {
    const result = parseSerialJson(
      JSON.stringify({
        locations: {
          passive: [{ x: 'left', y: 200, label: 'AP-1' }], // x must be number
        },
      }),
    );
    expect(result.ok).toBe(false);
    if (!result.ok) {
      const flat = result.issues.join(' ').toLowerCase();
      expect(flat).toContain('passive');
    }
  });

  it('keeps issue path so the consumer can render field-level hints', () => {
    const result = parseSerialJson(
      JSON.stringify({ views: [{ name: 'coverage' }] }), // missing option, mode
    );
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.issues.length).toBeGreaterThan(0);
      const flat = result.issues.join('\n');
      expect(flat).toMatch(/views\.0\.(option|mode)/);
    }
  });
});
