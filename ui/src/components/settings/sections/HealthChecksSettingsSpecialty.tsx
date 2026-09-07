/**
 * Specialty protocol endpoint sub-sections of HealthChecksSettings.
 *
 * Composes the streaming, clinical, education and industrial endpoint
 * editors; each child owns its own useArrayItem CRUD helpers so this layer
 * only forwards testsSettings + setter.
 */

import type React from 'react';
import type { JSX } from 'react';
import type { TestsSettings } from '../../../types/settings';
import { ClinicalEndpoints } from './specialty/ClinicalEndpoints';
import { EducationEndpoints } from './specialty/EducationEndpoints';
import { IndustrialEndpoints } from './specialty/IndustrialEndpoints';
import { StreamingEndpoints } from './specialty/StreamingEndpoints';

interface HealthChecksSettingsSpecialtyProps {
  testsSettings: TestsSettings;
  setTestsSettings: React.Dispatch<React.SetStateAction<TestsSettings>>;
}

export function HealthChecksSettingsSpecialty({
  testsSettings,
  setTestsSettings,
}: HealthChecksSettingsSpecialtyProps): JSX.Element {
  return (
    <>
      <StreamingEndpoints testsSettings={testsSettings} setTestsSettings={setTestsSettings} />
      <ClinicalEndpoints testsSettings={testsSettings} setTestsSettings={setTestsSettings} />
      <EducationEndpoints testsSettings={testsSettings} setTestsSettings={setTestsSettings} />
      <IndustrialEndpoints testsSettings={testsSettings} setTestsSettings={setTestsSettings} />
    </>
  );
}
