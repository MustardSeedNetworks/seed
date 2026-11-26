/** The catch-all route: an unknown address says so instead of landing on Link. */
import { SearchX } from 'lucide-react';
import { useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Link } from 'wouter';
import { documentTitle } from '../shared/documentTitle';
import { EmptyState } from '../shared/EmptyState';
import { PageHeader } from '../shared/PageHeader';
import { renderBreadcrumbLink } from '../ui/renderBreadcrumbLink';

export function NotFoundPage() {
  const { t } = useTranslation(['pages', 'common']);

  useEffect(() => {
    document.title = documentTitle(t('pages:notFound.label'), t('common:app.title'));
  }, [t]);

  return (
    <section data-testid="not-found-page">
      <PageHeader
        kicker={t('pages:notFound.kicker')}
        title={t('pages:notFound.label')}
        breadcrumbs={[
          { label: t('common:accessibility.home'), href: '/' },
          { label: t('pages:notFound.label') },
        ]}
        breadcrumbsLabel={t('common:accessibility.breadcrumb')}
        renderBreadcrumbLink={renderBreadcrumbLink}
      />
      <EmptyState
        icon={<SearchX className="h-6 w-6" />}
        title={t('pages:notFound.emptyTitle')}
        hint={t('pages:notFound.emptyHint')}
        action={
          <Link
            to="/link"
            className="inline-flex items-center justify-center rounded-lg border border-surface-border bg-surface-raised px-3 py-1.5 text-sm font-medium text-text-primary hover:bg-surface-hover"
          >
            {t('pages:notFound.back')}
          </Link>
        }
      />
    </section>
  );
}
