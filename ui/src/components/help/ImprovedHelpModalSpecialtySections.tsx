/**
 * Help modal specialty sections: profiles, WiFi survey, RTSP, DICOM, how-to, glossary, plus inline helper components.
 */

import type React from 'react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { cn, layout, radius, spacing } from '../../styles/theme';
import { Search } from '../ui/icons';

interface TroubleshootingIssue {
  symptom: string;
  causes: string[];
  solutions: string[];
}

function _troubleshootingCategory({
  title,
  issues,
}: {
  title: string;
  issues: TroubleshootingIssue[];
}): React.JSX.Element {
  return (
    <div className={cn(spacing.margin.top.section)}>
      <h4 className={cn('font-semibold text-text-primary', spacing.margin.bottom.content)}>
        {title}
      </h4>
      <div className="stack-lg">
        {issues.map((issue) => (
          <div
            key={issue.symptom}
            className={cn('border border-surface-border', radius.default, spacing.pad.default)}
          >
            <h5 className={cn('font-semibold text-status-warning', spacing.margin.bottom.inline)}>
              {issue.symptom}
            </h5>
            <div className="grid md:grid-cols-2 gap-4 body-small">
              <div>
                <p className="font-semibold text-text-primary mb-1">Possible Causes:</p>
                <ul
                  className={cn(
                    'text-text-secondary',
                    spacing.margin.left.comfortable,
                    'list-disc',
                  )}
                >
                  {issue.causes.map((cause) => (
                    <li key={cause}>{cause}</li>
                  ))}
                </ul>
              </div>
              <div>
                <p className="font-semibold text-text-primary mb-1">Solutions:</p>
                <ul
                  className={cn(
                    'text-text-secondary',
                    spacing.margin.left.comfortable,
                    'list-disc',
                  )}
                >
                  {issue.solutions.map((solution) => (
                    <li key={solution}>{solution}</li>
                  ))}
                </ul>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

// ============================================================================
// HELPER COMPONENTS
// ============================================================================

function _featureCard({
  title,
  description,
}: {
  title: string;
  description: string;
}): React.JSX.Element {
  return (
    <div
      className={cn(
        'bg-surface-hover border border-surface-border',
        radius.lg,
        spacing.pad.default,
      )}
    >
      <h4 className={cn('font-semibold text-text-primary', spacing.margin.bottom.inline)}>
        {title}
      </h4>
      <p className="body-small text-text-secondary">{description}</p>
    </div>
  );
}

function _stepCard({
  number,
  title,
  description,
}: {
  number: number;
  title: string;
  description: string;
}): React.JSX.Element {
  return (
    <div className={cn('flex', spacing.gap.comfortable)}>
      <div
        className={cn(
          'shrink-0 w-8 h-8',
          radius.full,
          'bg-brand-primary text-text-inverse',
          layout.flex.center,
          'font-semibold',
        )}
      >
        {number}
      </div>
      <div className="flex-1">
        <h4 className={cn('font-semibold', spacing.margin.bottom.inline)}>{title}</h4>
        <p className="body-small">{description}</p>
      </div>
    </div>
  );
}

function _helpContentSection({
  title,
  children,
}: {
  title: string;
  children: ReactNode;
}): React.JSX.Element {
  return (
    <div className="max-w-3xl">
      <h3 className={cn('heading-2', spacing.margin.bottom.content)}>{title}</h3>
      {children}
    </div>
  );
}

function _helpTermList({
  items,
}: {
  items: Array<{ term: string; description: string }>;
}): React.JSX.Element {
  return (
    <dl className="stack-lg">
      {items.map((item) => (
        <div
          key={item.term}
          className={cn('border-l-2 border-surface-border', spacing.pad.default)}
        >
          <dt className={cn('font-semibold text-text-primary', spacing.margin.bottom.inline)}>
            {item.term}
          </dt>
          <dd className="body-small text-text-secondary">{item.description}</dd>
        </div>
      ))}
    </dl>
  );
}

// ============================================================================
// NEW FEATURE SECTIONS
// ============================================================================

function _profilesSection(): React.JSX.Element {
  const { t } = useTranslation('help');
  const capabilities = t('content.profiles.capabilities', { returnObjects: true }) as string[];
  const useCases = t('content.profiles.useCases.items', { returnObjects: true }) as Array<{
    name: string;
    description: string;
  }>;

  return (
    <helpContentSection title={t('sections.profiles')}>
      <p className={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
        {t('content.profiles.description')}
      </p>
      <div className={spacing.margin.bottom.section}>
        <h4 className={cn('font-semibold text-text-primary', spacing.margin.bottom.content)}>
          {t('content.profiles.overview.title')}
        </h4>
        <p className="body-small text-text-secondary">{t('content.profiles.overview.content')}</p>
      </div>
      <div className={spacing.margin.bottom.section}>
        <h4 className={cn('font-semibold text-text-primary', spacing.margin.bottom.content)}>
          {t('content.profiles.capabilities_title', 'Profile Capabilities')}
        </h4>
        <ul
          className={cn(
            'body-small text-text-secondary stack-sm',
            spacing.margin.left.spacious,
            'list-disc',
          )}
        >
          {capabilities?.map((cap) => (
            <li key={cap}>{cap}</li>
          ))}
        </ul>
      </div>
      <div>
        <h4 className={cn('font-semibold text-text-primary', spacing.margin.bottom.content)}>
          {t('content.profiles.useCases.title')}
        </h4>
        <div className="stack-lg">
          {useCases?.map((useCase) => (
            <div
              key={useCase.name}
              className={cn('border-l-2 border-brand-primary', spacing.pad.default)}
            >
              <dt className={cn('font-semibold text-text-primary', spacing.margin.bottom.inline)}>
                {useCase.name}
              </dt>
              <dd className="body-small text-text-secondary">{useCase.description}</dd>
            </div>
          ))}
        </div>
      </div>
    </helpContentSection>
  );
}

function _wiFiSurveySection(): React.JSX.Element {
  const { t } = useTranslation('help');
  const visualizations = t('content.wifiSurvey.visualizations', { returnObjects: true }) as Array<{
    type: string;
    description: string;
  }>;
  const bestPractices = t('content.wifiSurvey.bestPractices.items', {
    returnObjects: true,
  }) as string[];

  return (
    <helpContentSection title={t('sections.wifiSurvey')}>
      <p className={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
        {t('content.wifiSurvey.description')}
      </p>
      <helpTermList
        items={[
          {
            term: t('content.wifiSurvey.terms.floorPlan.term'),
            description: t('content.wifiSurvey.terms.floorPlan.description'),
          },
          {
            term: t('content.wifiSurvey.terms.heatmap.term'),
            description: t('content.wifiSurvey.terms.heatmap.description'),
          },
          {
            term: t('content.wifiSurvey.terms.surveyPoint.term'),
            description: t('content.wifiSurvey.terms.surveyPoint.description'),
          },
          {
            term: t('content.wifiSurvey.terms.dataRate.term'),
            description: t('content.wifiSurvey.terms.dataRate.description'),
          },
        ]}
      />
      <div className={spacing.margin.top.section}>
        <h4 className={cn('font-semibold text-text-primary', spacing.margin.bottom.content)}>
          {t('content.wifiSurvey.visualizationsTitle', 'Visualization Modes')}
        </h4>
        <div className="grid md:grid-cols-2 gap-4">
          {visualizations?.map((viz) => (
            <div
              key={viz.type}
              className={cn(
                'bg-surface-hover border border-surface-border',
                radius.default,
                spacing.pad.sm,
              )}
            >
              <h5 className="font-semibold text-text-primary">{viz.type}</h5>
              <p className="body-small text-text-secondary">{viz.description}</p>
            </div>
          ))}
        </div>
      </div>
      <div
        className={cn(
          spacing.margin.top.section,
          'bg-status-info/10 border border-status-info/20',
          radius.default,
          spacing.pad.default,
        )}
      >
        <h4 className={cn('font-semibold text-text-primary', spacing.margin.bottom.inline)}>
          {t('content.wifiSurvey.bestPractices.title')}
        </h4>
        <ul
          className={cn(
            'body-small text-text-secondary stack-sm',
            spacing.margin.left.spacious,
            'list-disc',
          )}
        >
          {bestPractices?.map((practice) => (
            <li key={practice}>{practice}</li>
          ))}
        </ul>
      </div>
    </helpContentSection>
  );
}

function _rtspChecksSection(): React.JSX.Element {
  const { t } = useTranslation('help');
  const configuration = t('content.rtspChecks.configuration', { returnObjects: true }) as Array<{
    field: string;
    description: string;
  }>;

  return (
    <helpContentSection title={t('sections.rtspChecks')}>
      <p className={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
        {t('content.rtspChecks.description')}
      </p>
      <helpTermList
        items={[
          {
            term: t('content.rtspChecks.terms.rtsp.term'),
            description: t('content.rtspChecks.terms.rtsp.description'),
          },
          {
            term: t('content.rtspChecks.terms.options.term'),
            description: t('content.rtspChecks.terms.options.description'),
          },
          {
            term: t('content.rtspChecks.terms.describe.term'),
            description: t('content.rtspChecks.terms.describe.description'),
          },
          {
            term: t('content.rtspChecks.terms.authentication.term'),
            description: t('content.rtspChecks.terms.authentication.description'),
          },
        ]}
      />
      <div className={spacing.margin.top.section}>
        <h4 className={cn('font-semibold text-text-primary', spacing.margin.bottom.content)}>
          {t('content.rtspChecks.configurationTitle', 'Configuration Options')}
        </h4>
        <div className="stack-sm">
          {configuration?.map((config) => (
            <div
              key={config.field}
              className={cn('border-l-2 border-surface-border', spacing.pad.sm)}
            >
              <span className="font-mono text-brand-primary">{config.field}</span>
              <span className="body-small text-text-secondary ml-2">{config.description}</span>
            </div>
          ))}
        </div>
      </div>
    </helpContentSection>
  );
}

function _dicomChecksSection(): React.JSX.Element {
  const { t } = useTranslation('help');
  const configuration = t('content.dicomChecks.configuration', { returnObjects: true }) as Array<{
    field: string;
    description: string;
  }>;
  const commonIssues = t('content.dicomChecks.commonIssues', { returnObjects: true }) as Array<{
    issue: string;
    solution: string;
  }>;

  return (
    <helpContentSection title={t('sections.dicomChecks')}>
      <p className={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
        {t('content.dicomChecks.description')}
      </p>
      <helpTermList
        items={[
          {
            term: t('content.dicomChecks.terms.dicom.term'),
            description: t('content.dicomChecks.terms.dicom.description'),
          },
          {
            term: t('content.dicomChecks.terms.cEcho.term'),
            description: t('content.dicomChecks.terms.cEcho.description'),
          },
          {
            term: t('content.dicomChecks.terms.aeTitle.term'),
            description: t('content.dicomChecks.terms.aeTitle.description'),
          },
          {
            term: t('content.dicomChecks.terms.scp.term'),
            description: t('content.dicomChecks.terms.scp.description'),
          },
          {
            term: t('content.dicomChecks.terms.scu.term'),
            description: t('content.dicomChecks.terms.scu.description'),
          },
        ]}
      />
      <div className={spacing.margin.top.section}>
        <h4 className={cn('font-semibold text-text-primary', spacing.margin.bottom.content)}>
          {t('content.dicomChecks.configurationTitle', 'Configuration')}
        </h4>
        <div className="stack-sm">
          {configuration?.map((config) => (
            <div
              key={config.field}
              className={cn('border-l-2 border-surface-border', spacing.pad.sm)}
            >
              <span className="font-mono text-brand-primary">{config.field}</span>
              <span className="body-small text-text-secondary ml-2">{config.description}</span>
            </div>
          ))}
        </div>
      </div>
      <div
        className={cn(
          spacing.margin.top.section,
          'bg-status-warning/10 border border-status-warning/20',
          radius.default,
          spacing.pad.default,
        )}
      >
        <h4 className={cn('font-semibold text-text-primary', spacing.margin.bottom.content)}>
          {t('content.dicomChecks.commonIssuesTitle', 'Common Issues')}
        </h4>
        <div className="stack-lg">
          {commonIssues?.map((item) => (
            <div key={item.issue}>
              <p className="font-semibold text-status-warning">{item.issue}</p>
              <p className="body-small text-text-secondary">{item.solution}</p>
            </div>
          ))}
        </div>
      </div>
    </helpContentSection>
  );
}

function _howToSection(): React.JSX.Element {
  const { t } = useTranslation('help');
  const guides = t('content.howTo.guides', { returnObjects: true }) as Record<
    string,
    {
      title: string;
      description: string;
      steps: string[];
    }
  >;

  return (
    <helpContentSection title={t('sections.howTo')}>
      <p className={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
        {t('content.howTo.description')}
      </p>
      <div className="stack-xl">
        {guides
          ? Object.entries(guides).map(([key, guide]) => (
              <div
                key={key}
                className={cn('border border-surface-border', radius.lg, spacing.pad.default)}
              >
                <h4 className={cn('font-semibold text-text-primary', spacing.margin.bottom.inline)}>
                  {guide.title}
                </h4>
                <p className={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
                  {guide.description}
                </p>
                <ol
                  className={cn(
                    'body-small text-text-secondary stack-sm',
                    spacing.margin.left.spacious,
                    'list-decimal',
                  )}
                >
                  {guide.steps.map((step) => (
                    <li key={`${key}-${step.slice(0, 50)}`}>{step}</li>
                  ))}
                </ol>
              </div>
            ))
          : null}
      </div>
    </helpContentSection>
  );
}

function _glossarySection(): React.JSX.Element {
  const { t } = useTranslation('glossary');
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedCategory, setSelectedCategory] = useState<string>('all');

  const categories = t('categories', { returnObjects: true }) as Record<string, string>;
  const terms = t('terms', { returnObjects: true }) as Record<
    string,
    {
      term: string;
      fullName: string;
      definition: string;
      category: string;
    }
  >;

  const filteredTerms = terms
    ? Object.entries(terms).filter(([, termData]) => {
        const matchesSearch =
          searchTerm === '' ||
          termData.term.toLowerCase().includes(searchTerm.toLowerCase()) ||
          termData.fullName.toLowerCase().includes(searchTerm.toLowerCase()) ||
          termData.definition.toLowerCase().includes(searchTerm.toLowerCase());

        const matchesCategory =
          selectedCategory === 'all' || termData.category === selectedCategory;

        return matchesSearch && matchesCategory;
      })
    : [];

  return (
    <div className="max-w-3xl">
      <h3 className={cn('heading-2', spacing.margin.bottom.content)}>{t('title')}</h3>
      <p className={cn('body-small text-text-secondary', spacing.margin.bottom.content)}>
        {t('description')}
      </p>
      {/* Search and Filter */}
      <div className={cn('flex flex-wrap gap-4', spacing.margin.bottom.section)}>
        <div className="flex-1 min-w-[200px]">
          <div className="relative">
            <Search
              className={cn('absolute left-3 top-1/2 -translate-y-1/2', 'w-4 h-4 text-text-muted')}
            />
            <input
              type="text"
              placeholder="Search terms..."
              value={searchTerm}
              onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
                setSearchTerm(e.target.value)
              }
              className={cn(
                'w-full pl-9 pr-3 py-2',
                'body-small',
                radius.default,
                'border border-surface-border bg-surface-raised text-text-primary placeholder-text-muted',
                'focus:outline-none focus:ring-2 focus:ring-brand-primary',
              )}
            />
          </div>
        </div>
        <select
          value={selectedCategory}
          onChange={(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>): void =>
            setSelectedCategory(e.target.value)
          }
          className={cn(
            'px-3 py-2',
            'body-small',
            radius.default,
            'border border-surface-border bg-surface-raised text-text-primary',
            'focus:outline-none focus:ring-2 focus:ring-brand-primary',
          )}
        >
          <option value="all">All Categories</option>
          {categories
            ? Object.entries(categories).map(([key, label]) => (
                <option key={key} value={key}>
                  {label}
                </option>
              ))
            : null}
        </select>
      </div>
      {/* Terms List */}
      <div className="stack-lg">
        {filteredTerms.map(([key, termData]) => (
          <div
            key={key}
            className={cn(
              'border border-surface-border',
              radius.default,
              spacing.pad.default,
              'hover:border-brand-primary/50 transition-colors',
            )}
          >
            <div className="flex items-start justify-between gap-4">
              <div className="flex-1">
                <div className="flex items-baseline gap-2 mb-1">
                  <span className="font-bold text-brand-primary">{termData.term}</span>
                  <span className="body-small text-text-muted">({termData.fullName})</span>
                </div>
                <p className="body-small text-text-secondary">{termData.definition}</p>
              </div>
              <span
                className={cn(
                  'px-2 py-0.5 text-xs font-medium',
                  radius.default,
                  'bg-surface-hover text-text-muted capitalize',
                )}
              >
                {categories?.[termData.category] || termData.category}
              </span>
            </div>
          </div>
        ))}

        {filteredTerms.length === 0 && (
          <div className="text-center py-8 text-text-muted">
            No terms found matching your search.
          </div>
        )}
      </div>
    </div>
  );
}
