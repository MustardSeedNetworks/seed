/**
 * DeviceConfigsPreview — the sample body <GatedPreview> shows for
 * /device-configs: the real DiffView over a fixture diff typed by the wire
 * type, so a shape change breaks the build rather than the sample.
 */

import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';
import { DiffView } from '../../pages/deviceConfigs/DiffView';
import type { DeviceConfigDiff } from '../../types/deviceConfigs';
import { DetailFacts, DetailPane } from '../../ui/ListDetail';

const SAMPLE: DeviceConfigDiff = {
  targetId: 'dct-000000000001',
  backupId: 'dcb-000000000002',
  takenAt: '2026-03-04T09:00:00Z',
  previousBackupId: 'dcb-000000000001',
  previousTakenAt: '2026-03-03T09:00:00Z',
  added: 2,
  removed: 1,
  unified: [
    '--- dcb-000000000001',
    '+++ dcb-000000000002',
    '@@ -12,6 +12,7 @@',
    ' interface GigabitEthernet0/2',
    '  description uplink-core',
    '- switchport access vlan 20',
    '+ switchport access vlan 30',
    '+ spanning-tree portfast',
    ' !',
    ' interface GigabitEthernet0/3',
    '',
  ].join('\n'),
};

export function DeviceConfigsPreview(): JSX.Element {
  const { t } = useTranslation('pages');
  return (
    <DetailPane
      eyebrow={t('deviceConfigs.selectedDevice')}
      title="access-sw-01"
      meta="10.0.20.2:22"
    >
      <DetailFacts
        items={[
          {
            label: t('deviceConfigs.labelChanges'),
            value: t('deviceConfigs.changeCounts', {
              added: SAMPLE.added,
              removed: SAMPLE.removed,
            }),
          },
        ]}
      />
      <DiffView unified={SAMPLE.unified} />
    </DetailPane>
  );
}
