import type { Meta, StoryObj } from '@storybook/react-vite';
import { ReportDialog } from './ReportDialog';

const meta = {
  title: 'Survey/ReportDialog',
  component: ReportDialog,
  parameters: { layout: 'fullscreen' },
} satisfies Meta<typeof ReportDialog>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Open: Story = {
  args: {
    surveyId: 'survey-1',
    surveyName: 'Office Coverage Study',
    open: true,
    onClose: () => {},
  },
};
