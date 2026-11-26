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

import type { LucideIcon } from 'lucide-react';
import { createElement, type ReactNode, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useLocation } from 'wouter';
import msnLogo from '../assets/msn-logo.svg';
import { SeedLogo } from '../components/app/SeedLogo';
import { iconSizes } from '../constants/sizes';
import { accessibleNameIncluding } from '../lib/accessibleName';
import { RailFooter, type RailFooterProps } from '../shared/RailFooter';
import { Tooltip } from '../shared/Tooltip';
import { prefetchRoute } from '../utils/prefetch';
import { safeGetItem, safeSetItem } from '../utils/storage';
import { MobileTopBar } from './MobileTopBar';
import { ProductTitle } from './ProductTitle';

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

/** What the rail footer needs from the product; the layout adds collapse, mark and variant. */
export interface SidebarFooterConfig {
  status?: RailFooterProps['status'];
  productSlot?: RailFooterProps['productSlot'];
  help: RailFooterProps['help'];
  settings: RailFooterProps['settings'];
  theme: RailFooterProps['theme'];
  signOut?: RailFooterProps['signOut'];
  version?: RailFooterProps['version'];
}

interface SidebarLayoutProps {
  groups: SidebarNavGroup[];
  children: ReactNode;
  footer: SidebarFooterConfig;
  /**
   * The operator's name for this device (#195), shown under the product
   * name so several open tabs can be told apart. Empty or absent shows
   * nothing.
   */
  deviceName?: string;
}

const STORAGE_KEY = 'seed-sidebar-collapsed';
interface NavItemButtonProps {
  item: SidebarNavItem;
  active: boolean;
  collapsed: boolean;
  onNavigate: (path: string) => void;
}

function badgeClass(badge: string): string {
  const base = 'px-1.5 py-0.5 text-xs rounded font-medium';
  if (badge === 'New') return `${base} bg-status-success/15 text-status-success-strong`;
  if (badge === 'Beta') return `${base} bg-status-warning/15 text-status-warning-strong`;
  return `${base} bg-brand-primary/20 text-brand-primary-strong`;
}

function NavItemButton({ item, active, collapsed, onNavigate }: NavItemButtonProps) {
  return (
    <Tooltip text={collapsed ? item.label : undefined}>
      <button
        data-testid={`sidebar-nav-${item.path.slice(1)}`}
        aria-label={accessibleNameIncluding(
          item.label,
          [item.label, !collapsed ? item.badge : undefined].filter(Boolean).join(' '),
        )}
        type="button"
        onClick={() => onNavigate(item.path)}
        onMouseEnter={() => prefetchRoute(item.path)}
        aria-current={active ? 'page' : undefined}
        /* 44px minimum target, 11px radius, and a 3px left bar for the active route.
           The bar carries the state rather than a gradient fill: a filled row
           competes with status colour, and the rail is chrome. */
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
          // Module accent (M1 follow-up): the icon carries the per-module brand colour
          // the function-first nav moved off the group headers — dimmed at rest,
          // full-strength when active. Items without one keep neutral→brand colouring.
          className: `${iconSizes.lg} flex-shrink-0 ${
            item.accent
              ? active
                ? item.accent
                : `${item.accent} stroke-current/60 group-hover:stroke-current`
              : active
                ? 'text-brand-primary'
                : 'text-text-muted group-hover:text-text-secondary'
          }`,
        })}
        {!collapsed ? (
          <>
            <span className="flex-1 text-left truncate">{item.label}</span>
            {item.badge ? <span className={badgeClass(item.badge)}>{item.badge}</span> : null}
          </>
        ) : null}
      </button>
    </Tooltip>
  );
}

interface SidebarHeaderProps {
  collapsed: boolean;
  deviceName?: string;
}

/** The product mark: which tool this is. Status, collapse and the rest live in the footer. */
function SidebarHeader({ collapsed, deviceName }: SidebarHeaderProps) {
  return (
    <div
      className={`flex items-center ${
        collapsed ? 'justify-center' : ''
      } px-3 py-4 border-b border-surface-border`}
    >
      <div
        className={`flex items-center gap-compact rounded-lg ${collapsed ? 'justify-center' : ''}`}
      >
        <SeedLogo badge badgeClassName="h-9 w-9 shadow-lg" glyphClassName={iconSizes.lg} />
        {!collapsed ? (
          <ProductTitle
            bar="rail"
            deviceName={deviceName}
            className="font-display font-bold text-lg text-text-primary tracking-tight"
          />
        ) : null}
      </div>
    </div>
  );
}

interface SidebarBodyProps {
  groups: SidebarNavGroup[];
  collapsed: boolean;
  deviceName?: string;
  footer: ReactNode;
  onNavigate: (path: string) => void;
  isActive: (path: string) => boolean;
}

function SidebarBody({
  groups,
  collapsed,
  deviceName,
  footer,
  onNavigate,
  isActive,
}: SidebarBodyProps) {
  const { t } = useTranslation();
  // group.label is either a plain display string ("Account") or an
  // i18n key ("common:sections.modules"). t() returns the translation
  // if the key resolves; otherwise the defaultValue (label itself).
  const translateLabel = (label: string): string =>
    label ? t(label, { defaultValue: label }) : '';
  return (
    <>
      <SidebarHeader collapsed={collapsed} deviceName={deviceName} />
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
      {footer}
    </>
  );
}

export function SidebarLayout({ groups, children, footer, deviceName }: SidebarLayoutProps) {
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

  const railFooter = (variant: 'rail' | 'drawer'): ReactNode => (
    <RailFooter
      {...footer}
      collapsed={collapsed}
      variant={variant}
      mark={{ src: msnLogo, name: t('footer.company') }}
      collapse={{
        collapseLabel: t('accessibility.collapseSidebar'),
        expandLabel: t('accessibility.expandSidebar'),
        onToggle: () => setCollapsed(!collapsed),
      }}
    />
  );

  const body = (variant: 'rail' | 'drawer') => (
    <SidebarBody
      groups={groups}
      collapsed={variant === 'rail' && collapsed}
      footer={railFooter(variant)}
      deviceName={deviceName}
      onNavigate={(p) => navigate(p)}
      isActive={isActive}
    />
  );

  return (
    <div className="min-h-screen text-text-primary bg-gradient-to-br from-surface-base via-surface-raised to-surface-deep">
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:fixed focus:top-2 focus:left-2 focus:z-[100] focus:px-4 focus:py-2 focus:rounded-lg focus:bg-brand-primary focus:text-on-brand focus:outline-none"
      >
        {t('accessibility.skipToMainContent')}
      </a>

      <MobileTopBar
        mobileOpen={mobileOpen}
        toggleMobile={() => setMobileOpen(!mobileOpen)}
        deviceName={deviceName}
      />

      {mobileOpen ? (
        <button
          type="button"
          className="lg:hidden fixed inset-0 z-40 bg-scrim/60 backdrop-blur-sm"
          onClick={() => setMobileOpen(false)}
          aria-label={t('accessibility.closeMenu')}
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
        {mobileOpen ? <div className="flex flex-col h-full">{body('drawer')}</div> : null}
      </aside>

      <aside
        data-testid="sidebar-desktop"
        className={`hidden lg:flex fixed top-0 left-0 z-40 h-full flex-col bg-gradient-to-b from-rail-from to-rail-to backdrop-blur-xl border-r border-hairline transition-all duration-300 ease-in-out ${
          collapsed ? 'w-16' : 'w-56' // one step with lg:pl-56 below; was 252 vs 256
        }`}
      >
        {body('rail')}
      </aside>

      <main
        id="main-content"
        className={`transition-all duration-300 ease-in-out pt-16 lg:pt-0 ${
          collapsed ? 'lg:pl-16' : 'lg:pl-56'
        }`}
      >
        <div className="pad">{children}</div>
      </main>
    </div>
  );
}
