import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import type { SurveyConfig, SurveyType } from '../../hooks/useSurvey';
import { SurveyConfigPanel } from './SurveyConfigPanel';

const meta = {
  title: 'Survey/SurveyConfigPanel',
  component: SurveyConfigPanel,
} satisfies Meta<typeof SurveyConfigPanel>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Passive: Story = {
  render: () => {
    const [config, setConfig] = useState<SurveyConfig>({
      bands: ['2.4', '5'],
      ssidFilter: 'OfficeMain',
    });
    return (
      <SurveyConfigPanel
        config={config}
        surveyType="passive"
        availableAdapters={['wlan0', 'wlan1']}
        currentInterface="wlan0"
        onUpdate={(partial) => setConfig((prev) => ({ ...prev, ...partial }))}
        onSurveyTypeChange={() => {}}
      />
    );
  },
};

export const Throughput: Story = {
  render: () => {
    const [type, setType] = useState<SurveyType>('throughput');
    const [config, setConfig] = useState<SurveyConfig>({
      bands: ['5'],
    });
    return (
      <SurveyConfigPanel
        config={config}
        surveyType={type}
        availableAdapters={['wlan0']}
        currentInterface="wlan0"
        iperfServer="10.0.0.10"
        testDuration={5}
        onUpdate={(partial) => setConfig((prev) => ({ ...prev, ...partial }))}
        onSurveyTypeChange={(next) => setType(next)}
        onIperfSettingsChange={() => {}}
      />
    );
  },
};
