/**
 * BetaBadge — marks a feature surface as a future release.
 *
 * Phase 0 wraps the existing Tag primitive (yellow scheme) so that
 * Phase 2.5 features (`wifi_roam_analysis`, `wifi_association_forensics`)
 * can render as scaffolds with a clear "v1.0 beta" indicator until the
 * Phase 2.5 implementation lands. See
 * msn-docs-internal/01-Strategy/SEED_NMS_EXPANSION.md.
 */
import { Tag } from './Tag';

interface BetaBadgeProps {
  label?: string;
}

export function BetaBadge({ label = 'v1.0 beta' }: BetaBadgeProps) {
  return (
    <span data-testid="beta-badge">
      <Tag colorScheme="yellow">{label}</Tag>
    </span>
  );
}
