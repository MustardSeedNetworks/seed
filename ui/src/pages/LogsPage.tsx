import { LogViewerCard } from '../components/cards/LogViewerCard';
import { SystemHealthCard } from '../components/cards/SystemHealthCard';
import { layout } from '../styles/theme';

export function LogsPage() {
  return (
    <>
      <div className={layout.grid.cards}>
        <SystemHealthCard />
      </div>
      <LogViewerCard />
    </>
  );
}
