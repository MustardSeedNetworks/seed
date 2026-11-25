/**
 * PageHeader: the one title block every page opens with.
 *
 * Based on seed's PageHeader (the most complete: kicker, breadcrumbs, readout,
 * help entry) with its i18n, tooltip, router and icon coupling removed. All
 * strings arrive translated.
 *
 * The fleet's 76px title block (the 2026-09-17 density pass) is not a set height;
 * it is what these three lines measure at sm and up: a 10px kicker, a 24px title at
 * leading-tight and a 14px subtitle at leading-relaxed, with 4px between them.
 * Do not add padding, a min-height or a fourth line, and keep the title
 * `sm:text-2xl`: that is the fleet's page-title scale.
 *
 * At 390px the help button stays on the title's line (it sits inside the
 * title row, not in the actions group) and the actions wrap beneath.
 *
 * The help button is 28px; its `after:-inset-2` pseudo-element grows the hit
 * area to 44px for touch (WCAG 2.5.8) and, being absolutely positioned, extends
 * past the button's box. The button carries `data-touch-target-overflow` so a
 * product's density and overflow spec can skip it (and its `::after`) instead of
 * counting a deliberate hit area as content leaving the viewport. The button
 * takes focus on click, which Safari and Firefox on macOS do not do themselves.
 * With the breadcrumb row the header measures 104px, not 76.
 */
import { CircleHelp } from 'lucide-react';
import type { ReactNode, Ref } from 'react';
import { type BreadcrumbItem, type BreadcrumbLinkProps, Breadcrumbs } from './Breadcrumbs';
import { cx } from './cx';

export interface PageHeaderProps {
  /** Product-domain label above the title, e.g. "Test module". Every page has one. */
  kicker: string;
  /** Equal to the page's navigation label. */
  title: string;
  /** The trail. Every page has one; the last item is this page. */
  breadcrumbs: readonly BreadcrumbItem[];
  /** Accessible name of the breadcrumb landmark. */
  breadcrumbsLabel: string;
  renderBreadcrumbLink?: (props: BreadcrumbLinkProps) => ReactNode;
  /** One line. If it needs two, it belongs in the page body. */
  subtitle?: string;
  /**
   * A state readout beside the actions ("Uptime 02:41:19", "64 targets").
   * Prefer a readout to a second button.
   */
  readout?: ReactNode;
  /** The page's primary action, then at most one secondary. */
  actions?: ReactNode;
  /** Renders the help button inline with the title. Omit when the page has no help content. */
  help?: { label: string; onClick: () => void };
  /** The shell's focus target on a route change. */
  titleRef?: Ref<HTMLHeadingElement>;
  className?: string;
}

export function PageHeader({
  kicker,
  title,
  breadcrumbs,
  breadcrumbsLabel,
  renderBreadcrumbLink,
  subtitle,
  readout,
  actions,
  help,
  titleRef,
  className,
}: PageHeaderProps) {
  return (
    <header className={cx('mb-4', className)} data-testid="page-header">
      <Breadcrumbs
        items={breadcrumbs}
        label={breadcrumbsLabel}
        className="mb-1"
        {...(renderBreadcrumbLink ? { renderLink: renderBreadcrumbLink } : {})}
      />
      <div className="flex flex-wrap items-start justify-between gap-x-4 gap-y-3">
        <div className="min-w-0 flex-1">
          <p className="kicker mb-1" data-testid="page-header-eyebrow">
            {kicker}
          </p>
          <div className="flex items-center gap-2">
            <h1
              ref={titleRef}
              tabIndex={-1}
              className="min-w-0 text-xl font-bold leading-tight tracking-tight text-text-primary sm:text-2xl"
              data-testid="page-header-title"
            >
              {title}
            </h1>
            {help ? (
              <button
                type="button"
                onClick={(event) => {
                  // Safari and Firefox on macOS do not focus a button on click; take
                  // focus so the drawer it opens can hand it back on close.
                  event.currentTarget.focus();
                  help.onClick();
                }}
                aria-label={help.label}
                aria-haspopup="dialog"
                data-testid="page-header-help-button"
                data-touch-target-overflow=""
                className="relative inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-text-muted after:absolute after:-inset-2 after:content-[''] hover:bg-surface-hover hover:text-text-primary focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-primary"
              >
                <CircleHelp className="h-5 w-5" aria-hidden="true" />
              </button>
            ) : null}
          </div>
          {subtitle ? (
            <p
              className="mt-1 max-w-2xl text-sm leading-relaxed text-text-secondary"
              data-testid="page-header-description"
            >
              {subtitle}
            </p>
          ) : null}
        </div>
        {readout || actions ? (
          <div className="flex flex-wrap items-center gap-2">
            {readout ? (
              <div
                className="figure text-sm text-text-secondary"
                data-testid="page-header-secondary"
              >
                {readout}
              </div>
            ) : null}
            {actions}
          </div>
        ) : null}
      </div>
    </header>
  );
}
