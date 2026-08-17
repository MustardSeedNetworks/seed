import { CableCard } from '../components/cards/CableCard';
import { LinkCard } from '../components/cards/LinkCard';
import { WiFiCard } from '../components/cards/WiFiCard';
import { useAppContext } from '../contexts/AppContext';
import { layout } from '../styles/theme';

export function LinkPage() {
  const { cards, loading, isWifi, displayOptions } = useAppContext();

  return (
    <div className={layout.grid.cards}>
      {isWifi ? <WiFiCard data={cards.wifi} loading={loading} visible={true} /> : null}
      {!isWifi && (
        <>
          <LinkCard data={cards.link} loading={loading} />
          {cards.link && cards.link.linkUp === false ? (
            <CableCard
              data={cards.cable}
              loading={loading}
              unitSystem={displayOptions.unitSystem}
            />
          ) : null}
        </>
      )}
    </div>
  );
}
