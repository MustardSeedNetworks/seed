/** Contextual, translated help for the current Seed page. */

import { type ReactElement, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { FormField } from '../../shared/FormField';
import { RightDrawer } from '../../shared/RightDrawer';
import { SearchInput } from '../../shared/SearchInput';
import { Select } from '../../shared/Select';
import { HelpSectionBody } from './HelpSectionBody';
import { sectionSearchText } from './helpModel';
import { helpSections } from './helpSections';

interface HelpDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  /** Application version from backend, shown in the header. */
  version?: string;
  /**
   * Section this open was requested on — supplied by a page header's (?).
   * The shell clears it on close, so clicking (?) always lands on the page's
   * own section even if the reader browsed elsewhere during the last open.
   */
  section?: string;
}

export function HelpDrawer({
  isOpen,
  onClose,
  version,
  section,
}: HelpDrawerProps): ReactElement | null {
  const { t } = useTranslation(['help', 'cards', 'pages', 'common']);
  const [activeSection, setActiveSection] = useState<string>(section ?? 'about');
  const [requestedSection, setRequestedSection] = useState(section);
  const [wasOpen, setWasOpen] = useState(isOpen);
  const [searchQuery, setSearchQuery] = useState('');

  if (section !== requestedSection || isOpen !== wasOpen) {
    setRequestedSection(section);
    setWasOpen(isOpen);
    setSearchQuery('');
    if (section) {
      setActiveSection(section);
    }
  }

  const query = searchQuery.trim().toLowerCase();

  // Filter the TOC: a section matches on its translated title, its id, or any
  // of its rendered body text / keywords.
  const filteredSections = query
    ? helpSections.filter((candidate) => {
        const title = t(candidate.titleKey).toLowerCase();
        return (
          title.includes(query) ||
          candidate.id.toLowerCase().includes(query) ||
          sectionSearchText(candidate, t).includes(query)
        );
      })
    : helpSections;

  // Keep the active section valid as the filter narrows the list.
  const currentSection =
    filteredSections.find((s) => s.id === activeSection) ??
    filteredSections.at(0) ??
    helpSections.at(0);

  // No sections at all means the drawer has nothing to show. Rendering it
  // anyway produced a header for a section that does not exist.
  if (!(isOpen && currentSection)) {
    return null;
  }

  return (
    <RightDrawer
      open={isOpen}
      onClose={onClose}
      title={t('modal.title')}
      closeLabel={t('modal.closeHelp')}
      data-testid="help-drawer"
    >
      <div className="space-y-4">
        {version ? (
          <p className="text-xs text-text-muted">
            {version.startsWith('v') ? version : `v${version}`}
          </p>
        ) : null}

        <SearchInput
          value={searchQuery}
          onValueChange={setSearchQuery}
          label={t('modal.searchPlaceholder')}
          clearLabel={t('modal.clearSearch')}
          placeholder={t('modal.searchPlaceholder')}
        />

        <FormField label={t('modal.contents')}>
          {(control) => (
            <Select
              {...control}
              data-testid="help-section-select"
              value={currentSection.id}
              onValueChange={setActiveSection}
              options={(filteredSections.length > 0 ? filteredSections : helpSections).map(
                (entry) => ({ value: entry.id, label: t(entry.titleKey) }),
              )}
            />
          )}
        </FormField>

        <div
          // Reset scroll on reopen: keying the pane to the active section
          // remounts it, so it always starts at the top.
          key={currentSection.id}
          data-testid="help-drawer-content"
        >
          <h3 className="mb-2 text-base font-semibold text-text-primary">
            {t(currentSection.titleKey)}
          </h3>
          <a
            href="#help-section-select"
            data-testid="help-return-to-select"
            className="mb-3 inline-block text-sm text-brand-primary-strong underline"
          >
            {t('modal.backToSections')}
          </a>
          <HelpSectionBody blocks={currentSection.blocks} />
        </div>
      </div>
    </RightDrawer>
  );
}

export default HelpDrawer;
