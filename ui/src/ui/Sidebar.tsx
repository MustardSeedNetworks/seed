/**
 * Sidebar layout shell — persistent collapsible left navigation.
 *
 * Shared shell pattern — kept visually and behaviorally consistent across
 * seed / stem / niac by convention; each repo owns this file independently
 * (no master, no sync). All colors/spacing reference theme tokens;
 * per-product brand identity comes from each repo's index.css token values.
 *
 * Drawer triggers (help, settings, history) call up to the host App
 * via callback props so the actual drawer components stay mounted at
 * AppShell level alongside the existing test/state plumbing.
 */
import {
  ChevronLeft,
  ChevronRight,
  HelpCircle,
  History,
  type LucideIcon,
  Menu,
  Settings,
  Users,
  X,
} from 'lucide-react';
import { createElement, type FC, type ReactNode, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useLocation } from 'wouter';
import { SeedLogo } from '../components/app/SeedLogo';
import { Tooltip } from '../components/ui/Tooltip';
import { iconSizes } from '../constants/sizes';
import { prefetchRoute } from '../utils/prefetch';
import { safeGetItem, safeSetItem } from '../utils/storage';
import { MsnMark } from './MsnMark';

export interface SidebarNavItem {
  path: string;
  label: string;
  icon: LucideIcon;
  badge?: string;
  /**
   * Optional module accent — a semantic text-colour utility (e.g.
   * `text-module-wifi`). Carries the per-module brand identity that the
   * function-first nav (M1) moved off the group headers. Items with no module
   * (e.g. NMS) omit it and fall back to the neutral/brand colouring. Optional
   * so the sibling repos' navGroups stay valid until they adopt it.
   */
  accent?: string;
}

export interface SidebarNavGroup {
  label: string;
  items: SidebarNavItem[];
}

interface SidebarLayoutProps {
  groups: SidebarNavGroup[];
  version?: string;
  children: ReactNode;
  /**
   * Drawer callbacks — all optional. Pass only the ones your product uses;
   * the corresponding footer button only renders when its callback is provided.
   * Stem typically uses help/settings/history; seed uses help/settings/profiles;
   * niac uses help/settings. Add more here if a new product needs another drawer.
   */
  onOpenHelp?: () => void;
  onOpenSettings?: () => void;
  onOpenHistory?: () => void;
  onOpenProfiles?: () => void;
  /**
   * Connection state for the rail's product mark. The dot used to be a
   * hard-coded success green, so it reported "connected" on a dead socket;
   * it now reads the live state and, when that state is not `connected`,
   * reconnects on click.
   */
  status?: RailStatus;
  /**
   * Product-specific shell controls for the rail footer (account, interface,
   * theme). A render prop rather than a node: the panels have to open beside
   * the rail when it is collapsed, and only the rail knows that it is.
   */
  railControls?: (collapsed: boolean) => ReactNode;
}

export interface RailStatus {
  tone: 'success' | 'warning' | 'error';
  /** Machine-readable state, for the accessible name and the E2E assertion. */
  state: string;
  /** The state, in words — the accessible name of the rail's lockup. */
  label: string;
  /** What clicking does, when there is something to do. */
  hint?: string;
  onActivate?: () => void;
}

const STORAGE_KEY = 'stem-sidebar-collapsed';

interface NavItemButtonProps {
  item: SidebarNavItem;
  active: boolean;
  collapsed: boolean;
  onNavigate: (path: string) => void;
}

function badgeClass(badge: string): string {
  if (badge === 'New') return 'bg-status-success/15 text-status-success-strong';
  if (badge === 'Beta') return 'bg-status-warning/15 text-status-warning-strong';
  return 'bg-brand-primary/20 text-brand-primary-strong';
}

const NavItemButton: FC<NavItemButtonProps> = ({ item, active, collapsed, onNavigate }) => (
  <Tooltip text={collapsed ? item.label : undefined}>
    <button
      data-testid={`sidebar-nav-${item.path.slice(1)}`}
      aria-label={item.label}
      type="button"
      onClick={() => onNavigate(item.path)}
      onMouseEnter={() => prefetchRoute(item.path)}
      aria-current={active ? 'page' : undefined}
      /* 44px minimum target, 11px radius, and a 3px left bar for the active
       route. The bar carries the state rather than a gradient fill: a filled
       row competes with status colour, and the rail is chrome. */
      className={`group relative flex items-center gap-default w-full min-h-11 px-3 py-2.5 rounded-[11px] text-sm font-medium transition-all duration-200 ${
        active
          ? 'bg-[color-mix(in_oklab,var(--color-brand-primary)_16%,transparent)] text-text-primary'
          : 'text-text-muted hover:text-text-primary hover:bg-surface-hover'
      }`}
    >
      {active ? (
        <span
          aria-hidden="true"
          className="absolute inset-y-1 left-0 w-[3px] rounded-full bg-brand-primary"
        />
      ) : null}
      {createElement(item.icon, {
        // Module accent (M1 follow-up): the icon carries the per-module brand
        // colour the function-first nav moved off the group headers — dimmed at
        // rest, full-strength when active. Items without a module accent keep the
        // neutral→brand colouring.
        className: `${iconSizes.lg} flex-shrink-0 ${
          item.accent
            ? active
              ? item.accent
              : `${item.accent} opacity-60 group-hover:opacity-100`
            : active
              ? 'text-brand-primary'
              : 'text-text-muted group-hover:text-text-secondary'
        }`,
      })}
      {!collapsed ? (
        <>
          <span className="flex-1 text-left truncate">{item.label}</span>
          {item.badge ? (
            <span className={`px-1.5 py-0.5 text-xs rounded font-medium ${badgeClass(item.badge)}`}>
              {item.badge}
            </span>
          ) : null}
        </>
      ) : null}
    </button>
  </Tooltip>
);

interface FooterIconButtonProps {
  collapsed: boolean;
  onClick: () => void;
  icon: LucideIcon;
  label: string;
  title: string;
  /** E2E hook. The accessible name is no longer unique — the page header's
   *  (?) reads "Open help for <page>", which contains "Open help". */
  testId: string;
}

const FooterIconButton: FC<FooterIconButtonProps> = ({
  collapsed,
  onClick,
  icon,
  label,
  title,
  testId,
}) => (
  <Tooltip text={title}>
    <button
      type="button"
      onClick={onClick}
      data-testid={testId}
      className={`${collapsed ? 'w-full' : 'flex-1'} flex items-center ${
        collapsed ? 'justify-center' : 'gap-compact'
      } px-3 py-row rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover transition-colors text-sm font-medium`}
      aria-label={title}
    >
      {createElement(icon, { className: `${iconSizes.md} flex-shrink-0` })}
      {!collapsed ? <span>{label}</span> : null}
    </button>
  </Tooltip>
);

interface SidebarHeaderProps {
  collapsed: boolean;
  onCollapse: () => void;
  status?: RailStatus;
}

const STATUS_DOT: Record<RailStatus['tone'], string> = {
  success: 'bg-status-success',
  warning: 'bg-status-warning',
  error: 'bg-status-error',
};

const SidebarHeader: FC<SidebarHeaderProps> = ({ collapsed, onCollapse, status }) => {
  const { t } = useTranslation();
  const lockupClass = `flex items-center gap-compact rounded-lg ${
    collapsed ? 'justify-center' : ''
  }`;
  const lockup = (
    <>
      {/* 2px of overhang with nothing to scroll: declared, not tolerated. */}
      <div className="relative flex-shrink-0" data-phone-width-exempt="badge-overhang">
        <SeedLogo badge badgeClassName="h-9 w-9 shadow-lg" glyphClassName={iconSizes.lg} />
        <span
          aria-hidden="true"
          className={`absolute -top-0.5 -right-0.5 h-2.5 w-2.5 rounded-full border-2 border-surface-raised ${
            STATUS_DOT[status?.tone ?? 'success']
          } ${status?.state === 'connecting' ? 'animate-pulse' : ''}`}
        />
      </div>
      {!collapsed ? (
        <span className="font-display font-bold text-lg text-text-primary tracking-tight">
          {t('app.title')}
        </span>
      ) : null}
      {status ? (
        <span className="sr-only">{[status.label, status.hint].filter(Boolean).join(' — ')}</span>
      ) : null}
    </>
  );
  return (
    <div
      className={`flex items-center ${
        collapsed ? 'justify-center' : 'justify-between'
      } px-3 py-4 border-b border-surface-border`}
    >
      {/* The dot is decoration; the state has to be readable. When there is
          something to do about it the lockup is a button that does it, and
          when there is not it stays a live region rather than becoming a
          disabled button — Tab skips those, and hover never fires, so the
          tooltip would not open either. */}
      <Tooltip text={status ? [status.label, status.hint].filter(Boolean).join(' — ') : undefined}>
        {status?.onActivate ? (
          <button
            type="button"
            data-testid="rail-status"
            data-status={status.state}
            onClick={status.onActivate}
            className={`${lockupClass} hover:opacity-80`}
          >
            {lockup}
          </button>
        ) : (
          <div
            data-testid="rail-status"
            data-status={status?.state ?? 'connected'}
            role="status"
            className={lockupClass}
          >
            {lockup}
          </div>
        )}
      </Tooltip>
      {!collapsed ? (
        <Tooltip text={t('accessibility.collapseSidebar')}>
          <button
            type="button"
            onClick={onCollapse}
            className="p-1.5 rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover transition-colors lg:flex hidden"
            aria-label={t('accessibility.collapseSidebar')}
          >
            <ChevronLeft className={iconSizes.md} />
          </button>
        </Tooltip>
      ) : null}
    </div>
  );
};

interface SidebarFooterProps {
  collapsed: boolean;
  version?: string;
  railControls?: ReactNode;
  onOpenHelp?: () => void;
  onOpenSettings?: () => void;
  onOpenHistory?: () => void;
  onOpenProfiles?: () => void;
  onExpand: () => void;
}

interface FullWidthDrawerButtonProps {
  onClick: () => void;
  icon: LucideIcon;
  label: string;
  title: string;
}

const FullWidthDrawerButton: FC<FullWidthDrawerButtonProps> = ({ onClick, icon, label, title }) => (
  <Tooltip text={title}>
    <button
      type="button"
      onClick={onClick}
      className="w-full mb-heading flex items-center gap-compact px-3 py-row rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover transition-colors text-sm font-medium"
      aria-label={title}
    >
      {createElement(icon, { className: `${iconSizes.md} flex-shrink-0` })}
      <span>{label}</span>
    </button>
  </Tooltip>
);

const SidebarFooter: FC<SidebarFooterProps> = ({
  collapsed,
  version,
  railControls,
  onOpenHelp,
  onOpenSettings,
  onOpenHistory,
  onOpenProfiles,
  onExpand,
}) => {
  const { t } = useTranslation();
  return (
    <div className={`px-3 py-4 border-t border-surface-border ${collapsed ? 'text-center' : ''}`}>
      {railControls ? (
        <div className={`mb-heading ${collapsed ? '' : 'flex justify-start'}`}>{railControls}</div>
      ) : null}
      <div className={`${collapsed ? 'stack-sm' : 'flex items-center gap-compact'} mb-heading`}>
        {onOpenHelp ? (
          <FooterIconButton
            collapsed={collapsed}
            onClick={() => onOpenHelp()}
            icon={HelpCircle}
            label={t('labels.help')}
            testId="sidebar-help-button"
            title={t('accessibility.openHelp')}
          />
        ) : null}
        {onOpenSettings ? (
          <FooterIconButton
            collapsed={collapsed}
            onClick={onOpenSettings}
            icon={Settings}
            label={t('labels.settings')}
            testId="sidebar-settings-button"
            title={t('accessibility.openSettings')}
          />
        ) : null}
      </div>

      {onOpenHistory && !collapsed ? (
        <FullWidthDrawerButton
          onClick={onOpenHistory}
          icon={History}
          label={t('labels.history')}
          title={t('accessibility.openHistory')}
        />
      ) : null}

      {onOpenProfiles && !collapsed ? (
        <FullWidthDrawerButton
          onClick={onOpenProfiles}
          icon={Users}
          label={t('labels.manageProfiles')}
          title={t('accessibility.manageProfiles')}
        />
      ) : null}

      {version ? (
        <div className={`text-xs font-mono text-text-muted ${collapsed ? '' : 'flex-between'}`}>
          {!collapsed ? <span>{t('labels.version')}</span> : null}
          <span>{version}</span>
        </div>
      ) : null}
      {/* Whose tool this is, under what it is. Quiet by design: the product mark
        at the top of the rail is the one that has to be recognised. */}
      <MsnMark collapsed={collapsed} className="mt-heading" />
      {collapsed ? (
        <Tooltip text={t('accessibility.expandSidebar')}>
          <button
            type="button"
            onClick={onExpand}
            className="mt-inline p-1.5 rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover transition-colors"
            aria-label={t('accessibility.expandSidebar')}
          >
            <ChevronRight className={iconSizes.md} />
          </button>
        </Tooltip>
      ) : null}
    </div>
  );
};

interface SidebarBodyProps {
  groups: SidebarNavGroup[];
  collapsed: boolean;
  version?: string;
  status?: RailStatus;
  railControls?: ReactNode;
  onCollapse: () => void;
  onExpand: () => void;
  onNavigate: (path: string) => void;
  isActive: (path: string) => boolean;
  onOpenHelp?: () => void;
  onOpenSettings?: () => void;
  onOpenHistory?: () => void;
  onOpenProfiles?: () => void;
}

const SidebarBody: FC<SidebarBodyProps> = ({
  groups,
  collapsed,
  version,
  status,
  railControls,
  onCollapse,
  onExpand,
  onNavigate,
  isActive,
  onOpenHelp,
  onOpenSettings,
  onOpenHistory,
  onOpenProfiles,
}) => {
  const { t } = useTranslation();
  // group.label is either a plain display string ("Account") or an
  // i18n key ("common:sections.modules"). t() returns the translation
  // if the key resolves; otherwise the defaultValue (label itself).
  const translateLabel = (label: string): string =>
    label ? t(label, { defaultValue: label }) : '';
  return (
    <>
      <SidebarHeader collapsed={collapsed} onCollapse={onCollapse} status={status} />
      <nav className="flex-1 overflow-y-auto py-4 px-cell stack-xl">
        {groups.map((group, groupIndex) => (
          <div key={group.label || `nav-group-${String(groupIndex)}`}>
            {!collapsed && group.label ? (
              <h3 className="section-title font-semibold px-3 mb-2">
                {translateLabel(group.label)}
              </h3>
            ) : null}
            {collapsed ? <div className="h-px bg-surface-border mx-2 mb-2" /> : null}
            <div className="stack-xs">
              {group.items.map((item) => (
                <NavItemButton
                  key={item.path}
                  item={item}
                  active={isActive(item.path)}
                  collapsed={collapsed}
                  onNavigate={onNavigate}
                />
              ))}
            </div>
          </div>
        ))}
      </nav>
      <SidebarFooter
        collapsed={collapsed}
        version={version}
        railControls={railControls}
        onOpenHelp={onOpenHelp}
        onOpenSettings={onOpenSettings}
        onOpenHistory={onOpenHistory}
        onOpenProfiles={onOpenProfiles}
        onExpand={onExpand}
      />
    </>
  );
};

interface MobileTopBarProps {
  mobileOpen: boolean;
  toggleMobile: () => void;
}

const MobileTopBar: FC<MobileTopBarProps> = ({ mobileOpen, toggleMobile }) => {
  const { t } = useTranslation();
  return (
    <header className="lg:hidden fixed top-0 left-0 right-0 z-50 flex-between px-4 py-row-lg bg-surface-raised/95 backdrop-blur-xl border-b border-surface-border">
      <div className="flex items-center gap-compact">
        <SeedLogo badge badgeClassName="h-8 w-8" glyphClassName={iconSizes.md} />
        <span className="font-display font-bold text-text-primary">{t('app.title')}</span>
      </div>
      <Tooltip text={mobileOpen ? t('accessibility.closeMenu') : t('accessibility.openMenu')}>
        <button
          type="button"
          onClick={toggleMobile}
          data-testid="mobile-menu-toggle"
          className="pad-xs rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover transition-colors"
          aria-label={mobileOpen ? t('accessibility.closeMenu') : t('accessibility.openMenu')}
        >
          {mobileOpen ? <X className={iconSizes.lg} /> : <Menu className={iconSizes.lg} />}
        </button>
      </Tooltip>
    </header>
  );
};

export const SidebarLayout: FC<SidebarLayoutProps> = ({
  groups,
  version,
  children,
  onOpenHelp,
  onOpenSettings,
  onOpenHistory,
  onOpenProfiles,
  status,
  railControls,
}) => {
  const { t } = useTranslation();
  const [location, navigate] = useLocation();
  const [collapsed, setCollapsed] = useState(() => safeGetItem(STORAGE_KEY) === 'true');
  const [mobileOpen, setMobileOpen] = useState(false);

  useEffect(() => {
    safeSetItem(STORAGE_KEY, String(collapsed));
  }, [collapsed]);

  useEffect(() => {
    setMobileOpen(false);
  }, []);

  const isActive = (path: string) =>
    location === path || (path !== '/' && location.startsWith(path));

  const body = (
    <SidebarBody
      groups={groups}
      collapsed={collapsed}
      version={version}
      status={status}
      railControls={railControls?.(collapsed)}
      onCollapse={() => setCollapsed(true)}
      onExpand={() => setCollapsed(false)}
      onNavigate={(p) => navigate(p)}
      isActive={isActive}
      onOpenHelp={onOpenHelp}
      onOpenSettings={onOpenSettings}
      onOpenHistory={onOpenHistory}
      onOpenProfiles={onOpenProfiles}
    />
  );

  return (
    <div className="min-h-screen text-text-primary bg-gradient-to-br from-surface-base via-surface-raised to-surface-deep">
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:fixed focus:top-2 focus:left-2 focus:z-[100] focus:px-4 focus:py-row focus:rounded-lg focus:bg-brand-primary focus:text-on-brand focus:outline-none"
      >
        {t('accessibility.skipToMainContent')}
      </a>

      <MobileTopBar mobileOpen={mobileOpen} toggleMobile={() => setMobileOpen(!mobileOpen)} />

      {mobileOpen ? (
        <button
          type="button"
          className="lg:hidden fixed inset-0 z-40 bg-scrim/60 backdrop-blur-sm"
          onClick={() => setMobileOpen(false)}
          aria-label="Close menu"
        />
      ) : null}

      {/* The phone drawer mounts on open: a closed drawer in the tree keeps its
          buttons in the tab order and a second product mark on screen, which
          "one product mark per screen" (owner 2026-09-15) rules out. Closed, the
          empty panel is `invisible` so it is no landmark parked off-screen. */}
      <aside
        className={`lg:hidden fixed top-0 left-0 z-50 h-full w-72 bg-surface-raised/95 backdrop-blur-xl border-r border-surface-border transform transition-transform duration-300 ease-in-out ${
          mobileOpen ? 'translate-x-0' : '-translate-x-full invisible'
        }`}
      >
        {mobileOpen ? <div className="flex flex-col h-full">{body}</div> : null}
      </aside>

      <aside
        data-testid="sidebar-desktop"
        className={`hidden lg:flex fixed top-0 left-0 z-40 h-full flex-col bg-gradient-to-b from-rail-from to-rail-to backdrop-blur-xl border-r border-hairline transition-all duration-300 ease-in-out ${
          collapsed ? 'w-16' : 'w-56' // one step with lg:pl-56 below; was 252 vs 256
        }`}
      >
        {body}
      </aside>

      <main
        id="main-content"
        className={`transition-all duration-300 ease-in-out pt-16 lg:pt-0 ${
          collapsed ? 'lg:pl-16' : 'lg:pl-56'
        }`}
      >
        <div className="pad sm:pad-lg lg:pad-xl">{children}</div>
      </main>
    </div>
  );
};
