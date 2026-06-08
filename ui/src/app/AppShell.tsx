/**
 * AppShell — the authenticated application UI.
 *
 * Pure presentation extracted from the former `App` god component (B1): it takes
 * the {@link AppOrchestration} bundle (produced by `useAppOrchestration` in
 * `App`) plus `logout`, assembles the AppContext value and top bar, and renders
 * the router, sidebar layout, routes, drawers, FAB, and command palette. It
 * holds no state of its own — all wiring lives in the orchestration hook.
 */

import type { JSX } from 'react';
import { Suspense } from 'react';
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom';
import { AppFooter } from '../components/app/AppFooter';
import { CapabilityWarnings } from '../components/app/CapabilityWarnings';
import { HeaderBar } from '../components/app/HeaderBar';
import { HelpDrawer } from '../components/help/HelpDrawer';
import { ProfileManagement } from '../components/profiles/ProfileManagement';
import { SettingsDrawer } from '../components/settings/SettingsDrawer';
import { CommandPalette } from '../components/ui/CommandPalette';
import { Fab } from '../components/ui/fab';
import { AppContext, type AppContextValue } from '../contexts/AppContext';
import { navGroups } from '../navGroups';
import { pages } from '../pageRegistry';
import { cn, section } from '../styles/theme';
import { PageLoader } from '../ui/PageLoader';
import { SidebarLayout } from '../ui/Sidebar';
import type { AppOrchestration } from './useAppOrchestration';

interface AppShellProps {
  orchestration: AppOrchestration;
  logout: () => void;
}

export function AppShell({ orchestration, logout }: AppShellProps): JSX.Element {
  const {
    cards,
    loading,
    isWifi,
    currentInterface,
    cardSettings,
    displayOptions,
    networkDiscovery,
    triggerDeviceScan,
    registerTraceHopHandler,
    channelGraphData,
    channelGraphLoading,
    appVersion,
    capabilities,
    sseStatus,
    reconnect,
    profiles,
    activeProfile,
    profilesLoading,
    switchProfile,
    interfaces,
    hasEthernet,
    hasWifiInterface,
    changeInterface,
    switchToInterfaceType,
    toggleTheme,
    isDark,
    recommendedEthernet,
    recommendedWifi,
    profilesOpen,
    settingsOpen,
    helpOpen,
    openProfiles,
    closeProfiles,
    openSettings,
    closeSettings,
    openHelp,
    closeHelp,
    paletteOpen,
    setPaletteOpen,
  } = orchestration;

  const appContextValue: AppContextValue = {
    cards,
    loading,
    isWifi,
    currentInterface,
    cardSettings,
    displayOptions,
    networkDiscovery,
    triggerDeviceScan,
    registerTraceHopHandler,
    channelGraphData,
    channelGraphLoading,
    appVersion,
  };

  const topBar = (
    <HeaderBar
      wsStatus={sseStatus}
      onReconnect={reconnect}
      profiles={profiles}
      activeProfile={activeProfile}
      profilesLoading={profilesLoading}
      onProfileSwitch={switchProfile}
      onProfileManage={openProfiles}
      interfaces={interfaces}
      currentInterface={currentInterface}
      isWifi={isWifi}
      onInterfaceChange={changeInterface}
      hasEthernet={hasEthernet}
      hasWifiInterface={hasWifiInterface}
      switchToInterfaceType={switchToInterfaceType}
      toggleTheme={toggleTheme}
      isDark={isDark}
      onHelpOpen={openHelp}
      onSettingsOpen={openSettings}
      logout={logout}
      recommendedEthernet={recommendedEthernet}
      recommendedWifi={recommendedWifi}
    />
  );

  return (
    <BrowserRouter>
      <AppContext.Provider value={appContextValue}>
        <SidebarLayout
          groups={navGroups}
          version={appVersion}
          onOpenHelp={openHelp}
          onOpenSettings={openSettings}
          onOpenProfiles={openProfiles}
          topBar={topBar}
        >
          <div className={cn(section.width.xl, 'mx-auto')}>
            <CapabilityWarnings capabilities={capabilities} />

            <Suspense fallback={<PageLoader />}>
              <Routes>
                <Route path="/" element={<Navigate to="/link" replace={true} />} />
                {pages.map((page) => (
                  <Route key={page.path} path={page.path} element={<page.component />} />
                ))}
                <Route path="*" element={<Navigate to="/link" replace={true} />} />
              </Routes>
            </Suspense>

            <AppFooter appVersion={appVersion} />
          </div>
        </SidebarLayout>

        {/* Settings Drawer - shows interface-specific settings (#754) */}
        <SettingsDrawer
          isOpen={settingsOpen}
          onClose={closeSettings}
          version={appVersion}
          isWifi={isWifi}
        />

        {/* Help Drawer - data-driven, with TOC, search, and real content */}
        <HelpDrawer isOpen={helpOpen} onClose={closeHelp} version={appVersion} />

        {/* Profile Management Modal (#754) */}
        {profilesOpen ? <ProfileManagement onClose={closeProfiles} /> : null}

        {/* FAB - Run All Tests - positioned bottom-right */}
        <div className="fixed bottom-0 right-0 pointer-events-none z-50">
          <Fab className="pointer-events-auto absolute bottom-20 right-6" />
        </div>

        {/* Command palette (Cmd+K / Ctrl+K) */}
        <CommandPalette
          groups={navGroups}
          open={paletteOpen}
          onOpenChange={setPaletteOpen}
          onOpenSettings={openSettings}
          onOpenHelp={openHelp}
          onToggleTheme={toggleTheme}
          isDark={isDark}
        />
      </AppContext.Provider>
    </BrowserRouter>
  );
}
