import { Network } from 'lucide-react';
import { CableCard } from '../components/cards/CableCard';
import { LinkCard } from '../components/cards/LinkCard';
import { WiFiCard } from '../components/cards/WiFiCard';
import { useAppContext } from '../contexts/AppContext';
import { layout } from '../styles/theme';
import { Breadcrumbs } from '../ui/Breadcrumbs';
import { PageHeader } from '../ui/PageHeader';

export function LinkPage() {
  const { cards, loading, isWifi, displayOptions } = useAppContext();

  return (
    <section className="stack-xl">
      <Breadcrumbs />
      <PageHeader
        icon={Network}
        title="Link"
        description="Physical link state, cable diagnostics, and Wi-Fi association for the active interface."
        iconColorClass="text-module-sap"
      />
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
    </section>
  );
}
