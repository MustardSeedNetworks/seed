import type { Meta, StoryObj } from '@storybook/react-vite';
import { SurveyAnalysisPanel } from './SurveyAnalysisPanel';
import { sampleSurvey } from './storyData';

const meta = {
  title: 'Survey/SurveyAnalysisPanel',
  component: SurveyAnalysisPanel,
} satisfies Meta<typeof SurveyAnalysisPanel>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    survey: sampleSurvey,
    onFindingClick: () => {},
    onLocationClick: () => {},
    onGenerateReport: () => {},
  },
};
