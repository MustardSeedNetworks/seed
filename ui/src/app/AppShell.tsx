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
import { type ReactNode, Suspense, useEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { Redirect, Route, Switch, useLocation } from 'wouter';
import { AppFooter } from '../components/app/AppFooter';
import { CapabilityWarnings } from '../components/app/CapabilityWarnings';
import { ConnectionNotice } from '../components/app/ConnectionNotice';
import { RailControls } from '../components/app/RailControls';
import { HelpDrawer } from '../components/help/HelpDrawer';
import { ProfileManagement } from '../components/profiles/ProfileManagement';
import { SettingsDrawer } from '../components/settings/SettingsDrawer';
import { CommandPalette } from '../components/ui/CommandPalette';
import { Fab } from '../components/ui/Fab';
import { AppContext, type AppContextValue } from '../contexts/AppContext';
import { useIsPhone } from '../hooks/useIsPhone';
import { useNavGroups } from '../navGroups';
import { type PageConfig, usePages } from '../pageRegistry';
import { cn, section } from '../styles/theme';
import { Breadcrumbs } from '../ui/Breadcrumbs';
import { PageHeader } from '../ui/PageHeader';
import { PageLoader } from '../ui/PageLoader';
import { type RailStatus, SidebarLayout } from '../ui/Sidebar';
import type { AppOrchestration } from './useAppOrchestration';

interface AppShellProps {
  orchestration: AppOrchestration;
  logout: () => void;
}

export function AppShell({ orchestration, logout }: AppShellProps): JSX.Element {
  const navGroups = useNavGroups();
  const pages = usePages();
  const isPhone = useIsPhone();
  const [location] = useLocation();
  const {
    cards,
    loading,
    isWifi,
    currentInterface,
    cardSettings,
    displayOptions,
    networkDiscovery,
    scanError,
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
    hasWifiInterface,
    changeInterface,
    switchToInterfaceType,
    toggleTheme,
    isDark,
    recommendedEthernet,
    profilesOpen,
    settingsOpen,
    helpOpen,
    helpSection,
    openProfiles,
    closeProfiles,
    openSettings,
    closeSettings,
    openHelp,
    closeHelp,
    paletteOpen,
    setPaletteOpen,
  } = orchestration;

  const routePath = location.replace(/\/+$/, '') || '/';
  const previousPath = useRef(routePath);
  useEffect(() => {
    if (routePath !== previousPath.current) {
      previousPath.current = routePath;
      closeHelp();
    }
  }, [routePath, closeHelp]);

  const openPageHelp = (): void =>
    openHelp(pages.find((page) => page.path === routePath)?.help ?? 'link');

  const appContextValue: AppContextValue = {
    cards,
    loading,
    isWifi,
    currentInterface,
    cardSettings,
    displayOptions,
    networkDiscovery,
    scanError,
    triggerDeviceScan,
    registerTraceHopHandler,
    channelGraphData,
    channelGraphLoading,
    appVersion,
    openSettings,
  };

  const { t } = useTranslation();
  const statusLabel = t(`status.${sseStatus}`);
  const railStatus: RailStatus = {
    tone: sseStatus === 'connected' ? 'success' : sseStatus === 'connecting' ? 'warning' : 'error',
    state: sseStatus,
    label: sseStatus === 'connected' ? statusLabel : t('status.clickToReconnect'),
    onActivate: sseStatus === 'connected' ? undefined : reconnect,
  };

  return (
    <AppContext.Provider value={appContextValue}>
      <SidebarLayout
        groups={navGroups}
        version={appVersion}
        onOpenHelp={openPageHelp}
        onOpenSettings={openSettings}
        onOpenProfiles={openProfiles}
        status={railStatus}
        railControls={(collapsed) => (
          <RailControls
            collapsed={collapsed}
            profiles={profiles}
            activeProfile={activeProfile}
            profilesLoading={profilesLoading}
            onProfileSwitch={switchProfile}
            onProfileManage={openProfiles}
            logout={logout}
            interfaces={interfaces}
            currentInterface={currentInterface}
            isWifi={isWifi}
            hasWifiInterface={hasWifiInterface}
            onInterfaceChange={changeInterface}
            switchToInterfaceType={switchToInterfaceType}
            recommendedEthernet={recommendedEthernet}
            toggleTheme={toggleTheme}
            isDark={isDark}
          />
        )}
      >
        <div className={cn(section.width.xl, 'mx-auto')}>
          <ConnectionNotice status={sseStatus} onReconnect={reconnect} />
          <CapabilityWarnings capabilities={capabilities} />

          <Suspense fallback={<PageLoader />}>
            <Switch>
              <Route path="/">
                <Redirect to="/link" replace={true} />
              </Route>
              {pages.map((page) => (
                <Route key={page.path} path={page.path}>
                  <PageWithHeader page={page} onOpenHelp={openHelp}>
                    <page.component />
                  </PageWithHeader>
                </Route>
              ))}
              <Route>
                <Redirect to="/link" replace={true} />
              </Route>
            </Switch>
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
      <HelpDrawer
        isOpen={helpOpen}
        onClose={closeHelp}
        version={appVersion}
        section={helpSection}
      />

      {/* Profile Management Modal (#754) */}
      {profilesOpen ? <ProfileManagement onClose={closeProfiles} /> : null}

      {/* Run All Tests. On a phone this control lives in the page header
          instead (see PageWithHeader): the fixed layer has nowhere to sit at
          390px, because /link's status band occupies y 544-827 of an 844px
          viewport and a bottom-right FAB covers the figures at any offset
          (#2646). Rendered here only above the sm breakpoint, so exactly one
          run control is ever in the tree. */}
      {isPhone ? null : (
        <div className="fixed bottom-0 right-0 pointer-events-none z-50">
          <Fab className="pointer-events-auto absolute bottom-20 right-6" />
        </div>
      )}

      {/* Command palette (Cmd+K / Ctrl+K) */}
      <CommandPalette
        groups={navGroups}
        open={paletteOpen}
        onOpenChange={setPaletteOpen}
        onOpenSettings={openSettings}
        onOpenHelp={openPageHelp}
        onToggleTheme={toggleTheme}
        isDark={isDark}
      />
    </AppContext.Provider>
  );
}

/**
 * PageWithHeader renders the section frame every routed page shares —
 * breadcrumbs plus the page header — from the registry entry rather
 * than from the page body. Pages render only their own content.
 *
 * It also owns `document.title`. Nothing set it per route before, so every
 * page, bookmark and browser-history entry read the bare product name from
 * index.html and a user with several tabs open could not tell them apart
 * (#2645). The title is the registry's label, so it is the same string as the
 * rail item, the breadcrumb and the H1.
 */
function PageWithHeader({
  page,
  onOpenHelp,
  children,
}: {
  page: PageConfig;
  onOpenHelp: (section: string) => void;
  children: ReactNode;
}) {
  const isPhone = useIsPhone();
  const helpSection = page.help;
  const { t } = useTranslation('common');
  const productName = t('app.title');

  useEffect(() => {
    document.title = `${page.label} · ${productName}`;
  }, [page.label, productName]);

  return (
    <section className="stack-xl">
      <Breadcrumbs />
      <PageHeader
        icon={page.icon}
        iconColorClass={page.iconColorClass}
        eyebrow={page.eyebrow}
        title={page.title}
        description={page.description}
        // On a phone the run control is a labelled button here rather than a
        // FAB on the fixed layer, which had nowhere to sit clear of the page's
        // own content (#2646). Above `sm` it stays a FAB, rendered by AppShell.
        actions={isPhone ? <Fab variant="inline" /> : undefined}
        onHelp={helpSection ? () => onOpenHelp(helpSection) : undefined}
      />
      {children}
    </section>
  );
}
