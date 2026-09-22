/**
 * PathAnalysisPreview — the sample body <GatedPreview> shows for /path.
 *
 * Renders the real PATH_TIMELINE, which is presentational, over a fixture
 * typed by the very type the timeline consumes, so the sample is the feature's
 * own rendering rather than a drawing of it, and a shape change breaks the
 * build here rather than shipping a stale picture. PathDiscoveryCard itself is not reused: its result
 * only exists after a POST /api/v1/path/path, which is the request
 * `requireFeature` answers with 402 on this tier.
 */

import { useTranslation } from 'react-i18next';
import type { PathResponse } from '../../types';
import { PATH_TIMELINE } from '../cards/PathDiscoveryTimeline';
import { Card } from '../ui/Card';

const SAMPLE_PATH: PathResponse = {
  l2Path: {
    hops: [
      {
        device: 'access-sw-02',
        deviceIp: '10.20.0.12',
        ingressPort: {
          name: 'Gi1/0/14',
          index: 14,
          speed: '1Gbps',
          duplex: 'full',
          vlans: [20],
          isTrunk: false,
          connectedTo: 'ws-4021',
        },
        egressPort: {
          name: 'Gi1/0/48',
          index: 48,
          speed: '10Gbps',
          duplex: 'full',
          vlans: [20, 30],
          isTrunk: true,
          connectedTo: 'core-rtr-01',
        },
        source: 'lldp',
      },
    ],
  },
  l3Path: {
    target: 'app.example.internal',
    targetIp: '203.0.113.24',
    protocol: 'icmp',
    hops: [
      { ttl: 1, ip: '10.20.0.1', hostname: 'core-rtr-01', rtt: 0.8, state: 'reply' },
      { ttl: 2, ip: '198.51.100.9', hostname: 'edge-rtr-01', rtt: 3.4, state: 'reply' },
      { ttl: 3, ip: '203.0.113.24', hostname: 'app.example.internal', rtt: 11.7, state: 'reply' },
    ],
    completed: true,
  },
};

const SAMPLE_MAX_RTT = 11.7;

export function PathAnalysisPreview(): React.ReactElement {
  const { t } = useTranslation('cards');

  return (
    <Card title={t('pathDiscovery.title')} status="success">
      <PATH_TIMELINE
        result={SAMPLE_PATH}
        maxRtt={SAMPLE_MAX_RTT}
        expandedL2Hop={null}
        onToggleL2Hop={(): void => undefined}
        t={t}
      />
    </Card>
  );
}
