/**
 * The Card grid archetype's whole reason to exist is conditional membership:
 * a card that is not on the page is either something the subject cannot
 * produce — which has to say so — or something there is nothing to report
 * about, which stays quiet. These lock that distinction, because the failure
 * it prevents (a page that omits what it cannot measure and so looks fine) is
 * invisible by construction.
 */
import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { CardAbsent, CardGrid, CardSlot } from './CardGrid';

describe('CardSlot', () => {
  it('renders the card when it belongs on the page', () => {
    render(
      <CardGrid>
        <CardSlot present={true} absence="quiet">
          <p>Link is up</p>
        </CardSlot>
      </CardGrid>,
    );
    expect(screen.getByText('Link is up')).toBeInTheDocument();
  });

  it('says why a card the subject cannot produce is missing', () => {
    render(
      <CardGrid>
        <CardSlot
          present={false}
          absence={{ label: 'Wired link', reason: 'This interface is wireless.' }}
        >
          <p>Link is up</p>
        </CardSlot>
      </CardGrid>,
    );
    expect(screen.queryByText('Link is up')).not.toBeInTheDocument();
    expect(screen.getByTestId('card-absent-wired-link')).toHaveTextContent(
      'This interface is wireless.',
    );
  });

  it('stays quiet when there is simply nothing to report', () => {
    const { container } = render(
      <CardGrid>
        <CardSlot present={false} absence="quiet">
          <p>Cable fault</p>
        </CardSlot>
      </CardGrid>,
    );
    expect(screen.queryByText('Cable fault')).not.toBeInTheDocument();
    /* Quiet means gone, not an empty frame: a placeholder in the grid reads
       as a card that failed to load. */
    expect(container.querySelector('[data-testid^="card-absent-"]')).toBeNull();
  });
});

describe('CardAbsent', () => {
  it('is available on its own, for a card withheld by something with no children to guard', () => {
    render(<CardAbsent label="Roam analysis" reason="Available on Seed Pro." />);
    const note = screen.getByTestId('card-absent-roam-analysis');
    expect(note).toHaveTextContent('Roam analysis');
    expect(note).toHaveTextContent('Available on Seed Pro.');
  });
});
