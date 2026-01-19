import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import type {
  FloorPlan,
  PassiveSample,
  SamplePoint,
  ThroughputSample,
} from '../../hooks/useSurvey';
import { cn, spacing } from '../../styles/theme';
import { FloorPlanCanvas } from './FloorPlanCanvas';

/**
 * FloorPlanCanvas renders floor plan images with interactive sample points
 * and heatmap visualizations for WiFi survey data.
 *
 * Features:
 * - Floor plan image rendering with aspect ratio preservation
 * - Interactive click-to-add sample points
 * - Heatmap overlay for RSSI, throughput, or latency
 * - Numbered sample markers
 * - Responsive canvas sizing
 */
const meta: Meta<typeof FloorPlanCanvas> = {
  title: 'Survey/FloorPlanCanvas',
  component: FloorPlanCanvas,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'Canvas-based floor plan visualization with sample point markers and heatmap overlays for WiFi survey data.',
      },
    },
  },
  tags: ['autodocs'],
  argTypes: {
    interactive: {
      control: 'boolean',
      description: 'Enable click-to-add sample points',
    },
    heatmapMetric: {
      control: 'select',
      options: [null, 'rssi', 'throughput', 'latency'],
      description: 'Metric to visualize as heatmap overlay',
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Base64 encoded simple floor plan SVG for demos
const SAMPLE_FLOOR_PLAN_SVG =
  'data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iODAwIiBoZWlnaHQ9IjYwMCIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj48cmVjdCB3aWR0aD0iODAwIiBoZWlnaHQ9IjYwMCIgZmlsbD0iI2Y5ZmFmYiIvPjxyZWN0IHg9IjUwIiB5PSI1MCIgd2lkdGg9IjcwMCIgaGVpZ2h0PSI1MDAiIGZpbGw9Im5vbmUiIHN0cm9rZT0iIzMzMyIgc3Ryb2tlLXdpZHRoPSIzIi8+PHJlY3QgeD0iNTAiIHk9IjUwIiB3aWR0aD0iMjAwIiBoZWlnaHQ9IjE1MCIgZmlsbD0ibm9uZSIgc3Ryb2tlPSIjOTk5IiBzdHJva2Utd2lkdGg9IjIiLz48cmVjdCB4PSI1NTAiIHk9IjUwIiB3aWR0aD0iMjAwIiBoZWlnaHQ9IjE1MCIgZmlsbD0ibm9uZSIgc3Ryb2tlPSIjOTk5IiBzdHJva2Utd2lkdGg9IjIiLz48cmVjdCB4PSI1MCIgeT0iNDAwIiB3aWR0aD0iMjAwIiBoZWlnaHQ9IjE1MCIgZmlsbD0ibm9uZSIgc3Ryb2tlPSIjOTk5IiBzdHJva2Utd2lkdGg9IjIiLz48cmVjdCB4PSI1NTAiIHk9IjQwMCIgd2lkdGg9IjIwMCIgaGVpZ2h0PSIxNTAiIGZpbGw9Im5vbmUiIHN0cm9rZT0iIzk5OSIgc3Ryb2tlLXdpZHRoPSIyIi8+PHRleHQgeD0iNDAwIiB5PSI1MCIgdGV4dC1hbmNob3I9Im1pZGRsZSIgZm9udC1zaXplPSIyNCIgZmlsbD0iIzY2NiIgZm9udC13ZWlnaHQ9ImJvbGQiPk9mZmljZSBGbG9vciBQbGFuPC90ZXh0Pjx0ZXh0IHg9IjE1MCIgeT0iMTMwIiB0ZXh0LWFuY2hvcj0ibWlkZGxlIiBmb250LXNpemU9IjE0IiBmaWxsPSIjOTk5Ij5Db25mZXJlbmNlPC90ZXh0Pjx0ZXh0IHg9IjY1MCIgeT0iMTMwIiB0ZXh0LWFuY2hvcj0ibWlkZGxlIiBmb250LXNpemU9IjE0IiBmaWxsPSIjOTk5Ij5PZmZpY2VzPC90ZXh0Pjx0ZXh0IHg9IjQwMCIgeT0iMzAwIiB0ZXh0LWFuY2hvcj0ibWlkZGxlIiBmb250LXNpemU9IjE4IiBmaWxsPSIjY2NjIj5PcGVuIFdvcmtzcGFjZTwvdGV4dD48dGV4dCB4PSIxNTAiIHk9IjQ4MCIgdGV4dC1hbmNob3I9Im1pZGRsZSIgZm9udC1zaXplPSIxNCIgZmlsbD0iIzk5OSI+S2l0Y2hlbjwvdGV4dD48dGV4dCB4PSI2NTAiIHk9IjQ4MCIgdGV4dC1hbmNob3I9Im1pZGRsZSIgZm9udC1zaXplPSIxNCIgZmlsbD0iIzk5OSI+U3RvcmFnZTwvdGV4dD48L3N2Zz4=';

const sampleFloorPlan: FloorPlan = {
  imageData: SAMPLE_FLOOR_PLAN_SVG,
  width: 800,
  height: 600,
  scaleM: 0.1,
};

// Sample points with passive scan data
const passiveSamples: SamplePoint[] = [
  {
    x: 150,
    y: 120,
    timestamp: '2025-12-15T10:00:00Z',
    sampleData: {
      networks: [
        {
          ssid: 'OfficeMain',
          bssid: 'AA:BB:CC:DD:EE:01',
          rssi: -42,
          channel: 6,
          frequency: 2437,
        },
        {
          ssid: 'Guest',
          bssid: 'AA:BB:CC:DD:EE:02',
          rssi: -58,
          channel: 11,
          frequency: 2462,
        },
      ],
    } as PassiveSample,
  },
  {
    x: 400,
    y: 250,
    timestamp: '2025-12-15T10:05:00Z',
    sampleData: {
      networks: [
        {
          ssid: 'OfficeMain',
          bssid: 'AA:BB:CC:DD:EE:01',
          rssi: -38,
          channel: 6,
          frequency: 2437,
        },
      ],
    } as PassiveSample,
  },
  {
    x: 650,
    y: 120,
    timestamp: '2025-12-15T10:10:00Z',
    sampleData: {
      networks: [
        {
          ssid: 'OfficeMain',
          bssid: 'AA:BB:CC:DD:EE:03',
          rssi: -55,
          channel: 6,
          frequency: 2437,
        },
      ],
    } as PassiveSample,
  },
  {
    x: 150,
    y: 450,
    timestamp: '2025-12-15T10:15:00Z',
    sampleData: {
      networks: [
        {
          ssid: 'OfficeMain',
          bssid: 'AA:BB:CC:DD:EE:01',
          rssi: -68,
          channel: 6,
          frequency: 2437,
        },
      ],
    } as PassiveSample,
  },
  {
    x: 650,
    y: 450,
    timestamp: '2025-12-15T10:20:00Z',
    sampleData: {
      networks: [
        {
          ssid: 'OfficeMain',
          bssid: 'AA:BB:CC:DD:EE:03',
          rssi: -72,
          channel: 6,
          frequency: 2437,
        },
      ],
    } as PassiveSample,
  },
];

// Sample points with throughput data
const throughputSamples: SamplePoint[] = [
  {
    x: 150,
    y: 120,
    timestamp: '2025-12-15T10:00:00Z',
    sampleData: {
      ssid: 'OfficeMain',
      bssid: 'AA:BB:CC:DD:EE:01',
      rssi: -42,
      downloadMbps: 485.3,
      uploadMbps: 387.2,
      latency: 12,
      jitter: 1.2,
      packetLoss: 0.1,
    } as ThroughputSample,
  },
  {
    x: 400,
    y: 250,
    timestamp: '2025-12-15T10:05:00Z',
    sampleData: {
      ssid: 'OfficeMain',
      bssid: 'AA:BB:CC:DD:EE:01',
      rssi: -38,
      downloadMbps: 612.8,
      uploadMbps: 453.6,
      latency: 8,
      jitter: 0.8,
      packetLoss: 0,
    } as ThroughputSample,
  },
  {
    x: 650,
    y: 120,
    timestamp: '2025-12-15T10:10:00Z',
    sampleData: {
      ssid: 'OfficeMain',
      bssid: 'AA:BB:CC:DD:EE:03',
      rssi: -55,
      downloadMbps: 328.5,
      uploadMbps: 245.1,
      latency: 18,
      jitter: 2.5,
      packetLoss: 0.3,
    } as ThroughputSample,
  },
  {
    x: 150,
    y: 450,
    timestamp: '2025-12-15T10:15:00Z',
    sampleData: {
      ssid: 'OfficeMain',
      bssid: 'AA:BB:CC:DD:EE:01',
      rssi: -68,
      downloadMbps: 145.2,
      uploadMbps: 98.7,
      latency: 35,
      jitter: 5.2,
      packetLoss: 1.2,
    } as ThroughputSample,
  },
  {
    x: 650,
    y: 450,
    timestamp: '2025-12-15T10:20:00Z',
    sampleData: {
      ssid: 'OfficeMain',
      bssid: 'AA:BB:CC:DD:EE:03',
      rssi: -72,
      downloadMbps: 85.6,
      uploadMbps: 42.3,
      latency: 55,
      jitter: 8.1,
      packetLoss: 2.5,
    } as ThroughputSample,
  },
];

/**
 * Empty floor plan without sample points.
 * Ready for interactive sampling.
 */
export const EmptyFloorPlan: Story = {
  args: {
    floorPlan: sampleFloorPlan,
    samples: [],
    interactive: false,
    heatmapMetric: null,
  },
};

/**
 * Floor plan with sample points but no heatmap.
 * Shows numbered markers at each measurement location.
 */
export const WithSamplePoints: Story = {
  args: {
    floorPlan: sampleFloorPlan,
    samples: passiveSamples,
    interactive: false,
    heatmapMetric: null,
  },
};

/**
 * Interactive floor plan allowing click-to-add sample points.
 * Click on the canvas to add new measurement locations.
 */
export const Interactive: Story = {
  render: () => {
    const [samples, setSamples] = useState<SamplePoint[]>([]);

    const handlePointClick = (x: number, y: number) => {
      const newSample: SamplePoint = {
        x,
        y,
        timestamp: new Date().toISOString(),
        sampleData: {
          networks: [
            {
              ssid: 'TestNetwork',
              bssid: 'AA:BB:CC:DD:EE:FF',
              rssi: -40 - Math.random() * 40,
              channel: 6,
              frequency: 2437,
            },
          ],
        } as PassiveSample,
      };
      setSamples([...samples, newSample]);
    };

    return (
      <div class={cn(spacing.pad.default, 'w-full max-w-4xl bg-surface-base')}>
        <div class={spacing.margin.bottom.content}>
          <p class={cn(spacing.margin.bottom.inline, 'body-small text-text-muted')}>
            Click on the floor plan to add sample points. {samples.length} points added.
          </p>
          {samples.length > 0 && (
            <button
              type="button"
              onClick={() => setSamples([])}
              class={cn(
                spacing.chip.sm,
                'bg-status-error/10 text-status-error rounded text-sm hover:bg-status-error/20',
              )}
            >
              Clear Points
            </button>
          )}
        </div>
        <FloorPlanCanvas
          floorPlan={sampleFloorPlan}
          samples={samples}
          onPointClick={handlePointClick}
          interactive={true}
          heatmapMetric={null}
        />
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story: 'Interactive canvas where clicking adds sample points at that location.',
      },
    },
  },
};

/**
 * RSSI heatmap visualization.
 * Shows signal strength gradient across the floor plan.
 * Green = strong signal, Red = weak signal.
 */
export const RssiHeatmap: Story = {
  args: {
    floorPlan: sampleFloorPlan,
    samples: passiveSamples,
    interactive: false,
    heatmapMetric: 'rssi',
  },
  parameters: {
    docs: {
      description: {
        story:
          'RSSI heatmap showing signal strength distribution. Green indicates strong signal (-30 to -50 dBm), yellow is moderate (-50 to -70 dBm), and red indicates weak signal (below -70 dBm).',
      },
    },
  },
};

/**
 * Throughput heatmap visualization.
 * Shows download speed gradient across the floor plan.
 * Green = high throughput, Red = low throughput.
 */
export const ThroughputHeatmap: Story = {
  args: {
    floorPlan: sampleFloorPlan,
    samples: throughputSamples,
    interactive: false,
    heatmapMetric: 'throughput',
  },
  parameters: {
    docs: {
      description: {
        story:
          'Throughput heatmap showing download speed distribution. Green indicates high speeds, red indicates low speeds or dead zones.',
      },
    },
  },
};

/**
 * Latency heatmap visualization.
 * Shows network latency gradient across the floor plan.
 * Green = low latency, Red = high latency.
 */
export const LatencyHeatmap: Story = {
  args: {
    floorPlan: sampleFloorPlan,
    samples: throughputSamples,
    interactive: false,
    heatmapMetric: 'latency',
  },
  parameters: {
    docs: {
      description: {
        story:
          'Latency heatmap showing response time distribution. Green indicates low latency (<20ms), red indicates high latency (>50ms).',
      },
    },
  },
};

/**
 * Dense sample grid for accurate heatmap interpolation.
 * More samples provide smoother heatmap gradients.
 */
export const DenseSampling: Story = {
  args: {
    floorPlan: sampleFloorPlan,
    samples: Array.from({ length: 25 }, (_, i) => ({
      x: 100 + (i % 5) * 150,
      y: 100 + Math.floor(i / 5) * 100,
      timestamp: new Date().toISOString(),
      sampleData: {
        networks: [
          {
            ssid: 'OfficeMain',
            bssid: 'AA:BB:CC:DD:EE:01',
            rssi: -35 - (Math.abs(2 - (i % 5)) + Math.abs(2 - Math.floor(i / 5))) * 8,
            channel: 6,
            frequency: 2437,
          },
        ],
      } as PassiveSample,
    })),
    interactive: false,
    heatmapMetric: 'rssi',
  },
  parameters: {
    docs: {
      description: {
        story:
          'Dense 5x5 grid of samples providing smoother heatmap interpolation. Signal strength decreases from the center outward.',
      },
    },
  },
};

/**
 * Metric toggle demonstration.
 * Switch between different heatmap visualizations.
 */
export const MetricToggle: Story = {
  render: () => {
    const [metric, setMetric] = useState<'rssi' | 'throughput' | 'latency' | null>('rssi');

    return (
      <div class={cn(spacing.pad.default, 'w-full max-w-4xl bg-surface-base')}>
        <div class={cn(spacing.margin.bottom.content, spacing.gap.compact, 'flex')}>
          <button
            type="button"
            onClick={() => setMetric(null)}
            class={cn(
              spacing.chip.sm,
              'rounded text-sm',
              metric === null
                ? 'bg-brand-primary text-text-inverse'
                : 'bg-surface-raised border border-surface-border hover:bg-surface-hover',
            )}
          >
            No Heatmap
          </button>
          <button
            type="button"
            onClick={() => setMetric('rssi')}
            class={cn(
              spacing.chip.sm,
              'rounded text-sm',
              metric === 'rssi'
                ? 'bg-brand-primary text-text-inverse'
                : 'bg-surface-raised border border-surface-border hover:bg-surface-hover',
            )}
          >
            RSSI
          </button>
          <button
            type="button"
            onClick={() => setMetric('throughput')}
            class={cn(
              spacing.chip.sm,
              'rounded text-sm',
              metric === 'throughput'
                ? 'bg-brand-primary text-text-inverse'
                : 'bg-surface-raised border border-surface-border hover:bg-surface-hover',
            )}
          >
            Throughput
          </button>
          <button
            type="button"
            onClick={() => setMetric('latency')}
            class={cn(
              spacing.chip.sm,
              'rounded text-sm',
              metric === 'latency'
                ? 'bg-brand-primary text-text-inverse'
                : 'bg-surface-raised border border-surface-border hover:bg-surface-hover',
            )}
          >
            Latency
          </button>
        </div>
        <FloorPlanCanvas
          floorPlan={sampleFloorPlan}
          samples={throughputSamples}
          interactive={false}
          heatmapMetric={metric}
        />
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story: 'Toggle between different heatmap metrics or disable heatmap overlay entirely.',
      },
    },
  },
};

/**
 * Single sample point.
 * Minimal example with one measurement.
 */
export const SingleSample: Story = {
  args: {
    floorPlan: sampleFloorPlan,
    samples: [passiveSamples[0]],
    interactive: false,
    heatmapMetric: null,
  },
};

/**
 * Weak signal corner visualization.
 * Shows dead zone in bottom-right corner.
 */
export const WeakSignalCorner: Story = {
  args: {
    floorPlan: sampleFloorPlan,
    samples: [
      ...passiveSamples,
      {
        x: 700,
        y: 500,
        timestamp: new Date().toISOString(),
        sampleData: {
          networks: [
            {
              ssid: 'OfficeMain',
              bssid: 'AA:BB:CC:DD:EE:01',
              rssi: -85,
              channel: 6,
              frequency: 2437,
            },
          ],
        } as PassiveSample,
      },
    ],
    interactive: false,
    heatmapMetric: 'rssi',
  },
  parameters: {
    docs: {
      description: {
        story: 'Heatmap showing a dead zone in the bottom-right corner with -85 dBm signal.',
      },
    },
  },
};
