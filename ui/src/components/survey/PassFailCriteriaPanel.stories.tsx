import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import type { PassFailCriterion } from '../../hooks/useSurvey';
import { DEFAULT_PASSIVE_CRITERIA } from '../../hooks/useSurvey';
import { PassFailCriteriaPanel } from './PassFailCriteriaPanel';

const meta = {
  title: 'Survey/PassFailCriteriaPanel',
  component: PassFailCriteriaPanel,
} satisfies Meta<typeof PassFailCriteriaPanel>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Passive: Story = {
  render: () => {
    const [criteria, setCriteria] = useState<PassFailCriterion[]>(DEFAULT_PASSIVE_CRITERIA);
    return (
      <PassFailCriteriaPanel
        surveyType="passive"
        criteria={criteria}
        onChange={setCriteria}
        onValidate={() => {}}
        onImportFromAirMapper={() => {}}
      />
    );
  },
};
