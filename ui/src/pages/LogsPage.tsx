import { LogViewerCard } from '../components/cards/LogViewerCard';
import { SystemHealthCard } from '../components/cards/SystemHealthCard';
import { CardGrid } from '../ui/CardGrid';

/**
 * Logs — Card grid.
 *
 * No rollup and no conditional membership: both cards always apply, and each
 * reports its own state. The log viewer sits outside the grid because it is
 * full-width by nature — it owns its own filtering and virtualisation, and
 * squeezing it into a grid column would only make it scroll twice.
 */
export function LogsPage() {
  return (
    <>
      <CardGrid>
        <SystemHealthCard />
      </CardGrid>
      <LogViewerCard />
    </>
  );
}
