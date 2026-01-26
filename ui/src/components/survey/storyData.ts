import type { SamplePoint, Floor, ApLocation, Survey, SurveyValidation } from '../../hooks/useSurvey';
import type { SurveyReport } from '../../utils/reportGenerator';

export const sampleFloors: Floor[] = [
  {
    id: 'floor-1',
    name: 'Main Floor',
    level: 1,
    floorPlan: {
      imageData: '',
      width: 800,
      height: 600,
      scaleM: 0.1,
    },
  },
  {
    id: 'floor-2',
    name: 'Upper Floor',
    level: 2,
    floorPlan: {
      imageData: '',
      width: 800,
      height: 600,
      scaleM: 0.1,
    },
  },
];

export const samplePassiveSamples: SamplePoint[] = [
  {
    x: 120,
    y: 180,
    timestamp: new Date().toISOString(),
    sampleData: {
      networks: [
        { ssid: 'OfficeMain', bssid: 'AA:BB:CC:DD:EE:01', rssi: -45, channel: 6, frequency: 2437 },
        { ssid: 'Guest', bssid: 'AA:BB:CC:DD:EE:02', rssi: -58, channel: 11, frequency: 2462 },
      ],
    },
  },
  {
    x: 420,
    y: 300,
    timestamp: new Date().toISOString(),
    sampleData: {
      networks: [
        { ssid: 'OfficeMain', bssid: 'AA:BB:CC:DD:EE:01', rssi: -52, channel: 6, frequency: 2437 },
        { ssid: 'Neighbor_5G', bssid: '11:22:33:44:55:66', rssi: -72, channel: 36, frequency: 5180 },
      ],
    },
  },
];

export const sampleApLocations: ApLocation[] = [
  {
    id: 'ap-1',
    label: 'AP-01',
    bssid: 'AA:BB:CC:DD:EE:01',
    ssids: ['OfficeMain', 'Guest'],
    band: '5',
    channel: 36,
    model: 'UniFi U6',
    x: 160,
    y: 140,
  },
  {
    id: 'ap-2',
    label: 'AP-02',
    bssid: 'AA:BB:CC:DD:EE:02',
    ssids: ['OfficeMain'],
    band: '2.4',
    channel: 6,
    model: 'UniFi U6',
    x: 520,
    y: 320,
  },
];

export const sampleValidation: SurveyValidation = {
  overallPass: false,
  overallPercentage: 76,
  passedCount: 2,
  failedCount: 1,
  timestamp: new Date().toISOString(),
  surveyId: 'survey-1',
  criteria: [
    {
      id: 'primary-signal',
      name: 'primarySignal',
      displayKey: 'criteria.primarySignal',
      metric: 'rssi',
      comparison: 'gte',
      threshold: -65,
      suffix: 'dBm',
      enabled: true,
      mode: 'passive',
      apIndex: 0,
    },
  ],
  results: [
    {
      criterionId: 'primary-signal',
      criterionName: 'primarySignal',
      passed: false,
      averageValue: -70,
      worstValue: -82,
      bestValue: -48,
      threshold: -65,
      comparison: 'gte',
      suffix: 'dBm',
      failedSampleCount: 2,
      totalSampleCount: 5,
      failedLocations: [
        { x: 120, y: 180, value: -78 },
        { x: 420, y: 300, value: -82 },
      ],
      percentage: 60,
    },
  ],
};

export const sampleSurvey: Survey = {
  id: 'survey-1',
  name: 'Office Coverage Study',
  description: 'Passive scan of all available networks',
  surveyType: 'passive',
  status: 'in_progress',
  createdAt: new Date().toISOString(),
  updatedAt: new Date().toISOString(),
  interface: 'wlan0',
  floors: sampleFloors.map((floor, idx) => ({
    ...floor,
    samples: idx === 0 ? samplePassiveSamples : [],
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
  })),
  activeFloorId: sampleFloors[0]?.id,
  apLocations: sampleApLocations,
  lastValidation: sampleValidation,
};

export const sampleReport: SurveyReport = {
  metadata: {
    title: 'Office Survey Report',
    surveyName: 'Office Coverage Study',
    surveyId: 'survey-1',
    generatedAt: new Date().toISOString(),
    generatedBy: 'Seed',
    surveyType: 'passive',
    sampleCount: samplePassiveSamples.length,
    facilitySize: '10,000 sq ft',
    scaleInfo: '1px = 0.1m',
  },
  summary: {
    overallStatus: 'pass',
    passedCriteria: 8,
    totalCriteria: 10,
    overallPercentage: 80,
    keyFindings: ['Strong coverage in main areas', 'Weak signal near storage'],
  },
  validation: null,
  heatmaps: [],
  analysis: [],
  apInventory: [],
  recommendations: ['Add AP near storage area', 'Adjust channel plan to reduce co-channel'],
};
