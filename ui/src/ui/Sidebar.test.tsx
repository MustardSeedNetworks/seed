/**
 * Sidebar device name (#195): the operator's name for this Seed sits under the
 * product name in both bars, and nothing takes its place when none is set.
 */
import { render, screen } from '@testing-library/react';
import { Server } from 'lucide-react';
import { afterEach, describe, expect, it } from 'vitest';

import { type SidebarFooterConfig, SidebarLayout } from './Sidebar';

const groups = [{ label: 'g', items: [{ path: '/', label: 'x', icon: Server }] }];
const footer: SidebarFooterConfig = {
  help: { label: 'Help', onClick: () => undefined },
  settings: { label: 'Settings', onClick: () => undefined },
  theme: { label: 'Theme', onClick: () => undefined, isDark: true },
};

describe('SidebarLayout footer', () => {
  it('renders the shared rail footer with its fixed test ids', () => {
    render(
      <SidebarLayout
        groups={groups}
        footer={{ ...footer, signOut: { label: 'Sign out', onClick: () => undefined } }}
      >
        <div />
      </SidebarLayout>,
    );

    for (const id of ['rail-footer', 'rail-help', 'rail-settings', 'rail-theme', 'rail-sign-out']) {
      expect(screen.getByTestId(id)).toBeInTheDocument();
    }
    expect(screen.getByTestId('rail-mark')).toBeInTheDocument();
  });
});

describe('SidebarLayout device name', () => {
  afterEach(() => {
    localStorage.clear();
  });

  it('shows the name in the rail and the phone top bar', () => {
    render(
      <SidebarLayout groups={groups} footer={footer} deviceName="seed-idf-3b">
        <div />
      </SidebarLayout>,
    );

    expect(screen.getByTestId('rail-device-name')).toHaveTextContent('seed-idf-3b');
    expect(screen.getByTestId('topbar-device-name')).toHaveTextContent('seed-idf-3b');
  });

  it('shows no name line when none is set', () => {
    render(
      <SidebarLayout groups={groups} footer={footer} deviceName="">
        <div />
      </SidebarLayout>,
    );

    expect(screen.queryByTestId('rail-device-name')).not.toBeInTheDocument();
    expect(screen.queryByTestId('topbar-device-name')).not.toBeInTheDocument();
  });

  it('leaves the collapsed rail to the mark alone', () => {
    localStorage.setItem('seed-sidebar-collapsed', 'true');
    render(
      <SidebarLayout groups={groups} footer={footer} deviceName="seed-idf-3b">
        <div />
      </SidebarLayout>,
    );

    expect(screen.queryByTestId('rail-device-name')).not.toBeInTheDocument();
    expect(screen.getByTestId('topbar-device-name')).toHaveTextContent('seed-idf-3b');
  });
});
