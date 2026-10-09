/**
 * DiscoveryCustomOptions must not write settings on mount. It used to copy
 * the preset's port lists into settings from an effect, so expanding the
 * Discovery section armed the drawer's auto-save PUT with nothing changed,
 * and any caller passing a fresh setter each render looped forever.
 */

import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import type { NetworkDiscoverySettings } from '../../../../types/settings';
import { DEFAULT_NETWORK_DISCOVERY_SETTINGS } from '../../../../types/settings';
import { DiscoveryCustomOptions } from './DiscoveryCustomOptions';

describe('DiscoveryCustomOptions', () => {
  it.each(['common', 'secure', 'custom'] as const)(
    'leaves %s preset settings untouched on mount',
    (preset) => {
      const settings: NetworkDiscoverySettings = {
        ...DEFAULT_NETWORK_DISCOVERY_SETTINGS,
        options: {
          ...DEFAULT_NETWORK_DISCOVERY_SETTINGS.options,
          portScan: { enabled: true, preset, tcpPorts: '2222' },
        },
      };
      const onSettingsChange = vi.fn();

      render(<DiscoveryCustomOptions settings={settings} onSettingsChange={onSettingsChange} />);

      expect(onSettingsChange).not.toHaveBeenCalled();
      expect(screen.getByDisplayValue('2222')).toBeInTheDocument();
    },
  );
});
