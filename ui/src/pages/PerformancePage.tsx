import { Activity } from 'lucide-react';
import { HealthCheckCard } from '../components/cards/HealthCheckCard';
import { PerformanceCard } from '../components/cards/PerformanceCard';
import { useAppContext } from '../contexts/AppContext';
import { layout } from '../styles/theme';
import { Breadcrumbs } from '../ui/Breadcrumbs';
import { PageHeader } from '../ui/PageHeader';

export function PerformancePage() {
  const { loading, isWifi, cards, cardSettings } = useAppContext();

  return (
    <section class="space-y-6">
      <Breadcrumbs />
      <PageHeader
        icon={Activity}
        title="Performance"
        description="Active throughput tests (speedtest, iperf3) and health-check probes."
      />
      <div class={layout.grid.cards}>
        {(!isWifi || cards.wifi) && (
          <>
            <HealthCheckCard loading={loading} />
            {cardSettings.performance.enabled ? (
              <PerformanceCard
                loading={loading}
                runSpeedtestEnabled={
                  cardSettings.performance.speedtest.enabled &&
                  cardSettings.performance.speedtest.autoRunOnLink
                }
                runIperfEnabled={
                  cardSettings.performance.iperf.enabled &&
                  cardSettings.performance.iperf.autoRunOnLink
                }
              />
            ) : null}
          </>
        )}
      </div>
    </section>
  );
}
