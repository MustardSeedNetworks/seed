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
  Activity,
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
import { useLocation, useNavigate } from 'react-router-dom';
import { iconSizes } from '../constants/sizes';
import { prefetchRoute } from '../utils/prefetch';
import { safeGetItem, safeSetItem } from '../utils/storage';

export interface SidebarNavItem {
  path: string;
  label: string;
  icon: LucideIcon;
  badge?: string;
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
  topBar?: ReactNode;
}

const STORAGE_KEY = 'stem-sidebar-collapsed';

interface NavItemButtonProps {
  item: SidebarNavItem;
  active: boolean;
  collapsed: boolean;
  onNavigate: (path: string) => void;
}

function badgeClass(badge: string): string {
  if (badge === 'New') return 'bg-status-success/20 text-status-success';
  if (badge === 'Beta') return 'bg-status-warning/20 text-status-warning';
  return 'bg-brand-primary/20 text-brand-accent';
}

const NavItemButton: FC<NavItemButtonProps> = ({ item, active, collapsed, onNavigate }) => (
  <button
    type="button"
    onClick={() => onNavigate(item.path)}
    onMouseEnter={() => prefetchRoute(item.path)}
    className={`group flex items-center gap-default w-full px-3 py-2.5 rounded-lg text-sm font-medium transition-all duration-200 ${
      active
        ? 'bg-gradient-to-r from-brand-primary/30 to-brand-primary/20 text-text-primary shadow-[inset_0_1px_0_rgba(255,255,255,0.1)]'
        : 'text-text-muted hover:text-text-primary hover:bg-surface-hover'
    }`}
    title={collapsed ? item.label : undefined}
  >
    {createElement(item.icon, {
      className: `${iconSizes.lg} flex-shrink-0 ${
        active ? 'text-brand-accent' : 'text-text-muted group-hover:text-text-secondary'
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
);

interface FooterIconButtonProps {
  collapsed: boolean;
  onClick: () => void;
  icon: LucideIcon;
  label: string;
  title: string;
}

const FooterIconButton: FC<FooterIconButtonProps> = ({
  collapsed,
  onClick,
  icon,
  label,
  title,
}) => (
  <button
    type="button"
    onClick={onClick}
    className={`${collapsed ? 'w-full' : 'flex-1'} flex items-center ${
      collapsed ? 'justify-center' : 'gap-compact'
    } px-3 py-row rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover transition-colors text-sm font-medium`}
    title={title}
    aria-label={title}
  >
    {createElement(icon, { className: `${iconSizes.md} flex-shrink-0` })}
    {!collapsed ? <span>{label}</span> : null}
  </button>
);

interface SidebarHeaderProps {
  collapsed: boolean;
  onCollapse: () => void;
}

const SidebarHeader: FC<SidebarHeaderProps> = ({ collapsed, onCollapse }) => {
  const { t } = useTranslation();
  return (
    <div
      className={`flex items-center ${
        collapsed ? 'justify-center' : 'justify-between'
      } px-3 py-4 border-b border-surface-border`}
    >
      <div className={`flex items-center gap-compact ${collapsed ? 'justify-center' : ''}`}>
        <div className="relative flex-shrink-0">
          <div className="h-9 w-9 rounded-lg bg-gradient-to-br from-brand-primary to-brand-accent flex-center shadow-lg">
            <Activity className={`${iconSizes.lg} text-text-inverse`} />
          </div>
          <div className="absolute -top-0.5 -right-0.5 h-2.5 w-2.5 rounded-full bg-status-success border-2 border-surface-raised" />
        </div>
        {!collapsed ? (
          <span className="font-display font-bold text-lg text-text-primary tracking-tight">
            {t('app.title')}
          </span>
        ) : null}
      </div>
      {!collapsed ? (
        <button
          type="button"
          onClick={onCollapse}
          className="p-1.5 rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover transition-colors lg:flex hidden"
          title="Collapse sidebar"
          aria-label="Collapse sidebar"
        >
          <ChevronLeft className={iconSizes.md} />
        </button>
      ) : null}
    </div>
  );
};

interface SidebarFooterProps {
  collapsed: boolean;
  version?: string;
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
  <button
    type="button"
    onClick={onClick}
    className="w-full mb-heading flex items-center gap-compact px-3 py-row rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover transition-colors text-sm font-medium"
    title={title}
    aria-label={title}
  >
    {createElement(icon, { className: `${iconSizes.md} flex-shrink-0` })}
    <span>{label}</span>
  </button>
);

const SidebarFooter: FC<SidebarFooterProps> = ({
  collapsed,
  version,
  onOpenHelp,
  onOpenSettings,
  onOpenHistory,
  onOpenProfiles,
  onExpand,
}) => (
  <div className={`px-3 py-4 border-t border-surface-border ${collapsed ? 'text-center' : ''}`}>
    <div className={`${collapsed ? 'stack-sm' : 'flex items-center gap-compact'} mb-heading`}>
      {onOpenHelp ? (
        <FooterIconButton
          collapsed={collapsed}
          onClick={onOpenHelp}
          icon={HelpCircle}
          label="Help"
          title="Open help"
        />
      ) : null}
      {onOpenSettings ? (
        <FooterIconButton
          collapsed={collapsed}
          onClick={onOpenSettings}
          icon={Settings}
          label="Settings"
          title="Open settings"
        />
      ) : null}
    </div>

    {onOpenHistory && !collapsed ? (
      <FullWidthDrawerButton
        onClick={onOpenHistory}
        icon={History}
        label="History"
        title="Open test history"
      />
    ) : null}

    {onOpenProfiles && !collapsed ? (
      <FullWidthDrawerButton
        onClick={onOpenProfiles}
        icon={Users}
        label="Profiles"
        title="Manage profiles"
      />
    ) : null}

    {version ? (
      <div className={`text-xs font-mono text-text-muted ${collapsed ? '' : 'flex-between'}`}>
        {!collapsed ? <span>Version</span> : null}
        <span>{version}</span>
      </div>
    ) : null}
    {collapsed ? (
      <button
        type="button"
        onClick={onExpand}
        className="mt-inline p-1.5 rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover transition-colors"
        title="Expand sidebar"
        aria-label="Expand sidebar"
      >
        <ChevronRight className={iconSizes.md} />
      </button>
    ) : null}
  </div>
);

interface SidebarBodyProps {
  groups: SidebarNavGroup[];
  collapsed: boolean;
  version?: string;
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
      <SidebarHeader collapsed={collapsed} onCollapse={onCollapse} />
      <nav className="flex-1 overflow-y-auto py-4 px-cell stack-xl">
        {groups.map((group, groupIndex) => (
          <div key={group.label || `nav-group-${String(groupIndex)}`}>
            {!collapsed && group.label ? (
              <h3 className="px-3 mb-2 text-xs font-semibold text-text-muted uppercase tracking-wider">
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
        <div className="h-8 w-8 rounded-lg bg-gradient-to-br from-brand-primary to-brand-accent flex-center">
          <Activity className={`${iconSizes.md} text-text-inverse`} />
        </div>
        <span className="font-display font-bold text-text-primary">{t('app.title')}</span>
      </div>
      <button
        type="button"
        onClick={toggleMobile}
        className="pad-xs rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover transition-colors"
        title={mobileOpen ? 'Close menu' : 'Open menu'}
        aria-label={mobileOpen ? 'Close menu' : 'Open menu'}
      >
        {mobileOpen ? <X className={iconSizes.lg} /> : <Menu className={iconSizes.lg} />}
      </button>
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
  topBar,
}) => {
  const location = useLocation();
  const navigate = useNavigate();
  const [collapsed, setCollapsed] = useState(() => safeGetItem(STORAGE_KEY) === 'true');
  const [mobileOpen, setMobileOpen] = useState(false);

  useEffect(() => {
    safeSetItem(STORAGE_KEY, String(collapsed));
  }, [collapsed]);

  useEffect(() => {
    setMobileOpen(false);
  }, []);

  const isActive = (path: string) =>
    location.pathname === path || (path !== '/' && location.pathname.startsWith(path));

  const body = (
    <SidebarBody
      groups={groups}
      collapsed={collapsed}
      version={version}
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
        className="sr-only focus:not-sr-only focus:fixed focus:top-2 focus:left-2 focus:z-[100] focus:px-4 focus:py-row focus:rounded-lg focus:bg-brand-primary focus:text-text-inverse focus:outline-none"
      >
        Skip to main content
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

      <aside
        className={`lg:hidden fixed top-0 left-0 z-50 h-full w-72 bg-surface-raised/95 backdrop-blur-xl border-r border-surface-border transform transition-transform duration-300 ease-in-out ${
          mobileOpen ? 'translate-x-0' : '-translate-x-full'
        }`}
      >
        <div className="flex flex-col h-full">{body}</div>
      </aside>

      <aside
        className={`hidden lg:flex fixed top-0 left-0 z-40 h-full flex-col bg-surface-raised/80 backdrop-blur-xl border-r border-surface-border transition-all duration-300 ease-in-out ${
          collapsed ? 'w-16' : 'w-64'
        }`}
      >
        {body}
      </aside>

      <main
        id="main-content"
        className={`transition-all duration-300 ease-in-out pt-16 lg:pt-0 ${
          collapsed ? 'lg:pl-16' : 'lg:pl-64'
        }`}
      >
        {topBar}
        <div className="pad sm:pad-lg lg:pad-xl">{children}</div>
      </main>
    </div>
  );
};
