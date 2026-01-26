import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import type { CardSettings, LinkSettings as LinkSettingsType } from '../../../types/settings';
import { DEFAULT_CARD_SETTINGS, DEFAULT_LINK_SETTINGS } from '../../../types/settings';
import { LinkSettings } from './LinkSettings';

const meta = {
  title: 'Settings/LinkSettings',
  component: LinkSettings,
} satisfies Meta<typeof LinkSettings>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => {
    const [linkSettings, setLinkSettings] = useState<LinkSettingsType>({
      ...DEFAULT_LINK_SETTINGS,
      availableModes: ['auto', '100/full', '1000/full'],
    });
    const [cardSettings, setCardSettings] = useState<CardSettings>(DEFAULT_CARD_SETTINGS);
    return (
      <LinkSettings
        linkSettings={linkSettings}
        setLinkSettings={setLinkSettings}
        linkStatus="saved"
        cardSettings={cardSettings}
        updateCardSettings={(updates) => setCardSettings((prev) => ({ ...prev, ...updates }))}
      />
    );
  },
};
