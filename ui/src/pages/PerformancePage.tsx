import { HealthCheckCard } from '../components/cards/HealthCheckCard';
import { PerformanceCard } from '../components/cards/PerformanceCard';
import { useAppContext } from '../contexts/AppContext';
import { layout } from '../styles/theme';

export function PerformancePage() {
  const { loading, isWifi, cards, cardSettings } = useAppContext();

  return (
    <div className={layout.grid.cards}>
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
  );
}
