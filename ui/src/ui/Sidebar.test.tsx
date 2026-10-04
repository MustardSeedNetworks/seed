/**
 * Sidebar device name (#195): the operator's name for this Seed sits under the
 * product name in both bars, and nothing takes its place when none is set.
 */
import { render, screen } from '@testing-library/react';
import { Server } from 'lucide-react';
import { afterEach, describe, expect, it } from 'vitest';

import { SidebarLayout } from './Sidebar';

const groups = [{ label: 'g', items: [{ path: '/', label: 'x', icon: Server }] }];

describe('SidebarLayout device name', () => {
  afterEach(() => {
    localStorage.clear();
  });

  it('shows the name in the rail and the phone top bar', () => {
    render(
      <SidebarLayout groups={groups} deviceName="seed-idf-3b">
        <div />
      </SidebarLayout>,
    );

    expect(screen.getByTestId('rail-device-name')).toHaveTextContent('seed-idf-3b');
    expect(screen.getByTestId('topbar-device-name')).toHaveTextContent('seed-idf-3b');
  });

  it('shows no name line when none is set', () => {
    render(
      <SidebarLayout groups={groups} deviceName="">
        <div />
      </SidebarLayout>,
    );

    expect(screen.queryByTestId('rail-device-name')).not.toBeInTheDocument();
    expect(screen.queryByTestId('topbar-device-name')).not.toBeInTheDocument();
  });

  it('leaves the collapsed rail to the mark alone', () => {
    localStorage.setItem('seed-sidebar-collapsed', 'true');
    render(
      <SidebarLayout groups={groups} deviceName="seed-idf-3b">
        <div />
      </SidebarLayout>,
    );

    expect(screen.queryByTestId('rail-device-name')).not.toBeInTheDocument();
    expect(screen.getByTestId('topbar-device-name')).toHaveTextContent('seed-idf-3b');
  });
});
