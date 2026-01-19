import type { Meta, StoryObj } from '@storybook/react-vite';
import type { Survey } from '../../hooks/useSurvey';
import { WiFiSurveyCard } from './WiFiSurveyCard';

/**
 * WiFiSurveyCard displays WiFi site survey management with floor plan-based signal mapping.
 *
 * Features:
 * - Create new surveys with different types (passive, active, throughput)
 * - Manage survey lifecycle (start, pause, resume, complete)
 * - View survey list with status indicators
 * - Upload floor plans for coordinate-based sampling
 * - Display sample counts and metadata
 *
 * This story demonstrates all survey states and interactions.
 */
const meta: Meta<typeof WiFiSurveyCard> = {
  title: 'Cards/WiFiSurveyCard',
  component: WiFiSurveyCard,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  decorators: [
    (StoryComponent: React.ComponentType): JSX.Element => (
      <div class="w-96">
        <StoryComponent />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof WiFiSurveyCard>;

/**
 * Empty state when no surveys exist yet.
 * Shows call-to-action to create first survey.
 */
export const NoSurveys: Story = {
  args: {
    isWifi: true,
  },
  parameters: {
    mockData: {
      surveys: [],
      loading: false,
      error: null,
    },
  },
};

/**
 * Loading state while fetching surveys from API.
 */
export const Loading: Story = {
  args: {
    isWifi: true,
  },
  parameters: {
    mockData: {
      surveys: [],
      loading: true,
      error: null,
    },
  },
};

/**
 * Error state when survey fetch fails.
 */
export const ErrorState: Story = {
  args: {
    isWifi: true,
  },
  parameters: {
    mockData: {
      surveys: [],
      loading: false,
      error: 'Failed to load surveys: Network error',
    },
  },
};

/**
 * Warning state when WiFi interface is not available.
 * User cannot create surveys without WiFi connection.
 */
export const NoWiFiInterface: Story = {
  args: {
    isWifi: false,
  },
  parameters: {
    mockData: {
      surveys: [],
      loading: false,
      error: null,
    },
  },
};

/**
 * Single survey in "created" state (not yet started).
 * Shows start button for beginning the survey.
 */
export const SurveyCreated: Story = {
  args: {
    isWifi: true,
  },
  parameters: {
    mockData: {
      surveys: [
        {
          id: 'survey-1',
          name: 'Office Floor 1',
          description: 'Main office coverage mapping',
          surveyType: 'passive',
          status: 'created',
          createdAt: '2025-12-15T10:00:00Z',
          updatedAt: '2025-12-15T10:00:00Z',
          samples: [],
          interface: 'wlan0',
        },
      ] as Survey[],
      loading: false,
      error: null,
    },
  },
};

/**
 * Survey actively in progress.
 * Shows pause and complete buttons, warning status indicator.
 */
export const SurveyInProgress: Story = {
  args: {
    isWifi: true,
  },
  parameters: {
    mockData: {
      surveys: [
        {
          id: 'survey-2',
          name: 'Warehouse Coverage',
          description: 'Warehouse WiFi signal mapping',
          surveyType: 'passive',
          status: 'in_progress',
          createdAt: '2025-12-15T09:00:00Z',
          updatedAt: '2025-12-15T11:30:00Z',
          samples: Array.from({ length: 12 }, (_, i) => ({
            x: 100 + i * 50,
            y: 100 + i * 30,
            timestamp: new Date(Date.now() - (12 - i) * 60000).toISOString(),
            sampleData: {
              networks: [
                {
                  ssid: 'WarehouseWiFi',
                  bssid: 'AA:BB:CC:DD:EE:FF',
                  rssi: -45 - i * 2,
                  channel: 6,
                  frequency: 2437,
                },
              ],
            },
          })),
          interface: 'wlan0',
        },
      ] as Survey[],
      loading: false,
      error: null,
    },
  },
};

/**
 * Survey paused mid-collection.
 * Shows resume and complete buttons.
 */
export const SurveyPaused: Story = {
  args: {
    isWifi: true,
  },
  parameters: {
    mockData: {
      surveys: [
        {
          id: 'survey-3',
          name: 'Conference Room',
          description: 'Meeting space coverage test',
          surveyType: 'active',
          status: 'paused',
          createdAt: '2025-12-14T14:00:00Z',
          updatedAt: '2025-12-15T10:45:00Z',
          samples: Array.from({ length: 8 }, (_, i) => ({
            x: 150 + i * 40,
            y: 200 + i * 25,
            timestamp: new Date(Date.now() - (8 - i) * 120000).toISOString(),
            sampleData: {
              ssid: 'ConfRoom-5G',
              bssid: '11:22:33:44:55:66',
              rssi: -50 - i * 3,
              dataRate: 866 - i * 50,
              roamingEvent: i === 4,
            },
          })),
          interface: 'wlan0',
        },
      ] as Survey[],
      loading: false,
      error: null,
    },
  },
};

/**
 * Completed survey with full sample set.
 * Shows success status, view/export options.
 */
export const SurveyCompleted: Story = {
  args: {
    isWifi: true,
  },
  parameters: {
    mockData: {
      surveys: [
        {
          id: 'survey-4',
          name: 'Retail Store Coverage',
          description: 'Customer area signal quality',
          surveyType: 'throughput',
          status: 'completed',
          createdAt: '2025-12-10T08:00:00Z',
          updatedAt: '2025-12-10T12:30:00Z',
          samples: Array.from({ length: 25 }, (_, i) => ({
            x: 50 + (i % 5) * 80,
            y: 50 + Math.floor(i / 5) * 60,
            timestamp: new Date(Date.now() - (25 - i) * 300000).toISOString(),
            sampleData: {
              ssid: 'RetailGuest',
              bssid: 'AA:11:BB:22:CC:33',
              rssi: -40 - Math.random() * 30,
              downloadMbps: 100 + Math.random() * 400,
              uploadMbps: 50 + Math.random() * 200,
              latency: 10 + Math.random() * 40,
              jitter: Math.random() * 5,
              packetLoss: Math.random() * 2,
            },
          })),
          interface: 'wlan0',
          iperfServer: '192.168.1.100:5201',
          testDuration: 5,
        },
      ] as Survey[],
      loading: false,
      error: null,
    },
  },
};

/**
 * Multiple surveys with different states.
 * Shows survey list management (max 3 shown, "+N more" indicator).
 */
export const MultipleSurveys: Story = {
  args: {
    isWifi: true,
  },
  parameters: {
    mockData: {
      surveys: [
        {
          id: 'survey-5',
          name: 'Office Floor 2',
          surveyType: 'passive',
          status: 'in_progress',
          createdAt: '2025-12-15T14:00:00Z',
          updatedAt: '2025-12-15T14:45:00Z',
          samples: Array.from({ length: 6 }, () => ({
            x: 100,
            y: 100,
            timestamp: new Date().toISOString(),
            sampleData: { networks: [] },
          })),
          interface: 'wlan0',
        },
        {
          id: 'survey-6',
          name: 'Lobby Area',
          surveyType: 'active',
          status: 'paused',
          createdAt: '2025-12-15T13:00:00Z',
          updatedAt: '2025-12-15T13:30:00Z',
          samples: Array.from({ length: 4 }, () => ({
            x: 100,
            y: 100,
            timestamp: new Date().toISOString(),
            sampleData: {
              ssid: 'Lobby',
              bssid: 'AA:BB:CC:DD:EE:FF',
              rssi: -50,
              dataRate: 300,
              roamingEvent: false,
            },
          })),
          interface: 'wlan0',
        },
        {
          id: 'survey-7',
          name: 'Parking Lot',
          surveyType: 'passive',
          status: 'completed',
          createdAt: '2025-12-14T10:00:00Z',
          updatedAt: '2025-12-14T11:00:00Z',
          samples: Array.from({ length: 15 }, () => ({
            x: 100,
            y: 100,
            timestamp: new Date().toISOString(),
            sampleData: { networks: [] },
          })),
          interface: 'wlan0',
        },
        {
          id: 'survey-8',
          name: 'Cafeteria',
          surveyType: 'throughput',
          status: 'completed',
          createdAt: '2025-12-13T09:00:00Z',
          updatedAt: '2025-12-13T10:30:00Z',
          samples: Array.from({ length: 20 }, () => ({
            x: 100,
            y: 100,
            timestamp: new Date().toISOString(),
            sampleData: {
              ssid: 'Cafe',
              bssid: '11:22:33:44:55:66',
              rssi: -55,
              downloadMbps: 250,
              uploadMbps: 100,
              latency: 15,
              jitter: 2,
              packetLoss: 0.5,
            },
          })),
          interface: 'wlan0',
        },
      ] as Survey[],
      loading: false,
      error: null,
    },
  },
};

/**
 * Survey with floor plan preview.
 * Shows thumbnail of uploaded floor plan.
 */
export const SurveyWithFloorPlan: Story = {
  args: {
    isWifi: true,
  },
  parameters: {
    mockData: {
      surveys: [
        {
          id: 'survey-9',
          name: 'Data Center',
          description: 'Server room coverage analysis',
          surveyType: 'passive',
          status: 'in_progress',
          createdAt: '2025-12-15T08:00:00Z',
          updatedAt: '2025-12-15T12:00:00Z',
          samples: Array.from({ length: 18 }, () => ({
            x: 100,
            y: 100,
            timestamp: new Date().toISOString(),
            sampleData: { networks: [] },
          })),
          interface: 'wlan0',
          floorPlan: {
            imageData:
              'data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iNDAwIiBoZWlnaHQ9IjMwMCIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj48cmVjdCB3aWR0aD0iNDAwIiBoZWlnaHQ9IjMwMCIgZmlsbD0iI2Y1ZjVmNSIvPjxyZWN0IHg9IjIwIiB5PSIyMCIgd2lkdGg9IjM2MCIgaGVpZ2h0PSIyNjAiIGZpbGw9Im5vbmUiIHN0cm9rZT0iIzMzMyIgc3Ryb2tlLXdpZHRoPSIyIi8+PHRleHQgeD0iMjAwIiB5PSIxNTAiIHRleHQtYW5jaG9yPSJtaWRkbGUiIGZvbnQtc2l6ZT0iMjQiIGZpbGw9IiM2NjYiPkRhdGEgQ2VudGVyPC90ZXh0Pjwvc3ZnPg==',
            width: 400,
            height: 300,
            scaleM: 0.1,
          },
        },
      ] as Survey[],
      loading: false,
      error: null,
    },
  },
};

/**
 * Create survey dialog open state.
 * Shows modal for creating a new survey with name and type selection.
 */
export const CreateDialogOpen: Story = {
  args: {
    isWifi: true,
  },
  parameters: {
    mockData: {
      surveys: [],
      loading: false,
      error: null,
      showCreateDialog: true,
    },
  },
};
