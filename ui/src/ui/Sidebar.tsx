/**
 * Sidebar layout shell — persistent collapsible left navigation.
 *
 * Ported from niac to keep the three sibling projects (niac, seed,
 * stem) on the same shell pattern. Theme tokens are adapted to seed's
 * surface/brand palette.
 *
 * Drawer triggers (help, settings, profiles) call up to the host App
 * via the `onOpenHelp` / `onOpenSettings` / `onOpenProfiles` props so
 * the actual drawer components stay mounted at AppShell level where
 * the existing data plumbing lives.
 */
import {
  ChevronLeft,
  ChevronRight,
  HelpCircle,
  type LucideIcon,
  Menu,
  Settings,
  Sprout,
  Users,
  X,
} from 'lucide-react';
import { createElement, type FC, type ReactNode, useEffect, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { iconSizes } from '../constants/sizes';
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
  onOpenHelp?: () => void;
  onOpenSettings?: () => void;
  onOpenProfiles?: () => void;
  topBar?: ReactNode;
}

const STORAGE_KEY = 'seed-sidebar-collapsed';

interface NavItemButtonProps {
  item: SidebarNavItem;
  active: boolean;
  collapsed: boolean;
  onNavigate: (path: string) => void;
}

const NavItemButton: FC<NavItemButtonProps> = ({ item, active, collapsed, onNavigate }) => (
  <button
    type="button"
    onClick={() => onNavigate(item.path)}
    className={`group flex items-center gap-3 w-full px-3 py-2.5 rounded-lg text-sm font-medium transition-all duration-200 ${
      active
        ? 'bg-brand-primary/15 text-text-primary shadow-[inset_0_1px_0_rgba(255,255,255,0.1)]'
        : 'text-text-muted hover:text-text-primary hover:bg-surface-hover'
    }`}
    title={collapsed ? item.label : undefined}
  >
    {createElement(item.icon, {
      className: `${iconSizes.lg} flex-shrink-0 ${
        active ? 'text-brand-primary' : 'text-text-muted group-hover:text-text-secondary'
      }`,
    })}
    {!collapsed ? (
      <>
        <span className="flex-1 text-left truncate">{item.label}</span>
        {item.badge ? (
          <span className="px-1.5 py-0.5 text-xs rounded font-medium bg-brand-primary/20 text-brand-primary">
            {item.badge}
          </span>
        ) : null}
      </>
    ) : null}
  </button>
);

interface SidebarHeaderProps {
  collapsed: boolean;
  onCollapse: () => void;
}

const SidebarHeader: FC<SidebarHeaderProps> = ({ collapsed, onCollapse }) => (
  <div
    className={`flex items-center ${
      collapsed ? 'justify-center' : 'justify-between'
    } px-3 py-4 border-b border-surface-border`}
  >
    <div className={`flex items-center gap-2 ${collapsed ? 'justify-center' : ''}`}>
      <div className="relative flex-shrink-0">
        <div className="h-9 w-9 rounded-lg bg-gradient-to-br from-brand-primary to-brand-accent flex items-center justify-center shadow-lg">
          <Sprout className={`${iconSizes.lg} text-text-inverse`} />
        </div>
        <div className="absolute -top-0.5 -right-0.5 h-2.5 w-2.5 rounded-full bg-status-success border-2 border-surface-raised" />
      </div>
      {!collapsed ? (
        <span className="font-display font-bold text-lg text-text-primary tracking-tight">
          The Seed
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

interface FooterButtonProps {
  collapsed: boolean;
  onClick: () => void;
  icon: LucideIcon;
  label: string;
  title: string;
}

const FooterIconButton: FC<FooterButtonProps> = ({ collapsed, onClick, icon, label, title }) => (
  <button
    type="button"
    onClick={onClick}
    className={`${collapsed ? 'w-full' : 'flex-1'} flex items-center ${
      collapsed ? 'justify-center' : 'gap-2'
    } px-3 py-2 rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover transition-colors text-sm font-medium`}
    title={title}
    aria-label={title}
  >
    {createElement(icon, { className: `${iconSizes.md} flex-shrink-0` })}
    {!collapsed ? <span>{label}</span> : null}
  </button>
);

interface SidebarFooterProps {
  collapsed: boolean;
  version?: string;
  onOpenHelp?: () => void;
  onOpenSettings?: () => void;
  onOpenProfiles?: () => void;
  onExpand: () => void;
}

const SidebarFooter: FC<SidebarFooterProps> = ({
  collapsed,
  version,
  onOpenHelp,
  onOpenSettings,
  onOpenProfiles,
  onExpand,
}) => (
  <div className={`px-3 py-4 border-t border-surface-border ${collapsed ? 'text-center' : ''}`}>
    <div className={`${collapsed ? 'space-y-2' : 'flex items-center gap-2'} mb-3`}>
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

    {onOpenProfiles && !collapsed ? (
      <button
        type="button"
        onClick={onOpenProfiles}
        className="w-full mb-3 flex items-center gap-2 px-3 py-2 rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover transition-colors text-sm font-medium"
        title="Manage profiles"
        aria-label="Manage profiles"
      >
        <Users className={`${iconSizes.md} flex-shrink-0`} />
        <span>Profiles</span>
      </button>
    ) : null}

    {version ? (
      <div
        className={`text-xs font-mono text-text-muted ${
          collapsed ? '' : 'flex items-center justify-between'
        }`}
      >
        {!collapsed ? <span>Version</span> : null}
        <span>{version}</span>
      </div>
    ) : null}
    {collapsed ? (
      <button
        type="button"
        onClick={onExpand}
        className="mt-2 p-1.5 rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover transition-colors"
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
  onOpenProfiles,
}) => (
  <>
    <SidebarHeader collapsed={collapsed} onCollapse={onCollapse} />
    <nav className="flex-1 overflow-y-auto py-4 px-2 space-y-6">
      {groups.map((group) => (
        <div key={group.label}>
          {!collapsed ? (
            <h3 className="px-3 mb-2 text-xs font-semibold text-text-muted uppercase tracking-wider">
              {group.label}
            </h3>
          ) : null}
          {collapsed ? <div className="h-px bg-surface-border mx-2 mb-2" /> : null}
          <div className="space-y-1">
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
      onOpenProfiles={onOpenProfiles}
      onExpand={onExpand}
    />
  </>
);

interface MobileTopBarProps {
  mobileOpen: boolean;
  toggleMobile: () => void;
}

const MobileTopBar: FC<MobileTopBarProps> = ({ mobileOpen, toggleMobile }) => (
  <header className="lg:hidden fixed top-0 left-0 right-0 z-50 flex items-center justify-between px-4 py-3 bg-surface-raised/95 backdrop-blur-xl border-b border-surface-border">
    <div className="flex items-center gap-2">
      <div className="h-8 w-8 rounded-lg bg-gradient-to-br from-brand-primary to-brand-accent flex items-center justify-center">
        <Sprout className={`${iconSizes.md} text-text-inverse`} />
      </div>
      <span className="font-display font-bold text-text-primary">The Seed</span>
    </div>
    <button
      type="button"
      onClick={toggleMobile}
      className="p-2 rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover transition-colors"
      title={mobileOpen ? 'Close menu' : 'Open menu'}
      aria-label={mobileOpen ? 'Close menu' : 'Open menu'}
    >
      {mobileOpen ? <X className={iconSizes.lg} /> : <Menu className={iconSizes.lg} />}
    </button>
  </header>
);

export const SidebarLayout: FC<SidebarLayoutProps> = ({
  groups,
  version,
  children,
  onOpenHelp,
  onOpenSettings,
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
      onOpenProfiles={onOpenProfiles}
    />
  );

  return (
    <div className="min-h-screen text-text-primary font-body">
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:fixed focus:top-2 focus:left-2 focus:z-[100] focus:px-4 focus:py-2 focus:rounded-lg focus:bg-brand-primary focus:text-text-inverse focus:outline-none"
      >
        Skip to main content
      </a>
      <MobileTopBar mobileOpen={mobileOpen} toggleMobile={() => setMobileOpen(!mobileOpen)} />
      {mobileOpen ? (
        <button
          type="button"
          className="lg:hidden fixed inset-0 z-40 bg-black/60 backdrop-blur-sm"
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
        <div className="p-4 sm:p-6 lg:p-8">{children}</div>
      </main>
    </div>
  );
};
