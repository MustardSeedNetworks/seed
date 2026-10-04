import { useTranslation } from 'react-i18next';
import type { Rule } from '../../types/generated/wifi-anomalies-response';

interface WiFiNeedsCaptureProps {
  rules: Rule[] | undefined;
}

/**
 * WiFiNeedsCapture names the detection rules that cannot run without a
 * monitor-mode capture source, so an empty result is not read as a clean one
 * (seed#2351). The daemon omits the list while capture is active.
 */
export function WiFiNeedsCapture({ rules }: WiFiNeedsCaptureProps) {
  const { t } = useTranslation('pages');
  if (!rules?.length) {
    return null;
  }
  return (
    <div data-testid="wifi-needs-capture" className="stack-2xs">
      <p className="text-xs text-text-muted">{t('wifi.needsCapture')}</p>
      <ul className="text-xs text-text-secondary">
        {rules.map((r) => (
          <li key={r.id} data-testid="wifi-needs-capture-rule">
            {r.title}
          </li>
        ))}
      </ul>
    </div>
  );
}
