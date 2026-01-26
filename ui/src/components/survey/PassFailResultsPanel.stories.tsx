import type { Meta, StoryObj } from '@storybook/react-vite';
import { PassFailResultsPanel } from './PassFailResultsPanel';
import { sampleValidation } from './storyData';

const meta = {
  title: 'Survey/PassFailResultsPanel',
  component: PassFailResultsPanel,
} satisfies Meta<typeof PassFailResultsPanel>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    validation: sampleValidation,
    onLocationClick: () => {},
    onShowFailedLocations: () => {},
    onGenerateReport: () => {},
    onExportCsv: () => {},
  },
};
