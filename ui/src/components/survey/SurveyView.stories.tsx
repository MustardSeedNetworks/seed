import type { Meta, StoryObj } from '@storybook/react-vite';
import type { ActiveSample, PassiveSample, ThroughputSample } from '../../hooks/useSurvey';
import { SurveyView } from './SurveyView';

// No-op function for story event handlers
const noop = (): void => {
  // intentionally empty
};

/**
 * SurveyView is the full-screen WiFi survey editor and viewer.
 *
 * Features:
 * - Floor plan canvas with interactive sampling
 * - Upload custom floor plan images
 * - Click to add sample points at physical locations
 * - Heatmap visualization (RSSI, throughput, latency)
 * - Sample list with detailed metrics
 * - Survey controls (start, pause, complete)
 * - Real-time sample collection
 * - Support for passive, active, and throughput survey types
 *
 * This story demonstrates all survey view states and visualizations.
 */
const meta: Meta<typeof SurveyView> = {
  title: 'Survey/SurveyView',
  component: SurveyView,
  parameters: {
    layout: 'fullscreen',
  },
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof SurveyView>;

// Base64 encoded simple floor plan SVG for demos
const SAMPLE_FLOOR_PLAN =
  'data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iODAwIiBoZWlnaHQ9IjYwMCIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj48cmVjdCB3aWR0aD0iODAwIiBoZWlnaHQ9IjYwMCIgZmlsbD0iI2Y5ZmFmYiIvPjxyZWN0IHg9IjUwIiB5PSI1MCIgd2lkdGg9IjcwMCIgaGVpZ2h0PSI1MDAiIGZpbGw9Im5vbmUiIHN0cm9rZT0iIzMzMyIgc3Ryb2tlLXdpZHRoPSIzIi8+PHJlY3QgeD0iNTAiIHk9IjUwIiB3aWR0aD0iMjAwIiBoZWlnaHQ9IjE1MCIgZmlsbD0ibm9uZSIgc3Ryb2tlPSIjOTk5IiBzdHJva2Utd2lkdGg9IjIiLz48cmVjdCB4PSI1NTAiIHk9IjUwIiB3aWR0aD0iMjAwIiBoZWlnaHQ9IjE1MCIgZmlsbD0ibm9uZSIgc3Ryb2tlPSIjOTk5IiBzdHJva2Utd2lkdGg9IjIiLz48cmVjdCB4PSI1MCIgeT0iNDAwIiB3aWR0aD0iMjAwIiBoZWlnaHQ9IjE1MCIgZmlsbD0ibm9uZSIgc3Ryb2tlPSIjOTk5IiBzdHJva2Utd2lkdGg9IjIiLz48cmVjdCB4PSI1NTAiIHk9IjQwMCIgd2lkdGg9IjIwMCIgaGVpZ2h0PSIxNTAiIGZpbGw9Im5vbmUiIHN0cm9rZT0iIzk5OSIgc3Ryb2tlLXdpZHRoPSIyIi8+PHRleHQgeD0iNDAwIiB5PSI1MCIgdGV4dC1hbmNob3I9Im1pZGRsZSIgZm9udC1zaXplPSIyNCIgZmlsbD0iIzY2NiIgZm9udC13ZWlnaHQ9ImJvbGQiPk9mZmljZSBGbG9vciBQbGFuPC90ZXh0Pjx0ZXh0IHg9IjE1MCIgeT0iMTMwIiB0ZXh0LWFuY2hvcj0ibWlkZGxlIiBmb250LXNpemU9IjE0IiBmaWxsPSIjOTk5Ij5Db25mZXJlbmNlPC90ZXh0Pjx0ZXh0IHg9IjY1MCIgeT0iMTMwIiB0ZXh0LWFuY2hvcj0ibWlkZGxlIiBmb250LXNpemU9IjE0IiBmaWxsPSIjOTk5Ij5PZmZpY2VzPC90ZXh0Pjx0ZXh0IHg9IjQwMCIgeT0iMzAwIiB0ZXh0LWFuY2hvcj0ibWlkZGxlIiBmb250LXNpemU9IjE4IiBmaWxsPSIjY2NjIj5PcGVuIFdvcmtzcGFjZTwvdGV4dD48dGV4dCB4PSIxNTAiIHk9IjQ4MCIgdGV4dC1hbmNob3I9Im1pZGRsZSIgZm9udC1zaXplPSIxNCIgZmlsbD0iIzk5OSI+S2l0Y2hlbjwvdGV4dD48dGV4dCB4PSI2NTAiIHk9IjQ4MCIgdGV4dC1hbmNob3I9Im1pZGRsZSIgZm9udC1zaXplPSIxNCIgZmlsbD0iIzk5OSI+U3RvcmFnZTwvdGV4dD48L3N2Zz4=';

/**
 * Empty survey without floor plan.
 * Shows upload prompt to begin survey setup.
 */
export const NoFloorPlan: Story = {
  args: {
    survey: {
      id: 'survey-1',
      name: 'Office Coverage Study',
      description: 'Main office WiFi coverage analysis',
      surveyType: 'passive',
      status: 'created',
      createdAt: '2025-12-15T10:00:00Z',
      updatedAt: '2025-12-15T10:00:00Z',
      samples: [],
      interface: 'wlan0',
    },
    onClose: noop,
    onUpdate: noop,
  },
};

/**
 * Survey with floor plan but no samples.
 * Ready to start collecting data.
 */
export const FloorPlanNoSamples: Story = {
  args: {
    survey: {
      id: 'survey-2',
      name: 'Warehouse Coverage',
      description: 'Warehouse WiFi mapping',
      surveyType: 'passive',
      status: 'created',
      createdAt: '2025-12-15T09:00:00Z',
      updatedAt: '2025-12-15T09:30:00Z',
      samples: [],
      interface: 'wlan0',
      floorPlan: {
        imageData: SAMPLE_FLOOR_PLAN,
        width: 800,
        height: 600,
        scaleM: 0.1,
      },
    },
    onClose: noop,
    onUpdate: noop,
  },
};

/**
 * Passive survey in progress with sample points.
 * Shows multiple networks detected at each location.
 */
export const PassiveSurveyWithSamples: Story = {
  args: {
    survey: {
      id: 'survey-3',
      name: 'Office Coverage Study',
      description: 'Passive scan of all available networks',
      surveyType: 'passive',
      status: 'in_progress',
      createdAt: '2025-12-15T08:00:00Z',
      updatedAt: '2025-12-15T10:30:00Z',
      samples: [
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
              {
                ssid: 'Neighbor_5G',
                bssid: '11:22:33:44:55:66',
                rssi: -72,
                channel: 36,
                frequency: 5180,
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
              {
                ssid: 'Guest',
                bssid: 'AA:BB:CC:DD:EE:02',
                rssi: -45,
                channel: 11,
                frequency: 2462,
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
                rssi: -48,
                channel: 6,
                frequency: 2437,
              },
              {
                ssid: 'Guest',
                bssid: 'AA:BB:CC:DD:EE:04',
                rssi: -52,
                channel: 11,
                frequency: 2462,
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
                rssi: -65,
                channel: 6,
                frequency: 2437,
              },
              {
                ssid: 'Guest',
                bssid: 'AA:BB:CC:DD:EE:02',
                rssi: -70,
                channel: 11,
                frequency: 2462,
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
                rssi: -55,
                channel: 6,
                frequency: 2437,
              },
            ],
          } as PassiveSample,
        },
      ],
      interface: 'wlan0',
      floorPlan: {
        imageData: SAMPLE_FLOOR_PLAN,
        width: 800,
        height: 600,
        scaleM: 0.1,
      },
    },
    onClose: noop,
    onUpdate: noop,
  },
};

/**
 * Active survey monitoring connected network.
 * Shows RSSI, data rate, and roaming events.
 */
export const ActiveSurveyWithRoaming: Story = {
  args: {
    survey: {
      id: 'survey-4',
      name: 'Roaming Test',
      description: 'Testing handoff between access points',
      surveyType: 'active',
      status: 'in_progress',
      createdAt: '2025-12-15T11:00:00Z',
      updatedAt: '2025-12-15T11:45:00Z',
      samples: [
        {
          x: 100,
          y: 100,
          timestamp: '2025-12-15T11:00:00Z',
          sampleData: {
            ssid: 'OfficeMain',
            bssid: 'AA:BB:CC:DD:EE:01',
            rssi: -45,
            dataRate: 866,
            roamingEvent: false,
          } as ActiveSample,
        },
        {
          x: 200,
          y: 150,
          timestamp: '2025-12-15T11:10:00Z',
          sampleData: {
            ssid: 'OfficeMain',
            bssid: 'AA:BB:CC:DD:EE:01',
            rssi: -52,
            dataRate: 650,
            roamingEvent: false,
          } as ActiveSample,
        },
        {
          x: 300,
          y: 200,
          timestamp: '2025-12-15T11:20:00Z',
          sampleData: {
            ssid: 'OfficeMain',
            bssid: 'AA:BB:CC:DD:EE:02',
            rssi: -48,
            dataRate: 866,
            roamingEvent: true,
          } as ActiveSample,
        },
        {
          x: 400,
          y: 250,
          timestamp: '2025-12-15T11:30:00Z',
          sampleData: {
            ssid: 'OfficeMain',
            bssid: 'AA:BB:CC:DD:EE:02',
            rssi: -42,
            dataRate: 866,
            roamingEvent: false,
          } as ActiveSample,
        },
        {
          x: 500,
          y: 300,
          timestamp: '2025-12-15T11:40:00Z',
          sampleData: {
            ssid: 'OfficeMain',
            bssid: 'AA:BB:CC:DD:EE:03',
            rssi: -50,
            dataRate: 650,
            roamingEvent: true,
          } as ActiveSample,
        },
      ],
      interface: 'wlan0',
      floorPlan: {
        imageData: SAMPLE_FLOOR_PLAN,
        width: 800,
        height: 600,
        scaleM: 0.1,
      },
    },
    onClose: noop,
    onUpdate: noop,
  },
};

/**
 * Throughput survey with iperf3 results.
 * Shows download/upload speeds, latency, jitter, and packet loss.
 */
export const ThroughputSurvey: Story = {
  args: {
    survey: {
      id: 'survey-5',
      name: 'Performance Testing',
      description: 'iperf3 throughput mapping',
      surveyType: 'throughput',
      status: 'in_progress',
      createdAt: '2025-12-15T13:00:00Z',
      updatedAt: '2025-12-15T14:30:00Z',
      samples: [
        {
          x: 150,
          y: 120,
          timestamp: '2025-12-15T13:30:00Z',
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
          timestamp: '2025-12-15T13:40:00Z',
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
          timestamp: '2025-12-15T13:50:00Z',
          sampleData: {
            ssid: 'OfficeMain',
            bssid: 'AA:BB:CC:DD:EE:03',
            rssi: -48,
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
          timestamp: '2025-12-15T14:00:00Z',
          sampleData: {
            ssid: 'OfficeMain',
            bssid: 'AA:BB:CC:DD:EE:01',
            rssi: -65,
            downloadMbps: 145.2,
            uploadMbps: 98.7,
            latency: 35,
            jitter: 5.2,
            packetLoss: 1.2,
          } as ThroughputSample,
        },
      ],
      interface: 'wlan0',
      iperfServer: '192.168.1.100:5201',
      testDuration: 5,
      floorPlan: {
        imageData: SAMPLE_FLOOR_PLAN,
        width: 800,
        height: 600,
        scaleM: 0.1,
      },
    },
    onClose: noop,
    onUpdate: noop,
  },
};

/**
 * Paused survey with resume controls.
 * Shows survey paused mid-collection with resume/complete options.
 */
export const PausedSurvey: Story = {
  args: {
    survey: {
      id: 'survey-6',
      name: 'Conference Room',
      description: 'Meeting room coverage',
      surveyType: 'passive',
      status: 'paused',
      createdAt: '2025-12-14T14:00:00Z',
      updatedAt: '2025-12-15T10:45:00Z',
      samples: [
        {
          x: 200,
          y: 200,
          timestamp: '2025-12-15T10:30:00Z',
          sampleData: {
            networks: [
              {
                ssid: 'ConfRoom',
                bssid: '11:22:33:44:55:66',
                rssi: -40,
                channel: 36,
                frequency: 5180,
              },
            ],
          } as PassiveSample,
        },
        {
          x: 400,
          y: 300,
          timestamp: '2025-12-15T10:35:00Z',
          sampleData: {
            networks: [
              {
                ssid: 'ConfRoom',
                bssid: '11:22:33:44:55:66',
                rssi: -38,
                channel: 36,
                frequency: 5180,
              },
            ],
          } as PassiveSample,
        },
      ],
      interface: 'wlan0',
      floorPlan: {
        imageData: SAMPLE_FLOOR_PLAN,
        width: 800,
        height: 600,
        scaleM: 0.1,
      },
    },
    onClose: noop,
    onUpdate: noop,
  },
};

/**
 * Completed survey ready for analysis.
 * Shows full sample set with complete status.
 */
export const CompletedSurvey: Story = {
  args: {
    survey: {
      id: 'survey-7',
      name: 'Final Coverage Report',
      description: 'Complete office coverage analysis',
      surveyType: 'passive',
      status: 'completed',
      createdAt: '2025-12-10T08:00:00Z',
      updatedAt: '2025-12-10T12:30:00Z',
      samples: Array.from({ length: 25 }, (_, i) => ({
        x: 100 + (i % 5) * 150,
        y: 100 + Math.floor(i / 5) * 100,
        timestamp: new Date(Date.now() - (25 - i) * 300000).toISOString(),
        sampleData: {
          networks: [
            {
              ssid: 'OfficeMain',
              bssid: 'AA:BB:CC:DD:EE:01',
              rssi: -40 - Math.random() * 30,
              channel: 6,
              frequency: 2437,
            },
          ],
        } as PassiveSample,
      })),
      interface: 'wlan0',
      floorPlan: {
        imageData: SAMPLE_FLOOR_PLAN,
        width: 800,
        height: 600,
        scaleM: 0.1,
      },
    },
    onClose: noop,
    onUpdate: noop,
  },
};

/**
 * Survey with RSSI heatmap enabled.
 * Shows signal strength visualization across floor plan.
 */
export const WithRssiHeatmap: Story = {
  args: {
    survey: {
      id: 'survey-8',
      name: 'Signal Strength Map',
      description: 'RSSI heatmap visualization',
      surveyType: 'passive',
      status: 'completed',
      createdAt: '2025-12-10T08:00:00Z',
      updatedAt: '2025-12-10T12:30:00Z',
      samples: Array.from({ length: 20 }, (_, i) => ({
        x: 100 + (i % 4) * 200,
        y: 100 + Math.floor(i / 4) * 100,
        timestamp: new Date(Date.now() - (20 - i) * 300000).toISOString(),
        sampleData: {
          networks: [
            {
              ssid: 'OfficeMain',
              bssid: 'AA:BB:CC:DD:EE:01',
              rssi: -35 - (i % 4) * 10,
              channel: 6,
              frequency: 2437,
            },
          ],
        } as PassiveSample,
      })),
      interface: 'wlan0',
      floorPlan: {
        imageData: SAMPLE_FLOOR_PLAN,
        width: 800,
        height: 600,
        scaleM: 0.1,
      },
    },
    onClose: noop,
    onUpdate: noop,
  },
  parameters: {
    docs: {
      description: {
        story: 'Click "RSSI Heatmap" button to visualize signal strength distribution.',
      },
    },
  },
};

/**
 * Survey currently taking a sample.
 * Shows sampling in progress indicator.
 */
export const SamplingInProgress: Story = {
  args: {
    survey: {
      id: 'survey-9',
      name: 'Active Sampling',
      description: 'Currently collecting sample',
      surveyType: 'passive',
      status: 'in_progress',
      createdAt: '2025-12-15T14:00:00Z',
      updatedAt: '2025-12-15T14:30:00Z',
      samples: [
        {
          x: 200,
          y: 200,
          timestamp: '2025-12-15T14:25:00Z',
          sampleData: {
            networks: [
              {
                ssid: 'TestNet',
                bssid: 'AA:BB:CC:DD:EE:FF',
                rssi: -50,
                channel: 6,
                frequency: 2437,
              },
            ],
          } as PassiveSample,
        },
      ],
      interface: 'wlan0',
      floorPlan: {
        imageData: SAMPLE_FLOOR_PLAN,
        width: 800,
        height: 600,
        scaleM: 0.1,
      },
    },
    onClose: noop,
    onUpdate: noop,
  },
  parameters: {
    docs: {
      description: {
        story: 'Click on the floor plan to trigger sample collection.',
      },
    },
  },
};

/**
 * Survey with error state.
 * Shows error message when operation fails.
 */
export const WithError: Story = {
  args: {
    survey: {
      id: 'survey-10',
      name: 'Error Example',
      description: 'Survey with error state',
      surveyType: 'passive',
      status: 'in_progress',
      createdAt: '2025-12-15T14:00:00Z',
      updatedAt: '2025-12-15T14:30:00Z',
      samples: [],
      interface: 'wlan0',
      floorPlan: {
        imageData: SAMPLE_FLOOR_PLAN,
        width: 800,
        height: 600,
        scaleM: 0.1,
      },
    },
    onClose: noop,
    onUpdate: noop,
  },
  parameters: {
    docs: {
      description: {
        story: 'Shows how errors are displayed during survey operations.',
      },
    },
  },
};
